# go-apiops

Home of Kong's Go based APIOps library.

## What is APIOps

API Lifecycle Automation, or APIOps, is the process of applying API best practices via automation frameworks. This library contains functions to aid the development of tools to apply APIOps to Kong Gateway deployments.

See the [Kong Blog](https://konghq.com/blog/tag/apiops) for more information on APIOps concepts.

## What is this library?

The [go-apiops](https://github.com/Kong/go-apiops) library provides a set of tools (validation and transformation) for working with API specifications and [Kong Gateway](https://docs.konghq.com/gateway/latest/) declarative configurations. Conceptually, these tools are intended to be organized into a pipeline of individual steps configured for a particular users needs. The overall purpose of the library is to enable users to build a CI/CD workflow which deliver APIs from specification to deployment. This pipeline design allows users to customize the delivery of APIs based on their specific needs.

## What is the current status of this library?

This library is a public preview project under an [Apache 2.0 license](LICENSE). The library is under heavy development and is not currently supported by Kong Inc. In the future, this library will be tightly integrated into Kong tooling to allow users to apply Kong Gateway based APIOps directly in their deployment pipelines with existing well known command line and CICD tools.

## Usage

The library is under heavy development, and we do not provide API reference documentation. For testing and example usage, the library is released in a temporary CLI named `kced`. The latest release of the CLI can be downloaded for your OS from the [releases page](https://github.com/Kong/go-apiops/releases) Downloaded and extract the release archive to install.

The [Documentation](./docs/README.md) page provides examples and command details.

## Reporting issues

Issues using `kced` or the library can be reported in the [Github repo](https://github.com/Kong/go-apiops/issues).

