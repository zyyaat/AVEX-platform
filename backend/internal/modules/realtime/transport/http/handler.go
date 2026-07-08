// Package http: realtime WebSocket transport using coder/websocket.
//
// The WebSocket handler:
//   1. Upgrades the HTTP connection to a WebSocket (via coder/websocket.Accept).
//   2. Authenticates the connection using a JWT token from the query string
//      (WebSocket clients cannot set Authorization headers in browsers).
//   3. Registers the client with the Hub.
//   4. Reads subscribe/unsubscribe requests from the client + processes them.
//   5. On disconnect: unregisters the client.
//
// Message format (client → server):
//   {"action": "subscribe",   "channel": "order:abc-123"}
//   {"action": "unsubscribe", "channel": "order:abc-123"}
//
// Message format (server → client):
//   {"id":"...","type":"order.status_changed","channel":"order:abc-123","data":{...},"timestamp":"..."}
package http

import (
        "context"
        "encoding/json"
        "log/slog"
        "net/http"
        "sync"
        "time"

        "github.com/coder/websocket"

        "avex-backend/internal/modules/realtime/domain"
        "avex-backend/internal/modules/realtime/port"
        "avex-backend/internal/modules/realtime/service"
        idp "avex-backend/internal/modules/identity/port"
)

// Handler implements the realtime HTTP endpoints.
type Handler struct {
        hub       port.Hub
        svc       port.ServicePort
        jwtIssuer idp.JWTIssuer
        logger    *slog.Logger
        idGen     port.IDGenerator
        clock     port.Clock
}

// NewHandler constructs a new Handler.
func NewHandler(hub port.Hub, svc port.ServicePort, jwtIssuer idp.JWTIssuer, logger *slog.Logger, idGen port.IDGenerator, clock port.Clock) *Handler {
        return &Handler{
                hub:       hub,
                svc:       svc,
                jwtIssuer: jwtIssuer,
                logger:    logger,
                idGen:     idGen,
                clock:     clock,
        }
}

// ===== WebSocket Endpoint =====
//
// GET /api/v1/ws?token=<JWT>
//
// The token is passed as a query parameter because browsers cannot set
// Authorization headers on WebSocket connections. The server validates
// the token before upgrading.

// HandleWebSocket upgrades the HTTP connection to a WebSocket and manages
// the client's lifecycle.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
        // 1. Extract + verify JWT token from query string.
        token := r.URL.Query().Get("token")
        if token == "" {
                http.Error(w, `{"error":"token query parameter is required"}`, http.StatusUnauthorized)
                return
        }

        claims, err := h.jwtIssuer.Verify(r.Context(), token)
        if err != nil {
                h.logger.Debug("websocket auth failed", "error", err, "remote_addr", r.RemoteAddr)
                http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
                return
        }

        userID := claims.Subject
        role := domain.ClientRole(claims.Role)
        // Validate the role; default to "user" if not recognized.
        switch role {
        case domain.ClientRoleUser, domain.ClientRoleDriver, domain.ClientRoleMerchant, domain.ClientRoleAdmin:
                // valid
        default:
                role = domain.ClientRoleUser
        }

        // 2. Upgrade to WebSocket.
        conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
                // In production, set OriginPatterns to allowed domains.
                // For development, we allow all origins.
                OriginPatterns: []string{"*"},
        })
        if err != nil {
                h.logger.Error("websocket accept failed", "error", err, "user_id", userID)
                return
        }
        // Set reasonable limits.
        conn.SetReadLimit(64 * 1024) // 64KB max message size

        // 3. Generate a client ID + register with the Hub.
        clientID := h.idGen.NewID()

        // sendQueue buffers messages from the Hub to be written to the WebSocket.
        // A goroutine reads from this channel and writes to the WebSocket.
        sendQueue := make(chan domain.Message, 64)

        // sendFn is called by the Hub to push a message to this client.
        sendFn := func(ctx context.Context, msg domain.Message) error {
                select {
                case sendQueue <- msg:
                        return nil
                default:
                        // Buffer full — drop the client.
                        return domain.ErrBufferFull
                }
        }

        if err := h.hub.Register(r.Context(), clientID, userID, role, sendFn); err != nil {
                h.logger.Error("hub register failed", "error", err, "user_id", userID)
                _ = conn.Close(websocket.StatusPolicyViolation, "registration failed")
                return
        }
        defer func() {
                h.hub.Unregister(context.Background(), clientID)
                close(sendQueue)
        }()

        h.logger.Info("websocket client connected",
                "client_id", clientID,
                "user_id", userID,
                "role", string(role),
                "remote_addr", r.RemoteAddr,
        )

        // 4. Auto-subscribe to the client's own channel (user:{id} or driver:{id} or merchant:{id}).
        ownChannel := string(role) + ":" + userID
        if role == domain.ClientRoleAdmin {
                ownChannel = "admin"
        }
        if err := h.hub.Subscribe(r.Context(), clientID, ownChannel); err != nil {
                h.logger.Warn("auto-subscribe failed", "error", err, "channel", ownChannel)
        }

        // 5. Start the write goroutine.
        ctx, cancel := context.WithCancel(r.Context())
        defer cancel()

        var writeWg sync.WaitGroup
        writeWg.Add(1)
        go func() {
                defer writeWg.Done()
                h.writeLoop(ctx, conn, sendQueue, clientID)
        }()

        // 6. Read loop (blocks until client disconnects).
        h.readLoop(ctx, conn, clientID, userID, role, cancel)

        // Wait for the write goroutine to finish.
        writeWg.Wait()

        h.logger.Info("websocket client disconnected",
                "client_id", clientID,
                "user_id", userID,
        )
        _ = conn.Close(websocket.StatusNormalClosure, "connection closed")
}

