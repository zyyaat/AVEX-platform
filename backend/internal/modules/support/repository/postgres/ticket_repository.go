// Package postgres ticket_repository: TicketRepository implementation.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"avex-backend/internal/modules/support/domain"
	"avex-backend/internal/modules/support/port"
)

type TicketRepository struct{}

var _ port.TicketRepository = (*TicketRepository)(nil)

func (r *TicketRepository) Create(ctx context.Context, exec port.Executor, t domain.Ticket) error {
	dbtx := toDBTX(exec)
	_, err := dbtx.Exec(ctx, `
		INSERT INTO support.tickets (
			id, ticket_no, user_id, order_id, driver_id, restaurant_id,
			subject, description, category, priority, status,
			assigned_to, created_by, closed_by, closed_reason,
			message_count, first_response_at, resolved_at, closed_at,
			created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22
		)
	`,
		t.ID(), t.TicketNo(), t.UserID(), nilIfEmptyStr(t.OrderID()), nilIfEmptyStr(t.DriverID()), nilIfEmptyStr(t.RestaurantID()),
		t.Subject(), t.Description(), string(t.Category()), string(t.Priority()), string(t.Status()),
		nilIfEmptyStr(t.AssignedTo()), t.CreatedBy(), nilIfEmptyStr(t.ClosedBy()), nilIfEmptyStr(t.ClosedReason()),
		t.MessageCount(), t.FirstResponseAt(), t.ResolvedAt(), t.ClosedAt(),
		t.CreatedAt(), t.UpdatedAt(), t.Version(),
	)
	if err != nil {
		return fmt.Errorf("create ticket: %w", err)
	}
	return nil
}

func (r *TicketRepository) GetByID(ctx context.Context, exec port.Executor, id string) (*domain.Ticket, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support.tickets WHERE id = $1`, id)
	t, err := scanTicket(row)
	if err != nil {
		return nil, mapTicketReadError(err)
	}
	return &t, nil
}

func (r *TicketRepository) GetByTicketNo(ctx context.Context, exec port.Executor, ticketNo string) (*domain.Ticket, error) {
	dbtx := toDBTX(exec)
	row := dbtx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM support.tickets WHERE ticket_no = $1`, ticketNo)
	t, err := scanTicket(row)
	if err != nil {
		return nil, mapTicketReadError(err)
	}
	return &t, nil
}

func (r *TicketRepository) Update(ctx context.Context, exec port.Executor, t domain.Ticket) error {
	dbtx := toDBTX(exec)
	tag, err := dbtx.Exec(ctx, `
		UPDATE support.tickets SET
			subject = $2, description = $3, category = $4, priority = $5, status = $6,
			assigned_to = $7, closed_by = $8, closed_reason = $9,
			message_count = $10, first_response_at = $11, resolved_at = $12, closed_at = $13,
			updated_at = $14, version = version + 1
		WHERE id = $1 AND version = $15
	`,
		t.ID(), t.Subject(), t.Description(), string(t.Category()), string(t.Priority()), string(t.Status()),
		nilIfEmptyStr(t.AssignedTo()), nilIfEmptyStr(t.ClosedBy()), nilIfEmptyStr(t.ClosedReason()),
		t.MessageCount(), t.FirstResponseAt(), t.ResolvedAt(), t.ClosedAt(),
		t.UpdatedAt(), t.Version(),
	)
	if err != nil {
		return fmt.Errorf("update ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: optimistic lock failed for ticket %s", domain.ErrTicketNotFound, t.ID())
	}
	return nil
}

