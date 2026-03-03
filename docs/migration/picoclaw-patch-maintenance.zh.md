# PicoClaw 补丁维护手册（v0.2.0 对齐版）

本文档用于维护“尽量跟随主线 + 保留必要个性化补丁”的分发策略。

当前基线：
- 上游标签：`v0.2.0`
- 上游 commit：`8207c1c`
- 运行二进制（盒子）：`/usr/local/bin/picoclaw`

## 1. 补丁策略（总原则）

- 优先使用上游官方实现。
- 仅保留业务必需、官方未完全覆盖的补丁。
- 每次升级先对齐主线，再最小回填个性化补丁。

## 2. 当前补丁状态（v0.2.0）

### 2.1 仍保留的个性化补丁

#### P1. 模型别名路由修复（保留）
- 作用：把运行时 `model_name`（含异常前导 `/`）映射到 `model_list` 对应 provider/model。
- 文件：
  - `pkg/agent/loop.go`
  - `pkg/agent/loop_test.go`
- 关键标记：
  - `resolveModelTarget(...)`
  - `targetProvider.Chat(... targetModel ...)`

#### P2. Agent 级参数覆盖 defaults（保留）
- 作用：让每个 agent 的 `max_tool_iterations`、`max_tokens`、`temperature` 生效。
- 文件：
  - `pkg/config/config.go`
  - `pkg/agent/instance.go`
- 关键标记：
  - `AgentConfig` 包含 `MaxToolIterations/MaxTokens/Temperature`
  - `if agentCfg != nil && agentCfg.MaxToolIterations > 0 { ... }`

#### P3. 工具迭代耗尽兜底优化（保留）
- 作用：避免英文空话兜底，改为更明确中文提示。
- 文件：
  - `pkg/agent/loop.go`
- 关键标记：
  - `if finalContent == "" && iteration >= agent.MaxIterations { ... }`

#### P5. Tool call 规范化增强（保留）
- 作用：在现有 Normalize 基础上，自动补齐 `id/type`，减少兼容性问题。
- 文件：
  - `pkg/providers/toolcall_utils.go`
  - `pkg/providers/toolcall_utils_test.go`
- 关键标记：
  - `toolCallIDSeq`
  - `NormalizeToolCall(...)`

#### P6. WhatsApp Native 代理支持（保留）
- 作用：在国内网络环境下，让 WhatsApp Native 连接走 `channels.whatsapp.proxy`（或环境变量代理）以避免直连超时。
- 文件：
  - `pkg/config/config.go`
  - `pkg/channels/whatsapp_native/whatsapp_native.go`
- 关键标记：
  - `PICOCLAW_CHANNELS_WHATSAPP_PROXY`
  - `resolveWhatsAppProxy(...)`
  - `client.SetProxyAddress(...)`

#### P7. ClawHub Skills 安装代理支持（保留）
- 作用：让 `skills search/install` 访问 ClawHub 时可走专用代理，降低超时与 429 风险。
- 文件：
  - `pkg/config/config.go`
  - `pkg/skills/registry.go`
  - `pkg/skills/clawhub_registry.go`
- 关键标记：
  - `tools.skills.registries.clawhub.proxy`
  - `PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_PROXY`
  - `http.ProxyFromEnvironment` / `http.ProxyURL(...)`

### 2.2 已废弃（改用官方）

以下历史补丁在 v0.2.0 已切换为官方实现，不再维护本地分叉：
- WhatsApp Native 通道主体逻辑（改用官方 channel 体系；仅保留 P6 代理补丁）
- `pkg/channels/manager.go` 的大规模自定义调度逻辑
- `pkg/providers/openai_compat/provider.go` 的本地大改版本
- `pkg/tools/web.go` / `pkg/tools/web_test.go` 的本地增强版
- `go.mod` / `go.sum` / `config/config.example.json` 的本地分叉依赖结构

### 2.3 已被官方吸收

#### P4. 会话压缩提示静默化（官方已覆盖）
- 结论：v0.2.0 下不再向用户发送压缩提示消息，当前保留为日志行为。

## 3. 配置兼容性结论（重点）

问题：使用官方改进后，当前配置还能用吗？

结论：**可以继续用（整体兼容）**，但建议逐步迁移到 v0.2.0 推荐写法。

### 3.1 可直接沿用
- 现有 `agents/defaults/model[_name]` 配置可继续运行。
- 现有 provider 配置（OpenAI 兼容、Qwen、Moonshot 等）可继续运行。
- 当前线上 `/root/.picoclaw/config.json` 不需要立即重写。

### 3.2 建议迁移（非必须）
- WhatsApp：推荐使用官方字段 `use_native` + `session_store_path`（不要继续使用历史 `mode/data_dir`）。
- `model_list`：确保 `model_name` 唯一，`model` 写完整协议前缀（如 `openai/...`、`qwen/...`）。

### 3.3 需注意
- 若 `model_list` 中使用当前构建不支持的协议（例如特定 `codex/...` 路由），会自动回退到当前 provider，不会直接崩溃；但应尽快统一到可用协议配置。

## 4. 升级标准流程

1. 备份运行配置与二进制
```bash
cp /root/.picoclaw/config.json /root/.picoclaw/config.json.bak-$(date +%F-%H%M%S)
cp /usr/local/bin/picoclaw /usr/local/bin/picoclaw.bak-$(date +%F-%H%M%S)
```

2. 对齐上游（示例：v0.2.0）

3. 仅回填 P1/P2/P3/P5/P6/P7

4. 测试与编译
```bash
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
go test ./pkg/agent ./pkg/config ./pkg/providers/...
go build -tags whatsapp_native -o /tmp/picoclaw ./cmd/picoclaw
```

5. 发布替换并重启
```bash
install -m 0755 /tmp/picoclaw /usr/local/bin/picoclaw
systemctl restart picoclaw.service
systemctl status picoclaw.service --no-pager -l
```

## 5. 分支策略（建议）

- `upstream-sync`：纯上游同步
- `release/custom`：仅保留 P1/P2/P3/P5/P6/P7

每次升级：
1. `upstream-sync` 拉新
2. `release/custom` 最小补丁回填
3. 测试通过后打 tag（如 `custom-v0.2.0-r1`）

## 6. 分发前检查清单

- 不把真实 API key 写入仓库。
- `config.json` 保持模板化，密钥通过部署注入。
- 代理地址改为可配置项，不写死客户环境。
- 记录发布二进制 SHA256，便于回滚与审计。
