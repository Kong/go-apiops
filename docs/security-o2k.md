# OpenAPI2Kong support for security directives

This document documents the use of `security` directives within the `deck` subcommand `openapi2kong`. There are 2 sections;

1. [OpenAPI 3 security directives explained](#openapi-3-security-directives-explained)
2. [Conversion logic to Kong plugins](#conversion-logic-to-kong-plugins)

---

# Openapi 3 security directives explained

Within OpenAPI the security directives can be specified on the document root (the `OpenAPI` object). It can also be specified on the `Operation` object, in which case it will override the document level one.
Specifically; it cannot be specified on the `path` object level, the level in between document and operation.

## Format

The `security` property is an array of `Security Requirement` objects.

## Behaviour

- The requirements listed are a logical `OR`; passing any one requirement listed is enough to be allowed access.
- The default value is `[]`; an empty array
- An empty array allows access without authentication
- An empty array (on the `Operation` level) can override a non-empty array (document level)

## Security Requirement

A `security requirement` object lists (as properties) `security scheme` objects that MUST all be satisfied to allow access.

Example of a `security` directive listing 2 `security requirements`, each having 2 `security scheme` names :
```yaml
security:
  - myOpenId: [ "scope3" ]
    myBasicAuth: []
  - myBasicAuth: []
    myKeyAuth: []
```
To authenticate a request must either pass:
- `myOpenId` (with 'scope3') **AND** `myBasicAuth`, or
- `myBasicAuth` **AND** `myKeyAuth`

**NOTE**: the values (the arrays), contain scope names to use for `oauth2` and `openIdConnect` security schemes.
(for all other types the array must be empty).

## Security Scheme

The `security scheme` object specifies a single security/authication mechanism to validate. The schemes are defined as properties on the `components.securitySchemes` object.

```yaml
components:
  securitySchemes:
    myBasicAuth:
      type: http
      scheme: basic
    myKeyAuth:
      type: apiKey
      name: apikey
      in: header
    myOpenId:
      type: openIdConnect
      openIdConnectUrl: https://konghq.com/oauth2/.well-known/openid-configuration
```

---

# Conversion logic to Kong plugins

The `securityScheme` object has a `type` property. These are the possible values
and their support for Kong plugins conversions:

Type | supported | Kong plugin
-|-|-
`http`| no |
`apiKey`| no |
`openIdConnect`| yes | `openid-connect` |
`oauth2`| no |

## Boolean logic

No boolean AND/OR logic is supported. So a `security` directive can only have 1 `security requirement`, and with in that only a single `securityScheme`.

## Extensions

Within a `securityScheme` of type `openIdConnect`, the extension `x-kong-security-openid-connect` can be used to configure the plugin options.
(The name is the plugin-name, prefixed with `x-kong-security-`)

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
