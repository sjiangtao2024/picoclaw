# 上游同步记录（2026-03-03）

仓库：`/home/yukun/dev/picobox-ai/github_repos/picoclaw`

## 目标

在不丢失本地定制补丁的前提下，将 `upstream/main` 同步到长期分支体系。

## 保护措施

- 备份分支：`backup/custom-main-2026-03-03`
- 备份标签：`backup-before-upstream-sync-2026-03-03`
- 同步工作分支：`sync/upstream-2026-03-03`
- 合并策略：`git merge --no-ff -X ours upstream/main`
  - 含义：冲突位置默认保留我方（定制）实现，再逐项引入上游新增能力。

## 结果摘要

- 同步前：`custom/main` 相对 `upstream/main` 为 `ahead 1 / behind 132`
- 合并后：上游新增能力已进入同步分支（含 MCP、WeCom AIBot 等）
- 未出现未解决冲突（无 `U` 状态文件）

## 同步后修复

在全量编译中发现一处冲突后遗症：

- 文件：`pkg/providers/toolcall_utils.go`
- 问题：`buildCLIToolsPrompt` 在我方版本中缺失，导致 `claude_cli_provider.go` / `codex_cli_provider.go` 编译失败。
- 处理：补回上游 `buildCLIToolsPrompt`，同时保留我方 `NormalizeToolCall` 增强逻辑（自动补齐 `id/type`）。

## 验证

已通过：

```bash
../../.tools/go/bin/go build ./cmd/picoclaw
../../.tools/go/bin/go build ./...
```

## 后续建议

- 仅在同步分支验证通过后，才回合到 `custom/main` 并推送到 fork。
- 后续每次上游同步都复用同一策略：`backup -> sync branch -> merge -X ours -> build -> promote`。
