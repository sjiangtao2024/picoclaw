# PicoClaw 补丁维护说明

目录：`/home/yukun/dev/picobox-ai/picoclaw/patches`

升级流程文档：`UPGRADE-SOP.zh.md`

推荐给后续新版本升级使用的干净补丁集：

- `/home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.4-h618-minimal`
- `/home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes`

说明：

- 上面这套是从已验证的 `custom/release-v0.2.4-h618` 导出的最小补丁集。
- `v0.2.5-h618-chat-fixes` 是 2026-04-04 在 H618 远端排障后整理出的聊天链路修复补丁集。
- 其中已包含 `0004-skillhub-defaults-and-compat.patch`，用于固化腾讯 SkillHub 默认接入与 `clawhub -> skillhub` 兼容回退。
- 其中现已包含 `0005-web-chat-default-agent-selector.patch`，用于给 Web 聊天页补上“切换全局默认 Agent”的能力。
- 其中现已包含 `0006-web-skills-default-agent-workspace.patch`，用于让 Web skills 页面跟随当前默认 Agent 的 workspace，而不是固定落到基础 workspace。
- 以后切上游新 tag，优先从这套补丁开始评估和重放。
- 本页下方的 `0001~0014` 仍保留，主要用于追溯历史演进过程。

当前包含 13 个稳定补丁：
- `0001-routing-and-agent-overrides.patch`
- `0002-toolcall-normalization.patch`
- `0003-agent-loop-modularization.patch`
- `0004-skills-registry-proxy.patch`
- `0005-skills-proxy-toggle.patch`
- `0006-im-output-formatting.patch`
- `0007-web-model-search-provider.patch`
- `0008-feishu-interactive-card.patch`
- `0009-modelscope-image-tool.patch`
- `0010-im-image-output.patch`
- `0011-subagent-sync-only.patch`
- `0013-pico-media-output.patch`
- `0014-web-linux-nocgo-launcher.patch`

另有 1 个待清理补丁（当前不纳入默认升级顺序）：
- `0012-subagent-default-target.patch`

另有 1 个运行态增强（2026-03-02，尚未固化为独立 patch 文件）：
- agent 级工具权限白/黑名单（`allowed_tools` / `disabled_tools`）

## 补丁作用

## 0001-routing-and-agent-overrides.patch
覆盖文件：
- `pkg/agent/loop.go`
- `pkg/agent/instance.go`
- `pkg/config/config.go`

作用：
- 修复 `model_name -> model_list` 路由，避免错误 endpoint。
- 支持 agent 级 `max_tool_iterations/max_tokens/temperature` 覆盖 defaults。
- 多轮工具调用耗尽时返回可读中文兜底，不再英文空话。
- 不再向用户直发 `Memory threshold reached...` 内部提示。

## 0002-toolcall-normalization.patch
覆盖文件：
- `pkg/providers/toolcall_utils.go`
- `pkg/providers/toolcall_utils_test.go`

作用：
- 自动补齐 tool call 的 `id/type`，提升兼容性。
- 增加对应测试用例。

## 0003-agent-loop-modularization.patch
覆盖文件：
- `pkg/agent/loop.go`
- `pkg/agent/llm_runner.go`（新增）
- `pkg/agent/session_runtime.go`（新增）
- `pkg/agent/summarization_runtime.go`（新增）
- `pkg/agent/logging_formatters.go`（新增）

作用：
- 将超大 `loop.go` 模块化拆分，降低升级冲突概率。
- 保持行为不变，仅做结构重构（编译、测试、健康检查均验证通过）。
- 当前目标状态：`loop.go` 约 482 行。

注意：
- `0003` 是基于你当前 dev 分支生成，主要用于你后续升级时“还原我们的模块化结构”。
- 该 patch 文件路径前缀是 `old/`、`new/`，应用时需要 `-p1`。

## 0004-skills-registry-proxy.patch
覆盖文件：
- `pkg/config/config.go`
- `pkg/skills/registry.go`
- `pkg/skills/clawhub_registry.go`

作用：
- 给 `tools.skills.registries.clawhub` 增加 `proxy` 配置项。
- skills 相关 HTTP 请求支持代理转发（优先 `clawhub.proxy`，否则回退环境变量代理）。
- 用于内网环境安装/搜索 skills 时走代理。

注意：
- `0004` 当前路径前缀是 `old/`、`new/`，应用时需要 `-p1`。

## 0005-skills-proxy-toggle.patch
覆盖文件：
- `pkg/config/config.go`
- `pkg/skills/registry.go`
- `pkg/skills/clawhub_registry.go`

