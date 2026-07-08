// Package port events: event type constants, event payload structs, and
// the EventContext helper for constructing EventEnvelopes for the financial module.
//
// All events start at event_version=1, schema_version=1.
// Breaking changes increment event_version; additive changes increment
// schema_version.
//
// Imports: stdlib only.
package port

import (
	"time"
)

// ===== Event Type Constants =====

const (
	EventWalletCreated     = "financial.wallet.created"
	EventWalletCredited    = "financial.wallet.credited"
	EventWalletDebited     = "financial.wallet.debited"
	EventWalletFrozen      = "financial.wallet.frozen"
	EventWalletUnfrozen    = "financial.wallet.unfrozen"
	EventTransferCompleted = "financial.transfer.completed"
	EventPromotionRedeemed = "financial.promotion.redeemed"
	EventPricingCalculated = "financial.pricing.calculated"
)

// ===== Event Versions =====

const (
	WalletCreatedEventVersion     = 1
	WalletCreatedSchemaVersion    = 1
	WalletCreditedEventVersion    = 1
	WalletCreditedSchemaVersion   = 1
	WalletDebitedEventVersion     = 1
	WalletDebitedSchemaVersion    = 1
	WalletFrozenEventVersion      = 1
	WalletFrozenSchemaVersion     = 1
	WalletUnfrozenEventVersion    = 1
	WalletUnfrozenSchemaVersion   = 1
	TransferCompletedEventVersion = 1
	TransferCompletedSchemaVersion = 1
	PromotionRedeemedEventVersion = 1
	PromotionRedeemedSchemaVersion = 1
	PricingCalculatedEventVersion = 1
	PricingCalculatedSchemaVersion = 1
)

// ===== Event Payloads =====

type WalletCreatedPayload struct {
	WalletID  string `json:"wallet_id"`
	OwnerType string `json:"owner_type"`
	OwnerID   string `json:"owner_id"`
	Currency  string `json:"currency"`
}

type WalletCreditedPayload struct {
	WalletID      string `json:"wallet_id"`
	TransactionID string `json:"transaction_id"`
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
	Category      string `json:"category"`
	ReferenceType string `json:"reference_type,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`
	NewBalance    int64  `json:"new_balance_cents"`
}

type WalletDebitedPayload struct {
	WalletID      string `json:"wallet_id"`
	TransactionID string `json:"transaction_id"`
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
	Category      string `json:"category"`
	ReferenceType string `json:"reference_type,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`
	NewBalance    int64  `json:"new_balance_cents"`
}

type WalletFrozenPayload struct {
	WalletID string `json:"wallet_id"`
	OwnerID  string `json:"owner_id"`
}

type WalletUnfrozenPayload struct {
	WalletID string `json:"wallet_id"`
	OwnerID  string `json:"owner_id"`
}

type TransferCompletedPayload struct {
	FromWalletID      string `json:"from_wallet_id"`
	ToWalletID        string `json:"to_wallet_id"`
	AmountCents       int64  `json:"amount_cents"`
	Currency          string `json:"currency"`
	Category          string `json:"category"`
	ReferenceType     string `json:"reference_type,omitempty"`
	ReferenceID       string `json:"reference_id,omitempty"`
	DebitTransactionID  string `json:"debit_transaction_id"`
	CreditTransactionID string `json:"credit_transaction_id"`
}

type PromotionRedeemedPayload struct {
	RedemptionID    string `json:"redemption_id"`
	PromotionID     string `json:"promotion_id"`
	PromotionCode   string `json:"promotion_code"`
	UserID          string `json:"user_id"`
	OrderID         string `json:"order_id,omitempty"`
	DiscountCents   int64  `json:"discount_cents"`
	Currency        string `json:"currency"`
}

type PricingCalculatedPayload struct {
	ZoneID         string  `json:"zone_id"`
	Currency       string  `json:"currency"`
	DistanceKM     float64 `json:"distance_km"`
	DurationMin    int     `json:"duration_min"`
	SubtotalCents  int64   `json:"subtotal_cents"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	DiscountCents  int64   `json:"discount_cents"`
	TaxCents       int64   `json:"tax_cents"`
	TotalCents     int64   `json:"total_cents"`
	IsFreeDelivery bool    `json:"is_free_delivery"`
}

// ===== Event Metadata =====

type EventMetadata struct {
	CorrelationID string
	TraceID       string
	OccurredAt    time.Time
}

// EventContext bundles the actor and metadata for constructing an EventEnvelope.
type EventContext struct {
	Actor    ActorContext
	Metadata EventMetadata
}

// BuildEnvelope constructs an EventEnvelope from the given parameters.
func BuildEnvelope(
	eventID string,
	eventType string,
	eventVersion int,
	schemaVersion int,
	payload []byte,
	ec EventContext,
) EventEnvelope {
	occurredAt := ec.Metadata.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return EventEnvelope{
		EventID:       eventID,
		EventType:     eventType,
		EventVersion:  eventVersion,
		SchemaVersion: schemaVersion,
		OccurredAt:    occurredAt,
		Producer:      "financial",
		CorrelationID: ec.Metadata.CorrelationID,
		TraceID:       ec.Metadata.TraceID,
		ActorType:     ec.Actor.Type,
		ActorID:       ec.Actor.ID,
		ActorIP:       ec.Actor.IP,
		ActorUA:       ec.Actor.UserAgent,
		Payload:       payload,
	}
}
