// Package migrations holds the SQL schema files for the registry database
// and each tenant database. Both are embedded at compile time so the backend
// self-migrates on startup without shelling out to psql.
package migrations

import "embed"

//go:embed registry/*.sql
var RegistryFS embed.FS

//go:embed tenant/*.sql
var TenantFS embed.FS
