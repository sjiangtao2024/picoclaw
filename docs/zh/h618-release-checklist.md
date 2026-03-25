# H618 交付清单

这份清单面向你们自己的发布流程，目标是把 H618 交付做成可重复操作，而不是靠记忆。

## 发布包内容

发布目录建议包含：

```text
picoclaw-web-linux-arm64
picoclaw
install-h618-web.sh
upgrade-h618-web.sh
picoclaw-web.service
config.example.json
h618-web-deploy.md
```

## 默认交付约定

- Web UI 为主入口
- 默认启用 `skillhub`
- 默认关闭 `clawhub`
- 客户配置放在 `/data/picoclaw/`
- 程序二进制放在 `/opt/picoclaw/current/`
- 升级只替换二进制，不覆盖配置和日志

## 发版前检查

1. 确认当前分支是 `custom/h618-migration`
2. 确认工作树干净
3. 构建 `linux/arm64` 的 `picoclaw-web` 和 `picoclaw`
4. 检查 `config/config.example.json` 默认是 `skillhub=true`
5. 检查安装脚本、升级脚本、systemd 文件都在

## 真机回归

在 H618 真机上至少做一次：

1. 安装
2. 启动
3. 打开 Web UI
4. 搜索 skill
5. 安装 skill
6. 升级
7. 再次打开 Web UI
8. 点击“启动服务”并确认网关能正常运行

## 当前已验证设备

- `192.168.1.61`
- Armbian `aarch64`
- 已验证 Web 服务正常
- 已验证 Web UI 可正常启动网关
- 已验证 `skillhub` 搜索和安装正常
