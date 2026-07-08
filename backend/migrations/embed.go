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

// DispatchMigrations embeds all SQL files under migrations/dispatch/.
//
//go:embed dispatch
var DispatchMigrations embed.FS

// RealtimeMigrations embeds all SQL files under migrations/realtime/.
//
//go:embed realtime
var RealtimeMigrations embed.FS

// NotificationsMigrations embeds all SQL files under migrations/notifications/.
//
//go:embed notifications
var NotificationsMigrations embed.FS

// SupportMigrations embeds all SQL files under migrations/support/.
//
//go:embed support
var SupportMigrations embed.FS
