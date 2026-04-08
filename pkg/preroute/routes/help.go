package routes

import (
	"context"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/preroute"
)

var globalHelpRE = regexp.MustCompile(`^(功能帮助|帮助|你会什么|你能做什么|菜单|功能菜单|能力介绍|我现在有哪些功能|有哪些功能|全部功能|能力清单)$`)

const DefaultHelpText = `当前可用的系统功能：

1. 图片
- 生图
- 改图

2. 新闻
- 腾讯新闻 热点
- 今日新闻
- AI新闻
`

type helpRoute struct {
	text string
}

func NewHelp(text string) preroute.Route {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		trimmed = DefaultHelpText
	}
	return helpRoute{text: trimmed}
}

func (r helpRoute) ID() string { return "help" }

func (r helpRoute) Handle(_ context.Context, rc preroute.Context) (preroute.Result, error) {
	if !globalHelpRE.MatchString(strings.TrimSpace(rc.Query)) {
		return preroute.Result{}, nil
	}
	return preroute.Result{
		Handled: true,
		Text:    r.text,
		RouteID: r.ID(),
	}, nil
}
