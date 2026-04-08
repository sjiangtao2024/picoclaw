package executors

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/preroute"
)

var (
	tencentNewsPrefixRE = regexp.MustCompile(`^腾讯新闻(?:\s*[:：]\s*|\s*)([\s\S]*)$`)
	newsHelpRE          = regexp.MustCompile(`^(新闻帮助|腾讯新闻帮助|腾讯新闻功能帮助)$`)
	hotNewsRE           = regexp.MustCompile(`^(热点|热点新闻|热榜)$`)
	morningNewsRE       = regexp.MustCompile(`^(早报|今日早报)$`)
	eveningNewsRE       = regexp.MustCompile(`^(晚报|今日晚报)$`)
	todayNewsRE         = regexp.MustCompile(`^(今日新闻|今天新闻|今天的新闻|今日的新闻|新闻简报|今天有什么新闻|今天都有什么新闻|今天有哪些新闻)$`)
	aiNewsRE            = regexp.MustCompile(`^(AI新闻|AI日报|AI资讯)$`)
)

const DefaultTencentNewsCLIRelativePath = "skills/tencent-news/tencent-news-cli"

const DefaultNewsHelpText = `腾讯新闻支持示例：
- 腾讯新闻 热点
- 腾讯新闻 早报
- 腾讯新闻 晚报
- 今日新闻
- AI新闻`

type TencentNewsCLIExecutor struct {
	RelativePath string
	Now          func() time.Time
}

func ResolveTencentNewsCLIPath(workspace, relativePath string) (string, bool) {
	workspace = strings.TrimSpace(workspace)
	relativePath = strings.TrimSpace(relativePath)
	if workspace == "" || relativePath == "" {
		return "", false
	}
	path := filepath.Join(workspace, filepath.FromSlash(relativePath))
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

func (e TencentNewsCLIExecutor) Execute(ctx context.Context, rc preroute.Context) (string, bool, error) {
	query := strings.TrimSpace(rc.Query)
	if query == "" {
		return "", false, nil
	}
	if newsHelpRE.MatchString(query) {
		return DefaultNewsHelpText, true, nil
	}
	args, ok := buildTencentNewsCLIArgs(query, nowFunc(e.Now))
	if !ok {
		return "", false, nil
	}
	rel := strings.TrimSpace(e.RelativePath)
	if rel == "" {
		rel = DefaultTencentNewsCLIRelativePath
	}
	cliPath, found := ResolveTencentNewsCLIPath(rc.Workspace, rel)
	if !found {
		return "", false, nil
	}

	cmd, cleanup, err := buildTencentNewsCommand(ctx, cliPath, args, userHomeDir())
	if err != nil {
		return "", true, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", true, fmt.Errorf("tencent-news-cli failed: %s", msg)
	}
	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return "腾讯新闻暂无可返回内容。", true, nil
	}
	return text, true, nil
}

func buildTencentNewsCLIArgs(query string, now time.Time) ([]string, bool) {
	trimmed := strings.TrimSpace(query)
	explicitBody := trimmed
	if match := tencentNewsPrefixRE.FindStringSubmatch(trimmed); len(match) > 1 {
		explicitBody = strings.TrimSpace(match[1])
		if explicitBody == "" {
			explicitBody = "热点"
		}
	}
	switch {
	case hotNewsRE.MatchString(explicitBody):
		return []string{"hot"}, true
	case morningNewsRE.MatchString(explicitBody):
		return []string{"morning"}, true
	case eveningNewsRE.MatchString(explicitBody):
		return []string{"evening"}, true
	case todayNewsRE.MatchString(explicitBody):
		if now.Hour() >= 18 {
			return []string{"evening"}, true
		}
		return []string{"morning"}, true
	case aiNewsRE.MatchString(explicitBody):
		return []string{"ai-daily", "--query", "AI"}, true
	case strings.HasPrefix(explicitBody, "AI新闻 "):
		return []string{"ai-daily", "--query", strings.TrimSpace(strings.TrimPrefix(explicitBody, "AI新闻 "))}, true
	case strings.HasPrefix(trimmed, "腾讯新闻"):
		return []string{"ai-daily", "--query", explicitBody}, true
	default:
		return nil, false
	}
}

func nowFunc(fn func() time.Time) time.Time {
	if fn != nil {
		return fn()
	}
	return time.Now()
}

func resolveTencentNewsAPIKey(getenv func(string) string, home string) string {
	if getenv != nil {
		if key := strings.TrimSpace(getenv("TENCENT_NEWS_APIKEY")); key != "" {
			return key
		}
	}
	if strings.TrimSpace(home) == "" {
		return ""
	}
	for _, candidate := range []string{".bashrc", ".profile"} {
		data, err := os.ReadFile(filepath.Join(home, candidate))
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`(?m)^\s*export\s+TENCENT_NEWS_APIKEY=(?:"([^"]+)"|([^\s#]+))\s*$`)
		match := re.FindStringSubmatch(string(data))
		if len(match) < 3 {
			continue
		}
		if strings.TrimSpace(match[1]) != "" {
			return strings.TrimSpace(match[1])
		}
		if strings.TrimSpace(match[2]) != "" {
			return strings.TrimSpace(match[2])
		}
	}
	return ""
}

func userHomeDir() string {
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return home
}

func buildTencentNewsCommand(
	ctx context.Context,
	cliPath string,
	args []string,
	home string,
) (*exec.Cmd, func(), error) {
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Env = os.Environ()
	// Ensure HOME=/root so the CLI can read the API key from ~/.bashrc
	if home != "" {
		cmd.Env = append(cmd.Env, "HOME="+home)
	}
	return cmd, nil, nil
}
