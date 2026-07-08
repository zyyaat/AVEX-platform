// Package domain message: TicketMessage + TicketAttachment value objects.
package domain

import (
	"fmt"
	"time"
)

// TicketMessage is a single message in a ticket conversation.
type TicketMessage struct {
	id         string
	ticketID   string
	senderType MessageType // user | agent | system | internal
	senderID   string      // user_id or agent_id
	body       string
	editedAt   *time.Time
	createdAt  time.Time
}

func NewTicketMessage(
	id, ticketID string,
	senderType MessageType,
	senderID, body string,
	now time.Time,
) (TicketMessage, error) {
	if id == "" {
		return TicketMessage{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if ticketID == "" {
		return TicketMessage{}, fmt.Errorf("%w: ticket id is required", ErrInvalidInput)
	}
	if !senderType.IsValid() {
		return TicketMessage{}, fmt.Errorf("%w: %s", ErrInvalidMessageType, senderType)
	}
	if senderID == "" {
		return TicketMessage{}, fmt.Errorf("%w: sender id is required", ErrInvalidInput)
	}
	if body == "" {
		return TicketMessage{}, ErrEmptyMessage
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return TicketMessage{
		id:         id,
		ticketID:   ticketID,
		senderType: senderType,
		senderID:   senderID,
		body:       body,
		createdAt:  now,
	}, nil
}

func RehydrateTicketMessage(
	id, ticketID string,
	senderType MessageType,
	senderID, body string,
	editedAt *time.Time,
	createdAt time.Time,
) TicketMessage {
	return TicketMessage{
		id:         id,
		ticketID:   ticketID,
		senderType: senderType,
		senderID:   senderID,
		body:       body,
		editedAt:   editedAt,
		createdAt:  createdAt,
	}
}

func (m TicketMessage) ID() string             { return m.id }
func (m TicketMessage) TicketID() string       { return m.ticketID }
func (m TicketMessage) SenderType() MessageType { return m.senderType }
func (m TicketMessage) SenderID() string       { return m.senderID }
func (m TicketMessage) Body() string           { return m.body }
func (m TicketMessage) EditedAt() *time.Time   { return m.editedAt }
func (m TicketMessage) CreatedAt() time.Time   { return m.createdAt }

// Edit updates the message body + sets editedAt.
func (m TicketMessage) Edit(newBody string, now time.Time) (TicketMessage, error) {
	if newBody == "" {
		return m, ErrEmptyMessage
	}
	m.body = newBody
	m.editedAt = &now
	return m, nil
}

// TicketAttachment is a file attached to a message.
type TicketAttachment struct {
	id        string
	messageID string
	fileName  string
	fileType  string // image | document | video | audio
	fileURL   string
	fileSize  int64  // bytes
	createdAt time.Time
}

func NewTicketAttachment(
	id, messageID, fileName, fileType, fileURL string,
	fileSize int64,
	now time.Time,
) (TicketAttachment, error) {
	if id == "" {
		return TicketAttachment{}, fmt.Errorf("%w: id is required", ErrInvalidID)
	}
	if messageID == "" {
		return TicketAttachment{}, fmt.Errorf("%w: message id is required", ErrInvalidInput)
	}
	if fileName == "" {
		return TicketAttachment{}, fmt.Errorf("%w: file name is required", ErrInvalidInput)
	}
	if fileURL == "" {
		return TicketAttachment{}, fmt.Errorf("%w: file url is required", ErrInvalidInput)
	}
	if fileSize <= 0 {
		return TicketAttachment{}, fmt.Errorf("%w: file size must be positive", ErrInvalidInput)
	}
	switch fileType {
	case "image", "document", "video", "audio":
		// valid
	default:
		return TicketAttachment{}, fmt.Errorf("%w: %s", ErrInvalidFileType, fileType)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return TicketAttachment{
		id:        id,
		messageID: messageID,
		fileName:  fileName,
		fileType:  fileType,
		fileURL:   fileURL,
		fileSize:  fileSize,
		createdAt: now,
	}, nil
}

func RehydrateTicketAttachment(
	id, messageID, fileName, fileType, fileURL string,
	fileSize int64,
	createdAt time.Time,
) TicketAttachment {
	return TicketAttachment{
		id:        id,
		messageID: messageID,
		fileName:  fileName,
		fileType:  fileType,
		fileURL:   fileURL,
		fileSize:  fileSize,
		createdAt: createdAt,
	}
}

func (a TicketAttachment) ID() string         { return a.id }
func (a TicketAttachment) MessageID() string  { return a.messageID }
func (a TicketAttachment) FileName() string   { return a.fileName }
func (a TicketAttachment) FileType() string   { return a.fileType }
func (a TicketAttachment) FileURL() string    { return a.fileURL }
func (a TicketAttachment) FileSize() int64    { return a.fileSize }
func (a TicketAttachment) CreatedAt() time.Time { return a.createdAt }
