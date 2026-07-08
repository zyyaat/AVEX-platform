// Package migrations embeds SQL migration files for all modules.
// Each module's migrations are exposed as a separate embed.FS variable
// so that the migrator can run them independently.
package migrations

import "embed"

// IdentityMigrations embeds all SQL files under migrations/identity/.
// Used by the database migrator to run identity schema migrations.
//
//go:embed identity
var IdentityMigrations embed.FS

// OrdersMigrations embeds all SQL files under migrations/orders/.
// Used by the database migrator to run orders schema migrations.
//
//go:embed orders
var OrdersMigrations embed.FS

// CatalogMigrations embeds all SQL files under migrations/catalog/.
//
//go:embed catalog
var CatalogMigrations embed.FS

// FinancialMigrations embeds all SQL files under migrations/financial/.
//
//go:embed financial
var FinancialMigrations embed.FS
