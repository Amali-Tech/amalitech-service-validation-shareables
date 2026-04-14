# AWS Competency Center — Service Validations References

This repository serves as a central reference for the AWS Competency Center's service validation work. It contains:

- **Validation code** — scripts, configurations, and templates used to validate AWS services
- **Architecture references** — diagrams and documentation describing validated architectures

## Folder Structure

The top-level folder structure of this repository must be maintained as follows:

```
README.md
eks/
lambda/
...
... (other service folders)
```

However, the sub-structures within each main folder (such as `eks/`, `lambda/`, etc.) are flexible and can be organized according to the needs of each team or use case. Teams are free to create subfolders and files as required within their respective areas.

Please ensure that the top-level directories remain unchanged to maintain consistency across the repository.

## Contributing

To contribute to this repository:

1. If your AWS service does not already have a top-level directory (e.g., `eks/`, `lambda/`), create one.
2. Organize your service's resources within its directory using a sub-structure that is descriptive, well-structured, and organized. There are no required or recommended subfolder names—choose names and organization that best fit your team's needs and make the contents easy to understand.
3. Do not modify or remove the top-level directories for other services.
4. Follow any repository-wide contribution guidelines (such as PR review, code style, or documentation standards) as applicable.
   s
