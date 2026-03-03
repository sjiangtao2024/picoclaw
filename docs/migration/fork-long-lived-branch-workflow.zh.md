# PicoClaw Fork 长期分支工作流（本地定制版）

适用目录：`/home/yukun/dev/picobox-ai/github_repos/picoclaw`

目标：
- 你自己的定制开发走 fork 仓库；
- 持续跟进上游 `sipeed/picoclaw` 更新；
- 固定“长期分支”避免时间久了忘记流程。

## 1. 远端约定（固定）

- `upstream`：官方仓库（只拉取）
  - `https://github.com/sipeed/picoclaw.git`
- `origin`：你的 fork（日常推送）
  - `https://github.com/<your-github-username>/picoclaw.git`

说明：
- 你的本地仓库当前 `origin` 仍是官方仓库。
- 在 fork 创建完成前，不要改动 `origin`。

## 2. 长期分支约定（固定）

- 分支名：`custom/main`
- 作用：你所有长期个性化改动都汇总到该分支
- 禁止直接在 `upstream/main` 上做开发

可选短期分支命名：
- `feat/<topic>`
- `fix/<topic>`
- `ops/<topic>`

## 3. 首次初始化（fork 创建后执行一次）

```bash
cd /home/yukun/dev/picobox-ai/github_repos/picoclaw

# 1) 把官方设为 upstream（如果已存在会报错，可忽略）
git remote add upstream https://github.com/sipeed/picoclaw.git

# 2) 把 origin 改为你自己的 fork
git remote set-url origin https://github.com/<your-github-username>/picoclaw.git

# 3) 拉取远端
git fetch upstream
git fetch origin

# 4) 从官方 main 创建长期分支
git checkout -B custom/main upstream/main

# 5) 首次推送并建立跟踪
git push -u origin custom/main
```

## 4. 日常开发流程（固定）

```bash
cd /home/yukun/dev/picobox-ai/github_repos/picoclaw
git checkout custom/main
git pull --ff-only origin custom/main

# 开发（可直接在 custom/main，或新建短分支）
# git checkout -b feat/xxx

git add -A
git commit -m "feat: ..."
git push origin custom/main
```

## 5. 同步上游更新（建议每周/每次发版前）

```bash
cd /home/yukun/dev/picobox-ai/github_repos/picoclaw
git fetch upstream
git checkout custom/main

# 推荐 merge，保留历史最清晰
git merge upstream/main

# 解决冲突后
git push origin custom/main
```

若你偏好线性历史，也可改用：

```bash
git rebase upstream/main
git push --force-with-lease origin custom/main
```

## 6. 你当前仓库的特别说明

当前工作区存在未提交改动（定制代码中）。建议先完成一次“定制基线提交”：

```bash
cd /home/yukun/dev/picobox-ai/github_repos/picoclaw
git checkout -B custom/main
git add -A
git commit -m "chore: snapshot local customizations baseline"
```

然后再推送到 fork：

```bash
git push -u origin custom/main
```

## 7. 自检命令（防遗忘）

每次开始开发前先跑：

```bash
cd /home/yukun/dev/picobox-ai/github_repos/picoclaw
git remote -v
git branch --show-current
git status -sb
```

期望：
- `origin` 指向你的 fork
- 当前分支是 `custom/main`（或短期分支）
- 工作区状态可解释（没有意外脏改动）