作用：
- 增加 `tools.skills.registries.clawhub.use_proxy` 开关。
- `use_proxy=false` 时强制直连（忽略 `clawhub.proxy` 和环境代理）。
- `use_proxy=true` 或未配置时，保持原逻辑（优先 `clawhub.proxy`，否则环境变量代理）。

注意：
- `0005` 当前路径前缀是 `old/`、`new/`，应用时需要 `-p1`。

## 0006-im-output-formatting.patch
覆盖文件：
- `pkg/channels/manager.go`
- `pkg/formatters/output.go`（新增）
- `pkg/formatters/output_test.go`（新增）

作用：
- 新增 IM 输出格式化模块 `pkg/formatters`。
- 在统一消息出口 `dispatchOutbound` 按 channel 进行格式化。
- 对飞书/钉钉/QQ/Discord 自动做 Markdown 降级，提升显示可读性。
- Telegram 保持原逻辑（继续由其 channel 内部做 markdown->HTML 处理）。

## 0007-web-model-search-provider.patch
覆盖文件：
- `pkg/config/config.go`
- `pkg/config/defaults.go`
- `pkg/tools/web.go`
- `pkg/agent/loop.go`
- `pkg/tools/web_test.go`

作用：
- 给 `tools.web` 增加 `model_search` 配置项。
- `web_search` 新增模型搜索 provider（`/chat/completions` 调用）。
- 路由规则：有 `model_search` 配置优先用模型搜索；无配置回退 Tavily。
- 保持旧后端（Tavily/Perplexity/Brave/DuckDuckGo）兼容可回退。

## 0008-feishu-interactive-card.patch
覆盖文件：
- `pkg/channels/feishu/feishu_64.go`

作用：
- 飞书输出优先走 `interactive` 卡片，显著提升表格/列表的可读性。
- 自动把 Markdown 表格转换为结构化卡片元素。
- 发送失败自动回退文本消息，避免影响可用性。

判定原则（升级时）：
- 若上游已提供同等卡片渲染（行为一致）：跳过该补丁。
- 若上游未实现或行为不一致：继续应用该补丁。

## 新版本升级时：先判定再补丁

在新版本代码根目录执行：

## 0009-modelscope-image-tool.patch
覆盖文件：
- `pkg/config/config.go`
- `pkg/config/defaults.go`
- `pkg/agent/loop.go`
- `pkg/tools/modelscope_image.go`（新增）
- `pkg/tools/modelscope_image_test.go`（新增）

作用：
- 给 `tools.images.modelscope` 增加本地生图服务配置项。
- 注册 `modelscope-image` tool，走本机 `127.0.0.1` 图片服务。
- 保持 PicoClaw 只做薄适配，不把 ModelScope provider 逻辑焊进主程序。
- 增加最小测试，覆盖禁用态、缺少 prompt、成功调用三条路径。

判定原则（升级时）：
- 若上游已具备等价的本地图片 tool 和配置结构：跳过该补丁。
- 若上游未实现或行为不一致：继续应用该补丁。

## 0010-im-image-output.patch
覆盖文件：
- `pkg/tools/result.go`
- `pkg/bus/types.go`
- `pkg/agent/loop.go`
- `pkg/agent/llm_runner.go`
- `pkg/tools/modelscope_image.go`
- `pkg/tools/modelscope_image_test.go`
- `pkg/channels/manager.go`
- `pkg/channels/outbound_media.go`（新增）
- `pkg/channels/outbound_media_test.go`（新增）
- `pkg/channels/wecom.go`
- `pkg/channels/wecom_app.go`
- `pkg/channels/feishu_64.go`

作用：
- 给 tool result 和 outbound bus 增加结构化媒体字段。
- 生图结果不再只靠文本 URL，而是把图片作为媒体透传到 channel。
- 飞书、企微机器人、企微自建应用优先真发图片；其他 IM 自动回退为文本+链接。
- 为后续 TTS、股票图表等媒体能力预留统一出站结构。

判定原则（升级时）：
- 若上游已经有统一媒体出站结构且支持图片透传：跳过该补丁。
- 若上游仍只有纯文本出站：继续应用该补丁。

## 0011-subagent-sync-only.patch
覆盖文件：
- `pkg/config/config.go`
- `pkg/agent/loop.go`
- `pkg/agent/loop_test.go`

