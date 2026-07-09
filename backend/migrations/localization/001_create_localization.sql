-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS localization;

CREATE TABLE localization.languages (
    id          UUID         PRIMARY KEY,
    code        VARCHAR(2)   NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    is_rtl      BOOLEAN      NOT NULL DEFAULT FALSE,
    is_default  BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_languages_active ON localization.languages (is_active) WHERE is_active = TRUE;

CREATE TABLE localization.translations (
    id             UUID         PRIMARY KEY,
    language_code  VARCHAR(2)   NOT NULL REFERENCES localization.languages(code) ON DELETE CASCADE,
    key            VARCHAR(200) NOT NULL,
    value          TEXT         NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(language_code, key)
);
CREATE INDEX idx_translations_lang ON localization.translations (language_code);
CREATE INDEX idx_translations_key ON localization.translations (key);
CREATE INDEX idx_translations_lang_key ON localization.translations (language_code, key);

-- Seed languages
INSERT INTO localization.languages (id, code, name, is_rtl, is_default, is_active) VALUES
    ('00000000-0000-0000-0003-000000000001', 'en', 'English',  FALSE, TRUE,  TRUE),
    ('00000000-0000-0000-0003-000000000002', 'ar', 'العربية',  TRUE,  FALSE, TRUE),
    ('00000000-0000-0000-0003-000000000003', 'fr', 'Français', FALSE, FALSE, FALSE)
ON CONFLICT (code) DO NOTHING;

-- Seed translations (en + ar) for common keys
INSERT INTO localization.translations (id, language_code, key, value) VALUES
    -- Order statuses
    ('00000000-0000-0000-0004-000000000001', 'en', 'orders.status.pending',         'Pending'),
    ('00000000-0000-0000-0004-000000000002', 'en', 'orders.status.confirmed',       'Confirmed'),
    ('00000000-0000-0000-0004-000000000003', 'en', 'orders.status.preparing',       'Preparing'),
    ('00000000-0000-0000-0004-000000000004', 'en', 'orders.status.ready_for_pickup','Ready for Pickup'),
    ('00000000-0000-0000-0004-000000000005', 'en', 'orders.status.dispatching',     'Finding Driver'),
    ('00000000-0000-0000-0004-000000000006', 'en', 'orders.status.assigned',        'Driver Assigned'),
    ('00000000-0000-0000-0004-000000000007', 'en', 'orders.status.picked_up',       'Picked Up'),
    ('00000000-0000-0000-0004-000000000008', 'en', 'orders.status.delivered',       'Delivered'),
    ('00000000-0000-0000-0004-000000000009', 'en', 'orders.status.cancelled',       'Cancelled'),
    -- Arabic order statuses
    ('00000000-0000-0000-0004-000000000010', 'ar', 'orders.status.pending',         'قيد الانتظار'),
    ('00000000-0000-0000-0004-000000000011', 'ar', 'orders.status.confirmed',       'تم التأكيد'),
    ('00000000-0000-0000-0004-000000000012', 'ar', 'orders.status.preparing',       'قيد التحضير'),
    ('00000000-0000-0000-0004-000000000013', 'ar', 'orders.status.ready_for_pickup','جاهز للاستلام'),
    ('00000000-0000-0000-0004-000000000014', 'ar', 'orders.status.dispatching',     'البحث عن مندوب'),
    ('00000000-0000-0000-0004-000000000015', 'ar', 'orders.status.assigned',        'تم تعيين مندوب'),
    ('00000000-0000-0000-0004-000000000016', 'ar', 'orders.status.picked_up',       'تم الاستلام'),
    ('00000000-0000-0000-0004-000000000017', 'ar', 'orders.status.delivered',       'تم التوصيل'),
    ('00000000-0000-0000-0004-000000000018', 'ar', 'orders.status.cancelled',       'ملغي'),
    -- Notification titles
    ('00000000-0000-0000-0004-000000000019', 'en', 'notifications.order_created.title',  'Order Placed'),
    ('00000000-0000-0000-0004-000000000020', 'en', 'notifications.order_confirmed.title','Order Confirmed'),
    ('00000000-0000-0000-0004-000000000021', 'en', 'notifications.order_delivered.title','Order Delivered'),
    ('00000000-0000-0000-0004-000000000022', 'en', 'notifications.dispatch_offer.title', 'New Delivery Offer'),
    ('00000000-0000-0000-0004-000000000023', 'en', 'notifications.wallet_credited.title','Wallet Credited'),
    -- Arabic notification titles
    ('00000000-0000-0000-0004-000000000024', 'ar', 'notifications.order_created.title',  'تم إنشاء الطلب'),
    ('00000000-0000-0000-0004-000000000025', 'ar', 'notifications.order_confirmed.title','تم تأكيد الطلب'),
    ('00000000-0000-0000-0004-000000000026', 'ar', 'notifications.order_delivered.title','تم توصيل الطلب'),
    ('00000000-0000-0000-0004-000000000027', 'ar', 'notifications.dispatch_offer.title', 'عرض توصيل جديد'),
    ('00000000-0000-0000-0004-000000000028', 'ar', 'notifications.wallet_credited.title','تم شحن المحفظة'),
    -- Common buttons
    ('00000000-0000-0000-0004-000000000029', 'en', 'common.button.submit',   'Submit'),
    ('00000000-0000-0000-0004-000000000030', 'en', 'common.button.cancel',   'Cancel'),
    ('00000000-0000-0000-0004-000000000031', 'en', 'common.button.confirm',  'Confirm'),
    ('00000000-0000-0000-0004-000000000032', 'en', 'common.button.delete',   'Delete'),
    ('00000000-0000-0000-0004-000000000033', 'ar', 'common.button.submit',   'إرسال'),
    ('00000000-0000-0000-0004-000000000034', 'ar', 'common.button.cancel',   'إلغاء'),
    ('00000000-0000-0000-0004-000000000035', 'ar', 'common.button.confirm',  'تأكيد'),
    ('00000000-0000-0000-0004-000000000036', 'ar', 'common.button.delete',   'حذف'),
    -- Errors
    ('00000000-0000-0000-0004-000000000037', 'en', 'errors.not_found',       'Resource not found'),
    ('00000000-0000-0000-0004-000000000038', 'en', 'errors.unauthorized',    'You are not authorized'),
    ('00000000-0000-0000-0004-000000000039', 'en', 'errors.internal',        'Internal server error'),
    ('00000000-0000-0000-0004-000000000040', 'ar', 'errors.not_found',       'المورد غير موجود'),
    ('00000000-0000-0000-0004-000000000041', 'ar', 'errors.unauthorized',    'غير مصرح لك'),
    ('00000000-0000-0000-0004-000000000042', 'ar', 'errors.internal',        'خطأ في الخادم')
ON CONFLICT (language_code, key) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS localization.translations;
DROP TABLE IF EXISTS localization.languages;
-- +goose StatementEnd
