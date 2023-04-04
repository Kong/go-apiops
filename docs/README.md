# Kong go-apiops documentation

The [go-apiops](https://github.com/Kong/go-apiops) library provides a set of tools (validation and transformation) for working with API specifications and [Kong Gateway](https://docs.konghq.com/gateway/latest/) declarative configurations. Conceptually, these tools are intended to be organized into a pipeline of individual steps configured for a particular users needs. The overall purpose of the library is to enable users to build a CICD workflow which deliver APIs from specification to deployment. This pipline design allows users to customize the delivery of APIs based on their specific needs.

This project is currently in a public beta preview state. As a result, the tools are available in a temporary command line named `kced`. For installation instructions for the `kced` CLI, see the main [README](../README.md). Before these tools are available in a general release, the tools will be integrated into Kong's declarative management CLI, [deck](https://docs.konghq.com/deck/latest/) and the temporary CLI will be deprecated.

This document contains usage and examples for the current set of tools available, however, Kong will be expanding the library of available tools leading up to a GA release.

## Commands

### `openapi2kong`

Convert an OpenAPI Specification (OAS) to a Kong declarative configuration which can be further used with `deck` to configure a Kong Gateway. [OpenAPI Specifications](https://swagger.io/specification/) allow you to define language-agnostic interfaces to your services. The `openapi2kong` tool allows conversion of those specifications directly into Kong Gateway declarative configurations and includes support for Kong extensions (`x-kong`). For details on the format and conversion features, see the included [annotated example file](learnservice_oas.yml).

For full usage instructions, see the the command help:

```
kced openapi2kong --help
```

The general pattern for this command is to provide an OAS file and output to a deck file:

```
./kced openapi2kong --spec <input-oas-file> --output-file <output-deck-file>
```

### `merge`

Merge 2 or more Kong Declarative configurations into a single output.

For full usage instructions, see the the command help:

```
kced merge --help
```

An example of where `merge` will be useful is when you have independent development teams building APIs which need to be served from a unified Kong Gateway instance. A central job could `merge` the configurations from the two teams into one before deploying onto the gateway.

`merge` combines all the top-level objects in the input files and the files are processed in the order the transformation receives them (last file wins). This is **not a "deep merge"**. For example, with the following two files:

`merge-1.yml`:
```yml
a:
  b:
    c: abc
d: [ 1, 2, 3 ]
```

`merge-2.yml`:
```yml
a:
  b:
    c: xyz
d: [ 4, 5, 6 ]
```

```
kced merge merge-1.yml merge-2.yml
```
will result in :
```
a:
  b:
    c: xyz
d:
- 1
- 2
- 3
- 4
- 5
- 6
```


### `patch`

Update values in a Kong Declarative configuration using a JSONPath selector syntax.

An example of where this might be useful is when you have a

For usage instructions, see the the command help:

```
kced patch --help
```

## Example Workflow

The following example commands assume you are running the CLI from the root of the `go-apiops` project folder.

Convert the provided example OpenAPI Spec to a Kong configuration:

```
kced openapi2kong -s ./docs/mock-a-rena-oas.yml -o ./docs/mock-a-rena-kong.yml
```

The `./docs/mock-a-rena-kong.yml` file now contains a Kong declarative configuration with routes and services based on the contents of the OpenAPI Specification.

Now, merge the resulting file with the provided sample Kong declarative configuration file:

```
kced merge ./docs/mock-a-rena-kong.yml ./docs/patch
```

```
kced patch
```

## Sync to Kong Gateway

The files produced by the commands above can be sync'd to Kong using `deck`. Continuing the above example:

Run a new Kong Gateway in Docker with:

```bash
curl -Ls get.konghq.com/quickstart | bash
```

Then sync the file from the previous `patch` step with:

```bash
deck sync -s kong.yml
```