作用：
- 给 `agents.list[].subagents` 增加 `sync_only` 配置项。
- `sync_only=true` 时只注册同步 `subagent`，不注册异步 `spawn/spawn_status`。
- 用于像 `salesbox -> image-agent` 这种单次委托场景，避免重复异步派发把下游队列打满。

判定原则（升级时）：
- 若上游已支持 agent 级“只允许同步 subagent、不暴露 spawn”配置：跳过该补丁。
- 若上游仍同时暴露 `subagent` 与 `spawn`：继续应用该补丁。

## 0013-pico-media-output.patch
覆盖文件：
- `pkg/channels/pico/pico.go`
- `pkg/channels/pico/pico_test.go`
- `web/frontend/src/components/chat/assistant-message.tsx`
- `web/frontend/src/components/chat/chat-page.tsx`
- `web/frontend/src/features/chat/protocol.ts`
- `web/frontend/src/store/chat.ts`

作用：
- 给 `pico` channel 增加 `SendMedia`，不再报 `channel "pico" does not support media sending`。
- 服务端通过 `media.create` 下发结构化媒体，图片以 `data_url` 形式传给 web UI。
- Web 前端新增 `media.create` 处理与图片渲染，Pico 页面可直接显示生图结果。

判定原则（升级时）：
- 若上游 `pico` 已支持媒体发送且 web UI 能渲染 `media.create`：跳过该补丁。
- 若上游仍只有纯文本 `message.create`：继续应用该补丁。

## 0014-web-linux-nocgo-launcher.patch
覆盖文件：
- `web/backend/systray.go`
- `web/backend/tray_stub_nocgo.go`

作用：
- 修正 `linux + CGO_ENABLED=0` 时 launcher 的 build tag。
- 避免 Linux 无 cgo 构建误命中 systray 实现，导致 `picoclaw-web` 无法构建。
- 让 H618/盒子交付场景下的 `picoclaw-web` 可以稳定交叉编译。

判定原则（升级时）：
- 若上游已修正 launcher 在 `linux + no-cgo` 下的 build tag：跳过该补丁。
- 若上游仍会错误编译 systray 路径：继续应用该补丁。

```bash
rg -n "resolveModelTarget|NormalizeToolCall|runLLMIteration\(|resolveRoutedAgentAndSession\(|maybeSummarize\(|formatMessagesForLog\(|PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_PROXY|PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_USE_PROXY|parseProxyURL|UseProxy|formatters\.Format\(|pkg/formatters|model_search|ModelSearch|interactive|Feishu.*card|buildInteractiveCardPayload|ModelScopeImage|modelscope-image|tools\\.images\\.modelscope|ToolMedia|OutboundMedia|normalizeOutboundMessage|sendWebhookImageReply|sendImageMessage|sync_only|SyncOnly|SendMedia\(ctx context\.Context, msg bus\.OutboundMediaMessage\)|TypeMediaCreate|attachments\?: ChatAttachment|data_url" pkg web
```

判定规则：
- 若上游已实现且行为一致：对应补丁可跳过。
- 若缺失或行为不一致：继续应用补丁。

推荐最小功能判定：
- 工具调用稳定性：`NormalizeToolCall`
- 路由和 agent 覆盖：`resolveModelTarget` + agent 级覆盖字段
- 大文件可维护性：`llm_runner.go/session_runtime.go/summarization_runtime.go`

## 如何打补丁

以仓库目录 `REPO_DIR` 为例：

```bash
cd REPO_DIR

# 建议先建分支
# git checkout -b patch/local-maintenance

# 先试跑（不改文件）
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0001-routing-and-agent-overrides.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0002-toolcall-normalization.patch
git apply --check -p1 /home/yukun/dev/picobox-ai/picoclaw/patches/0003-agent-loop-modularization.patch
git apply --check -p1 /home/yukun/dev/picobox-ai/picoclaw/patches/0004-skills-registry-proxy.patch
git apply --check -p1 /home/yukun/dev/picobox-ai/picoclaw/patches/0005-skills-proxy-toggle.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0006-im-output-formatting.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0007-web-model-search-provider.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0008-feishu-interactive-card.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0009-modelscope-image-tool.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0010-im-image-output.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0011-subagent-sync-only.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0013-pico-media-output.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/0014-web-linux-nocgo-launcher.patch

# 应用补丁
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0001-routing-and-agent-overrides.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0002-toolcall-normalization.patch
git apply -p1 /home/yukun/dev/picobox-ai/picoclaw/patches/0003-agent-loop-modularization.patch
git apply -p1 /home/yukun/dev/picobox-ai/picoclaw/patches/0004-skills-registry-proxy.patch
git apply -p1 /home/yukun/dev/picobox-ai/picoclaw/patches/0005-skills-proxy-toggle.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0006-im-output-formatting.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0007-web-model-search-provider.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0008-feishu-interactive-card.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0009-modelscope-image-tool.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0010-im-image-output.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0011-subagent-sync-only.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0013-pico-media-output.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/0014-web-linux-nocgo-launcher.patch
```

