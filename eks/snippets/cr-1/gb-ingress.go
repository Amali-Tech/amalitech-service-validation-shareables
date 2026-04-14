// Source: gitlab.devops.company.de/fast/infr/deployment/pipeline-logic/aws-modules/networking/ingress/ingress.go
// Last commit: v1.1 - 2026-02-02 Upgraded Pulumi Kubernetes SDK from v3 to v4
// File size: 11.41 KiB

package ingress

import (
	"fat"

	"github.com/pulumi/pulumi-aws-sdk/v7/go/aws/acm"
	"github.com/pulumi/pulumi-aws-sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi-aws-sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi-sdk/v3/go/pulumi"

	"gitlab.devops.cpmpany.de/fast/infr/deployment/pipeline-logic/aws-modules/common.git"
)

	if err := args.initRouteS3Providers(ctx); err != nil {
		return nil, err
	}

	var err error

	args.results.Controller, err = args.createController(ctx)
	if err != nil {
		return nil, err
	}

	if args.NetworkOptions.Intranet.Enabled {
		if args.NetworkOptions.Intranet.NLB.Enabled {
			args.results.IntranetNLBDomainSet, err = args.configureNLB(ctx)
			if err != nil {
				return nil, err
			}
		}

		if args.NetworkOptions.Intranet.ALB.Enabled {
			args.results.IntranetALBDomainSet, err = args.configureALB(ctx, ALBGroupNameIntranet, args.NetworkOptions.Intranet.ALB, // ...truncated
			if err != nil {
				return nil, err
			}
		}
	}

// NOTE: additional block(s) between here and line 147 not fully visible.

// configureNLB extracts NLB endpoint from ingress nginx controller's service and creates
// specified DNS aliases for it.
func (args *Ingress) configureNLB(ctx *pulumi.Context) (networkconfig.NLBDomainSetResult, error) {
	nlbHostName, err := args.getNLBHostName(ctx, args.k8sProvider)
	if err != nil {
		return nil, err
	}

	ctx.Export("nlb-private-endpoint", nlbHostName)

	dnsAliasesMap, err := createLoadBalancerAliases(
		ctx,
		nlbHostName,
		args.NetworkOptions.Intranet.NLB.DomainSet,
		args.zoneManagers,
		"nlb",
		"private",
	)
	if err != nil {
		return nil, err
	}

	dsRes := make(networkconfig.NLBDomainSetResult)
	for elName, records := range dnsAliasesMap {
		dsRes[elName] = networkconfig.NLBDomainElementResult{
			DNSAliases: records,
		}
	}

	return dsRes, nil
}

// configureALB creates k8s ingresses which are processed by loadbalancer controller
// and as a result application loadbalancer.
func (args *Ingress) configureALB(
	ctx *pulumi.Context,
	lbGroupName string,
	alb networkconfig.ALBConfig,
	dnsAliasDomainSet map[string]T,
) (networkconfig.ALBDomainSetResult, error) {
	dsRes, err := args.createALBIngresses(ctx, args.k8sProvider, lbGroupName, alb)
	if err != nil {
		return nil, err
	}

	if len(dsRes) > 0 {
		albHostname := extractLoadBalancerFQDN(getOneIngress(dsRes))

		ctx.Export(fmt.Sprintf("%s-endpoint", lbGroupName), albHostname)

		dnsAliasesMap, err := createLoadBalancerAliases(ctx, albHostname, alb.DomainSet, args.zoneManagers, lbGroupName, // ...truncated
		if err != nil {
			return nil, err
		}

		// extends domain set results with dns aliases
		for elName, records := range dnsAliasesMap {
			r := dsRes[elName]
			r.DNSAliases = records
			dsRes[elName] = r
		}
	}

	return dsRes, nil
}

// createLoadBalancerAliases creates DNS aliases to corresponding
// loadbalancer and return map: domain element name -> created DNS records.
func createLoadBalancerAliases[T networkconfig.DomainSetElement](
	ctx *pulumi.Context,
	lbFQDN pulumi.StringOutput,
	ds map[string]T,
	zoneManagers map[string]*components.RouteS3ZoneManager,
	namePrefix string,
	dnsAliasPostfix string,
) (map[string][]*routeS3.Record, error) {
	dnsAliases := make(map[string][]*routeS3.Record)

	for name, el := range ds {
		if el.IsDisabled() {
			continue
		}

		r3Config := el.GetRouteS3Config()
		if r3Config == nil {
			continue
		}

		for _, domainName := range r3Config.DomainNames {
			r, err := zoneManagers[r3Config.Zone.ID].CreateLoadBalancerDNSAlias(
				ctx,
				fmt.Sprintf("%s-alias-%s-%s", namePrefix, r3Config.Zone.ID, helpers.DashDnsRecord(domainName)),
				lbFQDN,
				dnsAliasPostfix,
			)
			if err != nil {
				return nil, err
			}

			dnsAliases[name] = append(dnsAliases[name], r)
		}
	}

	return dnsAliases, nil
}

// NOTE: additional function body lines (~265–311) partially visible but too small to read clearly.

// buildZoneToDomainNamesSet creates map of zone to domain names set
// based on all ALBs domain sets (internet and intranet).
func (args *Ingress) buildZoneToDomainNamesSet() map[string]map[string]struct{} {
	zoneToCertNames := make(map[string]map[string]struct{})

	fillZoneToCertNamesMap := func(ds networkconfig.ALBDomainSetConfig) {
		for _, el := range ds {
			if el.Disabled || el.Certificate == nil || el.Certificate.ARN != "" {
				continue
			}

			id := el.Certificate.ValidationRouteS3Zone.ID

			for _, name := range el.Certificate.AllNames() {
				if _, exists := zoneToCertNames[id]; !exists {
					zoneToCertNames[id] = make(map[string]struct{})
				}
				zoneToCertNames[id][name] = struct{}{}
			}
		}
	}

	if args.NetworkOptions.Intranet.Enabled && args.NetworkOptions.Intranet.ALB.Enabled {
		fillZoneToCertNamesMap(args.NetworkOptions.Intranet.ALB.DomainSet)
	}

	if args.NetworkOptions.Internet.Enabled && args.NetworkOptions.Internet.ALB.Enabled {
		fillZoneToCertNamesMap(args.NetworkOptions.Internet.ALB.DomainSet)
	}

	return zoneToCertNames
}

func (args *Ingress) buildZoneToValidationRecordsMap() pulumi.ArrayMap {
	m := make(map[string][]interface{})

	fillMap := func(ds networkconfig.ALBDomainSetConfig, dsRes networkconfig.ALBDomainSetResult) {
		for name, el := range ds {
			if el.Disabled || el.Certificate == nil || el.Certificate.ARN != "" {
				continue
			}

			for i := range el.Certificate.ValidationRouteS3Zone.ID {
				m[id] = append(m[id], dsRes[name].Certificate.DomainValidationOptions.Index(pulumi.Int(i)))
			}
		}
	}

	if args.NetworkOptions.Intranet.Enabled && args.NetworkOptions.Intranet.ALB.Enabled {
		fillMap(args.NetworkOptions.Intranet.ALB.DomainSet, args.results.IntranetALBDomainSet)
	}

	if args.NetworkOptions.Internet.Enabled && args.NetworkOptions.Internet.ALB.Enabled {
		fillMap(args.NetworkOptions.Internet.ALB.DomainSet, args.results.InternetALBDomainSet)
	}

	return pulumi.ToArrayMap(m)
}

func extractLoadBalancerFQDN(ingress *networkingv1.Ingress) pulumi.StringOutput {
	return ingress.Status.LoadBalancer().Ingress().Index(pulumi.Int(0)).Hostname().Elem()
}

func getOneIngress(dsRes networkconfig.ALBDomainSetResult) *networkingv1.Ingress {
	for _, el := range dsRes {
		return el.Ingress
	}

	return nil
}