// Package port service: ServicePort + DTOs for the financial module.
package port

import (
        "context"
        "time"

        "avex-backend/internal/modules/financial/domain"
)

// ===== Wallet DTOs =====

type CreateWalletInput struct {
        OwnerType string // "user" | "driver" | "merchant"
        OwnerID   string
        Currency  string // ISO 4217, e.g. "EGP"
}

type WalletDTO struct {
        ID             string    `json:"id"`
        OwnerType      string    `json:"owner_type"`
        OwnerID        string    `json:"owner_id"`
        Currency       string    `json:"currency"`
        Balance        int64     `json:"balance"`         // cents
        PendingBalance int64     `json:"pending_balance"` // cents
        Status         string    `json:"status"`          // active | frozen | closed
        Version        int       `json:"version"`
        CreatedAt      time.Time `json:"created_at"`
        UpdatedAt      time.Time `json:"updated_at"`
}

// ===== Transaction DTOs =====

type CreditInput struct {
        WalletID       string
        Amount         int64 // cents
        Currency       string
        Category       string // topup | refund | adjustment | promotion | commission | tip
        ReferenceType  string // order | promotion | manual | system | payout | topup
        ReferenceID    string
        Description    string
        Metadata       Metadata
        IdempotencyKey string
}

type DebitInput struct {
        WalletID       string
        Amount         int64 // cents
        Currency       string
        Category       string // order_payment | payout | tip | adjustment
        ReferenceType  string
        ReferenceID    string
        Description    string
        Metadata       Metadata
        IdempotencyKey string
}

type TransferInput struct {
        FromWalletID   string
        ToWalletID     string
        Amount         int64
        Currency       string
        Category       string // order_payment | commission | payout | tip
        ReferenceType  string
        ReferenceID    string
        Description    string
        Metadata       Metadata
        IdempotencyKey string
}