如果 `git apply` 失败（上游代码有改动）：
1. 按报错文件手动合并。
2. 重新运行上面的 `rg` 判定命令确认功能仍在。
3. 运行测试并启动验证。

## 补丁后验证

```bash
# 单元测试
go test ./pkg/agent ./pkg/config ./pkg/providers/...
go test ./pkg/formatters ./pkg/channels
go test ./pkg/tools -run ModelScopeImage
go test ./pkg/channels -run NormalizeOutboundMessage

# 运行态检查（按你的部署路径调整）
curl -sS http://127.0.0.1:18790/health
curl -sS http://127.0.0.1:18790/ready
```

日志建议检查关键字：
- `LLM requested tool calls`
- `Tool call`
- `Unsupported param: tools`（应消失）

## 系统日志保留策略（盒子运维）

为避免 `journalctl -f` 长期运行导致日志占满磁盘，建议固定 `journald` 上限与保留周期。

目标策略：
- 最长保留：7 天
- 日志总量上限：200MB
- 单日志文件上限：20MB

配置文件：`/etc/systemd/journald.conf`

建议配置：

```ini
[Journal]
Storage=persistent
Compress=yes
SystemMaxUse=200M
SystemKeepFree=200M
SystemMaxFileSize=20M
MaxRetentionSec=7day
```

生效与一次性清理：

```bash
systemctl restart systemd-journald
journalctl --vacuum-time=7d
journalctl --vacuum-size=200M
journalctl --disk-usage
```

排障时建议使用：

```bash
journalctl -u picoclaw -b -f
```

说明：
- `-b` 仅查看本次开机日志，减少历史噪音。
- `-f` 为实时跟随，不会自动清理历史日志；清理由 `journald` 策略负责。

## 2026-03-01 飞书通道增强（盒子实装记录）

本次在盒子分支上完成了飞书可用性和显示效果增强，重点如下。

涉及文件（源码）：
- `pkg/channels/feishu_64.go`
- `pkg/config/config.go`
- `pkg/config/defaults.go`

### 1) 飞书显示优化
- 输出链路保持：`interactive card -> post -> text`。
- 卡片支持分区渲染（结论/要点/下一步/建议）。
- 长回复分段展示，减少飞书群内刷屏。
- 卡片标题按内容自动选择（快报/客服/开发/默认）。
- 卡片增加更新时间。

### 2) 分页与快捷指令（文本指令）
- 长回复超限时，缓存后续分页（TTL 20 分钟）。
- 用户发送 `继续` 可拉取下一页。
- 支持 `重试`、`精简版` 两个会话内快捷指令。
- “快捷操作提示”仅在确实存在后续分页时展示。

### 3) 身份规则：义父识别（已配置化）
- 不再硬编码，改为读配置：
  - `channels.feishu.godfather_name`
  - `channels.feishu.godfather_ids`（建议优先使用）
- 会话中注入身份守卫：
  - 命中义父：允许“义父”称呼。
  - 非义父：明确禁止称其为“义父”。

### 4) 盒子配置示例

`/root/.picoclaw/config.json`：

```json
{
  "channels": {
    "feishu": {
      "godfather_name": "ToddSun",
      "godfather_ids": ["ou_4236080c0dde229c4fc624b25531b281"]
    }
  }
}
```

### 5) 运行态验证命令

```bash
# 服务状态
systemctl is-active picoclaw

# 飞书接收与身份判定
journalctl -u picoclaw -f | rg "feishu: Feishu message received|sender_id|sender_name|is_godfather"

# 检查义父配置项
jq '.channels.feishu | {godfather_name,godfather_ids}' /root/.picoclaw/config.json
```

备注：
- 若 `sender_name` 仍显示为 ID（如 `ou_xxx`），通常是飞书通讯录权限或可见范围不足；不影响 `godfather_ids` 规则生效。

## 2026-03-02 盒子源码独立同步目录（避免污染 GitHub 源仓库）

目的：
- 盒子运行态源码与本地 GitHub 源仓库解耦；
- 方便回溯“线上实际生效版本”；
- 避免临时修复直接混入主仓库工作树。

