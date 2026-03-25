# H618 Web 交付与升级说明

这份说明面向 H618 一类 ARM64 设备，用于部署 `picoclaw-web` 单二进制版本，并尽量降低后续升级成本。

## 目标

- 在开发机上构建 `linux/arm64` 二进制
- 在 H618 上只保留运行时依赖，不在设备上编译前端
- 升级时替换程序，不覆盖客户配置

## 建议目录布局

建议把程序和数据分开：

```text
/opt/picoclaw/current/picoclaw-web-linux-arm64
/data/picoclaw/config.json
/data/picoclaw/launcher-config.json
/data/picoclaw/logs/
```

这样升级时只替换 `/opt/picoclaw/current/` 下的二进制，`/data/picoclaw/` 下的配置和日志可以保持不动。

## 在开发机构建

仓库内已经提供脚本：

```bash
./scripts/build-web-linux-arm64.sh
```

默认输出：

```text
releases/h618/picoclaw-web-linux-arm64
```

也可以自定义输出目录和文件名：

```bash
./scripts/build-web-linux-arm64.sh /tmp/release my-picoclaw-web
```

这个脚本会做三件事：

1. 安装 `web/frontend` 依赖
2. 构建并嵌入前端资源
3. 交叉编译 `linux/arm64` 二进制

## systemd 服务文件

仓库里已经提供固定目录布局的 systemd unit：

```text
deploy/systemd/picoclaw-web.service
```

它默认使用：

```text
/opt/picoclaw/current/picoclaw-web-linux-arm64
/data/picoclaw/config.json
```

如果你按本说明中的目录布局部署，通常不需要再手改 unit 文件。

## 部署到 H618

把二进制和配置目录准备好：

```bash
mkdir -p /opt/picoclaw/current
mkdir -p /data/picoclaw
```

拷贝文件：

```bash
cp picoclaw-web-linux-arm64 /opt/picoclaw/current/
chmod +x /opt/picoclaw/current/picoclaw-web-linux-arm64
```

首次启动示例：

```bash
/opt/picoclaw/current/picoclaw-web-linux-arm64 --no-browser /data/picoclaw/config.json
```

如果需要局域网访问：

```bash
/opt/picoclaw/current/picoclaw-web-linux-arm64 --no-browser --public /data/picoclaw/config.json
```

Web 启动后默认端口是 `18800`。浏览器访问：

```text
http://设备IP:18800
```

## 一键安装

在 H618 上可以直接使用仓库内脚本：

```bash
./scripts/install-h618-web.sh ./picoclaw-web-linux-arm64
```

它会完成：

1. 安装二进制到 `/opt/picoclaw/current/`
2. 初始化 `/data/picoclaw/`
3. 安装 `systemd` 服务
4. `enable --now` 启动服务

注意：

- 如果 `/data/picoclaw/config.json` 不存在，脚本会生成一个占位配置
- 这个占位配置只能用于初始化目录，正式接入渠道和模型前必须补全

## 一键升级

升级时可以使用：

```bash
./scripts/upgrade-h618-web.sh ./picoclaw-web-linux-arm64
```

它会：

1. 备份旧二进制到 `/opt/picoclaw/backups/`
2. 停掉当前服务
3. 替换新二进制
4. 重新启动服务

默认不会覆盖 `/data/picoclaw/config.json` 和 `/data/picoclaw/launcher-config.json`。

## launcher-config.json

Web 启动器自己的监听配置保存在：

```text
/data/picoclaw/launcher-config.json
```

最小示例：

```json
{
  "port": 18800,
  "public": true,
  "allowed_cidrs": [
    "192.168.1.0/24"
  ]
}
```

如果要面向客户交付，建议至少限制 `allowed_cidrs`，不要默认开放到所有来源。

## 升级建议

推荐升级步骤：

1. 停掉旧进程
2. 备份 `/data/picoclaw/config.json`
3. 替换 `/opt/picoclaw/current/picoclaw-web-linux-arm64`
4. 启动新版本
5. 用 Web UI 检查 channels、models、skills 是否正常

不要把客户配置直接放进程序目录，否则升级时很容易被覆盖。

## 当前适合 H618 的路线

基于当前分支，建议使用：

- `picoclaw-web` 作为统一管理入口
- 渠道接入走原生 channels
- skills 安装走内置 skills 工具
- Agent 单独参数覆盖通过 Web UI 的 Agent Settings 页面配置

这条路线不依赖完整 plugin 系统，更适合 H618 这类资源受限设备。

## 建议发布包内容

对外打包时，建议至少包含这些文件：

```text
picoclaw-web-linux-arm64
scripts/install-h618-web.sh
scripts/upgrade-h618-web.sh
deploy/systemd/picoclaw-web.service
config/config.example.json
docs/zh/h618-web-deploy.md
```

建议额外提供一份你们自己的客户交付说明，至少写清楚：

- 默认 Web 端口
- 默认局域网访问范围
- 默认启用 `skillhub`、默认关闭 `clawhub`
- 首次配置模型和渠道的方法
- 升级时只替换二进制，不覆盖 `/data/picoclaw/`

## 真机验证记录

已在一台 H618 设备上完成最小部署验证：

- 设备地址：`192.168.1.61`
- 系统：Armbian `aarch64`
- 验证项：
  - `linux/arm64` 二进制可启动
  - `install-h618-web.sh` 可完成目录初始化
  - `systemd` 服务可正常拉起 `picoclaw-web`
  - 设备本机 `curl http://127.0.0.1:18800/api/config` 可返回 JSON
  - 更新到 `custom/h618-migration` 最新二进制后，服务仍可正常重启
  - `/data/picoclaw/config.json` 已验证可以切换为：
    - `tools.skills.registries.skillhub.enabled = true`
    - `tools.skills.registries.clawhub.enabled = false`
  - 在真机上已验证 `skillhub` 搜索和安装链路：
    - 搜索 `github` 成功返回结果
    - 安装 `github` 成功
    - 安装落盘路径：`/data/picoclaw/workspace/skills-smoke/github`

这说明当前仓库中的 H618 构建、安装和服务启动链路已经打通。

## 建议发布前回归

每次对外发版前，建议最少检查这几项：

1. 从空目录运行 `install-h618-web.sh`
2. 确认 `systemctl status picoclaw-web` 为 `active`
3. 确认 Web UI 可打开并能保存配置
4. 确认至少一个渠道能收发消息
5. 确认 `skillhub` 搜索和安装正常
6. 运行 `upgrade-h618-web.sh` 后再次确认服务和配置正常
