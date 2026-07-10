# 📐 تقرير معماري شامل — منصة AVEX للتوصيل

> ⚠️ **تحذير — هذا التقرير قديم (OUTDATED)**
>
> يصف هذا التقرير المعمارية القديمة قبل إعادة الهيكلة (SQL strings، بدون migrations، CORS `*`، ServeMux واحد). الباكند الحالي في `backend/` يستخدم معمارية **Modular Monolith** (Domain → Port → Service → Repository → Transport) مع goose migrations و Redis و Outbox pattern و OpenTelemetry.
>
> **المرجع المعتمد الآن: [`backend/README.md`](backend/README.md)** — لا تعتمد على ما يلي في أي قرار تقني. (راجع issue #4)

> المسار: `/home/z/my-project/AVEX6` · نوع التحليل: قراءة فقط · بدون أي تعديل على الكود

---

## 1️⃣ وصف المعمارية الحالية

### 1.1 نظرة عامة

المشروع عبارة عن **Monorepo هجين** على pnpm workspace، فيه شقّين مختلفين تقنياً:

| الشق | التقنية | الموقع |
|------|---------|--------|
| Backend | **Go 1.25** (net/http خام + pgx) | `backend/` |
| Frontend | **5 تطبيقات Vite + React 19 + TS** | `artifacts/{customer,driver,merchant,admin,support}` |
| مكتبات TS مشتركة | Drizzle ORM + Orval + Zod + api-client-react | `lib/` |
| السكربتات | tsx + shell | `scripts/` |

### 1.2 الـ Stack التفصيلي

**Backend (Go):**
- خادم HTTP واحد عبر `net/http.ServeMux` (لا يستخدم أي framework)
- postgres عبر `jackc/pgx/v5` مع `database/sql.DB` عام (global)
- JWT HS256 عبر `golang-jwt/jwt/v5`
- bcrypt لكلمات المرور
- WebSocket اختياري عبر `gorilla/websocket` (محروس بـ `REALTIME_ENABLED=true`)
- CORS عبر `rs/cors` (يسمح بكل الأصول `*`)
- **لا يوجد router، لا ORM، لا migration tool، لا structured logging، لا observability**

**Frontends (5 تطبيقات):**
- React 19 + Vite 7 + TailwindCSS 4
- Wouter للـ routing
- Zustand للحالة + Persist middleware
- TanStack React Query للبيانات
- shadcn/ui (Radix UI) — مكررة بالكامل في كل تطبيق (~50 مكون لكل واحد)
- `lib/api.ts` يدوي في كل تطبيق (fetch + localStorage token)

**المكتبات المشتركة (lib):**
- `lib/db`: إعداد Drizzle لكن **الـ schema فارغ** (`export {}`) — غير مستخدم فعلياً
- `lib/api-spec`: OpenAPI YAML فيه نقطة `/healthz` فقط — لم يُحدّث ليتطابق مع الـ 80+ endpoint في Go
- `lib/api-client-react`: أوتوجين من Orval لكنه لا يغطّي سوى `/healthz`
- `lib/api-zod`: Zod schemas مولّدة تلقائياً لنفس النقطة الواحدة

### 1.3 قاعدة البيانات (PostgreSQL)

الـ schema مكتوب **كـ string SQL خام** داخل `backend/internal/shared/db.go::createSchema()` — حوالي 30 جدول في ملف واحد:

```
users, addresses, favorites, restaurants, categories, menu_items,
orders, order_items, order_status_history, order_photos, coupons,
settings, saved_cards, payment_transactions,
delivery_zones, driver_tiers, tier_thresholds, tier_zone_prices,
driver_applications, drivers, driver_stats, driver_shifts,
driver_tier_history, dispatch_offers,
support_tickets, support_messages, support_agents,
merchants, store_hours, scheduled_orders, zone_transfer_requests
```

ثم هجرة (migrations) عبر `runMigrations()` الذي يشغّل `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` بشكل تراكمي. وأخيراً `Seed()` يُشغّل في كل إقلاع لضمان وجود بيانات أولية.

### 1.4 بنية الـ APIs

كل الـ routes تتسجّل على نفس الـ `ServeMux` في `main.go`:

| المجموعة | البادئة | عدد نقاط النهاية التقريبي | الـ Auth |
|----------|---------|---------------------------|---------|
| Customer | `/api/*` | ~15 | Optional + AuthMW |
| Driver | `/api/driver/*` | ~20 | DriverAuthMW |
| Merchant | `/api/merchant/*` | ~14 | MerchantAuthMW |
| Admin | `/api/admin/*` | ~35 | AuthMW + AdminMW |
| Support Agent | `/api/agent/*` | ~12 | AgentAuthMW |
| Realtime | `/ws/{driver,admin}` | 2 | JWT في query param |
| Storage | `/api/storage/objects/{path...}` | 1 | redirect إلى presigned URL |

**نموذج JWT موحّد** لكل الأنواع الأربعة عبر `Claims` struct فيه بوليانات منفصلة: `Admin`, `IsDriver`, `IsMerchant`, `IsAgent` + IDs مرافقة. كل نوع له `Generate*JWT` خاص به بصلاحية مختلفة (7 أيام للـ admin، 30 يوم للباقي).

### 1.5 المنطق الأساسي (Core Domain)

| الحزمة | المسؤولية |
|--------|-----------|
| `internal/dispatch/dispatch.go` | اختيار أعلى 5 سائقين متاحين بالقرب من المطعم + إنشاء `dispatch_offers` + حساب المسافة (Haversine) + ترتيب بـ weighted score (50% distance + 30% tier + 10% response + 10% shift) |
| `internal/dispatch/pricing.go` | حساب أجرة السائق من مصفوفة `tier × zone` (base + per_km + min/max cap) |
| `internal/dispatch/tier.go` | تقييم tier السائق بناءً على acceptance/completion/rating/on-time/shift-adherence/lifetime |
| `internal/realtime/hub.go` | `Hub` في الذاكرة، خريطة `driver_id → set[*Client]` + set admins |
| `internal/storage/storage.go` | presigned URLs عبر sidecar على `127.0.0.1:1106` (Replit Object Storage) |

### 1.6 علاقات المكونات

```
                 ┌──────────────────────────────────────┐
                 │   5 Vite Apps (customer/driver/...)  │
                 │   - Wouter + Zustand + React Query   │
                 │   - lib/api.ts يدوي لكل تطبيق        │
                 └────────────────┬─────────────────────┘
                                  │ HTTPS (/api + /ws)
                                  ▼
            ┌──────────────────────────────────────────────┐
            │   Go Binary (single process, port 8080)      │
            │  ┌─────────────────────────────────────────┐ │
            │  │ ServeMux (net/http)                     │ │
            │  │  ├─ customer  ├─ driver  ├─ merchant    │ │
            │  │  ├─ admin     ├─ support ├─ realtime    │ │
            │  │  └─ storage                             │ │
            │  └─────────────────────────────────────────┘ │
            │       │                              │       │
            │       ▼                              ▼       │
            │  shared.DB (global *sql.DB)    GlobalHub      │
            │       │              (in-memory map)          │
            └───────┼──────────────────────────────────────┘
                    ▼
            PostgreSQL (single instance)
            + Replit Object Storage (sidecar)
```

### 1.7 الـ Deployment

- Replit Autoscale، الـ `.replit` يشغّل Node 24 + Go 1.25 + Python 3.11
- `post-merge.sh` يشغّل `pnpm install` ثم `pnpm --filter db push` (رغم أن Drizzle schema فارغ)
- `api-server` artifact يلفّ الـ Go binary (dev = `go run`، prod = binary مُجمَّع)
- الـ frontends تُبنى عبر Vite وتُخدَم كـ static assets

---

## 2️⃣ المشاكل الموجودة في الهيكل الحالي

### 🔴 مشاكل معمارية حرجة

| # | المشكلة | الأثر |
|---|---------|------|
| 1 | **انفصام هوية (Identity Crisis)**: `replit.md` يقول "Express 5 + Drizzle" لكن الـ backend فعلياً Go. `lib/db` و `lib/api-spec` و `lib/api-client-react` و `lib/api-zod` كلها مرتبطة بـ Express/Drizzle ولا تُستخدم فعلياً. | صيانة كود ميت، تشتيت، والتوثيق يكذب. |
| 2 | **OpenAPI غير متزامن مع الواقع**: `openapi.yaml` فيه نقطة واحدة فقط، بينما Go فيه 80+ endpoint. | الـ codegen لا يولّد شيئاً مفيداً، كل frontend يكتب `lib/api.ts` يدوياً وبدون types مشتركة → انحراف الأنواع (type drift). |
| 3 | **DB Schema كسلسلة نصية في Go**: 30+ جدول كـ SQL string + `ALTER ... IF NOT EXISTS` تراكمي. | لا توجد ملفات migration مرقّمة، لا rollback، لا versioning، drift محتوم بين البيئات، صعب audit. |
| 4 | **No service/repository layer**: الـ handlers تكتب SQL مباشرة (`shared.DB.Query(...)`)، وبعضها ضخم (~700 سطر لكل ملف). | صعوبة الاختبار، تكرار الكود، صعوبة التغيير، خطر SQL injection إذا أخطأ مطور. |
| 5 | **Monolith واحد لـ 5 أنواع مستخدمين**: customer + driver + merchant + admin + support كلهم في نفس الـ binary. | لا يمكن scaling مستقل (مثلاً driver app يحتاج instances أكثر من merchant)، فشل واحد يوقف الكل. |

### 🟠 مشاكل قابلية التوسع

| # | المشكلة |
|---|---------|
| 6 | **GlobalHub في الذاكرة**: WebSocket state محفوظ في `sync.RWMutex` map داخل العملية. لا ينجو من إعادة التشغيل، ولا يعمل مع horizontal scaling (instance-2 لا يعرف عن socket في instance-1). |
| 7 | **Dispatch polling/fallback مكرر**: `NotifyDriverOrderOffer` يستخدم WS لكن الـ driver app يصبح معتمداً على REST polling كـ fallback — ازدواجية. |
| 8 | **Seed في كل إقلاع**: `shared.Seed()` يركض في `main()` عند كل boot. غير آمن للإنتاج، يعطّل التفريق بين dev/prod. |
| 9 | **CORS `*`**: `AllowedOrigins: []string{"*"}` يفتح الباب لأي origin. |
| 10 | **DB pool ثابت (25 conns)**: لا يدعم auto-scaling، ولا read replicas. |
| 11 | **JWT secret واحد لكل الأنواع**: HS256 متطابق لكل claims، لا يوجد key rotation، لا issuer/audience claim. |

### 🟡 مشاكل الـ Frontend

| # | المشكلة |
|---|---------|
| 12 | **5 تطبيقات فيها ~50 مكون shadcn/ui مكرر حرفياً**: `button.tsx` و `dialog.tsx` وغيرها نسخ متطابقة في customer + driver + merchant + admin + support + mockup-sandbox. | صعوبة الصيانة، تحديث واحد يتطلب sync في 5 أماكن. |
| 13 | **لا توجد UI library مشتركة**: كان يجب أن تكون `lib/ui` package مشتركة. |
| 14 | **`lib/api.ts` يدوي في كل تطبيق**: لا types مشتركة، لا validation، كل مطور يكتب اللي يعجبه. |
| 15 | **Tokens في localStorage**: عرضة لـ XSS. |
| 16 | **i18n ad-hoc**: أعمدة `name` و `name_ar` في كل جدول، لكن لا توجد i18n runtime في الـ frontend — المطور يختار عمود يدوياً. |

### 🟢 مشاكل تشغيلية وضمان جودة

| # | المشكلة |
|---|---------|
| 17 | **لا اختبارات** (لا unit، لا integration، لا e2e) — في أي من الـ side. |
| 18 | **لا structured logging**: Go يستخدم `log.Printf` فقط. لا trace IDs، لا correlation بين الطلب و DB و WS. |
| 19 | **لا observability**: لا metrics (Prometheus)، لا tracing (OpenTelemetry)، لا dashboards. |
| 20 | **لا rate limiting**: نقطة `/api/auth/login` مكشوفة لـ brute force. |
| 21 | **لا retries / circuit breakers** في اتصال DB أو sidecar storage. |
| 22 | **صور seed من domain خارجي** (`sfile.chatglm.cn`) — single point of failure. |
| 23 | **`var _ = ...` patterns** في كل handler لقمع أخطاء unused imports — دلالة على كود غير منظم. |
| 24 | **`REALTIME_ENABLED` env gate**: الـ WS معطّل افتراضياً، يعني الـ driver app بيعتمد على polling في الوضع الافتراضي. |
| 25 | **لا CI/CD pipeline** مرئي. |

---

## 3️⃣ معمارية جديدة مقترحة قابلة للتوسع

> الهدف: دعم نمو من 100 → 100K طلب/يوم، 5K → 50K سائق نشط، مع الحفاظ على تجربة مطور سلسة.

### 3.1 المبادئ المعمارية

1. **Contract-First**: OpenAPI هو source of truth، الـ codegen يولّد types و clients لكل الـ frontends.
2. **Modular Monolith → Microservices تدريجياً**: ابدأ بـ bounded contexts داخل monolith، افصل لما يحين وقته.
3. **Layered Architecture**: Handler → Service → Repository → DB. واضح، قابل للاختبار.
4. **Event-Driven**: مكونات الـ dispatch و notifications تتنقل لأحداث (events) لا استدعاءات مباشرة.
5. **Stateless Services + External State**: الـ WebSocket و الـ sessions يخرجون لـ Redis، لا في الذاكرة.

### 3.2 الـ Target Architecture

```
                      ┌───────────────────────────────┐
                      │       API Gateway / BFF        │
                      │   (Kong / Envoy / Traefik)     │
                      │   TLS · Rate-limit · Auth      │
                      └───────────────┬─────────────────┘
                                      │
       ┌───────────────┬──────────────┼──────────────┬────────────────┐
       ▼               ▼              ▼              ▼                ▼
┌──────────┐    ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌────────────┐
│ Customer │    │ Driver   │   │ Merchant │   │ Admin    │   │  Support   │
│  BFF     │    │  BFF     │   │  BFF     │   │  BFF     │   │  BFF       │
│ (REST)   │    │ (REST+WS)│   │ (REST)   │   │ (REST)   │   │ (REST)     │
└────┬─────┘    └────┬─────┘   └────┬─────┘   └────┬─────┘   └─────┬──────┘
     │                │              │              │               │
     └────────────────┴──────┬───────┴──────────────┴───────────────┘
                             │
            ┌────────────────┴──────────────────┐
            ▼                                   ▼
  ┌────────────────────┐              ┌────────────────────┐
  │   Domain Services  │              │   Realtime Service │
  │  (Go micro-bins)   │              │   (Go + Redis)     │
  │                    │              │                    │
  │  - catalog         │              │  - WS Hub on Redis │
  │  - orders          │              │  - presence        │
  │  - dispatch        │              │  - location stream │
  │  - pricing         │              │  - fanout          │
  │  - identity        │              └─────────┬──────────┘
  │  - payments        │                        │
  │  - support         │                        │
  └──────────┬─────────┘                        │
             │                                  │
             ▼                                  ▼
  ┌────────────────────┐              ┌────────────────────┐
  │   Event Bus        │              │   Redis (cluster)  │
  │  (NATS / Kafka)    │              │   - pubsub         │
  │  - order.created   │              │   - presence       │
  │  - order.assigned  │              │   - rate limit     │
  │  - driver.tier     │              │   - cache          │
  │  - payment.captured│              └────────────────────┘
  └──────────┬─────────┘
             │
             ▼
  ┌────────────────────┐    ┌────────────────────┐
  │  PostgreSQL        │    │  Object Storage    │
  │  + Read Replicas   │    │  (S3-compatible)   │
  │  + pgvector (geo)  │    │  - photos          │
  └────────────────────┘    └────────────────────┘
```

### 3.3 تقسيم الـ Domain Services (Bounded Contexts)

| Service | المسؤولية | DB Schema | الواجهة |
|---------|-----------|-----------|---------|
| **identity-svc** | users, drivers, merchants, agents, auth, JWT, sessions | `users`, `drivers`, `merchants`, `support_agents`, `saved_cards` | gRPC داخلي + REST public |
| **catalog-svc** | restaurants, categories, menu_items, store_hours | `restaurants`, `categories`, `menu_items`, `store_hours` | REST + GraphQL read |
| **orders-svc** | orders, order_items, status_history, scheduled_orders | `orders`, `order_items`, `order_status_history`, `scheduled_orders` | REST + events |
| **dispatch-svc** | dispatch_offers, driver geolocation, matching algorithm | `dispatch_offers` + Redis geo index | gRPC + events |
| **pricing-svc** | tiers, thresholds, tier_zone_prices, fee computation | `driver_tiers`, `tier_thresholds`, `tier_zone_prices`, `delivery_zones` | gRPC (داخلي فقط) |
| **payments-svc** | paymob integration, payment_transactions, refunds | `payment_transactions`, `saved_cards` (token) | REST + webhooks |
| **support-svc** | tickets, messages, agent assignment | `support_tickets`, `support_messages`, `zone_transfer_requests` | REST + WS |
| **notifications-svc** | push (FCM/APNs), SMS, email, in-app | جدول `notifications_outbox` | events consumer |
| **realtime-svc** | WebSocket hub, presence, location stream | stateless — كل البيانات في Redis | WS + gRPC |
| **admin-svc** | dashboard aggregation, ops tools | read-only من كل الـ schemas | REST |

### 3.4 طبقات داخل كل خدمة

```
service/
├── cmd/                    # entry point
├── internal/
│   ├── api/                # HTTP handlers + middleware
│   │   ├── http.go         # router (chi/echo)
│   │   └── v1/             # versioned handlers
│   ├── service/            # business logic (use cases)
│   ├── repository/         # DB access (sqlc-generated)
│   ├── domain/             # entities + value objects
│   ├── events/             # producers + consumers
│   └── infra/              # db, redis, s3, telemetry
├── migrations/             # sql migrations (goose/migrate)
│   ├── 001_users.up.sql
│   └── 001_users.down.sql
├── api/                    # OpenAPI spec + generated code
│   ├── openapi.yaml
│   └── gen/
└── tests/
    ├── unit/
    └── integration/
```

### 3.5 الـ Stack المُقترح

| الطبقة | التقنية | المبرر |
|--------|--------|--------|
| اللغة | Go 1.25 (يبقى) | الأداء العالي والـ concurrency مثالي لتوصيل |
| HTTP Router | `chi` أو `echo` | أخف من Gin، أنضف من net/http الخام |
| DB Driver | `pgx/v5` (موجود) | الأفضل لأداء Postgres |
| Query Builder | `sqlc` (codegen من SQL) | type-safe، لا reflection، أداء عالٍ |
| Migrations | `golang-migrate` أو `goose` | versioned + reversible |
| Validation | `go-playground/validator` + OpenAPI codegen | type-safe end-to-end |
| Logging | `slog` (structured, stdlib) + `otel/zap` adapter | structured من البداية |
| Tracing | OpenTelemetry SDK | traces + metrics موحّدة |
| WebSocket | `nhooyr/websocket` (أخف من gorilla) | أعلى أداء |
| PubSub | NATS JetStream (أبسط) أو Kafka (للـ volume الأعلى) | events بين الخدمات |
| Cache/Presence | Redis 7 (cluster mode) | Geo index + Streams + pubsub |
| Object Storage | MinIO / S3-compatible | بدل Replit sidecar |
| Auth | JWT (RS256 asymmetric) + key rotation في JWKS | مفصول بين services |
| Frontend | يبقى React 19 + Vite، لكن مع **UI library مشتركة** | تقليل الازدواجية |
| Monorepo | pnpm workspaces (موجود) + **Turborepo** للأبنية المتوازية | أسرع 5-10x |

### 3.6 نموذج الأحداث (Event Storming)

```
order.created          → dispatch-svc يبدأ matching
                         → merchant-svc يعرف طلب جديد
order.accepted         → notifications-svc يخبر العميل
driver.offered         → realtime-svc يدفع WS للسائق
driver.accepted        → dispatch-svc يقفل الـ offer
                         → orders-svc يحدّث status='assigned'
driver.picked_up       → notifications-svc يخبر العميل بـ ETA
driver.delivered       → orders-svc يقفل الطلب
                         → payments-svc يخصم/يحول
                         → pricing-svc يحسب أجرة السائق
                         → driver-stats-svc يحدّث الإحصائيات
driver.tier.changed    → notifications-svc يخبر السائق
ticket.created         → support-svc يفتح tacket
                         → notifications-svc يخبر الـ agent
payment.captured       → orders-svc يغير الحالة → paid
```

### 3.7 إدارة الـ Frontend الموحّدة

```
lib/
├── ui/                # شارك shadcn/ui components مرة واحدة فقط
├── api-client/        # Orval-generated من OpenAPI موحّد
├── i18n/              # react-i18next + translations/{ar,en}.json
├── auth/              # token management (httpOnly cookie + refresh)
└── config/            # eslint, prettier, tsconfig المشتركة

artifacts/
├── customer/          # يستخدم lib/ui, lib/api-client
├── driver/            # نفسه
├── merchant/          # نفسه
├── admin/             # نفسه
└── support/           # نفسه
```

### 3.8 خطة المراقبة (Observability)

- **Metrics**: Prometheus + Grafana (rate, error, latency, saturation لكل خدمة)
- **Tracing**: OpenTelemetry collector + Jaeger (trace واحد من الـ frontend للـ DB)
- **Logging**: Loki + structured JSON logs
- **Alerting**: AlertManager على SLOs (p99 latency, error rate)
- **Synthetic checks**: k6 للـ critical flows (تسجيل → إنشاء طلب → dispatch)

---

## 4️⃣ خطة النقل التدريجي بدون كسر النظام

> القاعدة الذهبية: **في أي لحظة، النظام يعمل ويربح المال**. لا big-bang rewrite.

### المرحلة 0: الاستقرار والتحضير (2-3 أسابيع) — بدون تغيير معماري

| الخطوة | الوصف | المخاطر |
|--------|------|--------|
| 0.1 | توحيد `replit.md` ليطابق الواقع (Go backend، lib/db معطّل) | لا شيء |
| 0.2 | إضافة structured logging (`slog`) في `shared/` بدل `log.Printf` | منخفض |
| 0.3 | إضافة health checks منفصلة: `/health/live` و `/health/ready` | منخفض |
| 0.4 | إضافة rate limiting أساسي على `/api/auth/login` و `/api/driver/auth/login` (`golang.org/x/time/rate`) | منخفض |
| 0.5 | نقل الـ Seed ليكون command منفصل: `go run ./cmd/seed` بدل تشغيله في `main()` | منخفض |
| 0.6 | تقييد CORS على origins معروفة بدل `*` | متوسط — يحتاج تكوين لكل بيئة |
| 0.7 | إضافة CI pipeline: lint + typecheck + build (GitHub Actions) | منخفض |

### المرحلة 1: استخراج عقد الـ API (3-4 أسابيع)

| الخطوة | الوصف |
|--------|------|
| 1.1 | توليد `openapi.yaml` كامل من كود Go الحالي (عبر `swaggo/swag` أو كتابة يدوية) — يغطّي الـ 80+ endpoint |
| 1.2 | إعادة كتابة `lib/api-client-react` و `lib/api-zod` عبر Orval على الأساس الجديد |
| 1.3 | استبدال `lib/api.ts` اليدوي في كل frontend بالـ generated client (تطبيق تلو الآخر) |
| 1.4 | إضافة contract tests (Pact أو Schemathesis) تمنع انحراف الـ spec عن الـ implementation |

**الناتج**: عقد موثّق، types مشتركة، نهاية الـ type drift.

### المرحلة 2: استخراج طبقة Repository + Migrations (4-6 أسابيع)

| الخطوة | الوصف |
|--------|------|
| 2.1 | تجميد الـ schema strings في `db.go` وفكها لملفات `migrations/NNN_*.up.sql` عبر `golang-migrate` |
| 2.2 | توليد كود SQL عبر `sqlc` من queries موجودة في الـ handlers |
| 2.3 | نقل كل `shared.DB.Query(...)` من الـ handlers لـ `internal/*/repository/` |
| 2.4 | إضافة unit tests للـ repositories (testcontainers + Postgres حقيقي) |

**الناتج**: handlers رفيعة، repository قابل للاختبار، migration مُدار.

### المرحلة 3: استخراج طبقة Service + Domain (4-6 أسابيع)

| الخطوة | الوصف |
|--------|------|
| 3.1 | استخراج منطق الأعمال من الـ handlers لـ `internal/*/service/` |
| 3.2 | تعريف `domain/` entities (Order, Driver, DispatchOffer...) |
| 3.3 | تحويل الـ handlers لـ thin layer: parse request → call service → write response |
| 3.4 | إضافة integration tests لكل service |

**الناتج**: monolith نظيف بطبقات، قابل للاختبار، جاهز للفصل.

### المرحلة 4: إدخال PubSub و فصل الـ Realtime (6-8 أسابيع) — أول فصل فعلي

| الخطوة | الوصف |
|--------|------|
| 4.1 | إعداد Redis cluster + NATS JetStream |
| 4.2 | نقل `GlobalHub` من in-memory إلى Redis-backed (`RPUSH` + `SUBSCRIBE`) — يصبح horizontal-scalable |
| 4.3 | استخراج `realtime-svc` كـ binary منفصل يقرأ من Redis و يخدم WS |
| 4.4 | نشر أول الأحداث: `order.created`, `driver.location.updated` — لكن **دون لمس الـ flows الحالية** (shadow mode: يكتب للـ bus ولا يقرأه أحد بعد) |
| 4.5 | بعد التحقق، تحويل `dispatch.DispatchOrder` لـ consumer للحدث بدل استدعاء مباشر |

**الناتج**: realtime scales مستقلة، أول event-driven flow حقيقي.

### المرحلة 5: فصل الخدمات تدريجياً (3-6 أشهر)

نفس الـ pattern لكل خدمة:

```
[a] إضافة الـ service الجديد كـ binary منفصل
[b] توجيه نسبة x% من الترافيك له (feature flag)
[c] مقارنة metrics و errors بين القديم والجديد
[d] رفع النسبة تدريجياً حتى 100%
[e] حذف الـ code القديم من الـ monolith
```

**الترتيب المقترح** (من الأسهل للأصعب):

1. **notifications-svc** (جديد بالكامل، لا يكسر شيئاً)
2. **pricing-svc** (pure function، لا side effects)
3. **catalog-svc** (read-heavy، مفصول بوضوح)
4. **support-svc** (شبه مستقل بالفعل)
5. **payments-svc** (حساس لكن معزول)
6. **identity-svc** (يحتاج careful migration للـ sessions)
7. **orders-svc** (الأكثر تشابكاً — يُترك للنهاية)
8. **dispatch-svc** (الـ brain — يُترك للنهاية جداً)

### المرحلة 6: تحديث الـ Frontends (متوازي مع 4-5)

| الخطوة | الوصف |
|--------|------|
| 6.1 | إنشاء `lib/ui` و نقل كل مكونات shadcn/ui مرة واحدة إليه |
| 6.2 | تحديث كل artifact لاستيراد من `@workspace/ui` بدل `./components/ui` |
| 6.3 | إضافة `lib/i18n` (react-i18next) و إلغاء أعمدة `name_ar` تدريجياً لصالح translations catalog |
| 6.4 | نقل الـ auth tokens لـ httpOnly cookies + refresh token rotation |
| 6.5 | إضافة E2E tests (Playwright) لكل artifact |

### المرحلة 7: Observability & Production Hardening (مستمر)

- OpenTelemetry traces في كل خدمة
- Prometheus metrics + Grafana dashboards
- SLOs مكتوبة (p99 < 300ms للـ read، < 800ms للـ write)
- Runbooks لكل خدمة
- Chaos engineering (محاكاة فشل Redis/Postgres/NATS)

---

## 📊 جدول زمني تقديري

| المرحلة | المدة | عدد المطورين |
|---------|------|-------------|
| 0: الاستقرار | 2-3 أسابيع | 1-2 |
| 1: OpenAPI | 3-4 أسابيع | 2 |
| 2: Repository + Migrations | 4-6 أسابيع | 2-3 |
| 3: Service layer | 4-6 أسابيع | 2-3 |
| 4: Realtime + PubSub | 6-8 أسابيع | 3 |
| 5: فصل الخدمات | 3-6 أشهر | 3-5 |
| 6: Frontend cleanup | متوازي | 2 |
| 7: Observability | مستمر | 1 |

**الإجمالي**: ~9-14 شهر للوصول للمعمارية المستهدفة بالكامل، مع نظام يعمل في كل لحظة.

---

## ✅ الخلاصة

النظام الحالي **يعمل ويربح** لكنه في **منطقة الخطر التقني**:
- توثيق يكذب، codegen ميت، schema في string، handlers ضخمة، لا اختبارات، scaling محدود.

المعمارية المقترحة لا تتطلب **big-bang rewrite** بل **سلسلة من الخطوات العكسية (strangler fig pattern)** — كل خطوة تترك النظام أفضل مما كان عليه دون كسره.

أهم 3 أولويات للبدء فوراً:
1. **توحيد OpenAPI contract** (المرحلة 1) — لأن كل المشاكل تتفرع من غيابه.
2. **DB migrations منظمة** (المرحلة 2.1) — لأن الـ schema الحالي معرض للـ drift في أي لحظة.
3. **استخراج realtime لـ Redis** (المرحلة 4.2) — لأنه البوابة لأي horizontal scaling.