本地独立目录约定：
- 根目录：`/home/yukun/dev/picobox-ai/box-sync`
- 每次同步目录：`picoclaw-box-YYYY-MM-DD`
- 运行态覆盖配置单独存放：`_runtime_overrides/workhorse-agent/AGENTS.md`

当前已同步快照（示例）：
- `/home/yukun/dev/picobox-ai/box-sync/picoclaw-box-2026-03-02`

建议同步命令（本地执行）：

```bash
DEST_BASE=/home/yukun/dev/picobox-ai/box-sync
STAMP=$(date +%F)
DEST=$DEST_BASE/picoclaw-box-$STAMP
mkdir -p "$DEST"

# 1) 拉取盒子当前源码快照（不进入 GitHub 主仓库）
ssh root@192.168.1.57 'tar -C /userdata/picobox-ai/picoclaw/src -cf - .' | tar -C "$DEST" -xf -

# 2) 额外拉取运行态代理配置（避免遗漏本地定制）
mkdir -p "$DEST/_runtime_overrides/workhorse-agent"
scp root@192.168.1.57:/userdata/picoclaw-workspace/workhorse-agent/AGENTS.md \
  "$DEST/_runtime_overrides/workhorse-agent/AGENTS.md"
```

说明：
- 该目录只做“线上版本镜像与备份”，不作为主开发仓库。
- 若要回传上游，请在独立分支中对比后择优迁移，避免直接覆盖官方源码。

## 2026-03-02 agent 工具权限管控（配置优先，降低误操作风险）

目的：
- 避免客服 agent 执行高风险工具（例如 `exec`、在线安装技能）；
- 保留牛马 agent 的运维能力；
- 后续升级时尽量不再改业务逻辑，只调配置。

盒子源码变更点（v0.2.0 基线）：
- `pkg/config/config.go`：`AgentConfig` 新增 `allowed_tools`、`disabled_tools`
- `pkg/agent/instance.go`：增加 `IsToolEnabled` 与工具集合解析
- `pkg/agent/loop.go`：统一按 agent 策略过滤共享工具注册

当前推荐配置（示例）：

```json
{
  "agents": {
    "list": [
      {
        "id": "workhorse-agent",
        "disabled_tools": []
      },
      {
        "id": "whatsapp-cs-agent",
        "disabled_tools": ["find_skills", "install_skill", "exec"]
      }
    ]
  }
}
```

说明：
- `disabled_tools` 优先级高于 `allowed_tools`。
- 建议只在确有需要时配置 `allowed_tools`，默认以黑名单收敛风险即可。
- `OPENCLAW/ClawHub` token 不应暴露给客服 agent 工作流。

## 升级原则（保持不变）

1. 先对齐官方版本，再补本地能力。
2. 上游已提供同等能力时，删除本地重复逻辑。
3. 优先“配置化”而不是“硬编码”。
4. 生产改动先在 `box-sync` 快照留痕，再回灌文档与补丁。

## 2026-03-02 Skills 运行基线（按 agent 隔离）

目的：
- 防止升级后 skills 丢失或串装到错误 agent。
- 确保客服/牛马职责隔离持续生效。

当前基线：
- `whatsapp-cs-agent`: `customer-support`, `whatsapp-faq-bot`, `whatsapp-styling-guide`, `lead-scorer`
- `workhorse-agent`: `evolver`, `feishu-bitable`, `feishu-interactive-cards`, `feishu-sheets-skill`, `humanizer-zh`, `memory-manager`, `perplexica-search`(本地自建), `news-summary`, `rss-ai-reader`, `self-improving`, `skill-vetter`, `summarize`, `weather`

升级后核对命令（盒子）：

```bash
ls -1 /userdata/picoclaw-workspace/whatsapp-cs-agent/skills
ls -1 /userdata/picoclaw-workspace/workhorse-agent/skills
```

按 agent 安装技能的推荐方式（一次性环境变量，不改全局配置）：

```bash
# 安装到客服 agent
PICOCLAW_AGENTS_DEFAULTS_WORKSPACE=/userdata/picoclaw-workspace/whatsapp-cs-agent \
  picoclaw skills install --registry clawhub <slug>

# 安装到牛马 agent
PICOCLAW_AGENTS_DEFAULTS_WORKSPACE=/userdata/picoclaw-workspace/workhorse-agent \
  picoclaw skills install --registry clawhub <slug>
```

