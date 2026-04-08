package agent

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/preroute"
	"github.com/sipeed/picoclaw/pkg/preroute/executors"
	preroutes "github.com/sipeed/picoclaw/pkg/preroute/routes"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func (al *AgentLoop) handlePreRoute(
	ctx context.Context,
	agent *AgentInstance,
	msg bus.InboundMessage,
) (string, bool, error) {
	router := al.buildPreRouter(agent, msg.Channel)
	if router == nil {
		logger.WarnCF("agent", "Forced pre-route skipped: no router", map[string]any{
			"agent_id": agent.ID,
			"channel":  msg.Channel,
			"chat_id":  msg.ChatID,
			"query":    strings.TrimSpace(msg.Content),
		})
		return "", false, nil
	}

	result, err := router.Route(ctx, preroute.Context{
		Channel:    msg.Channel,
		ChatID:     msg.ChatID,
		AgentID:    agent.ID,
		Workspace:  agent.Workspace,
		Query:      msg.Content,
		QuotedText: inboundMetadata(msg, "quote_text"),
		MediaRefs:  append([]string(nil), msg.Media...),
		Metadata:   cloneStringMap(msg.Metadata),
	})
	if err != nil {
		logger.WarnCF("agent", "Forced pre-route failed", map[string]any{
			"agent_id": agent.ID,
			"channel":  msg.Channel,
			"chat_id":  msg.ChatID,
			"error":    err.Error(),
		})
		return "", false, err
	}
	if !result.Handled {
		logger.WarnCF("agent", "Forced pre-route miss", map[string]any{
			"agent_id": agent.ID,
			"channel":  msg.Channel,
			"chat_id":  msg.ChatID,
			"query":    strings.TrimSpace(msg.Content),
		})
		return "", false, nil
	}

	logger.WarnCF("agent", "Forced pre-route handled message", map[string]any{
		"agent_id": agent.ID,
		"route_id": result.RouteID,
		"channel":  msg.Channel,
		"chat_id":  msg.ChatID,
	})
	if len(result.MediaRefs) > 0 {
		parts := make([]bus.MediaPart, 0, len(result.MediaRefs))
		for _, ref := range result.MediaRefs {
			part := bus.MediaPart{Ref: ref}
			if al.mediaStore != nil {
				if _, meta, err := al.mediaStore.ResolveWithMeta(ref); err == nil {
					part.Filename = meta.Filename
					part.ContentType = meta.ContentType
					part.Type = inferMediaType(meta.Filename, meta.ContentType)
				}
			}
			parts = append(parts, part)
		}
		outboundMedia := bus.OutboundMediaMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Parts:   parts,
		}
		if al.channelManager != nil && msg.Channel != "" && !constants.IsInternalChannel(msg.Channel) {
			if err := al.channelManager.SendMedia(ctx, outboundMedia); err != nil {
				return "", true, err
			}
		} else if al.bus != nil {
			al.bus.PublishOutboundMedia(ctx, outboundMedia)
		}
	}
	return result.Text, true, nil
}

func (al *AgentLoop) buildPreRouter(agent *AgentInstance, channel string) *preroute.Router {
	if al == nil || al.cfg == nil || agent == nil {
		return nil
	}
	cfg := al.cfg.Routes.Forced
	if !cfg.Enabled {
		logger.WarnCF("agent", "Forced pre-route disabled", map[string]any{
			"agent_id": agent.ID,
			"channel":  channel,
		})
		return nil
	}
	if len(cfg.Channels) > 0 && !containsStringFold(cfg.Channels, channel) {
		logger.WarnCF("agent", "Forced pre-route channel filtered", map[string]any{
			"agent_id": agent.ID,
			"channel":  channel,
			"allowed":  strings.Join(cfg.Channels, ","),
		})
		return nil
	}

	routes := make([]preroute.Route, 0, len(cfg.Order))
	for _, id := range cfg.Order {
		switch strings.ToLower(strings.TrimSpace(id)) {
		case "help":
			if cfg.Features.Help {
				routes = append(routes, preroutes.NewHelp(""))
			}
		case "news":
			if cfg.Features.News {
				routes = append(routes, preroutes.NewNews(executors.TencentNewsCLIExecutor{
					RelativePath: cfg.News.CLIRelativePath,
				}))
			}
		case "image":
			if !cfg.Features.Image {
				continue
			}
			tool, ok := agent.Tools.Get("modelscope-image")
			if !ok {
				continue
			}
			execTool, ok := tool.(interface {
				Execute(ctx context.Context, args map[string]any) *tools.ToolResult
			})
			if !ok {
				continue
			}
			routes = append(routes, preroutes.NewImage(executors.ModelScopeImageExecutor{Tool: execTool}))
		}
	}
	if len(routes) == 0 {
		logger.WarnCF("agent", "Forced pre-route built zero routes", map[string]any{
			"agent_id": agent.ID,
			"channel":  channel,
			"order":    strings.Join(cfg.Order, ","),
		})
		return nil
	}
	logger.WarnCF("agent", "Forced pre-route router ready", map[string]any{
		"agent_id": agent.ID,
		"channel":  channel,
		"count":    len(routes),
		"order":    strings.Join(cfg.Order, ","),
	})
	return preroute.NewRouter(routes)
}

func containsStringFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
