// Package postgres implements the notifications module's repository interfaces.
package postgres

import (
	"avex-backend/internal/modules/notifications/port"
	"avex-backend/internal/platform/database"
)

type Repositories struct {
	notifications *NotificationRepository
	preferences   *PreferenceRepository
	outbox        *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		notifications: &NotificationRepository{},
		preferences:   &PreferenceRepository{},
		outbox:        &OutboxRepository{},
	}
}

func (r *Repositories) RepositorySet() port.RepositorySet {
	return port.RepositorySet{
		Notifications: r.notifications,
		Preferences:   r.preferences,
		Outbox:        r.outbox,
	}
}

func toDBTX(exec port.Executor) database.DBTX {
	dbtx, ok := exec.(database.DBTX)
	if !ok {
		panic("postgres: port.Executor does not satisfy database.DBTX")
	}
	return dbtx
}

type scanner interface {
	Scan(dest ...any) error
}

func nilIfEmptyStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
