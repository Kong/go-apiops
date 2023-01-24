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

For CLI usage try;

```bash
kced openapi2kong --help
``` 

An extensively annotated input file describing the many features can be found [here](https://github.com/Kong/fw/blob/main/learnservice_oas.yaml).

## Reporting issues

The `kced` repo is a CLI wrapper around the `fw` library.

- issues with the CLI, report them [as an issue at `kced`](https://github.com/Kong/kced/issues)
- issues with the content generated, report them [as an issue at `fw`](https://github.com/Kong/fw/issues)
- when in doubt use the first option; kced.

## Updating the version of `fw`

1. Commit your changes to the `fw` repo
2. Run `GOPRIVATE=github.com/Kong/fw go get github.com/Kong/fw@<sha>` to update the `kced` dependency
3. Tag a new release of `kced` and the artifacts will be automatically be built

> If you're not using `https` git URLs, you might need to run `git config --global url."git@github.com:Kong/fw".insteadOf "https://github.com/Kong/fw"` before running `go get`

