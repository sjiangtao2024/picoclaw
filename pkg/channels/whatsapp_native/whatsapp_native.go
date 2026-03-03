//go:build whatsapp_native

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package whatsapp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
	"rsc.io/qr"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	sqliteDriver   = "sqlite"
	whatsappDBName = "store.db"

	reconnectInitial    = 5 * time.Second
	reconnectMax        = 5 * time.Minute
	reconnectMultiplier = 2.0
)

// WhatsAppNativeChannel implements the WhatsApp channel using whatsmeow (in-process, no external bridge).
type WhatsAppNativeChannel struct {
	*channels.BaseChannel
	config       config.WhatsAppConfig
	storePath    string
	client       *whatsmeow.Client
	container    *sqlstore.Container
	mu           sync.Mutex
	runCtx       context.Context
	runCancel    context.CancelFunc
	reconnectMu  sync.Mutex
	reconnecting bool
	stopping     atomic.Bool    // set once Stop begins; prevents new wg.Add calls
	wg           sync.WaitGroup // tracks background goroutines (QR handler, reconnect)
	pairing      pairingState
	pairingMu    sync.RWMutex
}

type pairingState struct {
	Connected    bool   `json:"connected"`
	NeedsPairing bool   `json:"needs_pairing"`
	Event        string `json:"event"`
	Code         string `json:"code"`
	UpdatedAt    string `json:"updated_at"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	expiresAtUnix int64
}

// NewWhatsAppNativeChannel creates a WhatsApp channel that uses whatsmeow for connection.
// storePath is the directory for the SQLite session store (e.g. workspace/whatsapp).
func NewWhatsAppNativeChannel(
	cfg config.WhatsAppConfig,
	bus *bus.MessageBus,
	storePath string,
) (channels.Channel, error) {
	base := channels.NewBaseChannel("whatsapp_native", cfg, bus, cfg.AllowFrom, channels.WithMaxMessageLength(65536))
	if storePath == "" {
		storePath = "whatsapp"
	}
	c := &WhatsAppNativeChannel{
		BaseChannel: base,
		config:      cfg,
		storePath:   storePath,
		pairing: pairingState{
			Event:     "init",
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}
	return c, nil
}

func (c *WhatsAppNativeChannel) Start(ctx context.Context) error {
	logger.InfoCF("whatsapp", "Starting WhatsApp native channel (whatsmeow)", map[string]any{"store": c.storePath})

	// Reset lifecycle state from any previous Stop() so a restarted channel
	// behaves correctly.  Use reconnectMu to be consistent with eventHandler
	// and Stop() which coordinate under the same lock.
	c.reconnectMu.Lock()
	c.stopping.Store(false)
	c.reconnecting = false
	c.reconnectMu.Unlock()

	if err := os.MkdirAll(c.storePath, 0o700); err != nil {
		return fmt.Errorf("create session store dir: %w", err)
	}

	dbPath := filepath.Join(c.storePath, whatsappDBName)
	connStr := "file:" + dbPath + "?_foreign_keys=on"

	db, err := sql.Open(sqliteDriver, connStr)
	if err != nil {
		return fmt.Errorf("open whatsapp store: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	waLogger := waLog.Stdout("WhatsApp", "WARN", true)
	container := sqlstore.NewWithDB(db, sqliteDriver, waLogger)
	if err = container.Upgrade(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("open whatsapp store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return fmt.Errorf("get device store: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLogger)

	// Create runCtx/runCancel BEFORE registering event handler and starting
	// goroutines so that Stop() can cancel them at any time, including during
	// the QR-login flow.
	c.runCtx, c.runCancel = context.WithCancel(ctx)

	client.AddEventHandler(c.eventHandler)

	c.mu.Lock()
	c.container = container
	c.client = client
	c.mu.Unlock()

	// cleanupOnError clears struct references and releases resources when
	// Start() fails after fields are already assigned.  This prevents
	// Stop() from operating on stale references (double-close, disconnect
	// of a partially-initialized client, or stray event handler callbacks).
	startOK := false
	defer func() {
		if startOK {
			return
		}
		c.runCancel()
		client.Disconnect()
		c.mu.Lock()
		c.client = nil
		c.container = nil
		c.mu.Unlock()
		_ = container.Close()
	}()

	if client.Store.ID == nil {
		c.setPairingState(false, true, "awaiting_qr", "")
		qrChan, err := client.GetQRChannel(c.runCtx)
		if err != nil {
			return fmt.Errorf("get QR channel: %w", err)
		}
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		if err := c.startQREventLoop(qrChan); err != nil {
			return err
		}
	} else {
		c.setPairingState(true, false, "connected", "")
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
	}

	startOK = true
	c.SetRunning(true)
	logger.InfoC("whatsapp", "WhatsApp native channel connected")
	return nil
}

func (c *WhatsAppNativeChannel) Stop(ctx context.Context) error {
	logger.InfoC("whatsapp", "Stopping WhatsApp native channel")

	// Mark as stopping under reconnectMu so the flag is visible to
	// eventHandler atomically with respect to its wg.Add(1) call.
	// This closes the TOCTOU window where eventHandler could check
	// stopping (false), then Stop sets it true + enters wg.Wait,
	// then eventHandler calls wg.Add(1) — causing a panic.
	c.reconnectMu.Lock()
	c.stopping.Store(true)
	c.reconnectMu.Unlock()

	if c.runCancel != nil {
		c.runCancel()
	}

	// Disconnect the client first so any blocking Connect()/reconnect loops
	// can be interrupted before we wait on the goroutines.
	c.mu.Lock()
	client := c.client
	container := c.container
	c.mu.Unlock()

	if client != nil {
		client.Disconnect()
	}

	// Wait for background goroutines (QR handler, reconnect) to finish in a
	// context-aware way so Stop can be bounded by ctx.
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines have finished.
	case <-ctx.Done():
		// Context canceled or timed out; log and proceed with best-effort cleanup.
		logger.WarnC("whatsapp", fmt.Sprintf("Stop context canceled before all goroutines finished: %v", ctx.Err()))
	}

	// Now it is safe to clear and close resources.
	c.mu.Lock()
	c.client = nil
	c.container = nil
	c.mu.Unlock()

	if container != nil {
		_ = container.Close()
	}
	c.SetRunning(false)
	c.setPairingState(false, false, "stopped", "")
	return nil
}

func (c *WhatsAppNativeChannel) eventHandler(evt any) {
	switch evt.(type) {
	case *events.Message:
		c.handleIncoming(evt.(*events.Message))
	case *events.Connected:
		c.setPairingState(true, false, "connected", "")
	case *events.Disconnected:
		c.setPairingState(false, false, "disconnected", "")
		logger.InfoCF("whatsapp", "WhatsApp disconnected, will attempt reconnection", nil)
		c.reconnectMu.Lock()
		if c.reconnecting {
			c.reconnectMu.Unlock()
			return
		}
		// Check stopping while holding the lock so the check and wg.Add
		// are atomic with respect to Stop() setting the flag + calling
		// wg.Wait(). This prevents the TOCTOU race.
		if c.stopping.Load() {
			c.reconnectMu.Unlock()
			return
		}
		c.reconnecting = true
		c.wg.Add(1)
		c.reconnectMu.Unlock()
		go func() {
			defer c.wg.Done()
			c.reconnectWithBackoff()
		}()
	}
}

func (c *WhatsAppNativeChannel) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/whatsapp/pairing", c.handlePairingStatus)
	mux.HandleFunc("/api/whatsapp/pairing/qr.png", c.handlePairingQRPNG)
	mux.HandleFunc("/api/whatsapp/pairing/refresh", c.handlePairingRefresh)
	mux.HandleFunc("/whatsapp/pairing", c.handlePairingPage)
}

func (c *WhatsAppNativeChannel) handlePairingStatus(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{
		"channel":    "whatsapp_native",
		"configured": c.config.Enabled && c.config.UseNative,
	}
	for k, v := range c.getPairingStateMap() {
		resp[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *WhatsAppNativeChannel) handlePairingPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(pairingPageHTML))
}

func (c *WhatsAppNativeChannel) handlePairingQRPNG(w http.ResponseWriter, _ *http.Request) {
	state := c.getPairingState()
	code := strings.TrimSpace(state.Code)
	if code == "" {
		http.Error(w, "pairing code is empty", http.StatusNotFound)
		return
	}
	qrCode, err := qr.Encode(code, qr.M)
	if err != nil {
		http.Error(w, "failed to encode qr", http.StatusInternalServerError)
		return
	}
	qrCode.Scale = 8
	png := qrCode.PNG()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func (c *WhatsAppNativeChannel) handlePairingRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	c.mu.Lock()
	client := c.client
	runCtx := c.runCtx
	c.mu.Unlock()

	if client == nil || runCtx == nil {
		http.Error(w, "whatsapp channel not initialized", http.StatusServiceUnavailable)
		return
	}
	if client.Store.ID != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"message": "already connected",
		})
		return
	}

	if client.IsConnected() {
		client.Disconnect()
	}

	qrChan, err := client.GetQRChannel(runCtx)
	if err != nil {
		http.Error(w, "failed to get qr channel: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := client.Connect(); err != nil {
		http.Error(w, "failed to connect: "+err.Error(), http.StatusBadGateway)
		return
	}
	if err := c.startQREventLoop(qrChan); err != nil {
		http.Error(w, "failed to start qr loop: "+err.Error(), http.StatusInternalServerError)
		return
	}
	c.setPairingState(false, true, "refresh_requested", "")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"message": "refresh triggered",
	})
}

func (c *WhatsAppNativeChannel) setPairingState(connected, needsPairing bool, event, code string) {
	c.pairingMu.Lock()
	defer c.pairingMu.Unlock()
	c.pairing.Connected = connected
	c.pairing.NeedsPairing = needsPairing
	c.pairing.Event = event
	c.pairing.Code = code
	c.pairing.UpdatedAt = time.Now().Format(time.RFC3339)
	if code == "" {
		c.pairing.ExpiresAt = ""
		c.pairing.expiresAtUnix = 0
	}
}

func (c *WhatsAppNativeChannel) setPairingCode(code string, timeout time.Duration) {
	c.pairingMu.Lock()
	defer c.pairingMu.Unlock()
	c.pairing.Connected = false
	c.pairing.NeedsPairing = true
	c.pairing.Event = "code"
	c.pairing.Code = code
	c.pairing.UpdatedAt = time.Now().Format(time.RFC3339)
	if timeout > 0 {
		exp := time.Now().Add(timeout)
		c.pairing.ExpiresAt = exp.Format(time.RFC3339)
		c.pairing.expiresAtUnix = exp.Unix()
	} else {
		c.pairing.ExpiresAt = ""
		c.pairing.expiresAtUnix = 0
	}
}

func (c *WhatsAppNativeChannel) startQREventLoop(qrChan <-chan whatsmeow.QRChannelItem) error {
	// Guard wg.Add with reconnectMu + stopping check (same protocol as
	// eventHandler) so a concurrent Stop() cannot enter wg.Wait while we Add.
	c.reconnectMu.Lock()
	if c.stopping.Load() {
		c.reconnectMu.Unlock()
		return fmt.Errorf("channel stopped during QR setup")
	}
	c.wg.Add(1)
	c.reconnectMu.Unlock()

	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.runCtx.Done():
				return
			case evt, ok := <-qrChan:
				if !ok {
					return
				}
				if evt.Event == "code" {
					c.setPairingCode(evt.Code, evt.Timeout)
					logger.InfoCF("whatsapp", "Scan this QR code with WhatsApp (Linked Devices):", nil)
					qrterminal.GenerateWithConfig(evt.Code, qrterminal.Config{
						Level:      qrterminal.L,
						Writer:     os.Stdout,
						HalfBlocks: true,
					})
				} else {
					c.handlePairingEvent(evt.Event)
					logger.InfoCF("whatsapp", "WhatsApp login event", map[string]any{"event": evt.Event})
				}
			}
		}
	}()
	return nil
}

func (c *WhatsAppNativeChannel) handlePairingEvent(event string) {
	state := c.getPairingState()
	connected := state.Connected
	needsPairing := state.NeedsPairing
	code := state.Code
	switch strings.ToLower(strings.TrimSpace(event)) {
	case "success", "connected":
		connected = true
		needsPairing = false
		code = ""
	case "timeout", "expired":
		connected = false
		needsPairing = true
		code = ""
	}
	c.setPairingState(connected, needsPairing, event, code)
}

func (c *WhatsAppNativeChannel) getPairingState() pairingState {
	c.pairingMu.RLock()
	defer c.pairingMu.RUnlock()
	return c.pairing
}

func (c *WhatsAppNativeChannel) getPairingStateMap() map[string]any {
	s := c.getPairingState()
	var left int64
	if s.expiresAtUnix > 0 {
		left = s.expiresAtUnix - time.Now().Unix()
		if left < 0 {
			left = 0
		}
	}
	return map[string]any{
		"connected":          s.Connected,
		"needs_pairing":      s.NeedsPairing,
		"event":              s.Event,
		"code":               s.Code,
		"updated_at":         s.UpdatedAt,
		"expires_at":         s.ExpiresAt,
		"expires_in_seconds": left,
	}
}

const pairingPageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>WhatsApp Pairing</title>
  <style>
    body { font-family: system-ui, -apple-system, Segoe UI, sans-serif; margin: 0; background: #0b1220; color: #e5e7eb; }
    .wrap { max-width: 680px; margin: 24px auto; padding: 20px; background: #111827; border: 1px solid #1f2937; border-radius: 12px; }
    .title { font-size: 20px; font-weight: 700; margin-bottom: 12px; }
    .sub { color: #9ca3af; margin-bottom: 16px; }
    .row { margin: 8px 0; }
    .pill { display: inline-block; padding: 4px 10px; border-radius: 999px; background: #1f2937; color: #d1d5db; }
    .code { margin-top: 12px; padding: 12px; border-radius: 8px; background: #030712; border: 1px dashed #374151; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; word-break: break-all; }
    #qrcode { margin-top: 16px; display: block; width: 300px; height: 300px; background: #fff; padding: 8px; border-radius: 8px; object-fit: contain; }
    .hint { margin-top: 12px; color: #9ca3af; font-size: 14px; }
    button { margin-top: 10px; border: 1px solid #374151; background: #111827; color: #e5e7eb; border-radius: 8px; padding: 8px 12px; cursor: pointer; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="title">WhatsApp Pairing</div>
    <div class="sub">Auto-refreshes every 2s. Keep this page open while pairing.</div>
    <div class="row">Connection: <span id="connected" class="pill">unknown</span></div>
    <div class="row">Event: <span id="event" class="pill">init</span></div>
    <div class="row">Updated: <span id="updated" class="pill">-</span></div>
    <div class="row">Expires: <span id="expires" class="pill">-</span></div>
    <div class="row">Countdown: <span id="countdown" class="pill">-</span></div>
    <img id="qrcode" alt="QR code will appear here" />
    <div id="code" class="code">Waiting for pairing code...</div>
    <button id="copy-btn" type="button">Copy Code</button>
    <button id="refresh-btn" type="button">Manual Request QR</button>
    <button id="status-btn" type="button">Refresh Status</button>
    <div class="hint">If QR is not shown, install network access for CDN or copy code manually.</div>
  </div>
  <script>
    let lastCode = "";
    const qrcodeEl = document.getElementById("qrcode");
    const codeEl = document.getElementById("code");
    const connectedEl = document.getElementById("connected");
    const eventEl = document.getElementById("event");
    const updatedEl = document.getElementById("updated");
    const expiresEl = document.getElementById("expires");
    const countdownEl = document.getElementById("countdown");
    const copyBtn = document.getElementById("copy-btn");
    const refreshBtn = document.getElementById("refresh-btn");
    const statusBtn = document.getElementById("status-btn");

    copyBtn.addEventListener("click", async () => {
      if (!lastCode) return;
      try { await navigator.clipboard.writeText(lastCode); } catch (_) {}
    });

    refreshBtn.addEventListener("click", async () => {
      refreshBtn.disabled = true;
      try {
        await fetch("/api/whatsapp/pairing/refresh", { method: "POST" });
      } catch (_) {}
      await refresh();
      refreshBtn.disabled = false;
    });

    statusBtn.addEventListener("click", async () => {
      statusBtn.disabled = true;
      await refresh();
      statusBtn.disabled = false;
    });

    function renderQR(text) {
      if (!text) {
        qrcodeEl.removeAttribute("src");
        return;
      }
      qrcodeEl.src = "/api/whatsapp/pairing/qr.png?t=" + Date.now();
    }

    async function refresh() {
      try {
        const resp = await fetch("/api/whatsapp/pairing", { cache: "no-store" });
        const data = await resp.json();
        connectedEl.textContent = data.connected ? "connected" : "not connected";
        eventEl.textContent = data.event || "unknown";
        updatedEl.textContent = data.updated_at || "-";
        expiresEl.textContent = data.expires_at || "-";
        const left = Number(data.expires_in_seconds || 0);
        countdownEl.textContent = left > 0 ? (left + "s") : "-";
        const code = (data.code || "").trim();
        codeEl.textContent = code || "Waiting for pairing code...";
        if (code !== lastCode) {
          lastCode = code;
          renderQR(code);
        }
      } catch (_) {
        connectedEl.textContent = "error";
      }
    }
    refresh();
  </script>
</body>
</html>`

func (c *WhatsAppNativeChannel) reconnectWithBackoff() {
	defer func() {
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
	}()

	backoff := reconnectInitial
	for {
		select {
		case <-c.runCtx.Done():
			return
		default:
		}

		c.mu.Lock()
		client := c.client
		c.mu.Unlock()
		if client == nil {
			return
		}

		logger.InfoCF("whatsapp", "WhatsApp reconnecting", map[string]any{"backoff": backoff.String()})
		err := client.Connect()
		if err == nil {
			logger.InfoC("whatsapp", "WhatsApp reconnected")
			return
		}

		logger.WarnCF("whatsapp", "WhatsApp reconnect failed", map[string]any{"error": err.Error()})

		select {
		case <-c.runCtx.Done():
			return
		case <-time.After(backoff):
			if backoff < reconnectMax {
				next := time.Duration(float64(backoff) * reconnectMultiplier)
				if next > reconnectMax {
					next = reconnectMax
				}
				backoff = next
			}
		}
	}
}

func (c *WhatsAppNativeChannel) handleIncoming(evt *events.Message) {
	if evt.Message == nil {
		return
	}
	senderID := evt.Info.Sender.String()
	chatID := evt.Info.Chat.String()
	content := evt.Message.GetConversation()
	if content == "" && evt.Message.ExtendedTextMessage != nil {
		content = evt.Message.ExtendedTextMessage.GetText()
	}
	content = utils.SanitizeMessageContent(content)

	if content == "" {
		return
	}

	var mediaPaths []string

	metadata := make(map[string]string)
	metadata["message_id"] = evt.Info.ID
	if evt.Info.PushName != "" {
		metadata["user_name"] = evt.Info.PushName
	}
	if evt.Info.Chat.Server == types.GroupServer {
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = chatID
	} else {
		metadata["peer_kind"] = "direct"
		metadata["peer_id"] = senderID
	}

	peerKind := "direct"
	if evt.Info.Chat.Server == types.GroupServer {
		peerKind = "group"
	}
	peer := bus.Peer{Kind: peerKind, ID: chatID}
	messageID := evt.Info.ID
	sender := bus.SenderInfo{
		Platform:    "whatsapp",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("whatsapp", senderID),
		DisplayName: evt.Info.PushName,
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	logger.DebugCF(
		"whatsapp",
		"WhatsApp message received",
		map[string]any{"sender_id": senderID, "content_preview": utils.Truncate(content, 50)},
	)
	c.HandleMessage(c.runCtx, peer, messageID, senderID, chatID, content, mediaPaths, metadata, sender)
}

func (c *WhatsAppNativeChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}

	// Detect unpaired state: the client is connected (to WhatsApp servers)
	// but has not completed QR-login yet, so sending would fail.
	if client.Store.ID == nil {
		return fmt.Errorf("whatsapp not yet paired (QR login pending): %w", channels.ErrTemporary)
	}

	to, err := parseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", msg.ChatID, err)
	}

	waMsg := &waE2E.Message{
		Conversation: proto.String(msg.Content),
	}

	if _, err = client.SendMessage(ctx, to, waMsg); err != nil {
		return fmt.Errorf("whatsapp send: %w", channels.ErrTemporary)
	}
	return nil
}

// parseJID converts a chat ID (phone number or JID string) to types.JID.
func parseJID(s string) (types.JID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return types.JID{}, fmt.Errorf("empty chat id")
	}
	if strings.Contains(s, "@") {
		return types.ParseJID(s)
	}
	return types.NewJID(s, types.DefaultUserServer), nil
}
