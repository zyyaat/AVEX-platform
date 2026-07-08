// Package domain ticket: Ticket aggregate root.
//
// A Support Ticket is a customer service request. It has:
//   - A subject + description (the initial issue)
//   - A category (order_issue, payment_issue, etc.)
//   - A priority (low, normal, high, urgent)
//   - A status (open → in_progress → waiting → resolved → closed)
//   - An owner (the user who created it)
//   - An assigned agent (support staff)
//   - A list of messages (the conversation)
//   - Optional references to orders, drivers, restaurants
//
// Lifecycle:
//   open → in_progress (agent assigned) → waiting (agent asks user a question)
//        → resolved (agent marks fixed) → closed (user confirms or auto-close after 7d)
//
// Reopen: resolved → in_progress (if user reports the issue persists)
//
// Imports stdlib only.
package domain

import (
	"fmt"
	"time"
)

// Ticket is the aggregate root for a support ticket.
type Ticket struct {
	id          string
	ticketNo    string // human-readable, e.g. "TKT-20260101-00001"
	userID      string // the customer who created the ticket
	orderID     string // optional: related order
	driverID    string // optional: related driver
	restaurantID string // optional: related restaurant
	subject     string
	description string
	category    TicketCategory
	priority    TicketPriority
	status      TicketStatus
	assignedTo  string // agent user ID (empty = unassigned)
	createdBy   string // user | agent | system
	closedBy    string // user | agent | system
	closedReason string
	messageCount int   // total messages (denormalized for quick display)
	firstResponseAt *time.Time
	resolvedAt  *time.Time
	closedAt    *time.Time
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewTicket creates a new Ticket with validation.
// New tickets start in "open" status with no agent assigned.
func NewTicket(
	id, ticketNo, userID string,
	orderID, driverID, restaurantID string,
	subject, description string,
	category TicketCategory,
	priority TicketPriority,
	createdBy string,
	now time.Time,
) (Ticket, error) {
	if id == "" {
		return Ticket{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if ticketNo == "" {
		return Ticket{}, fmt.Errorf("%w: ticket number is required", ErrInvalidInput)
	}
	if userID == "" {
		return Ticket{}, ErrEmptyUserID
	}
	if subject == "" {
		return Ticket{}, ErrEmptySubject
	}
	if !category.IsValid() {
		return Ticket{}, fmt.Errorf("%w: %s", ErrInvalidTicketCategory, category)
	}
	if !priority.IsValid() {
		return Ticket{}, fmt.Errorf("%w: %s", ErrInvalidTicketPriority, priority)
	}
	if priority == "" {
		priority = PriorityNormal
	}
	if createdBy == "" {
		createdBy = "user"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Ticket{
		id:           id,
		ticketNo:     ticketNo,
		userID:       userID,
		orderID:      orderID,
		driverID:     driverID,
		restaurantID: restaurantID,
		subject:      subject,
		description:  description,
		category:     category,
		priority:     priority,
		status:       StatusOpen,
		createdBy:    createdBy,
		createdAt:    now,
		updatedAt:    now,
		version:      1,
	}, nil
}

// RehydrateTicket reconstructs from persistence.
func RehydrateTicket(
	id, ticketNo, userID, orderID, driverID, restaurantID string,
	subject, description string,
	category TicketCategory,
	priority TicketPriority,
	status TicketStatus,
	assignedTo, createdBy, closedBy, closedReason string,
	messageCount int,
	firstResponseAt, resolvedAt, closedAt *time.Time,
	createdAt, updatedAt time.Time,
	version int,
) Ticket {
	return Ticket{
		id:              id,
		ticketNo:        ticketNo,
		userID:          userID,
		orderID:         orderID,
		driverID:        driverID,
		restaurantID:    restaurantID,
		subject:         subject,
		description:     description,
		category:        category,
		priority:        priority,
		status:          status,
		assignedTo:      assignedTo,
		createdBy:       createdBy,
		closedBy:        closedBy,
		closedReason:    closedReason,
		messageCount:    messageCount,
		firstResponseAt: firstResponseAt,
		resolvedAt:      resolvedAt,
		closedAt:        closedAt,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		version:         version,
	}
}

// ===== Accessors =====

func (t Ticket) ID() string            { return t.id }
func (t Ticket) TicketNo() string      { return t.ticketNo }
func (t Ticket) UserID() string        { return t.userID }
func (t Ticket) OrderID() string       { return t.orderID }
func (t Ticket) DriverID() string      { return t.driverID }
func (t Ticket) RestaurantID() string  { return t.restaurantID }
func (t Ticket) Subject() string       { return t.subject }
func (t Ticket) Description() string   { return t.description }
func (t Ticket) Category() TicketCategory { return t.category }
func (t Ticket) Priority() TicketPriority { return t.priority }
func (t Ticket) Status() TicketStatus  { return t.status }
func (t Ticket) AssignedTo() string    { return t.assignedTo }
func (t Ticket) CreatedBy() string     { return t.createdBy }
func (t Ticket) ClosedBy() string      { return t.closedBy }
func (t Ticket) ClosedReason() string  { return t.closedReason }
func (t Ticket) MessageCount() int     { return t.messageCount }
func (t Ticket) FirstResponseAt() *time.Time { return t.firstResponseAt }
func (t Ticket) ResolvedAt() *time.Time { return t.resolvedAt }
func (t Ticket) ClosedAt() *time.Time  { return t.closedAt }
func (t Ticket) CreatedAt() time.Time  { return t.createdAt }
func (t Ticket) UpdatedAt() time.Time  { return t.updatedAt }
func (t Ticket) Version() int          { return t.version }

// IsOpen reports whether the ticket is still active (not closed).
func (t Ticket) IsOpen() bool { return !t.status.IsTerminal() }

// IsAssigned reports whether an agent has been assigned.
func (t Ticket) IsAssigned() bool { return t.assignedTo != "" }

// ===== Mutations =====

// Assign assigns the ticket to an agent + transitions to in_progress.
func (t Ticket) Assign(agentID string, now time.Time) (Ticket, error) {
	if t.status.IsTerminal() {
		return t, ErrTicketClosed
	}
	if t.assignedTo != "" && t.assignedTo != agentID {
		return t, fmt.Errorf("%w: already assigned to %s", ErrTicketAlreadyAssigned, t.assignedTo)
	}
	if agentID == "" {
		return t, fmt.Errorf("%w: agent id is required", ErrInvalidInput)
	}
	// Transition open → in_progress (or keep current if already in_progress/waiting)
	if t.status == StatusOpen {
		newStatus, err := t.status.Transition(StatusInProgress)
		if err != nil {
			return t, err
		}
		t.status = newStatus
	}
	t.assignedTo = agentID
	if t.firstResponseAt == nil {
		t.firstResponseAt = &now
	}
	t.updatedAt = now
	return t, nil
}

// Unassign removes the agent + transitions back to open.
func (t Ticket) Unassign(now time.Time) (Ticket, error) {
	if t.status.IsTerminal() {
		return t, ErrTicketClosed
	}
	t.assignedTo = ""
	if t.status == StatusInProgress || t.status == StatusWaiting {
		newStatus, err := t.status.Transition(StatusOpen)
		if err != nil {
			return t, err
		}
		t.status = newStatus
	}
	t.updatedAt = now
	return t, nil
}

// SetStatus transitions the ticket to a new status.
func (t Ticket) SetStatus(target TicketStatus, now time.Time) (Ticket, error) {
	if t.status.IsTerminal() {
		return t, ErrTicketClosed
	}
	newStatus, err := t.status.Transition(target)
	if err != nil {
		return t, err
	}
	t.status = newStatus
	t.updatedAt = now
	if target == StatusResolved && t.resolvedAt == nil {
		t.resolvedAt = &now
	}
	return t, nil
}

// SetPriority updates the priority.
func (t Ticket) SetPriority(p TicketPriority, now time.Time) (Ticket, error) {
	if !p.IsValid() {
		return t, fmt.Errorf("%w: %s", ErrInvalidTicketPriority, p)
	}
	if t.status.IsTerminal() {
		return t, ErrTicketClosed
	}
	t.priority = p
	t.updatedAt = now
	return t, nil
}

// Close transitions the ticket to closed.
func (t Ticket) Close(closedBy, reason string, now time.Time) (Ticket, error) {
	if t.status.IsTerminal() {
		return t, nil // idempotent
	}
	// All statuses can transition to closed
	t.status = StatusClosed
	t.closedBy = closedBy
	t.closedReason = reason
	t.closedAt = &now
	t.updatedAt = now
	return t, nil
}

// Reopen transitions from resolved → in_progress.
func (t Ticket) Reopen(now time.Time) (Ticket, error) {
	if t.status != StatusResolved {
		return t, fmt.Errorf("%w: can only reopen resolved tickets", ErrInvalidTicketStatus)
	}
	newStatus, err := t.status.Transition(StatusInProgress)
	if err != nil {
		return t, err
	}
	t.status = newStatus
	t.resolvedAt = nil
	t.updatedAt = now
	return t, nil
}

// IncrementMessageCount bumps the denormalized message count.
func (t Ticket) IncrementMessageCount(now time.Time) Ticket {
	t.messageCount++
	t.updatedAt = now
	return t
}

// Transition wraps status.Transition for external callers.
func (s TicketStatus) Transition(target TicketStatus) (TicketStatus, error) {
	if !s.CanTransitionTo(target) {
		return s, fmt.Errorf("%w: %s -> %s", ErrInvalidTicketStatus, s, target)
	}
	return target, nil
}