type TransactionDTO struct {
        ID            string     `json:"id"`
        WalletID      string     `json:"wallet_id"`
        Type          string     `json:"type"` // credit | debit
        Category      string     `json:"category"`
        Amount        int64      `json:"amount"` // cents
        Currency      string     `json:"currency"`
        Status        string     `json:"status"`
        ReferenceType string     `json:"reference_type,omitempty"`
        ReferenceID   string     `json:"reference_id,omitempty"`
        Description   string     `json:"description,omitempty"`
        Metadata      Metadata   `json:"metadata,omitempty"`
        CreatedAt     time.Time  `json:"created_at"`
        CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// ===== Promotion DTOs =====

type CreatePromotionInput struct {
        Code              string
        Description       string
        PromoType         string // percent | fixed | free_delivery
        Value             int64
        Currency          string
        MinOrderAmount    int64
        MaxDiscountAmount *int64
        UsageLimit        *int
        PerUserLimit      int
        ValidFrom         time.Time
        ValidTo           *time.Time
}

type ValidatePromoInput struct {
        Code        string
        OrderTotal  int64 // cents
        Currency    string
        DeliveryFee int64 // cents (for free_delivery type)
        UserID      string
}

type RedeemPromoInput struct {
        Code        string
        UserID      string
        OrderID     string
        OrderTotal  int64
        Currency    string
        DeliveryFee int64
}

type PromotionDTO struct {
        ID                string     `json:"id"`
        Code              string     `json:"code"`
        Description       string     `json:"description,omitempty"`
        PromoType         string     `json:"promo_type"`
        Value             int64      `json:"value"`
        Currency          string     `json:"currency"`
        MinOrderAmount    int64      `json:"min_order_amount"`
        MaxDiscountAmount *int64     `json:"max_discount_amount,omitempty"`
        UsageLimit        *int       `json:"usage_limit,omitempty"`
        UsageCount        int        `json:"usage_count"`
        PerUserLimit      int        `json:"per_user_limit"`
        ValidFrom         time.Time  `json:"valid_from"`
        ValidTo           *time.Time `json:"valid_to,omitempty"`
        IsActive          bool       `json:"is_active"`
        CreatedAt         time.Time  `json:"created_at"`
}

type ValidatePromoResult struct {
        Valid          bool   `json:"valid"`
        DiscountAmount int64  `json:"discount_amount"` // cents
        Currency       string `json:"currency"`
        Reason         string `json:"reason,omitempty"`
}

type RedeemPromoResult struct {
        RedemptionID    string `json:"redemption_id"`
        PromotionID     string `json:"promotion_id"`
        DiscountAmount  int64  `json:"discount_amount"`
        Currency        string `json:"currency"`
}

// ===== Pricing DTOs =====

type CalculateQuoteInput struct {
        ZoneID       string
        Currency     string
        DistanceKM   float64
        DurationMin  int
        OrderTotal   int64  // cents — used for free-delivery threshold check
        PromoCode    string // optional
        UserID       string // optional, for per-user promo limits
}

type CalculateQuoteResult struct {
        Currency          string  `json:"currency"`
        DistanceKM        float64 `json:"distance_km"`
        DurationMin       int     `json:"duration_min"`
        BaseFee           int64   `json:"base_fee"`
        DistanceFee       int64   `json:"distance_fee"`
        TimeFee           int64   `json:"time_fee"`
        Subtotal          int64   `json:"subtotal"`
        SurgeMultiplier   float64 `json:"surge_multiplier"`
        SurgeAdjustment   int64   `json:"surge_adjustment"`
        PostSurge         int64   `json:"post_surge"`
        Discount          int64   `json:"discount"`
        PostDiscount      int64   `json:"post_discount"`
        TaxRate           float64 `json:"tax_rate"`
        TaxAmount         int64   `json:"tax_amount"`
        Total             int64   `json:"total"`
        IsFreeDelivery    bool    `json:"is_free_delivery"`
        FreeDeliveryReason string `json:"free_delivery_reason,omitempty"`
}

type CreatePricingRuleInput struct {
        ZoneID                string
        Currency              string
        BaseFee               int64
        PerKmRate             int64
        PerMinRate            int64
        MinFee                int64
        MaxFee                *int64
        FreeDeliveryThreshold *int64
        ValidFrom             time.Time
        ValidTo               *time.Time
}

type PricingRuleDTO struct {
        ID                    string     `json:"id"`
        ZoneID                string     `json:"zone_id"`
        Currency              string     `json:"currency"`
        BaseFee               int64      `json:"base_fee"`
        PerKmRate             int64      `json:"per_km_rate"`
        PerMinRate            int64      `json:"per_min_rate"`
        MinFee                int64      `json:"min_fee"`
        MaxFee                *int64     `json:"max_fee,omitempty"`
        FreeDeliveryThreshold *int64     `json:"free_delivery_threshold,omitempty"`
        IsActive              bool       `json:"is_active"`
        ValidFrom             time.Time  `json:"valid_from"`
        ValidTo               *time.Time `json:"valid_to,omitempty"`
        CreatedAt             time.Time  `json:"created_at"`
}

type CreateSurgeZoneInput struct {
        ZoneID    string
        Multiplier float64
        Reason    string
        DayOfWeek *int
        StartTime string
        EndTime   string
}

type SurgeZoneDTO struct {
        ID        string    `json:"id"`
        ZoneID    string    `json:"zone_id"`
        Multiplier float64  `json:"multiplier"`
        Reason    string    `json:"reason,omitempty"`
        DayOfWeek *int      `json:"day_of_week,omitempty"`
        StartTime string    `json:"start_time,omitempty"`
        EndTime   string    `json:"end_time,omitempty"`
        IsActive  bool      `json:"is_active"`
        CreatedAt time.Time `json:"created_at"`
}

// ===== ServicePort =====

type ServicePort interface {
        // ===== Wallet Operations =====
        CreateWallet(ctx context.Context, input CreateWalletInput) (*WalletDTO, error)
        GetWallet(ctx context.Context, id string) (*WalletDTO, error)
        GetWalletByOwner(ctx context.Context, ownerType, ownerID, currency string) (*WalletDTO, error)
        ListWalletsByOwner(ctx context.Context, ownerType, ownerID string) ([]WalletDTO, error)
        Credit(ctx context.Context, input CreditInput) (*TransactionDTO, *WalletDTO, error)
        Debit(ctx context.Context, input DebitInput) (*TransactionDTO, *WalletDTO, error)
        Transfer(ctx context.Context, input TransferInput) (*TransactionDTO, *TransactionDTO, error)
        FreezeWallet(ctx context.Context, id string) error
        UnfreezeWallet(ctx context.Context, id string) error

        // ===== Transaction Queries =====
        GetTransaction(ctx context.Context, id string) (*TransactionDTO, error)
        ListTransactionsByWallet(ctx context.Context, walletID string, page PageQuery) (Page[TransactionDTO], error)

        // ===== Promotion Operations =====
        CreatePromotion(ctx context.Context, input CreatePromotionInput) (*PromotionDTO, error)
        GetPromotion(ctx context.Context, id string) (*PromotionDTO, error)
        ListActivePromotions(ctx context.Context) ([]PromotionDTO, error)
        ValidatePromotion(ctx context.Context, input ValidatePromoInput) (ValidatePromoResult, error)
        RedeemPromotion(ctx context.Context, input RedeemPromoInput) (*RedeemPromoResult, error)

        // ===== Pricing Operations =====
        CalculateQuote(ctx context.Context, input CalculateQuoteInput) (*CalculateQuoteResult, error)
        CreatePricingRule(ctx context.Context, input CreatePricingRuleInput) (*PricingRuleDTO, error)
        ListPricingRules(ctx context.Context, page PageQuery) (Page[PricingRuleDTO], error)
        CreateSurgeZone(ctx context.Context, input CreateSurgeZoneInput) (*SurgeZoneDTO, error)
        ListSurgeZones(ctx context.Context, page PageQuery) (Page[SurgeZoneDTO], error)
        DeactivateSurgeZone(ctx context.Context, id string) error
}

// ===== Domain → DTO Mappers =====

func ToWalletDTO(w domain.Wallet) WalletDTO {
        return WalletDTO{
                ID:             w.ID(),
                OwnerType:      string(w.OwnerType()),
                OwnerID:        w.OwnerID(),
                Currency:       w.Currency(),
                Balance:        w.Balance().Amount(),
                PendingBalance: w.PendingBalance().Amount(),
                Status:         string(w.Status()),
                Version:        w.Version(),
                CreatedAt:      w.CreatedAt(),
                UpdatedAt:      w.UpdatedAt(),
        }
}

func ToTransactionDTO(t domain.Transaction) TransactionDTO {
        return TransactionDTO{
                ID:            t.ID(),
                WalletID:      t.WalletID(),
                Type:          string(t.Type()),
                Category:      string(t.Category()),
                Amount:        t.Amount().Amount(),
                Currency:      t.Amount().Currency(),
                Status:        string(t.Status()),
                ReferenceType: string(t.ReferenceType()),
                ReferenceID:   t.ReferenceID(),
                Description:   t.Description(),
                Metadata:      t.Metadata(),
                CreatedAt:     t.CreatedAt(),
                CompletedAt:   t.CompletedAt(),
        }
}

func ToPromotionDTO(p domain.Promotion) PromotionDTO {
        return PromotionDTO{
                ID:                p.ID(),
                Code:              p.Code(),
                Description:       p.Description(),
                PromoType:         string(p.Type()),
                Value:             p.Value(),
                Currency:          p.Currency(),
                MinOrderAmount:    p.MinOrderAmount(),
                MaxDiscountAmount: p.MaxDiscountAmount(),
                UsageLimit:        p.UsageLimit(),
                UsageCount:        p.UsageCount(),
                PerUserLimit:      p.PerUserLimit(),
                ValidFrom:         p.ValidFrom(),
                ValidTo:           p.ValidTo(),
                IsActive:          p.IsActive(),
                CreatedAt:         p.CreatedAt(),
        }
}

func ToPricingRuleDTO(r domain.PricingRule) PricingRuleDTO {
        dto := PricingRuleDTO{
                ID:        r.ID(),
                ZoneID:    r.ZoneID(),
                Currency:  r.Currency(),
                BaseFee:   r.BaseFee().Amount(),
                PerKmRate: r.PerKmRate().Amount(),
                PerMinRate: r.PerMinRate().Amount(),
                MinFee:    r.MinFee().Amount(),
                IsActive:  r.IsActive(),
                ValidFrom: r.ValidFrom(),
                ValidTo:   r.ValidTo(),
                CreatedAt: r.CreatedAt(),
        }
        if r.MaxFee() != nil {
                m := r.MaxFee().Amount()
                dto.MaxFee = &m
        }
        if r.FreeDeliveryThreshold() != nil {
                t := r.FreeDeliveryThreshold().Amount()
                dto.FreeDeliveryThreshold = &t
        }
        return dto
}

func ToSurgeZoneDTO(s domain.SurgeZone) SurgeZoneDTO {
        return SurgeZoneDTO{
                ID:         s.ID(),
                ZoneID:     s.ZoneID(),
                Multiplier: s.Multiplier(),
                Reason:     s.Reason(),
                DayOfWeek:  s.DayOfWeek(),
                StartTime:  s.StartTime(),
                EndTime:    s.EndTime(),
                IsActive:   s.IsActive(),
                CreatedAt:  s.CreatedAt(),
        }
}

// ===== Pointer Helpers =====
//
// Convenience wrappers used by the service layer when it needs to return
// pointers (the ServicePort signatures return *DTO for create/get paths).

func ToWalletDTOPtr(w domain.Wallet) *WalletDTO {
        dto := ToWalletDTO(w)
        return &dto
}

func ToTransactionDTOPtr(t domain.Transaction) *TransactionDTO {
        dto := ToTransactionDTO(t)
        return &dto
}

func ToPromotionDTOPtr(p domain.Promotion) *PromotionDTO {
        dto := ToPromotionDTO(p)
        return &dto
}

func ToPricingRuleDTOPtr(r domain.PricingRule) *PricingRuleDTO {
        dto := ToPricingRuleDTO(r)
        return &dto
}

func ToSurgeZoneDTOPtr(s domain.SurgeZone) *SurgeZoneDTO {
        dto := ToSurgeZoneDTO(s)
        return &dto
}
