# kceD

Welcome to `kceD`, the APIOps v2 playground CLI. We intend to ship the majority of this functionality in `decK` eventually, but needed a space to try things out as we build functionality.

## Try it out

You can download the latest release for your OS from the [releases page](https://github.com/Kong/kced/releases)

Once you've downloaded and extracted an archive, try running the following:

```bash
kced openapi2kong -i /path/to/openapi.yaml -o ./kong.yaml
cat kong.yaml
```

You'll see that a Kong Gateway 3.0 compatible decK configuration has been generated
