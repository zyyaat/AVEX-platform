// Package postgres implements the support module's repository interfaces.
package postgres

import (
	"avex-backend/internal/modules/support/port"
	"avex-backend/internal/platform/database"
)

type Repositories struct {
	tickets     *TicketRepository
	messages    *MessageRepository
	attachments *AttachmentRepository
	outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		tickets:     &TicketRepository{},
		messages:    &MessageRepository{},
		attachments: &AttachmentRepository{},
		outbox:      &OutboxRepository{},
	}
}

func (r *Repositories) RepositorySet() port.RepositorySet {
	return port.RepositorySet{
		Tickets:     r.tickets,
		Messages:    r.messages,
		Attachments: r.attachments,
		Outbox:      r.outbox,
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
