// Package domain: Support module errors + types.
package domain

import (
	"errors"
	"fmt"
)

// ===== Ticket Errors =====

var ErrTicketNotFound = errors.New("ticket not found")
var ErrTicketAlreadyExists = errors.New("ticket already exists")
var ErrTicketClosed = errors.New("ticket is closed")
var ErrInvalidTicketStatus = errors.New("invalid ticket status")
var ErrInvalidTicketPriority = errors.New("invalid ticket priority")
var ErrInvalidTicketCategory = errors.New("invalid ticket category")
var ErrNotTicketOwner = errors.New("user is not the ticket owner")
var ErrNotAssignedAgent = errors.New("user is not the assigned agent")
var ErrTicketAlreadyAssigned = errors.New("ticket already assigned")
var ErrTicketNotAssigned = errors.New("ticket not assigned to any agent")

// ===== Message Errors =====

var ErrMessageNotFound = errors.New("message not found")
var ErrEmptyMessage = errors.New("message body is required")
var ErrInvalidMessageType = errors.New("invalid message type")
var ErrCannotEditMessage = errors.New("message cannot be edited")

// ===== Attachment Errors =====

var ErrAttachmentNotFound = errors.New("attachment not found")
var ErrInvalidFileType = errors.New("invalid file type")
var ErrFileTooLarge = errors.New("file too large")

// ===== Validation =====

var ErrInvalidID = errors.New("invalid id")
var ErrInvalidInput = errors.New("invalid input")
var ErrEmptySubject = errors.New("subject is required")
var ErrEmptyUserID = errors.New("user id is required")

// ===== Types =====

// TicketStatus enumerates ticket lifecycle states.
type TicketStatus string

const (
	StatusOpen       TicketStatus = "open"
	StatusInProgress TicketStatus = "in_progress"
	StatusWaiting    TicketStatus = "waiting" // waiting for user response
	StatusResolved   TicketStatus = "resolved"
	StatusClosed     TicketStatus = "closed"
)

func (s TicketStatus) IsValid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusWaiting, StatusResolved, StatusClosed:
		return true
	}
	return false
}

func (s TicketStatus) IsTerminal() bool {
	return s == StatusClosed
}

func (s TicketStatus) CanTransitionTo(target TicketStatus) bool {
	if s == target || s.IsTerminal() {
		return false
	}
	allowed := map[TicketStatus][]TicketStatus{
		StatusOpen:       {StatusInProgress, StatusResolved, StatusClosed},
		StatusInProgress: {StatusWaiting, StatusResolved, StatusClosed, StatusOpen},
		StatusWaiting:    {StatusInProgress, StatusResolved, StatusClosed},
		StatusResolved:   {StatusClosed, StatusInProgress}, // reopen possible
		StatusClosed:     {},
	}
	for _, t := range allowed[s] {
		if t == target {
			return true
		}
	}
	return false
}

// TicketPriority enumerates priority levels.
type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityNormal TicketPriority = "normal"
	PriorityHigh   TicketPriority = "high"
	PriorityUrgent TicketPriority = "urgent"
)

func (p TicketPriority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityUrgent:
		return true
	}
	return false
}

// TicketCategory enumerates ticket categories.
type TicketCategory string

const (
	CategoryOrderIssue     TicketCategory = "order_issue"
	CategoryPaymentIssue   TicketCategory = "payment_issue"
	CategoryDeliveryIssue  TicketCategory = "delivery_issue"
	CategoryAccountIssue   TicketCategory = "account_issue"
	CategoryAppBug         TicketCategory = "app_bug"
	CategoryFeatureRequest TicketCategory = "feature_request"
	CategoryOther          TicketCategory = "other"
)

func (c TicketCategory) IsValid() bool {
	switch c {
	case CategoryOrderIssue, CategoryPaymentIssue, CategoryDeliveryIssue,
		CategoryAccountIssue, CategoryAppBug, CategoryFeatureRequest, CategoryOther:
		return true
	}
	return false
}

// MessageType enumerates the kinds of messages in a ticket.
type MessageType string

const (
	MessageTypeUser      MessageType = "user"      // message from the customer
	MessageTypeAgent     MessageType = "agent"     // message from a support agent
	MessageTypeSystem    MessageType = "system"    // system-generated (e.g. status change)
	MessageTypeInternal  MessageType = "internal"  // internal note (not visible to user)
)

func (m MessageType) IsValid() bool {
	switch m {
	case MessageTypeUser, MessageTypeAgent, MessageTypeSystem, MessageTypeInternal:
		return true
	}
	return false
}

// Composite error
type ValidationError struct {
	Field   string
	Wrapped error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %v", e.Field, e.Wrapped)
	}
	return e.Wrapped.Error()
}

func (e *ValidationError) Unwrap() error { return e.Wrapped }

func NewValidationError(field string, err error) *ValidationError {
	return &ValidationError{Field: field, Wrapped: err}
}