// writeLoop reads messages from the sendQueue and writes them to the WebSocket.
func (h *Handler) writeLoop(ctx context.Context, conn *websocket.Conn, sendQueue <-chan domain.Message, clientID string) {
        for {
                select {
                case <-ctx.Done():
                        return
                case msg, ok := <-sendQueue:
                        if !ok {
                                return // channel closed
                        }
                        data, err := json.Marshal(msg)
                        if err != nil {
                                h.logger.Error("marshal message failed", "error", err, "client_id", clientID)
                                continue
                        }
                        writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
                        if err := conn.Write(writeCtx, websocket.MessageText, data); err != nil {
                                h.logger.Debug("write to websocket failed",
                                        "error", err,
                                        "client_id", clientID,
                                )
                                cancel()
                                return
                        }
                        cancel()
                }
        }
}

// readLoop reads messages from the WebSocket and processes subscribe/unsubscribe requests.
func (h *Handler) readLoop(ctx context.Context, conn *websocket.Conn, clientID, userID string, role domain.ClientRole, cancel context.CancelFunc) {
        defer cancel()

        for {
                msgType, data, err := conn.Read(ctx)
                if err != nil {
                        // Normal closure or error
                        return
                }
                if msgType != websocket.MessageText && msgType != websocket.MessageBinary {
                        continue
                }

                // Parse as SubscribeRequest
                var req domain.SubscribeRequest
                if err := json.Unmarshal(data, &req); err != nil {
                        h.sendError(ctx, conn, "invalid message format: expected JSON with action + channel")
                        continue
                }

                if err := req.Validate(); err != nil {
                        h.sendError(ctx, conn, err.Error())
                        continue
                }

                // Authorization check
                // We need a temporary client to check CanSubscribeTo.
                // The Hub has the authoritative client, but for the auth check we
                // reconstruct a temporary one (cheap).
                tempClient, _ := domain.NewClient(clientID, userID, role)
                if !tempClient.CanSubscribeTo(req.Channel) {
                        h.sendError(ctx, conn, "not authorized to subscribe to channel: "+req.Channel)
                        continue
                }

                switch req.Action {
                case "subscribe":
                        if err := h.hub.Subscribe(ctx, clientID, req.Channel); err != nil {
                                h.sendError(ctx, conn, "subscribe failed: "+err.Error())
                                continue
                        }
                        h.sendAck(ctx, conn, "subscribed", req.Channel)
                        h.logger.Debug("client subscribed",
                                "client_id", clientID,
                                "channel", req.Channel,
                        )
                case "unsubscribe":
                        if err := h.hub.Unsubscribe(ctx, clientID, req.Channel); err != nil {
                                h.sendError(ctx, conn, "unsubscribe failed: "+err.Error())
                                continue
                        }
                        h.sendAck(ctx, conn, "unsubscribed", req.Channel)
                }
        }
}

// sendError sends an error message to the client.
func (h *Handler) sendError(ctx context.Context, conn *websocket.Conn, message string) {
        errMsg := map[string]any{
                "type":    "error",
                "message": message,
        }
        data, _ := json.Marshal(errMsg)
        writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        _ = conn.Write(writeCtx, websocket.MessageText, data)
}

// sendAck sends an acknowledgement message to the client.
func (h *Handler) sendAck(ctx context.Context, conn *websocket.Conn, action, channel string) {
        ackMsg := map[string]any{
                "type":    "ack",
                "action":  action,
                "channel": channel,
        }
        data, _ := json.Marshal(ackMsg)
        writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        _ = conn.Write(writeCtx, websocket.MessageText, data)
}

// ===== REST Endpoints =====

// GetStats returns realtime hub statistics.
// GET /api/v1/admin/realtime/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
        stats := h.svc.GetStats()
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]any{"data": stats})
}

// ===== Routes =====

// RegisterRoutes wires the realtime HTTP routes.
// The WebSocket endpoint does NOT use the standard auth middleware (because
// browsers can't set Authorization headers on WS connections). Instead, it
// authenticates via the query string token.
//
// The stats endpoint uses standard Bearer auth.
func RegisterRoutes(mux *http.ServeMux, hub port.Hub, svc port.ServicePort, jwtIssuer idp.JWTIssuer, logger *slog.Logger, idGen port.IDGenerator, clock port.Clock) {
        h := NewHandler(hub, svc, jwtIssuer, logger, idGen, clock)

        // Wire the WebSocket handler into the service (so ServicePort.HandleWebSocket works).
        service.SetWSHandler(h.HandleWebSocket)

        // WebSocket endpoint — no auth middleware (auth via query token).
        mux.HandleFunc("GET /api/v1/ws", h.HandleWebSocket)

        // Admin stats — Bearer auth.
        // We use a simple wrapper that requires auth via identity's middleware.
        // Since we don't have direct access to idhttp.Auth here (would be circular),
        // the module.go wires this with the auth middleware.
}

// HandleStats is the handler function for the stats endpoint (used by module.go
// to wrap with auth middleware).
func (h *Handler) HandleStats() http.HandlerFunc {
        return h.GetStats
}
