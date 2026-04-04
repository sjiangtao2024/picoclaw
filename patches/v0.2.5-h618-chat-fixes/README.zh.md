# v0.2.5 H618 聊天链路修复补丁集

目录：

- `/home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes`

用途：

- 作为后续升级到上游新版本时的可重放补丁集
- 固化 2026-04-04 在 H618 盒子上验证通过的聊天链路修复
- 不包含运行机上的业务配置恢复，例如 `agents.list`、`大丫鬟`、`workspace-*`

适用基线：

- `/home/yukun/dev/picobox-ai/github_repos/picoclaw-v0.2.5-h618`

补丁顺序：

1. `0001-web-config-validation.patch`
2. `0002-frontend-chat-reconnect-and-polling.patch`
3. `0003-pico-agent-diagnostics-and-allowlist.patch`
4. `0004-skillhub-defaults-and-compat.patch`
5. `0005-web-chat-default-agent-selector.patch`
6. `0006-web-skills-default-agent-workspace.patch`

补丁说明：

## 0001-web-config-validation.patch

作用：

- 修复 Web 配置保存时允许 `gateway.port = 0` 的问题
- 增加回归测试，阻止无效端口再次落盘

## 0002-frontend-chat-reconnect-and-polling.patch

作用：

- 将稳定态 `/api/gateway/status` 轮询降到 `60s`
- 过渡态轮询改为 `3s`
- 聊天发送前自动重连 WebSocket
- 为假在线 socket 增加 `ping/pong` 健康检查
- 当 gateway `pid` 变化时强制重连聊天连接
- 增加前端回归测试

## 0003-pico-agent-diagnostics-and-allowlist.patch

作用：

- 给 Pico token、WebSocket 握手、Pico message、bus、AgentLoop 增加诊断日志
- 修复 `channels.pico.allow_from = [""]` 时会静默拒绝所有 Pico 消息的问题
- 将空白 allowlist 归一为“空列表 = 允许全部”
- 增加对应回归测试

## 0004-skillhub-defaults-and-compat.patch

作用：

- 固化“默认使用腾讯 SkillHub，ClawHub 作为兼容名”的行为
- 当旧调用仍传 `clawhub`，但实际只启用了 `skillhub` 时，自动回退到 `skillhub`
- 将用户侧 `install_skill` 参数示例改为 `skillhub`
- 增加回归测试，锁住默认 SkillHub 配置与 Web API 的 `skillhub` URL 生成

## 0005-web-chat-default-agent-selector.patch

作用：

- 给 Web 聊天页增加默认 Agent 选择器
- 选择后通过 `PATCH /api/config` 更新 `agents.list[].default`
- 仅修改“全局默认 Agent”，不强切当前正在聊天的会话
- 切换成功后提示“重启服务后，新的会话将使用新的默认 Agent”
- 增加前端回归测试，锁住 Agent 列表解析与默认 Agent patch 生成逻辑

## 0006-web-skills-default-agent-workspace.patch

作用：

- 修复 Web skills 页面仍固定使用默认 workspace 的问题
- `skills` 的列表、安装、删除、导入统一切到“当前默认 Agent 的 workspace”
- 避免出现“页面安装成功，但默认 Agent 实际看不到 skill”的错位
- 增加后端回归测试，锁住“默认 Agent workspace 优先”的行为

重要说明：

- 这个补丁集只覆盖源码。
- 运行机上的 `大丫鬟` agent 配置是运行态配置，不在 patch 里。
- 如需恢复 `大丫鬟`，应从运行机备份恢复 `config.json` 中的 `agents.list` 与默认 `agent_id`。

应用方式：

```bash
cd REPO_DIR

git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0001-web-config-validation.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0002-frontend-chat-reconnect-and-polling.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0003-pico-agent-diagnostics-and-allowlist.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0004-skillhub-defaults-and-compat.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0005-web-chat-default-agent-selector.patch
git apply --check /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0006-web-skills-default-agent-workspace.patch

git apply /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0001-web-config-validation.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0002-frontend-chat-reconnect-and-polling.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0003-pico-agent-diagnostics-and-allowlist.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0004-skillhub-defaults-and-compat.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0005-web-chat-default-agent-selector.patch
git apply /home/yukun/dev/picobox-ai/picoclaw/patches/v0.2.5-h618-chat-fixes/0006-web-skills-default-agent-workspace.patch
```

建议回归：

```bash
go test ./web/backend/api -run TestHandlePatchConfig_RejectsZeroGatewayPort -count=1
go test ./pkg/channels -run 'TestBaseChannelIsAllowed|TestIsAllowedSender' -count=1
go test ./pkg/skills -run 'TestRegistryManagerGetRegistry|TestRegistryManagerGetRegistryFallsBackFromClawHubToSkillHub' -count=1
go test ./pkg/tools -run TestInstallSkillToolParameters -count=1
go test ./pkg/config -run 'TestSkillsRegistriesConfig_ParsesSkillHub|TestDefaultConfigPrefersSkillHubOverClawHub' -count=1
go test ./web/backend/api -run TestRegistrySkillURLUsesSkillHubTemplate -count=1
node --test web/frontend/src/store/gateway.test.ts
node --test web/frontend/src/features/chat/socket-connection.test.ts
node --test web/frontend/src/features/chat/gateway-reconnect.test.ts
node --test web/frontend/src/components/chat/agent-selector-utils.test.ts
go test ./web/backend/api -run 'TestHandleListSkillsUsesDefaultAgentWorkspace|TestHandleInstallSkillUsesDefaultAgentWorkspace' -count=1
```
