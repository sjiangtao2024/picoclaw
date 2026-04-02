# 自定义分支升级工作流

本文档用于维护基于上游 PicoClaw release 的自定义分支，目标是让后续升级流程稳定、可重复，而不是长期在一条脏分支上持续堆补丁。

## 远端约定

- `upstream`: `https://github.com/sipeed/picoclaw`
- `origin`: 你们自己的 fork

不要把 plain `git pull` 当成“已跟上游”的判断依据。当前分支如果跟踪的是 `origin/custom/...`，那么 `git pull` 只会同步你们自己的 fork，不会同步上游 release。

## 分支约定

- 上游正式版本以 tag 为准，例如 `v0.2.4`
- 你们的补丁分支以 `custom/release-<tag>-h618` 命名
- 示例：`custom/release-v0.2.4-h618`
- 对外稳定入口保留为 `custom/main`
- `custom/main` 不再直接承载开发，而是始终指向当前生效的 `custom/release-*` 分支
- `custom/release-*` 表示“已回放最小补丁栈并通过盒子 smoke 的可运行状态”，不是纯上游镜像

这样做的好处是：

- 每次升级都有清晰基线
- 本地补丁范围可控
- 下次升级时只需要重新评估补丁是否仍然必要

## 目录约定

运行机上的以下目录默认原地保留，不随主程序升级覆盖：

- `/root/picoclaw/config`
- `/root/picoclaw/workspace`
- `/root/picoclaw/workspace-dayahuan`
- `/root/picoclaw/workspace-image-agent`
- `/root/picoclaw-plugins`

升级时只替换：

- `/root/picoclaw/bin/picoclaw`
- `/root/picoclaw/bin/picoclaw-web`
- 必要的前端静态资源

## 创建新的升级 worktree

仓库提供了脚本：

```bash
./scripts/sync-upstream-release.sh v0.2.4
```

脚本会执行：

1. `git fetch upstream --tags`
2. `git fetch origin`
3. 基于 release tag 创建新分支
4. 创建独立 worktree，避免污染当前工作树

如果当前环境的网络访问受代理或沙箱限制，可以先手动执行 `git fetch upstream --tags` 和 `git fetch origin`，再使用：

```bash
SKIP_FETCH=1 ./scripts/sync-upstream-release.sh v0.2.4
```

默认产物：

- 分支：`custom/release-v0.2.4-h618`
- worktree：仓库同级目录下的 `picoclaw-v0.2.4-h618`

## 回放补丁

进入新 worktree 后，只回放仍然需要保留的补丁：

```bash
cd ../picoclaw-v0.2.4-h618
git cherry-pick <commit1> <commit2> ...
```

推荐最小补丁栈：

1. `modelscope-image tool`
2. `subagent` 稳定性补丁
3. `pico media output`
4. `linux nocgo web/launcher`

如果某个补丁的功能已经被上游等效实现，不要强行保留，直接废弃即可。

建议在本地把补丁分成两类：

- 平台补丁：`subagent`、`pico media`、`linux nocgo`
- 业务补丁：`modelscope-image`

补丁目录本身建议继续保留在你们自己的运维仓中，由发布流程按顺序重放。fork 仓库负责保存“当前 release 分支的已验证代码状态”。

## 推送到你们自己的远端

补丁回放和测试通过后，推送到 `origin`：

```bash
git push -u origin custom/release-v0.2.4-h618
```

推荐把这一步视为正式发布前的固定动作：

1. 新 worktree 上完成补丁重放
2. 本地完成构建和 smoke
3. 将结果提交到 `custom/release-*`
4. 推送到 fork
5. 再从该分支产出部署二进制

如果你们还保留一个统一入口分支，比如 `custom/main`，建议只做快进：

```bash
git checkout custom/main
git merge --ff-only custom/release-v0.2.4-h618
git push origin custom/main
```

不要再在 `custom/main` 上直接叠加临时改动。

旧分支例如 `custom/h618-migration` 应视为历史分支。建议在切换入口后保留一个带日期的归档引用，例如：

```bash
git branch archive/custom-h618-migration-2026-03-31 custom/h618-migration
git push origin archive/custom-h618-migration-2026-03-31
```

## 后续升级流程

以上游新 release 为起点，例如 `v0.2.5`：

```bash
git fetch upstream --tags
./scripts/sync-upstream-release.sh v0.2.5
cd ../picoclaw-v0.2.5-h618
git cherry-pick <仍需保留的补丁>
git push -u origin custom/release-v0.2.5-h618
```

推荐固定流程：

1. `git fetch upstream --tags`
2. 创建 `custom/release-vX.Y.Z-h618`
3. 重放最小补丁栈
4. 本地构建 `picoclaw` 与 `picoclaw-web`
5. 先投测试盒子做 smoke
6. 通过后推送 fork 并更新 `custom/main`

## 61 机器上的推荐形态

推荐把 61 视为纯运行机，只保留：

- `/root/picoclaw/bin/picoclaw`
- `/root/picoclaw/bin/picoclaw-web`
- `/root/picoclaw/config/config.json`
- `/root/picoclaw/config/launcher-config.json`
- `/etc/systemd/system/picoclaw-web.service`

不建议长期保留源码仓库、Node.js、Go 构建链或 Docker 构建环境。

## 61 机器上的固定升级入口

建议在 61 上固定一个二进制发布目录：

```bash
/root/releases/picoclaw
```

把本地构建好的两个文件上传到这里：

- `/root/releases/picoclaw/picoclaw-web-linux-arm64`
- `/root/releases/picoclaw/picoclaw`

然后在 61 上执行固定命令：

```bash
/usr/local/bin/picoclaw-upgrade-binaries
```

这个命令默认会从 `/root/releases/picoclaw` 读取二进制，备份旧文件，然后替换 `/root/picoclaw/bin` 下的运行文件并重启 `picoclaw-web.service`。

如果需要显式指定路径，也可以：

```bash
/usr/local/bin/picoclaw-upgrade-binaries /path/to/picoclaw-web-linux-arm64 /path/to/picoclaw
```

核心原则：

- 升级基线只认上游 release tag
- 自定义功能只保留在 `custom/release-*` 分支
- 每次升级重新判断补丁是否仍然必要
- 不再依赖单纯的 `git pull` 判断是否“同步上游”
- fork 仓库保存“已验证代码状态”，补丁目录保存“可重放变更集合”
