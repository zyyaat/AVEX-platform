// Package domain tests: Ticket + Message + Attachment.
package domain

import (
	"testing"
	"time"
)

func testNowSupport() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-01-01T12:00:00Z")
	return t
}

// ===== Ticket Status Tests =====

func TestTicketStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		from, to TicketStatus
		want     bool
	}{
		{StatusOpen, StatusInProgress, true},
		{StatusOpen, StatusResolved, true},
		{StatusOpen, StatusClosed, true},
		{StatusOpen, StatusWaiting, false}, // must go through in_progress first
		{StatusInProgress, StatusWaiting, true},
		{StatusInProgress, StatusResolved, true},
		{StatusInProgress, StatusOpen, true},
		{StatusWaiting, StatusInProgress, true},
		{StatusWaiting, StatusResolved, true},
		{StatusResolved, StatusClosed, true},
		{StatusResolved, StatusInProgress, true}, // reopen
		{StatusClosed, StatusOpen, false},
		{StatusClosed, StatusInProgress, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("CanTransitionTo(%s->%s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// ===== Ticket Tests =====

func TestNewTicket(t *testing.T) {
	now := testNowSupport()
	tests := []struct {
		name       string
		id         string
		ticketNo   string
		userID     string
		subject    string
		category   TicketCategory
		priority   TicketPriority
		wantErr    error
	}{
		{"valid", "t1", "TKT-001", "u1", "Order delayed", CategoryOrderIssue, PriorityHigh, nil},
		{"empty id", "", "TKT-002", "u1", "S", CategoryOther, PriorityNormal, ErrInvalidID},
		{"empty ticket no", "t2", "", "u1", "S", CategoryOther, PriorityNormal, ErrInvalidInput},
		{"empty user id", "t3", "TKT-003", "", "S", CategoryOther, PriorityNormal, ErrEmptyUserID},
		{"empty subject", "t4", "TKT-004", "u1", "", CategoryOther, PriorityNormal, ErrEmptySubject},
		{"invalid category", "t5", "TKT-005", "u1", "S", TicketCategory("bogus"), PriorityNormal, ErrInvalidTicketCategory},
		{"invalid priority", "t6", "TKT-006", "u1", "S", CategoryOther, TicketPriority("urgent-x"), ErrInvalidTicketPriority},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTicket(tt.id, tt.ticketNo, tt.userID, "", "", "", tt.subject, "desc", tt.category, tt.priority, "user", now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTicketAssign(t *testing.T) {
	now := testNowSupport()
	ticket, _ := NewTicket("t1", "TKT-001", "u1", "", "", "", "Subject", "desc", CategoryOrderIssue, PriorityNormal, "user", now)

	// Assign
	assigned, err := ticket.Assign("agent-1", now)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	if assigned.AssignedTo() != "agent-1" {
		t.Errorf("expected agent-1, got %s", assigned.AssignedTo())
	}
	if assigned.Status() != StatusInProgress {
		t.Errorf("expected in_progress, got %s", assigned.Status())
	}
	if assigned.FirstResponseAt() == nil {
		t.Error("expected first_response_at set")
	}

	// Reassign to same agent (idempotent)
	_, err = assigned.Assign("agent-1", now)
	if err != nil {
		t.Fatalf("reassign same agent: %v", err)
	}

	// Reassign to different agent (should fail)
	_, err = assigned.Assign("agent-2", now)
	if !errIs(err, ErrTicketAlreadyAssigned) {
		t.Fatalf("expected ErrTicketAlreadyAssigned, got %v", err)
	}
}

func TestTicketCloseAndReopen(t *testing.T) {
	now := testNowSupport()
	ticket, _ := NewTicket("t1", "TKT-001", "u1", "", "", "", "S", "desc", CategoryOther, PriorityNormal, "user", now)

	// Close directly from open
	closed, err := ticket.Close("user", "issue resolved", now)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if closed.Status() != StatusClosed {
		t.Errorf("expected closed, got %s", closed.Status())
	}
	if closed.ClosedAt() == nil {
		t.Error("expected closed_at set")
	}

	// Cannot assign a closed ticket
	_, err = closed.Assign("agent-1", now)
	if !errIs(err, ErrTicketClosed) {
		t.Fatalf("expected ErrTicketClosed, got %v", err)
	}
}

func TestTicketReopenFromResolved(t *testing.T) {
	now := testNowSupport()
	ticket, _ := NewTicket("t1", "TKT-001", "u1", "", "", "", "S", "desc", CategoryOther, PriorityNormal, "user", now)

	// open → in_progress → resolved
	ticket, _ = ticket.Assign("agent-1", now)
	ticket, _ = ticket.SetStatus(StatusResolved, now)
	if ticket.Status() != StatusResolved {
		t.Fatalf("expected resolved, got %s", ticket.Status())
	}

	// Reopen
	reopened, err := ticket.Reopen(now)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Status() != StatusInProgress {
		t.Errorf("expected in_progress after reopen, got %s", reopened.Status())
	}
	if reopened.ResolvedAt() != nil {
		t.Error("expected resolved_at cleared after reopen")
	}
}

func TestTicketSetPriority(t *testing.T) {
	now := testNowSupport()
	ticket, _ := NewTicket("t1", "TKT-001", "u1", "", "", "", "S", "desc", CategoryOther, PriorityNormal, "user", now)

	updated, err := ticket.SetPriority(PriorityUrgent, now)
	if err != nil {
		t.Fatalf("set priority: %v", err)
	}
	if updated.Priority() != PriorityUrgent {
		t.Errorf("expected urgent, got %s", updated.Priority())
	}

	// Invalid priority
	_, err = ticket.SetPriority(TicketPriority("bogus"), now)
	if !errIs(err, ErrInvalidTicketPriority) {
		t.Fatalf("expected ErrInvalidTicketPriority, got %v", err)
	}
}

// ===== Message Tests =====

func TestNewTicketMessage(t *testing.T) {
	now := testNowSupport()
	tests := []struct {
		name       string
		id         string
		ticketID   string
		senderType MessageType
		senderID   string
		body       string
		wantErr    error
	}{
		{"valid user msg", "m1", "t1", MessageTypeUser, "u1", "I have an issue", nil},
		{"valid agent msg", "m2", "t1", MessageTypeAgent, "a1", "Let me help you", nil},
		{"valid system msg", "m3", "t1", MessageTypeSystem, "system", "Ticket assigned", nil},
		{"empty id", "", "t1", MessageTypeUser, "u1", "body", ErrInvalidID},
		{"empty ticket id", "m4", "", MessageTypeUser, "u1", "body", ErrInvalidInput},
		{"invalid sender type", "m5", "t1", MessageType("bogus"), "u1", "body", ErrInvalidMessageType},
		{"empty sender id", "m6", "t1", MessageTypeUser, "", "body", ErrInvalidInput},
		{"empty body", "m7", "t1", MessageTypeUser, "u1", "", ErrEmptyMessage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTicketMessage(tt.id, tt.ticketID, tt.senderType, tt.senderID, tt.body, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTicketMessageEdit(t *testing.T) {
	now := testNowSupport()
	msg, _ := NewTicketMessage("m1", "t1", MessageTypeUser, "u1", "original body", now)

	edited, err := msg.Edit("updated body", now)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if edited.Body() != "updated body" {
		t.Errorf("expected 'updated body', got %q", edited.Body())
	}
	if edited.EditedAt() == nil {
		t.Error("expected edited_at set")
	}

	// Cannot edit to empty
	_, err = msg.Edit("", now)
	if !errIs(err, ErrEmptyMessage) {
		t.Fatalf("expected ErrEmptyMessage, got %v", err)
	}
}

// ===== Attachment Tests =====

func TestNewTicketAttachment(t *testing.T) {
	now := testNowSupport()
	tests := []struct {
		name     string
		fileType string
		fileSize int64
		wantErr  error
	}{
		{"valid image", "image", 1024, nil},
		{"valid document", "document", 5000, nil},
		{"valid video", "video", 5000000, nil},
		{"invalid type", "executable", 1024, ErrInvalidFileType},
		{"zero size", "image", 0, ErrInvalidInput},
		{"negative size", "image", -1, ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTicketAttachment("a1", "m1", "file.png", tt.fileType, "https://example.com/file.png", tt.fileSize, now)
			if tt.wantErr != nil {
				if err == nil || !errIs(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// errIs helper
func errIs(err, target error) bool {
	if err == target {
		return true
	}
	for {
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
		if err == target {
			return true
		}
		if err == nil {
			return false
		}
	}
}
