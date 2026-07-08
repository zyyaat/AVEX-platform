// Package events factory: convenience constructors for the financial EventPublisher.
package events

import (
	"encoding/json"

	"avex-backend/internal/modules/financial/port"
)

// BuildEnvelope constructs an EventEnvelope from a typed payload.
func BuildEnvelope(
	eventType string,
	eventVersion int,
	schemaVersion int,
	payload any,
	ec port.EventContext,
) (port.EventEnvelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return port.EventEnvelope{}, err
	}
	return port.BuildEnvelope("", eventType, eventVersion, schemaVersion, payloadBytes, ec), nil
}

// ===== Per-Event Convenience Functions =====

func WalletCreatedEnvelope(payload port.WalletCreatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventWalletCreated, port.WalletCreatedEventVersion, port.WalletCreatedSchemaVersion, payload, ec)
}

func WalletCreditedEnvelope(payload port.WalletCreditedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventWalletCredited, port.WalletCreditedEventVersion, port.WalletCreditedSchemaVersion, payload, ec)
}

func WalletDebitedEnvelope(payload port.WalletDebitedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventWalletDebited, port.WalletDebitedEventVersion, port.WalletDebitedSchemaVersion, payload, ec)
}

func WalletFrozenEnvelope(payload port.WalletFrozenPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventWalletFrozen, port.WalletFrozenEventVersion, port.WalletFrozenSchemaVersion, payload, ec)
}

func WalletUnfrozenEnvelope(payload port.WalletUnfrozenPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventWalletUnfrozen, port.WalletUnfrozenEventVersion, port.WalletUnfrozenSchemaVersion, payload, ec)
}

func TransferCompletedEnvelope(payload port.TransferCompletedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventTransferCompleted, port.TransferCompletedEventVersion, port.TransferCompletedSchemaVersion, payload, ec)
}

func PromotionRedeemedEnvelope(payload port.PromotionRedeemedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventPromotionRedeemed, port.PromotionRedeemedEventVersion, port.PromotionRedeemedSchemaVersion, payload, ec)
}

func PricingCalculatedEnvelope(payload port.PricingCalculatedPayload, ec port.EventContext) (port.EventEnvelope, error) {
	return BuildEnvelope(port.EventPricingCalculated, port.PricingCalculatedEventVersion, port.PricingCalculatedSchemaVersion, payload, ec)
}
