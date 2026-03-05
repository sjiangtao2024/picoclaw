//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type FeishuChannel struct {
	*channels.BaseChannel
	config   config.FeishuConfig
	client   *lark.Client
	wsClient *larkws.Client
	botOpenID string

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewFeishuChannel(cfg config.FeishuConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	base := channels.NewBaseChannel("feishu", cfg, bus, cfg.AllowFrom,
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &FeishuChannel{
		BaseChannel: base,
		config:      cfg,
		client:      lark.NewClient(cfg.AppID, cfg.AppSecret),
	}, nil
}

func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("feishu app_id or app_secret is empty")
	}

	dispatcher := larkdispatcher.NewEventDispatcher(c.config.VerificationToken, c.config.EncryptKey).
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.botOpenID = c.resolveBotOpenID(ctx)
	c.cancel = cancel
	c.wsClient = larkws.NewClient(
		c.config.AppID,
		c.config.AppSecret,
		larkws.WithEventHandler(dispatcher),
	)
	wsClient := c.wsClient
	c.mu.Unlock()

	c.SetRunning(true)
	logger.InfoC("feishu", "Feishu channel started (websocket mode)")

	go func() {
		if err := wsClient.Start(runCtx); err != nil {
			logger.ErrorCF("feishu", "Feishu websocket stopped with error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *FeishuChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.mu.Unlock()

	c.SetRunning(false)
	logger.InfoC("feishu", "Feishu channel stopped")
	return nil
}

func (c *FeishuChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	if msg.ChatID == "" {
		return fmt.Errorf("chat ID is empty")
	}

	cardPayload, err := buildInteractiveCardPayload(msg.Content)
	if err == nil {
		cardPreview := utils.Truncate(normalizeForFeishuCard(msg.Content), 180)
		req := larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(larkim.ReceiveIdTypeChatId).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(msg.ChatID).
				MsgType(larkim.MsgTypeInteractive).
				Content(string(cardPayload)).
				Uuid(fmt.Sprintf("picoclaw-card-%d", time.Now().UnixNano())).
				Build()).
			Build()

		resp, sendErr := c.client.Im.V1.Message.Create(ctx, req)
		if sendErr == nil && resp != nil && resp.Success() {
			logger.InfoCF("feishu", "Feishu interactive card sent", map[string]any{
				"chat_id":      msg.ChatID,
				"card_preview": cardPreview,
			})
			return nil
		}
		code := -1
		msgText := ""
		if resp != nil {
			code = resp.Code
			msgText = resp.Msg
		}
		logger.InfoCF("feishu", "Interactive card send failed; fallback to text", map[string]any{
			"chat_id":      msg.ChatID,
			"error":        fmt.Sprintf("%v", sendErr),
			"code":         code,
			"msg":          msgText,
			"card_preview": cardPreview,
		})
	}

	textPayload, err := json.Marshal(map[string]string{"text": msg.Content})
	if err != nil {
		return fmt.Errorf("failed to marshal feishu text content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType(larkim.MsgTypeText).
			Content(string(textPayload)).
			Uuid(fmt.Sprintf("picoclaw-text-%d", time.Now().UnixNano())).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send: %w", channels.ErrTemporary)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu api error (code=%d msg=%s): %w", resp.Code, resp.Msg, channels.ErrTemporary)
	}

	logger.DebugCF("feishu", "Feishu message sent", map[string]any{
		"chat_id": msg.ChatID,
	})

	return nil
}

func shouldUseInteractiveCard(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	// Prefer card rendering for structured output so Feishu can display rich layout.
	return strings.Contains(trimmed, "\n") ||
		strings.Contains(trimmed, "|") ||
		strings.Contains(trimmed, "###") ||
		strings.Contains(trimmed, "**")
}

func buildInteractiveCardPayload(content string) ([]byte, error) {
	cardContent := normalizeForFeishuCard(content)
	elements := []map[string]any{}
	if blocks := buildRepoRecommendElements(content); len(blocks) > 0 {
		elements = blocks
	} else {
		elements = []map[string]any{
			{
				"tag":     "markdown",
				"content": cardContent,
			},
		}
	}

	card := map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
		},
		"header": map[string]any{
			"template": "turquoise",
			"title": map[string]any{
				"tag":     "plain_text",
				"content": "PicoClaw 推荐结果",
			},
		},
		"elements": elements,
	}
	return json.Marshal(card)
}

