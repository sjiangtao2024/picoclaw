# PicoClaw + CLIProxyAPI Tool Calling 运维说明（升级防覆盖）

## 1. 结论（先看这个）
- 是的，后续更新 `picoclaw`（拉新代码或替换二进制）时，这个修复有可能被覆盖。
- `cliproxyapi` 本身无需改动；问题根因在 `picoclaw` 的模型别名解析与 provider 路由。
- 建议把修复作为你自己的长期补丁，随每次升级自动应用并回归测试。

## 2. 背景与根因
你在 `config.json` 中使用了：
- `model_name`: 例如 `gpt-5.2-codex`（别名）
- `model`: 例如 `codex/gpt-5.2-codex`（真实协议+模型）

旧逻辑在 fallback/tool-calling 路径中没有始终按 `model_name -> model_list` 回查 provider，导致别名有时走到错误 provider（如默认 `openai`），最终请求打到错误端点（`/chat/completions`），并携带 `tools` 参数，引发：
- `Unsupported param: tools`

## 3. 本次修复点
核心改动：
- `pkg/agent/loop.go`
  - 新增 `resolveModelTarget(...)`：把运行时模型引用映射到 `model_list` 的真实 provider/model。
  - 对前导 `/` 的异常模型名做容错（例如 `/gpt-5.2-codex`）。
  - 在单模型与 fallback 两条调用链都统一使用该解析结果发请求。
- `pkg/agent/loop_test.go`
  - 新增 3 个回归测试，覆盖：
    - 别名解析到 model_list
    - 前导 `/` 仍可正确解析
    - 未命中 alias 时保持 fallback 行为

## 4. 会不会影响正常模型（Kimi/Qwen/OpenAI 等）
正常不会，且行为更可控：
- 在 `model_list` 有定义的模型：现在会稳定走对应 provider。
- 不在 `model_list` 的模型：保留 fallback 行为，不破坏旧逻辑。
- 关键前提：`model_list` 中 `model_name` 保持唯一，`model` 写成带协议前缀（如 `qwen/...`、`moonshot/...`、`openai/...`、`codex/...`）。

## 5. 升级后的标准操作（推荐）

## 5.1 升级前备份
```bash
cp /root/.picoclaw/config.json /root/.picoclaw/config.json.bak-$(date +%F-%H%M%S)
cp /userdata/picobox-ai/picoclaw/picoclaw /userdata/picobox-ai/picoclaw/picoclaw.bak-$(date +%F-%H%M%S)
```

## 5.2 升级代码后重新应用补丁
建议维护你自己的分支（如 `ops/picoclaw-patches`），每次升级后 `cherry-pick` 这两个文件改动。

至少确认这两个文件包含修复：
- `pkg/agent/loop.go`
- `pkg/agent/loop_test.go`

## 5.3 回归测试
```bash
cd /userdata/picobox-ai/picoclaw/src
go test ./pkg/agent
go test ./pkg/providers/...
```

## 5.4 重新构建与启动
```bash
cd /userdata/picobox-ai/picoclaw/src
go build -o /userdata/picobox-ai/picoclaw/picoclaw ./cmd/picoclaw

pkill -f '^./picoclaw gateway$' || true
cd /userdata/picobox-ai/picoclaw
nohup ./picoclaw gateway > /tmp/picoclaw.log 2>&1 &
```

## 5.5 运行态验收
```bash
curl -sS http://127.0.0.1:18790/health
tail -n 200 /tmp/picoclaw.log | grep -E 'LLM requested tool calls|Tool call|Unsupported param: tools|Endpoint:' -n
```

通过标准：
- 能看到 `LLM requested tool calls` 与 `Tool call`
- 不再出现 `Unsupported param: tools`
- 不再看到 codex 场景误打到 `Endpoint: /chat/completions`

## 6. 回滚方案
若升级后异常：
```bash
cp /userdata/picobox-ai/picoclaw/picoclaw.bak-YYYY-MM-DD-HHMMSS /userdata/picobox-ai/picoclaw/picoclaw
pkill -f '^./picoclaw gateway$' || true
cd /userdata/picobox-ai/picoclaw
nohup ./picoclaw gateway > /tmp/picoclaw.log 2>&1 &
```

## 7. 长期建议
- 把此修复提交到你自己的远程仓库分支，避免“手工改一次、下次丢一次”。
- 若你计划长期跟官方上游同步，建议提一个 PR 到上游，这样后续升级可直接吃到官方修复。