自建本地 skill 说明（不依赖 ClawHub）：
- `perplexica-search` 当前放在：
  - `/userdata/picoclaw-workspace/workhorse-agent/skills/perplexica-search/SKILL.md`
  - `/userdata/picoclaw-workspace/workhorse-agent/skills/perplexica-search/_meta.json`
- 升级后若 workspace 迁移，需手动同步该目录。

## 2026-03-02 服务环境变量治理（systemd EnvironmentFile）

目的：
- 避免把敏感 token 硬编码在 `picoclaw.service`。
- 后续改 token 只改 `.env` 文件，不改 unit 主文件。

落地结果（盒子）：
- 从 `/etc/systemd/system/picoclaw.service` 移除：
  - `Environment="GITHUB_TOKEN=..."`
  - `Environment="GITHUB_USERNAME=..."`
- 新增 drop-in：
  - `/etc/systemd/system/picoclaw.service.d/envfile.conf`
  - 内容：`EnvironmentFile=-/root/.picoclaw/picoclaw.env`
- 新增环境文件：
  - `/root/.picoclaw/picoclaw.env`（权限 `600`）

运维操作：
```bash
vi /root/.picoclaw/picoclaw.env
systemctl restart picoclaw
systemctl status picoclaw
```

注意：
- token 值不得包含错误前缀（例如 `github token:`）。

## 2026-03-02 代码改动确认（盒子源码）

本次确实修改了盒子源码并重新编译部署，主要包括：
- `pkg/channels/manager.go`
  - IM 出站清洗：WhatsApp 保持轻格式清洗；飞书清理 Markdown 泄漏符号。
- `pkg/channels/feishu/feishu_64.go`
  - 发送策略改为：长回复优先 `interactive card`，短回复优先 `post`，失败回退 `text`。

部署方式：
- 在盒子源码目录编译 `go build -tags whatsapp_native`
- 覆盖 `/usr/local/bin/picoclaw`
- `systemctl restart picoclaw`

## 更新补丁文件（当你改了补丁内容）

在你的补丁工作仓库执行：

```bash
cd REPO_DIR

git diff -- pkg/agent/loop.go pkg/agent/instance.go pkg/config/config.go \
  > /home/yukun/dev/picobox-ai/picoclaw/patches/0001-routing-and-agent-overrides.patch

git diff -- pkg/providers/toolcall_utils.go pkg/providers/toolcall_utils_test.go \
  > /home/yukun/dev/picobox-ai/picoclaw/patches/0002-toolcall-normalization.patch

# 如果是标准 git diff（a/b 前缀），建议用下面方式重建 0003
# （优先推荐：避免 old/new 前缀）
git diff -- pkg/agent/loop.go pkg/agent/llm_runner.go pkg/agent/session_runtime.go \
  pkg/agent/summarization_runtime.go pkg/agent/logging_formatters.go \
  > /home/yukun/dev/picobox-ai/picoclaw/patches/0003-agent-loop-modularization.patch

git diff -- pkg/config/config.go pkg/skills/registry.go pkg/skills/clawhub_registry.go \
  > /home/yukun/dev/picobox-ai/picoclaw/patches/0004-skills-registry-proxy.patch

git diff -- pkg/config/config.go pkg/skills/registry.go pkg/skills/clawhub_registry.go \
  > /home/yukun/dev/picobox-ai/picoclaw/patches/0005-skills-proxy-toggle.patch

git diff -- pkg/channels/manager.go pkg/formatters/output.go pkg/formatters/output_test.go \
  > /home/yukun/dev/picobox-ai/picoclaw/patches/0006-im-output-formatting.patch

git diff -- pkg/config/config.go pkg/config/defaults.go pkg/tools/web.go pkg/agent/loop.go pkg/tools/web_test.go \
  > /home/yukun/dev/picobox-ai/picoclaw/patches/0007-web-model-search-provider.patch
```

建议每次更新后在本文档顶部追加一条变更记录（日期 + 修改点）。

## 参考文档

- 升级判定清单 v2：`UPGRADE-CHECKLIST-v2.zh.md`
- 升级合并流程：`UPGRADE-SOP.zh.md`
- 0003 函数迁移细节：`PATCH-0003-DETAILS.zh.md`
- 0004 代理补丁细节：`PATCH-0004-DETAILS.zh.md`
- 0005 代理开关细节：`PATCH-0005-DETAILS.zh.md`
- 0006 IM 输出格式补丁细节：`PATCH-0006-DETAILS.zh.md`
- 0007 模型搜索 provider 细节：`PATCH-0007-DETAILS.zh.md`
