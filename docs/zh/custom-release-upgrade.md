# 自定义分支升级工作流

本文档用于维护基于上游 PicoClaw release 的自定义分支，目标是让后续升级流程稳定、可重复，而不是长期在一条脏分支上持续堆补丁。

## 远端约定

- `upstream`: `https://github.com/sipeed/picoclaw`
- `origin`: 你们自己的 fork (`https://github.com/sjiangtao2024/picoclaw`)

不要把 plain `git pull` 当成"已跟上游"的判断依据。当前分支如果跟踪的是 `origin/custom/...`，那么 `git pull` 只会同步你们自己的 fork，不会同步上游 release。

## 分支约定

- 上游正式版本以 tag 为准，例如 `v0.2.5`
- 你们的补丁分支以 `custom/release-<tag>-h618` 命名
- 示例：`custom/release-v0.2.5-h618`

这样做的好处是：

- 每次升级都有清晰基线
- 本地补丁范围可控
- 下次升级时只需要重新评估补丁是否仍然必要

## 当前补丁状态

`custom/release-v0.2.5-h618` 分支基于 `v0.2.5`，包含以下补丁：

| Commit | 描述 | 文件变化 |
|--------|------|---------|
| `b058da85` | feat: add SkillHub registry support with API hotfix | pkg/skills/, web/backend/api/, cmd/picoclaw/, docs/ |
| `a3cdf1aa` | feat: add web search settings page | web/frontend/ |

---

## 升级流程（v0.2.5 → v0.2.6）

### 步骤 1：同步上游

```bash
# 进入上游仓库（不是 worktree）
cd /home/yukun/dev/picobox-ai/github_repos/picoclaw

# 同步上游 tags 和 branches
git fetch upstream --tags
git fetch origin
```

### 步骤 2：创建新 worktree

使用脚本自动创建（推荐）：

```bash
cd /home/yukun/dev/picobox-ai/github_repos/picoclaw

# 创建 v0.2.6 的 worktree 和分支
./scripts/sync-upstream-release.sh v0.2.6
```

脚本会自动执行：
1. `git fetch upstream --tags`
2. `git fetch origin`
3. 基于 `v0.2.6` tag 创建新分支 `custom/release-v0.2.6-h618`
4. 在 `../picoclaw-v0.2.6-h618` 创建独立 worktree

**如果网络受限**，可以先手动 fetch，再跳过网络步骤：

```bash
SKIP_FETCH=1 ./scripts/sync-upstream-release.sh v0.2.6
```

### 步骤 3：进入新 worktree

```bash
cd ../picoclaw-v0.2.6-h618
git status
git log --oneline -3  # 确认基于 v0.2.6
```

### 步骤 4：评估和回放补丁

```bash
# 查看当前分支需要哪些补丁
git log --oneline upstream/v0.2.6..HEAD

# 逐个 cherry-pick 补丁
git cherry-pick <commit1>
git cherry-pick <commit2>
```

**如何判断某个补丁是否还需要：**

1. 上游是否已经实现了相同功能？
2. 补丁修改的文件在 v0.2.6 中是否有冲突？
3. 冲突是否可以轻松解决（简单文本替换）？

如果上游已经等效实现，**废弃该补丁**，不用 cherry-pick。

### 步骤 5：解决冲突

如果 cherry-pick 出现冲突：

```bash
# 查看冲突文件
git status

# 解决冲突后
git add <resolved-files>
git cherry-pick --continue
```

或者放弃此次 cherry-pick：

```bash
git cherry-pick --abort
```

### 步骤 6：构建测试

```bash
# 构建 ARM64 版本
cd cmd/picoclaw
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags "goolm,stdjson" -o /tmp/picoclaw .

# 构建 web launcher
cd ../web
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags "goolm,stdjson" -o /tmp/picoclaw-web ./backend/
```

### 步骤 7：部署测试

```bash
# 复制到 H618
scp -i ~/.ssh/id_ed25519_192168161 /tmp/picoclaw-web root@192.168.1.63:/root/picoclaw/bin/picoclaw-web

# 重启服务
ssh -i ~/.ssh/id_ed25519_192168161 root@192.168.1.63 "systemctl restart picoclaw-web"
ssh -i ~/.ssh/id_ed25519_192168161 root@192.168.1.63 "systemctl status picoclaw-web --no-pager"
```

### 步骤 8：推送

```bash
git push -u origin custom/release-v0.2.6-h618
```

---

## 补丁管理规范

### 何时创建新补丁

当需要给上游代码打自定义修改时：

1. **功能类补丁**（新增 registry、新增 channel 支持等）→ 单独 commit
2. ** bug 修复类补丁**（修复 upstream 漏掉的配置传递等）→ 单独 commit
3. **文档类补丁** → 可以合并到功能补丁中

### 如何合并多个相关补丁

如果多个补丁修改的是同一功能的不同方面，可以合并：

```bash
# 查看最近的多个补丁
git log --oneline -5

# 软合并：把这些补丁合并成一个
git reset --soft <在他们之前的commit>
git commit -m "feat: 描述合并后的功能"
```

### 热补丁（Hot Patch）

如果 upstream 发布了新版本但还没来得及完整升级，可以用 cherry-pick 方式单独应用某个关键修复：

```bash
# 在当前分支上 cherry-pick 特定的修复 commit
git cherry-pick <fix-commit-hash>

# 推送
git push origin custom/release-v0.2.x-h618
```

---

## 常用命令速查

```bash
# 查看当前分支基于哪个 tag
git describe --tags --abbrev=0

# 查看所有自定义分支
git branch -a | grep custom

# 查看补丁列表
git log --oneline upstream/v0.2.5..HEAD

# 查看某个 commit 的修改内容
git show <commit-hash> --stat

# 查看补丁在哪些文件有改动
git diff upstream/v0.2.5..HEAD --stat

# 删除旧的 worktree（升级完成后）
git worktree remove ../picoclaw-v0.2.4-h618

# 查看远程分支
git branch -r | grep custom
```

---

## 故障排除

### "refusing to merge unrelated histories"

首次合并 upstream 到 origin 时可能遇到：

```bash
git pull upstream v0.2.6 --allow-unrelated-histories
```

### "cherry-pick 冲突太多"

说明该补丁已经不适用于新版，建议：

1. 检查上游是否已实现相同功能
2. 只 cherry-pick 扔有效的部分
3. 重新编写补丁

### worktree 冲突

如果 worktree 目录已存在：

```bash
# 查看现有 worktrees
git worktree list

# 删除旧的（确保已推送或不需要）
git worktree remove ../picoclaw-v0.2.4-h618
```
