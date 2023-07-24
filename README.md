# go-apiops

Home of Kong's Go based APIOps library.

[![Build Status](https://img.shields.io/github/actions/workflow/status/kong/go-apiops/test.yml?branch=main&label=Tests)](https://github.com/kong/go-apiops/actions?query=branch%3Amain+event%3Apush)
[![Lint Status](https://img.shields.io/github/actions/workflow/status/kong/go-apiops/golangci-lint.yml?branch=main&label=Linter)](https://github.com/kong/go-apiops/actions?query=branch%3Amain+event%3Apush)
[![codecov](https://codecov.io/gh/Kong/go-apiops/branch/main/graph/badge.svg?token=8XTDGNP8VW)](https://codecov.io/gh/Kong/go-apiops)
[![Go Report Card](https://goreportcard.com/badge/github.com/kong/go-apiops)](https://goreportcard.com/report/github.com/kong/go-apiops)
[![SemVer](https://img.shields.io/github/v/tag/kong/go-apiops?color=brightgreen&label=SemVer&logo=semver&sort=semver)](https://github.com/Kong/go-apiops/releases)
[![License](https://img.shields.io/github/license/Kong/go-apiops)](LICENSE)

## What is APIOps

API Lifecycle Automation, or APIOps, is the process of applying API best practices via automation frameworks.
This library contains functions to aid the development of tools to apply APIOps to Kong Gateway deployments.

See the [Kong Blog](https://konghq.com/blog/tag/apiops) for more information on APIOps concepts.

## What is this library?

The [go-apiops](https://github.com/Kong/go-apiops) library provides a set of tools (generation and transformation)
for working with API specifications and [Kong Gateway](https://docs.konghq.com/gateway/latest/) declarative configurations.
Conceptually, these tools are intended to be organized into a pipeline of individual steps configured for a particular
users needs. The overall purpose of the library is to enable users to build a CI/CD workflow which deliver APIs from
specification to deployment. This pipeline design allows users to customize the delivery of APIs based on their specific needs.

## What is the current status of this library?

The library is an [Apache 2.0 license](LICENSE).
The library functionality will be be made available through
the [deck](https://docs.konghq.com/deck/latest/) cli tool.

## Installation & Usage

Currently, the functionality is released as a library only. The repository contains a CLI named `go-apiops` for local testing.
For general CLI usage, see the [deck](https://docs.konghq.com/deck/latest/) cli tool which exposes the library functions.

### Local Build

* make sure [Go-lang tools](https://go.dev/doc/install) are installed
* Checkout the Git repository (switch to a specific tag to select a version if required)
* use the makefile to build the project via `make build`

## Reporting issues

Issues using the `go-apiops` CLI or the library can be reported in the [Github repo](https://github.com/Kong/go-apiops/issues).

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
