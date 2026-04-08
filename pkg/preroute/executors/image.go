package executors

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/preroute"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type ModelScopeImageTool interface {
	Execute(ctx context.Context, args map[string]any) *tools.ToolResult
}

type ModelScopeImageExecutor struct {
	Tool ModelScopeImageTool
}

func (e ModelScopeImageExecutor) Execute(ctx context.Context, rc preroute.Context) (preroute.Result, error) {
	query := strings.TrimSpace(rc.Query)
	switch {
	case strings.HasPrefix(query, "生图 "), strings.HasPrefix(query, "阿里生图 "):
		prompt := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(query, "阿里生图"), "生图"))
		if prompt == "" {
			return preroute.Result{Handled: true, Text: "请在“生图”后面补充你想生成的画面描述。", RouteID: "image"}, nil
		}
		if e.Tool == nil {
			return preroute.Result{}, nil
		}
		result := e.Tool.Execute(ctx, map[string]any{"prompt": prompt})
		if result == nil {
			return preroute.Result{Handled: true, Text: "图片生成失败：工具未返回结果。", RouteID: "image"}, nil
		}
		return preroute.Result{
			Handled:   true,
			Text:      strings.TrimSpace(result.ForUser),
			MediaRefs: append([]string(nil), result.Media...),
			RouteID:   "image",
		}, nil
	case strings.HasPrefix(query, "改图 "), strings.HasPrefix(query, "阿里改图 "):
		if len(rc.MediaRefs) == 0 {
			return preroute.Result{Handled: true, Text: "请先回复一张图片，再发送“改图 ...”或“阿里改图 ...”。", RouteID: "image"}, nil
		}
		return preroute.Result{Handled: true, Text: "当前版本暂未在 forced-routes 中启用改图，请继续使用现有对话流程。", RouteID: "image"}, nil
	default:
		return preroute.Result{}, nil
	}
}
