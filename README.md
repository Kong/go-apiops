# go-apiops

Home of Kong's Go based APIOps library.

[![Build Status](https://img.shields.io/github/actions/workflow/status/kong/go-apiops/test.yml?branch=main&label=Tests)](https://github.com/kong/go-apiops/actions?query=branch%3Amain+event%3Apush)
[![Lint Status](https://img.shields.io/github/actions/workflow/status/kong/go-apiops/golangci-lint.yml?branch=main&label=Linter)](https://github.com/kong/go-apiops/actions?query=branch%3Amain+event%3Apush)
[![codecov](https://codecov.io/gh/Kong/go-apiops/branch/main/graph/badge.svg?token=8XTDGNP8VW)](https://codecov.io/gh/Kong/go-apiops)
[![Go Report Card](https://goreportcard.com/badge/github.com/kong/go-apiops)](https://goreportcard.com/report/github.com/kong/go-apiops)
[![SemVer](https://img.shields.io/github/v/tag/kong/go-apiops?color=brightgreen&label=SemVer&logo=semver&sort=semver)](https://github.com/Kong/go-apiops/releases)
[![License](https://img.shields.io/github/license/Kong/go-apiops)](LICENSE)

## What is APIOps

API Lifecycle Automation, or APIOps, is the process of applying API best practices via automation frameworks. This library contains functions to aid the development of tools to apply APIOps to Kong Gateway deployments.

See the [Kong Blog](https://konghq.com/blog/tag/apiops) for more information on APIOps concepts.

## What is this library?

The [go-apiops](https://github.com/Kong/go-apiops) library provides a set of tools (validation and transformation) for working with API specifications and [Kong Gateway](https://docs.konghq.com/gateway/latest/) declarative configurations. Conceptually, these tools are intended to be organized into a pipeline of individual steps configured for a particular users needs. The overall purpose of the library is to enable users to build a CI/CD workflow which deliver APIs from specification to deployment. This pipeline design allows users to customize the delivery of APIs based on their specific needs.

## What is the current status of this library?

The library is under heavy development and is a public preview project under an [Apache 2.0 license](LICENSE). The library is not currently supported by Kong Inc. In the future, this library will be tightly integrated into Kong tooling to allow users to apply Kong Gateway based APIOps directly in their deployment pipelines with existing well known command line and CICD tools.

## Installation & Usage

Currently, the functionality is released in a temporary CLI named `kced`. The CLI can be installed locally or ran as a Docker container.

### Local Install

* Download the latest release archive of the CLI for your OS from the [releases page](https://github.com/Kong/go-apiops/releases).
* Once you have downloaded, extract the release archive contents somewhere in your `PATH`, for example:

  ```bash
  tar xvf ~/Downloads/go-apiops_0.1.11_darwin_all.tar.gz -C /tmp
  ```

  And test the installation:

  ```bash
  /tmp/kced version
  ```

  Should print the installed version:

  ```bash
  kceD v0.1.11 (347b296)
  ```

### Docker Install

[Docker images](https://hub.docker.com/r/kong/kced) are available on Docker Hub and can be ran with:

```bash
docker run kong/kced:v0.1.11 version
```

Should result in:

```bash
kceD kong/kced:v0.1.11 (347b2965c3713219aff3844306878ca492e782a2)
```

### Usage

The [Documentation](./docs/README.md) page provides command details and examples. The CLI also provides a `help` command to see usage details on the command line:

```bash
kced help
```

Usage example:

```bash
A temporary CLI that drives the Kong go-apiops library.

go-apiops houses an improved APIOps toolset for operating Kong Gateway deployments.

Usage:
  kced [command]

Available Commands:
  completion   Generate the autocompletion script for the specified shell
  help         Help about any command
  merge        Merges multiple decK files into one
  openapi2kong Convert OpenAPI files to Kong's decK format
  patch        Applies patches on top of a decK file
  version      Print the kceD version

Flags:
  -h, --help   help for kced

Use "kced [command] --help" for more information about a command.
```

## Reporting issues

Issues using `kced` or the library can be reported in the [Github repo](https://github.com/Kong/go-apiops/issues).

## Releasing new versions

The releases are automated. To create a new release:

* tag at the desired place to release

``` bash
git tag vX.Y.Z
```

* push the tag and CI will create a new release

```bash
git push vX.Y.Z
```

* verify the release on [the releases page](https://github.com/Kong/go-apiops/releases), possibly edit the release-notes (which will be generated from the commit history)
