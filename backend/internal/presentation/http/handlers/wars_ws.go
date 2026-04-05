package handlers

// wars_ws.go — WebSocket real-time leaderboard hub (spec §3.5 Phase 3)
//
// Architecture:
//   - LeaderboardHub: central broadcaster; holds all active WS connections
//   - WarsHandler.LiveLeaderboard: upgrades HTTP → WebSocket and registers client
//   - Clients receive a full leaderboard snapshot on connect, then live diffs
//     whenever BroadcastLeaderboard() is called (by lifecycle worker after
//     any recharge event or on a 30-second poll ticker)
//
// Route:  GET /api/v1/wars/live   (WebSocket upgrade)
// Client sends: nothing (read-only stream)
// Server sends JSON messages:
//   {"type":"snapshot","data":[...entries...],"period":"2026-03"}
//   {"type":"update",  "data":[...entries...],"period":"2026-03"}
//
// Connection lifecycle:
//   1. Client connects → hub registers client
//   2. Hub sends full snapshot immediately
//   3. Hub broadcasts "update" on every leaderboard change (≤ 30s latency)
//   4. Client disconnects → hub unregisters client (no goroutine leak)
//
// Concurrency: all map mutations are guarded by a mutex; each client has its
// own buffered channel so a slow client cannot block the broadcaster.

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"      //nolint:staticcheck // nhooyr.io/websocket is maintained as github.com/coder/websocket; API identical
	"nhooyr.io/websocket/wsjson" //nolint:staticcheck

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
)

// ─── Hub ─────────────────────────────────────────────────────────────────────

// LeaderboardHub manages all active WebSocket clients for the leaderboard.
// Create a single instance and pass it to WarsHandler + LifecycleWorker.
type LeaderboardHub struct {
	mu      sync.RWMutex
	clients map[*leaderboardClient]struct{}
}

// NewLeaderboardHub creates an initialised, ready-to-use hub.
func NewLeaderboardHub() *LeaderboardHub {
	return &LeaderboardHub{
		clients: make(map[*leaderboardClient]struct{}),
	}
}

type leaderboardClient struct {
	send chan wsLeaderboardMsg
}

type wsLeaderboardMsg struct {
	Type    string                    `json:"type"`   // "snapshot" | "update"
	Period  string                    `json:"period"` // "YYYY-MM"
	Data    []entities.LeaderboardEntry `json:"data"`
	SentAt  int64                     `json:"sent_at"` // unix ms — lets frontend detect stale renders
}

func (h *LeaderboardHub) register(c *leaderboardClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *LeaderboardHub) unregister(c *leaderboardClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.send)
}

// BroadcastLeaderboard sends a fresh snapshot to every connected client.
// Safe to call from any goroutine; non-blocking (slow clients are dropped).
func (h *LeaderboardHub) BroadcastLeaderboard(entries []entities.LeaderboardEntry, period string, msgType string) {
	msg := wsLeaderboardMsg{
		Type:   msgType,
		Period: period,
		Data:   entries,
		SentAt: time.Now().UnixMilli(),
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
			// client channel full — they are too slow; drop this frame
			log.Printf("[WarsWS] client channel full — dropping frame for 1 client")
		}
	}
}

// ConnectedClients returns the current number of active WebSocket connections.
func (h *LeaderboardHub) ConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ─── Handler ─────────────────────────────────────────────────────────────────

// LiveLeaderboard upgrades an HTTP connection to WebSocket and streams
// leaderboard updates to the client.
//
//	GET /api/v1/wars/live
//	Requires: valid JWT (enforced by auth middleware before this handler)
func (h *WarsHandler) LiveLeaderboard(w http.ResponseWriter, r *http.Request) {
	if h.hub == nil {
		http.Error(w, "WebSocket not configured", http.StatusServiceUnavailable)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{ //nolint:staticcheck
		// Allow any origin in dev; tighten in production via CORS middleware
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("[WarsWS] upgrade failed: %v", err)
		return
	}

	client := &leaderboardClient{
		// Buffer 16 frames so a brief network hiccup doesn't disconnect the client
		send: make(chan wsLeaderboardMsg, 16),
	}
	h.hub.register(client)
	defer h.hub.unregister(client)

	ctx := conn.CloseRead(r.Context()) //nolint:staticcheck

	// Send full snapshot immediately on connect
	entries, err := h.warsSvc.GetLeaderboard(ctx, 37)
	if err == nil {
		snapshot := wsLeaderboardMsg{
			Type:   "snapshot",
			Period: currentWarPeriod(),
			Data:   entries,
			SentAt: time.Now().UnixMilli(),
		}
		if writeErr := wsjson.Write(ctx, conn, snapshot); writeErr != nil {
			log.Printf("[WarsWS] snapshot write failed: %v", writeErr)
			return
		}
	}

	// Stream updates until context cancelled or client disconnects
	for {
		select {
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "done") //nolint:staticcheck
			return
		case msg, ok := <-client.send:
			if !ok {
				_ = conn.Close(websocket.StatusNormalClosure, "hub closed") //nolint:staticcheck
				return
			}
			if err := wsjson.Write(ctx, conn, msg); err != nil {
				log.Printf("[WarsWS] write error — closing client: %v", err)
				return
			}
		}
	}
}

// ─── Polling broadcaster (fallback for when no recharge event fires) ─────────

// StartLeaderboardPoller runs a background goroutine that fetches the leaderboard
// every `interval` and broadcasts it to all connected clients.
// Call this once from cmd/api/main.go after creating the hub.
func StartLeaderboardPoller(ctx context.Context, hub *LeaderboardHub, warsSvc *services.RegionalWarsService, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if hub.ConnectedClients() == 0 {
					continue // no one listening — skip the DB query
				}
				entries, err := warsSvc.GetLeaderboard(ctx, 37)
				if err != nil {
					log.Printf("[WarsWS] poller fetch error: %v", err)
					continue
				}
				hub.BroadcastLeaderboard(entries, currentWarPeriod(), "update")
			}
		}
	}()
}

// ─── WebSocket hub is complete above ─────────────────────────────────────────
