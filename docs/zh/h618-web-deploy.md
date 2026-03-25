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
