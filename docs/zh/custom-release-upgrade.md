# 自定义分支升级工作流

本文档用于维护基于上游 PicoClaw release 的自定义分支，目标是让后续升级流程稳定、可重复，而不是长期在一条脏分支上持续堆补丁。

## 远端约定

- `upstream`: `https://github.com/sipeed/picoclaw`
- `origin`: 你们自己的 fork

不要把 plain `git pull` 当成"已跟上游"的判断依据。当前分支如果跟踪的是 `origin/custom/...`，那么 `git pull` 只会同步你们自己的 fork，不会同步上游 release。

## 分支约定

- 上游正式版本以 tag 为准，例如 `v0.2.4`
- 补丁以 `patches/` 目录下的 `.patch` 文件形式管理
- 每次升级应用补丁后，建议在 `origin` 上创建对应的 release 分支，如 `custom/release-v0.2.4-h618`

这样做的好处是：

- 每次升级都有清晰基线
- 补丁范围可控、可审查
- 下次升级时只需要重新评估补丁是否仍然必要
- 不需要维护长期叠加补丁的 dirty branch

## 补丁目录

本项目所有自定义修改都保存在仓库根目录的 `patches/` 目录中：

```
patches/
├── README.md                        # 补丁说明和应用指南
├── 0001-preroute-system.patch       # 强制路由系统（help/news/image）
├── 0003-config-forced-routes.patch # Config 中添加 Routes 字段
├── 0004-modelscope-image.patch     # ModelScope 图片生成工具
└── 0005-sync-script.patch         # 升级同步脚本
```

各补丁的具体说明：

| 序号 | 补丁文件 | 说明 | 适用版本 |
|------|----------|------|----------|
| 0001 | preroute-system.patch | 强制路由系统（help/news/image） | 上游 main |
| 0003 | config-forced-routes.patch | Config 中添加 Routes 字段 | 上游 main |
| 0004 | modelscope-image.patch | ModelScope 图片生成工具 | 上游 main |
| 0005 | sync-script.patch | 升级同步脚本 | 上游 main |

注：`0002-pico-channel.patch` 已移除（包含 Channel 接口变更，需要单独审核处理）。

## 升级流程

### 方式一：使用补丁文件（推荐）

每次以上游新 release 为起点，应用补丁：

```bash
# 1. 同步上游标签
git fetch upstream --tags

# 2. 切换到新版本
git checkout v0.2.6

# 3. 应用所有补丁
git apply patches/*.patch

# 4. 处理冲突（如果有）
# 如果冲突，逐一解决后继续
git apply --continue

# 5. 验证构建
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags goolm -o picoclaw-arm64 ./cmd/picoclaw

# 6. 测试通过后，推送到 origin
git push -u origin v0.2.6
```

### 方式二：使用 release worktree

如果需要隔离开发，可以使用脚本创建独立 worktree：

```bash
./scripts/sync-upstream-release.sh v0.2.6
cd ../picoclaw-v0.2.6
git apply patches/*.patch
# ... 测试和验证
git push -u origin custom/release-v0.2.6-h618
```

## 新增或修改补丁

补丁基于干净的 upstream tag 生成，确保在任意版本上都能应用。

### 修改现有文件

```bash
# 做完修改后，生成 diff
git diff HEAD -- path/to/file.go > patches/xxxx-descriptive-name.patch
```

### 新增文件

```bash
# 使用 diff --no-index 生成新文件补丁
git diff --no-index /dev/null path/to/newfile.go > patches/xxxx-newfile.patch
```

### 验证补丁

每次修改补丁后，务必在干净的上游版本上验证：

```bash
# 在临时目录验证
cd /tmp
rm -rf test-repo
git clone --depth=1 --branch=v0.2.5 https://github.com/sipeed/picoclaw.git test-repo
cd test-repo
git apply --check /path/to/your.patch
# 如果失败，git apply 会报错
```

## 运行时配置

补丁只负责源码修改。某些功能需要在 `config.json` 中单独启用，例如：

```json
{
  "routes": {
    "forced": {
      "enabled": true,
      "order": ["help", "news", "image"],
      "features": {
        "help": true,
        "news": true,
        "image": true
      },
      "news": {
        "cli_relative_path": "skills/tencent-news/tencent-news-cli"
      }
    }
  }
}
```

运行时配置的修改需要单独同步到部署环境，不在本仓库的补丁管理范围内。

## 核心原则

- 升级基线只认上游 release tag
- 源码修改只保留在 `patches/` 目录
- 每次升级重新判断补丁是否仍然必要
- 不再依赖单纯的 `git pull` 判断是否"同步上游"
- 补丁必须能在干净的上游版本上通过 `git apply`
