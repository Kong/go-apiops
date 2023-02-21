# kceD

Welcome to `kceD`, the APIOps v2 playground CLI. We intend to ship the majority of this functionality in `decK` eventually, but needed a space to try things out as we build functionality.

## What to expect?

Start with nothing, and you might be pleasantly surprised.

## Try it out

You can download the latest release for your OS from the [releases page](https://github.com/Kong/kced/releases)

Once you've downloaded and extracted an archive, try running the following:

```bash
cat /path/to/openapi.yaml | kced openapi2kong
```

You'll see that a Kong Gateway 3.0 compatible decK configuration has been generated.

## Usage

Though initially aimed at OpenAPI conversion, additional commands will be added as we go.

Checkout the CLI usage on the latest release. Try;

```bash
kced --help
```

Online documentation and examples can be found [here](https://github.com/Kong/kced/blob/main/docs).

## Reporting issues

Issues using `kced` can be reported at [its Github repo](https://github.com/Kong/kced/issues).

