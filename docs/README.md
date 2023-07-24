# Kong go-apiops documentation

The [go-apiops](https://github.com/Kong/go-apiops) library provides a set of
tools (validation and transformation) for working with API specifications
and [Kong Gateway](https://docs.konghq.com/gateway/latest/) declarative configurations.
Conceptually, these tools are intended to be organized into a pipeline of individual steps
configured for a particular users needs. The overall purpose of the library is to enable users to
build a CI/CD workflow which deliver APIs from specification to deployment. This pipeline design
allows users to customize the delivery of APIs based on their specific needs.

The functionality is integrated into Kong's declarative management CLI,
[deck](https://docs.konghq.com/deck/latest/). The local binary `go-apiops` is only for local testing.

This document contains usage and examples for the current set of tools available, however,
Kong will be expanding the library of available tools leading up to a GA release.

The below examples assume you have installed and are using the `deck` CLI tool.

## Notes
| :exclamation:  Important compatibility notes  :exclamation: |
|:---------------------------|
| The jsonpath library in use has a bug related to the "recursive decent" operator (`..`). Filter expressions following a recursive decent will not succeed unless prefixed with a generic wilcard (`[*]`).<br/><br/>Non-recursive-decent works as expected:<br/>> `$.plugins[?(@.regex_priority>100)]`</br>where recursive-decent will fail:<br/>> `$..plugins[?(@.regex_priority>100)]`</br>The workaround is to add an extra wildcard (`[*]`) before the filter expression like this:<br/>> `$..plugins[*][?(@.regex_priority>100)]`</br>|


## Commands

---
### `openapi2kong`

The `openapi2kong` transformation is used to convert an OpenAPI Specification (OAS) to a
Kong declarative configuration which can be further used with `deck` to configure a Kong Gateway.
[OpenAPI Specifications](https://swagger.io/specification/) allow you to define language-agnostic
interfaces to your services. The `openapi2kong` tool allows conversion of those specifications
directly into Kong Gateway declarative configurations and includes support for Kong extensions (`x-kong`).
For details on the format and conversion features, see the included [annotated example file](learnservice_oas.yaml).

For full usage instructions, see the command help:

```sh
deck file openapi2kong --help
```

The general pattern for this command is to provide an OAS file and output to a deck file:

```sh
deck file openapi2kong --spec <input-oas-file> --output-file <output-deck-file>
```
---
### `merge`

The `merge` transformation will merge 2 or more Kong Declarative configurations into a single output.

For full usage instructions, see the the command help:

```
deck file merge --help
```

An example of where `merge` will be useful is when you have independent development teams building
APIs which need to be served from a unified Kong Gateway instance. A central job could `merge` the
configurations from the two teams into one before deploying onto the gateway.

`merge` combines all the top-level objects in the input files and the files are processed in the
order the transformation receives them (last file wins). This is **not a "deep merge"**.
For example, with the following two files:

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
deck file merge merge-1.yml merge-2.yml
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

---
### `patch`

The `patch` transformation is used to apply a partial update to a Kong Declarative
configuration using a [JSONPath](https://goessner.net/articles/JsonPath/) selector syntax. There
are many useful use cases for `patch`. One example might be when you have a central team responsible
for applying standards to Kong Gateway configurations, independent of "upstream" developer teams.
The developer teams provide the OAS, and the central team "patches" the gateway configuration
with company standard security plugins.

The `patch` command supports the ability to apply a patch using only command line flags or
with 'patch-files'. For full usage instructions, see the the command help:

```
deck file patch --help
```

For example, to update the `read_timeout` for _all_ services in a given configuration, you could use the following command:

```
deck file patch --state <deck-file> --selector '$..services[*]' --value 'read_timeout: 30000'
```

To accomplish the same with a patch-file, first specify the file:

```yaml
_format_version: 1.0
patches:
  - selectors:
    - $..services[*]
    values:
      read_timeout: 30000
```

And apply it by passing it as an argument to `patch`:

```sh
deck file patch --state <deck-file> <patch-file>
```

`patch` can also be used to _remove_ keys from the output file. For example, if you wish to remove
the `_ignore` key from the root of a file, you can apply the following patch-file:

```yaml
_format_version: "1.0"

patches:
  - selectors:
    - $
    remove:
      - _ignore
```

---
## Example Workflow

### Transform Pipeline

The following example commands assume you are running the CLI from the root of the `go-apiops` project folder.

Convert the provided example OpenAPI Spec to a Kong configuration:

```
deck openapi2kong -s ./docs/mock-a-rena-oas.yml -o ./docs/mock-a-rena-kong.yml
```

The `./docs/mock-a-rena-kong.yml` file now contains a Kong declarative configuration with routes and
services based on the contents of the OpenAPI Specification.

Now, merge the resulting file with the provided sample Kong declarative configuration file:

```
deck merge ./docs/mock-a-rena-kong.yml ./docs/summertime-kong.yml -o ./docs/kong-combined.yml
```

In a seperate step, let's update the `read_timeout` configuration for all the services in the combined file:
```
deck patch -s ./docs/kong-combined.yml --selector '$..services[*]' --value 'read_timeout:30000' --output-file ./docs/kong.yml
```

### Sync to Kong Gateway

To continue with the example you will need:
* `deck`: the Kong declarative management tool: [installation](https://docs.konghq.com/deck/latest/installation/).
* Docker: To run a local Kong Gateway instance: [installation](https://docs.docker.com/get-docker/)

The `./docs/kong.yml` file produced from the pipeline of commands above can be sync'd to Kong using `deck`.
Continuing the above example:

Run a new Kong Gateway in Docker with:

```bash
curl -Ls get.konghq.com/quickstart | bash
```

Then sync the file from the previous `patch` step with:

```bash
deck sync -s ./docs/kong.yml
```

`deck` reports the status of the sync operation:
```
creating service summer-time
creating service mock-a-rena
creating route summer-time_get
creating route mock-a-rena_mock
creating route mock-a-rena_a-rena
Summary:
  Created: 5
  Updated: 0
  Deleted: 0
```

And you can test the entire workflow by routing requests to the local configured gateway:

```
curl -s localhost:8000/mock
```
```
curl -s localhost:8000/a-rena
```
```
curl -s localhost:8000/summer-time
```
