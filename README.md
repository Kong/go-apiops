# go-apiops

Home of Kong's Go based APIOps library.

## What is APIOps

API Lifecycle Automation, or APIOps, is the process of applying API best practices via automation frameworks. This library contains functions to aid the development of tools to apply APIOps to Kong Gateway deployments.

See the [Kong Blog](https://konghq.com/blog/tag/apiops) for more information on APIOps concepts.

## What is this library?

Currently, this library contains functions to convert [OpenAPI Specifications](https://swagger.io/specification/) (OAS) to Kong Gateway deployment formats, including the configuration of Kong Gateway plugins. In the future, this library will contain additional functions to operate over gateway configurations in support of advanced automation workflows. The aim is to provide a library that contains a set of building blocks that can be used to assemble advanced and fully custom automated Kong Gateway deployment pipelines.

## What is the current status of this library?

This library is a public preview project under an [Apache 2.0 license](LICENSE). The library is under heavy development and is not currently supported by Kong Inc. In the future, this library will be tightly integrated into Kong tooling to allow users to apply Kong Gateway based APIOps directly in their deployment pipelines with existing well known command line and CICD tools.

## Usage

The library is under heavy development, and we do not provide API reference documentation. For testing and example usage, the library is released in a temporary CLI named `kced`. The latest release of the CLI can be downloaded for your OS from the [releases page](https://github.com/Kong/go-apiops/releases).

Once you've downloaded and extracted release archive, try running the following:

```bash
cat /path/to/openapi.yaml | kced openapi2kong
```

`kced` will print a Kong Gateway 3.0 compatible decK configuration to `STDOUT`.

An example OAS file is provided at [docs/learnservice_oas.yaml](docs/learnservice_oas.yaml). More documentation and examples will be added in the future.

## Reporting issues

Issues using `kced` or the library can be reported in the [Github repo](https://github.com/Kong/go-apiops/issues).