func normalizeForFeishuCard(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			out = append(out, "")
			continue
		}

		// Convert markdown headings to plain titles.
		line = strings.TrimPrefix(line, "### ")
		line = strings.TrimPrefix(line, "## ")
		line = strings.TrimPrefix(line, "# ")

		// Convert markdown list bullets to a plain bullet.
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "*   ")

		// Remove markdown emphasis markers.
		line = strings.ReplaceAll(line, "**", "")

		// Convert markdown links [text](url) -> text（url）
		line = markdownLinkToPlain(line)

		// Detect and transform markdown table blocks.
		if strings.Contains(line, "|") && i+1 < len(lines) && isTableDivider(strings.TrimSpace(lines[i+1])) {
			headers := parseTableRow(line)
			i += 2 // skip header + divider
			for ; i < len(lines); i++ {
				rowLine := strings.TrimSpace(lines[i])
				if rowLine == "" || !strings.Contains(rowLine, "|") {
					i--
					break
				}
				cells := parseTableRow(rowLine)
				if len(cells) == 0 {
					continue
				}
				var parts []string
				for c := 0; c < len(cells) && c < len(headers); c++ {
					if cells[c] == "" {
						continue
					}
					parts = append(parts, fmt.Sprintf("%s: %s", headers[c], cells[c]))
				}
				if len(parts) > 0 {
					out = append(out, "- "+strings.Join(parts, " | "))
				}
			}
			continue
		}

		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func isTableDivider(line string) bool {
	trim := strings.ReplaceAll(strings.ReplaceAll(line, "|", ""), "-", "")
	trim = strings.ReplaceAll(trim, ":", "")
	trim = strings.TrimSpace(trim)
	return trim == "" && strings.Contains(line, "-")
}

func parseTableRow(line string) []string {
	parts := strings.Split(line, "|")
	var out []string
	for _, p := range parts {
		cell := strings.TrimSpace(p)
		if cell != "" {
			out = append(out, cell)
		}
	}
	return out
}

var mdLinkPattern = regexp.MustCompile(`\[(.*?)\]\((https?://[^\s)]+)\)`)

func markdownLinkToPlain(line string) string {
	return mdLinkPattern.ReplaceAllString(line, `$1（$2）`)
}

type repoRow struct {
	Name   string
	Scene  string
	Why    string
	Link   string
}

func buildRepoRecommendElements(content string) []map[string]any {
	rows := parseRepoRowsFromMarkdownTable(content)
	if len(rows) == 0 {
		return nil
	}

	elements := []map[string]any{
		{
			"tag":     "markdown",
			"content": fmt.Sprintf("已为你筛选 **%d** 个候选仓库，按可落地优先排序：", len(rows)),
		},
		{"tag": "hr"},
	}

	for i, r := range rows {
		line := fmt.Sprintf(
			"**%d. %s**\n场景：%s\n理由：%s\n链接：%s",
			i+1,
			emptyOr(r.Name, "未命名仓库"),
			emptyOr(r.Scene, "通用"),
			emptyOr(r.Why, "与需求匹配"),
			emptyOr(r.Link, "暂无"),
		)
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": line,
		})
		if i != len(rows)-1 {
			elements = append(elements, map[string]any{"tag": "hr"})
		}
	}

	return elements
}

func parseRepoRowsFromMarkdownTable(content string) []repoRow {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var out []repoRow
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || !strings.Contains(line, "|") || isTableDivider(line) {
			continue
		}
		cells := parseTableRow(line)
		if len(cells) < 4 {
			continue
		}
		if strings.Contains(cells[0], "项目") && strings.Contains(cells[1], "适用场景") {
			continue
		}
		out = append(out, repoRow{
			Name:  stripMd(cells[0]),
			Scene: stripMd(cells[1]),
			Why:   stripMd(cells[2]),
			Link:  markdownLinkToPlain(cells[3]),
		})
	}
	return out
}

func stripMd(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "**", "")
	s = strings.TrimPrefix(s, "- ")
	s = strings.TrimPrefix(s, "* ")
	return s
}

func emptyOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (c *FeishuChannel) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	sender := event.Event.Sender

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	senderID := extractFeishuSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	content := extractFeishuMessageContent(message)
	if content == "" {
		content = "[empty message]"
	}

	metadata := map[string]string{}
	messageID := ""
	if mid := stringValue(message.MessageId); mid != "" {
		messageID = mid
	}
	if messageType := stringValue(message.MessageType); messageType != "" {
		metadata["message_type"] = messageType
	}
	if chatType := stringValue(message.ChatType); chatType != "" {
		metadata["chat_type"] = chatType
	}
	if sender != nil && sender.TenantKey != nil {
		metadata["tenant_key"] = *sender.TenantKey
	}

	chatType := stringValue(message.ChatType)
	var peer bus.Peer
	if chatType == "p2p" {
		peer = bus.Peer{Kind: "direct", ID: senderID}
	} else {
		peer = bus.Peer{Kind: "group", ID: chatID}
		// In group chats, apply unified group trigger filtering
		isMentioned := isMentionedForGroup(message, c.botOpenID)
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return nil
		}
		content = cleaned
	}

	logger.InfoCF("feishu", "Feishu message received", map[string]any{
		"sender_id": senderID,
		"chat_id":   chatID,
		"preview":   utils.Truncate(content, 80),
	})

	senderInfo := bus.SenderInfo{
		Platform:    "feishu",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("feishu", senderID),
	}

	if !c.IsAllowedSender(senderInfo) {
		return nil
	}

	c.HandleMessage(ctx, peer, messageID, senderID, chatID, content, nil, metadata, senderInfo)
	return nil
}

func extractFeishuSenderID(sender *larkim.EventSender) string {
	if sender == nil || sender.SenderId == nil {
		return ""
	}

	if sender.SenderId.UserId != nil && *sender.SenderId.UserId != "" {
		return *sender.SenderId.UserId
	}
	if sender.SenderId.OpenId != nil && *sender.SenderId.OpenId != "" {
		return *sender.SenderId.OpenId
	}
	if sender.SenderId.UnionId != nil && *sender.SenderId.UnionId != "" {
		return *sender.SenderId.UnionId
	}

	return ""
}

func extractFeishuMessageContent(message *larkim.EventMessage) string {
	if message == nil || message.Content == nil || *message.Content == "" {
		return ""
	}

	if message.MessageType != nil && *message.MessageType == larkim.MsgTypeText {
		var textPayload struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &textPayload); err == nil {
			return textPayload.Text
		}
	}

	return *message.Content
}

func (c *FeishuChannel) resolveBotOpenID(ctx context.Context) string {
	if c.client == nil {
		return ""
	}
	// bot/v3/info is the authoritative API for current app bot identity.
	resp, err := c.client.Get(ctx, "/open-apis/bot/v3/info", nil, larkcore.AccessTokenTypeTenant)
	if err != nil || resp == nil {
		logger.WarnCF("feishu", "Failed to resolve bot open_id", map[string]any{
			"error": fmt.Sprintf("%v", err),
		})
		return ""
	}

	var data struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Bot  struct {
			OpenID string `json:"open_id"`
		} `json:"bot"`
		Data struct {
			Bot struct {
				OpenID string `json:"open_id"`
			} `json:"bot"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.RawBody, &data); err != nil {
		logger.WarnCF("feishu", "Failed to decode bot info response", map[string]any{
			"error": err.Error(),
		})
		return ""
	}
	if data.Code != 0 {
		logger.WarnCF("feishu", "Failed to resolve bot open_id from API", map[string]any{
			"code": data.Code,
			"msg":  data.Msg,
		})
		return ""
	}

	botOpenID := strings.TrimSpace(data.Bot.OpenID)
	if botOpenID == "" {
		botOpenID = strings.TrimSpace(data.Data.Bot.OpenID)
	}
	if botOpenID == "" {
		logger.WarnC("feishu", "Bot open_id is empty in bot info response")
		return ""
	}
	logger.InfoCF("feishu", "Resolved bot open_id", map[string]any{
		"bot_open_id": botOpenID,
	})
	return botOpenID
}

// isMentionedForGroup determines whether the incoming group message explicitly
// mentions this bot app. We match by mention open_id to avoid false positives
// when users @ other bots in the same group.
func isMentionedForGroup(message *larkim.EventMessage, botOpenID string) bool {
	if message == nil {
		return false
	}
	botOpenID = strings.TrimSpace(botOpenID)
	if botOpenID == "" {
		return false
	}
	for _, mention := range message.Mentions {
		if mention == nil || mention.Id == nil || mention.Id.OpenId == nil {
			continue
		}
		if strings.TrimSpace(*mention.Id.OpenId) == botOpenID {
			return true
		}
	}
	// Post rich-text messages may not always carry top-level mentions.
	if message.MessageType != nil && *message.MessageType == larkim.MsgTypePost &&
		message.Content != nil && strings.Contains(*message.Content, botOpenID) {
		return true
	}
	return false
}
