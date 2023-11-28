# OpenAPI2Kong support for security directives

This document documents the use of `security` directives within the `deck` subcommand `openapi2kong`. There are 2 sections;

1. [OpenAPI 3 security directives explained](#openapi-3-security-directives-explained)
2. [Conversion logic to Kong plugins](#conversion-logic-to-kong-plugins)

---

# Openapi 3 security directives explained

Within OpenAPI the security directives can be specified on the document root (the `OpenAPI` object). It can also be specified on the `Operation` object, in which case it will override the document level one.
Specifically; it cannot be specified on the `path` object level, the level in between document and operation.

A nice explanation is here: https://swagger.io/docs/specification/authentication/

---

# Conversion logic to Kong plugins

To enable the generation of Kong plugins the `deck` flag `--generate-security` must be specified.

The `securityScheme` object has a `type` property. These are the possible values
and their support for Kong plugins conversions:

Type | supported | Kong plugin
-|-|-
`http`| no |
`apiKey`| no |
`openIdConnect`| yes | `openid-connect` |
`oauth2`| no |

The non-supported types will result in errors when doing a conversion. To ignore those the flag `--ignore-security-errors` can be specified.

## Boolean logic

No boolean AND/OR logic is supported. So a `security` directive can only have 1 `security requirement`, and within that only a single `securityScheme`.
Again; the errors generated can be ignored by specifying the `--ignore-security-errors` flag.

## Extensions

Within a `securityScheme` of type `openIdConnect`, the extension `x-kong-security-openid-connect` can be used to configure the plugin options.
(The name is the plugin-name, prefixed with "`x-kong-security-`")

## Conversion
The following table describes property behaviour:

OpenID Connect plugin | securityScheme | Notes
-|-|-
`config` | `x-kong-security-openid-connect` | The basis configuration is taken from the extension. Defaults to an empty object if omitted.
`config.issuer` | `openIdConnectUrl` |
`config.scopes_required` | | Union of the scopes in the extension, and the scopes listed in the `securityRequirement` scopes array.

Example:
```yaml
security:
  - myOpenId: [ "scope3" ]
components:
  securitySchemes:
    myOpenId:
      type: openIdConnect
      openIdConnectUrl: https://konghq.com/oauth2/.well-known/openid-configuration
      x-kong-security-openid-connect:
        config:
          run_on_preflight: false
          scopes_required: ["scope1", "scope2"]
```

Will result in a plugin entry as follows:
```yaml
plugins:
- name: openid-connect
  config:
    issuer: https://konghq.com/oauth2/.well-known/openid-configuration
    run_on_preflight: false
    scopes_required: ["scope1", "scope2", "scope3"]
```
