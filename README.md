# fast-proxy

`fast-proxy` 是一个用于本地开发环境的命令行工具，可以快速配置本地域名访问本机服务。

典型场景：将 `app.test` 代理到本机的 `localhost:3000`，工具会自动维护 `/etc/hosts` 和 Caddy 配置。

## 功能特性

- 添加本地域名反向代理规则
- 按规则 ID 删除代理规则，支持 ID 前缀匹配
- 查看当前由工具管理的代理规则
- 检查运行环境与 Caddy 状态
- 根据状态文件重新同步 hosts 与 Caddy 配置
- 自动维护 `/etc/hosts` 中带 `# fast-proxy` 标记的记录
- 自动维护 Caddy 站点片段配置
- 修改配置后自动执行 `caddy reload`
- 支持使用 `FAST_PROXY_HOME` 自定义状态文件目录

## 工作原理

添加本地代理规则时，`fast-proxy` 会完成以下操作：

1. 将规则写入状态文件 `~/.fast-proxy/config.json`
2. 在 `/etc/hosts` 中写入域名解析记录
3. 在 `/etc/caddy/fast-proxy/` 下生成 Caddy 站点配置
4. 重新加载 Caddy 配置

例如：

```text
app.test -> localhost:3000
```

会生成类似配置：

```caddyfile
app.test {
    tls internal
    reverse_proxy localhost:3000
}
```

## 前置要求

- Go 1.22+
- 已安装 Caddy，并确保 `caddy` 命令在 `PATH` 中
- 当前用户具备修改 `/etc/hosts`、`/etc/caddy/Caddyfile` 和 `/etc/caddy/fast-proxy` 的权限

由于需要修改系统文件，常用命令通常需要通过 `sudo` 执行。

如果尚未安装 Caddy，可以先执行：

```bash
fp doctor
```

该命令会输出安装建议和当前环境检查结果。

## 安装

从源码构建：

```bash
make build
```

安装到 `/usr/local/bin/fast-proxy`：

```bash
make install
```

安装后会同时提供两个命令：

- `fast-proxy`
- `fp`

卸载：

```bash
make uninstall
```

也可以直接运行：

```bash
make run ARGS="list"
```

## 初始化 Caddy 配置

首次使用前，先执行初始化：

```bash
sudo fast-proxy init
```

也可以使用短命令：

```bash
sudo fp init
```

该命令会在系统 Caddyfile 中加入 fast-proxy 的 import 配置：

```caddyfile
import /etc/caddy/fast-proxy/*.caddy
```

并创建站点片段目录：

```text
/etc/caddy/fast-proxy
```

如果未检测到 Caddy，`init` 会给出安装提示；也可以先执行 `fp doctor` 查看环境检查结果。

## 使用方式

### 添加代理规则

```bash
sudo fast-proxy add <domain> <host:port>
```

示例：

```bash
sudo fast-proxy add app.test localhost:3000
```

短命令：

```bash
sudo fp add app.test localhost:3000
```

执行后：

- `/etc/hosts` 会增加由 `fast-proxy` 管理的记录
- `/etc/caddy/fast-proxy/app.test.caddy` 会被生成
- Caddy 会被重新加载

### 查看规则

```bash
sudo fast-proxy list
```

短命令：

```bash
fp list
```

也可以使用别名：

```bash
sudo fast-proxy ls
```

输出示例：

```text
+--------------+----------+----------------+
| ID           | DOMAIN   | TARGET         |
+--------------+----------+----------------+
| a1b2c3d4e5f6 | app.test | localhost:3000 |
+--------------+----------+----------------+
```

### 删除规则

删除规则需要使用规则 ID：

```bash
sudo fast-proxy remove <id>
```

示例：

```bash
sudo fast-proxy remove a1b2c3d4e5f6
```

短命令：

```bash
sudo fp rm a1b2c3d4e5f6
```

### 检查环境

```bash
fp doctor
```

该命令会检查：

- Caddy 是否已安装
- Caddyfile 是否存在
- fast-proxy import 是否已配置
- 站点片段目录是否存在
- 状态文件是否正常
- Caddy 配置是否通过校验
- Caddy 服务是否运行

### 重新同步

如果 `/etc/hosts` 或 `/etc/caddy/fast-proxy/*.caddy` 被手动修改或误删，可以根据状态文件重新同步：

```bash
sudo fp sync
```

支持 ID 前缀匹配，只要前缀唯一即可：

```bash
sudo fast-proxy remove a1b2c3
```

也可以一次删除多个规则：

```bash
sudo fast-proxy remove a1b2c3 d4e5f6
```

删除命令别名：

```bash
sudo fast-proxy rm <id>
sudo fast-proxy delete <id>
```

## 命令列表

| 命令 | 说明 |
| --- | --- |
| `fast-proxy init` | 初始化系统 Caddy 配置 |
| `fast-proxy doctor` | 检查运行环境与 Caddy 状态 |
| `fast-proxy sync` | 根据状态文件重新同步 hosts 与 Caddy 配置 |
| `fast-proxy add <domain> <host:port>` | 添加或更新代理规则 |
| `fast-proxy list` | 查看当前规则 |
| `fast-proxy remove <id> [id...]` | 删除规则 |

安装后还可使用短命令 `fp`，例如 `fp init`、`fp add`、`fp list`、`fp rm`。

## 文件位置

| 路径 | 说明 |
| --- | --- |
| `~/.fast-proxy/config.json` | fast-proxy 状态文件 |
| `/etc/hosts` | 系统 hosts 文件 |
| `/etc/caddy/Caddyfile` | 系统 Caddyfile |
| `/etc/caddy/fast-proxy/*.caddy` | fast-proxy 生成的站点配置 |

如果通过 `sudo` 执行，并且存在 `SUDO_USER`，状态文件会优先写入原用户的 home 目录，而不是 `/root`。

也可以使用 `FAST_PROXY_HOME` 自定义状态文件目录：

```bash
FAST_PROXY_HOME=/path/to/home sudo fast-proxy list
```

## 参数校验

### domain

- 不能为空
- 不能是 `localhost`
- 不能包含空格、`/`、`:`

### target

- 推荐使用 `localhost:port` 或 `127.0.0.1:port`
- 端口范围为 `1-65535`
- 也可以填写不带端口的主机或 IP，用于维护 hosts 解析记录

## 注意事项

- `fast-proxy` 只会删除 `/etc/hosts` 中包含 `# fast-proxy` 标记的记录。
- `fast-proxy` 会重新生成 `/etc/caddy/fast-proxy/*.caddy` 下的站点片段。
- 本地目标仅识别 `localhost` 和 `127.0.0.1`，这些规则会生成 Caddy 反向代理配置。
- 非本地目标会写入 hosts 解析记录，但不会生成 Caddy 反向代理站点片段。
- 如果 Caddy reload 失败，请确认 Caddy 已安装、服务正在运行，并检查 `/etc/caddy/Caddyfile` 是否有效。
- `fast-proxy doctor` 可用于检查当前环境、Caddy 安装状态和配置问题。
- `fast-proxy sync` 可用于根据状态文件重建 hosts 和站点片段。

## 开发

常用开发命令：

```bash
make fmt
make test
make tidy
make build
```

项目结构：

```text
cmd/fast-proxy/main.go      CLI 入口
internal/app/app.go         命令定义和核心流程
internal/config/            状态文件与路径配置
internal/hosts/             hosts 文件同步
internal/caddy/             Caddy 配置生成与重载
```