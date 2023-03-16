package identify

// Note: this file is generated

// strEntityStructure contains a yaml snippet holding the foreign relationships
// between Kong entities.
const strEntityStructure = `

---
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: acls
- TableName: acme_storage
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  - ForeignTable: rbac_users
    LocalField: rbac_user
  TableName: admins
- ForeignRelations:
  - ForeignTable: applications
    LocalField: application
  - ForeignTable: services
    LocalField: service
  TableName: application_instances
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  - ForeignTable: developers
    LocalField: developer
  TableName: applications
- TableName: audit_objects
- TableName: audit_requests
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: basicauth_credentials
- TableName: ca_certificates
- TableName: certificates
- TableName: clustering_data_planes
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  - ForeignTable: consumer_groups
    LocalField: consumer_group
  TableName: consumer_group_consumers
- ForeignRelations:
  - ForeignTable: consumer_groups
    LocalField: consumer_group
  TableName: consumer_group_plugins
- TableName: consumer_groups
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: consumer_reset_secrets
- TableName: consumers
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: credentials
- ForeignRelations:
  - ForeignTable: services
    LocalField: service
  TableName: degraphql_routes
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  - ForeignTable: rbac_users
    LocalField: rbac_user
  TableName: developers
- ForeignRelations:
  - ForeignTable: services
    LocalField: service
  TableName: document_objects
- TableName: event_hooks
- TableName: files
- ForeignRelations:
  - ForeignTable: services
    LocalField: service
  TableName: graphql_ratelimiting_advanced_cost_decoration
- ForeignRelations:
  - ForeignTable: groups
    LocalField: group
  - ForeignTable: rbac_roles
    LocalField: rbac_role
  - ForeignTable: workspaces
    LocalField: workspace
  TableName: group_rbac_roles
- TableName: groups
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: hmacauth_credentials
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: jwt_secrets
- TableName: jwt_signer_jwks
- TableName: key_sets
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: keyauth_credentials
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: keyauth_enc_credentials
- TableName: keyring_keys
- TableName: keyring_meta
- ForeignRelations:
  - ForeignTable: key_sets
    LocalField: set
  TableName: keys
- TableName: konnect_applications
- TableName: legacy_files
- TableName: licenses
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: login_attempts
- ForeignRelations:
  - ForeignTable: ca_certificates
    LocalField: ca_certificate
  - ForeignTable: consumers
    LocalField: consumer
  TableName: mtls_auth_credentials
- ForeignRelations:
  - ForeignTable: oauth2_credentials
    LocalField: credential
  - ForeignTable: services
    LocalField: service
  TableName: oauth2_authorization_codes
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  TableName: oauth2_credentials
- ForeignRelations:
  - ForeignTable: oauth2_credentials
    LocalField: credential
  - ForeignTable: services
    LocalField: service
  TableName: oauth2_tokens
- TableName: parameters
- ForeignRelations:
  - ForeignTable: consumers
    LocalField: consumer
  - ForeignTable: routes
    LocalField: route
  - ForeignTable: services
    LocalField: service
  TableName: plugins
- TableName: ratelimiting_metrics
- ForeignRelations:
  - ForeignTable: rbac_roles
    LocalField: role
  TableName: rbac_role_endpoints
- ForeignRelations:
  - ForeignTable: rbac_roles
    LocalField: role
  TableName: rbac_role_entities
- TableName: rbac_roles
- ForeignRelations:
  - ForeignTable: rbac_roles
    LocalField: role
  - ForeignTable: rbac_users
    LocalField: user
  TableName: rbac_user_roles
- TableName: rbac_users
- ForeignRelations:
  - ForeignTable: services
    LocalField: service
  TableName: routes
- ForeignRelations:
  - ForeignTable: certificates
    LocalField: client_certificate
  TableName: services
- TableName: sessions
- ForeignRelations:
  - ForeignTable: certificates
    LocalField: certificate
  TableName: snis
- TableName: tags
- ForeignRelations:
  - ForeignTable: upstreams
    LocalField: upstream
  TableName: targets
- ForeignRelations:
  - ForeignTable: certificates
    LocalField: client_certificate
  TableName: upstreams
- TableName: vault_auth_vaults
- TableName: vaults
- TableName: workspace_entity_counters
- TableName: workspaces
...

`