func (r *TicketRepository) listWithFilter(ctx context.Context, exec port.Executor, where string, args []any, page port.PageQuery) (port.Page[domain.Ticket], error) {
	page = page.Normalize()
	dbtx := toDBTX(exec)

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM support.tickets WHERE %s`, where)
	var total int64
	if err := dbtx.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return port.Page[domain.Ticket]{}, fmt.Errorf("count: %w", err)
	}

	listSQL := fmt.Sprintf(`
		SELECT %s FROM support.tickets WHERE %s
		ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, ticketColumns, where, len(args)+1, len(args)+2)
	args = append(args, page.Limit, page.Offset)
	rows, err := dbtx.Query(ctx, listSQL, args...)
	if err != nil {
		return port.Page[domain.Ticket]{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var items []domain.Ticket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return port.Page[domain.Ticket]{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return port.Page[domain.Ticket]{}, fmt.Errorf("rows: %w", err)
	}
	return port.Page[domain.Ticket]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (r *TicketRepository) ListByUser(ctx context.Context, exec port.Executor, userID string, page port.PageQuery) (port.Page[domain.Ticket], error) {
	return r.listWithFilter(ctx, exec, "user_id = $1", []any{userID}, page)
}

func (r *TicketRepository) ListByAgent(ctx context.Context, exec port.Executor, agentID string, page port.PageQuery) (port.Page[domain.Ticket], error) {
	return r.listWithFilter(ctx, exec, "assigned_to = $1", []any{agentID}, page)
}

func (r *TicketRepository) ListByStatus(ctx context.Context, exec port.Executor, status string, page port.PageQuery) (port.Page[domain.Ticket], error) {
	return r.listWithFilter(ctx, exec, "status = $1", []any{status}, page)
}

func (r *TicketRepository) ListAll(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.Ticket], error) {
	return r.listWithFilter(ctx, exec, "1=1", []any{}, page)
}

func (r *TicketRepository) ListUnassigned(ctx context.Context, exec port.Executor, page port.PageQuery) (port.Page[domain.Ticket], error) {
	return r.listWithFilter(ctx, exec, "assigned_to IS NULL AND status = 'open'", []any{}, page)
}

const ticketColumns = `id, ticket_no, user_id, order_id, driver_id, restaurant_id, subject, description, category, priority, status, assigned_to, created_by, closed_by, closed_reason, message_count, first_response_at, resolved_at, closed_at, created_at, updated_at, version`

func scanTicket(s scanner) (domain.Ticket, error) {
	var (
		id, ticketNo, userID, subject, description string
		category, priority, status, createdBy      string
		orderID, driverID, restaurantID            *string
		assignedTo, closedBy, closedReason         *string
		messageCount, version                      int
		firstResponseAt, resolvedAt, closedAt      *time.Time
		createdAt, updatedAt                       time.Time
	)
	if err := s.Scan(
		&id, &ticketNo, &userID, &orderID, &driverID, &restaurantID,
		&subject, &description, &category, &priority, &status,
		&assignedTo, &createdBy, &closedBy, &closedReason,
		&messageCount, &firstResponseAt, &resolvedAt, &closedAt,
		&createdAt, &updatedAt, &version,
	); err != nil {
		return domain.Ticket{}, err
	}
	var orderIDStr, driverIDStr, restaurantIDStr, assignedToStr, closedByStr, closedReasonStr string
	if orderID != nil {
		orderIDStr = *orderID
	}
	if driverID != nil {
		driverIDStr = *driverID
	}
	if restaurantID != nil {
		restaurantIDStr = *restaurantID
	}
	if assignedTo != nil {
		assignedToStr = *assignedTo
	}
	if closedBy != nil {
		closedByStr = *closedBy
	}
	if closedReason != nil {
		closedReasonStr = *closedReason
	}
	return domain.RehydrateTicket(
		id, ticketNo, userID, orderIDStr, driverIDStr, restaurantIDStr,
		subject, description,
		domain.TicketCategory(category), domain.TicketPriority(priority), domain.TicketStatus(status),
		assignedToStr, createdBy, closedByStr, closedReasonStr,
		messageCount, firstResponseAt, resolvedAt, closedAt,
		createdAt, updatedAt, version,
	), nil
}

func mapTicketReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrTicketNotFound
	}
	return fmt.Errorf("ticket read: %w", err)
}
