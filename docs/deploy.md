# OpenFire 部署文档

> 本文档覆盖 OpenFire 从本地开发、单机部署到生产级高可用部署的全流程。内容包括：依赖安装、Godot 客户端打包、Go 中央服务端部署、专用游戏服务器（GS）部署、反向代理与 TLS、防火墙规则、监控与故障排查。
>
> 与本项目对应：Godot 4.7 客户端 + Go 1.22+ 中央服务端 + PostgreSQL 13+ / Redis 6+。

---

## 目录

1. [部署架构总览](#1-部署架构总览)
2. [依赖软件清单](#2-依赖软件清单)
3. [本地开发环境搭建](#3-本地开发环境搭建)
4. [客户端构建与打包](#4-客户端构建与打包)
5. [专用游戏服务器（GS）构建](#5-专用游戏服务器gs构建)
6. [中央服务端（Go）部署](#6-中央服务端go部署)
   - [6.1 二进制 + systemd 方式（推荐）](#61-二进制--systemd-方式推荐)
   - [6.2 Docker Compose 方式](#62-docker-compose-方式)
7. [Nginx / Caddy 反向代理与 TLS](#7-nginx--caddy-反向代理与-tls)
8. [防火墙与安全](#8-防火墙与安全)
9. [客户端分发与首次启动配置](#9-客户端分发与首次启动配置)
10. [灰度发布与版本更新](#10-灰度发布与版本更新)
11. [监控、日志与备份](#11-监控日志与备份)
12. [故障排查（FAQ）](#12-故障排查faq)

---

## 1. 部署架构总览

OpenFire 在线对战采用 **中央服务 (Central) + 专用游戏服务器 (GS) + Godot 客户端** 三角色架构：

```
 ┌──────────────┐   HTTPS/WSS   ┌──────────────────┐
 │ Godot 客户端 │──────────────▶│  中央服务 :8080  │──PostgreSQL (账号/对局)
 │  (桌面/安卓) │◀──────────────│  (Go + chi + WS) │──Redis (未来分布式)
 └──────────────┘   match_found └────────┬─────────┘
             ▲                           │ fork Godot headless
             │ ENet UDP                  ▼
             └─────────────── ┌──────────────────────┐
                             │ 专用服务器 (GS) :27020 │
                             │  27021、27022 … 27099 │
                             └──────────────────────┘
                                       │
                                       ▼ POST /internal/matches/{id}/result
                                  中央服务（使用 X-Internal-Token 鉴权）
```

关键：
- **HTTP/WS 入口**：中央服务一个端口（8080）；前置反代做 TLS。
- **GS 进程由中央服务按需 fork**，端口池 27020–27099 UDP；**必须对玩家开放 UDP 端口段**。
- **GS 与中央服务之间的 HTTP 内部通信**：使用 `X-Internal-Token` 头鉴权，建议在内网。

---

## 2. 依赖软件清单

| 角色 | 依赖 | 最低版本 | 说明 |
|---|---|---|---|
| 全角色共用 | Git | 2.x | 拉取代码 |
| 客户端构建 | Godot (editor + export templates) | **4.7.1.stable**（CI 已固定） | 用于打包，含 Headless 版本 |
| 客户端（安卓包） | JDK 17 + Android SDK + debug.keystore | Temurin 17, build-tools 34 | APK 签名与打包 |
| 服务端（中央） | Go | 1.22+ | 编译二进制 |
| 服务端（中央） | PostgreSQL | 13+（因为 `gen_random_uuid()` 内置） | 关系存储 |
| 服务端（中央） | Redis | 6+ | 当前预留；未来用于分布式 presence / 队列 |
| 专用服务器（GS）运行机 | Godot Headless（或专用服务器导出的 openfire-server 二进制） | 与客户端版本一致 | 推荐用导出的 GS 二进制（体积小、不含渲染） |
| 反代（推荐） | Nginx 1.22+ 或 Caddy 2.x | — | TLS、WSS 升级、压缩、限流 |
| 系统 | Linux（推荐） | Ubuntu 22.04 LTS / Debian 12 / RHEL 8+ | 中央+GS 建议同机或内网 |

硬件参考（小型公测：同时在线 500 人，峰值并发 30 场对局）：

| 节点 | CPU | 内存 | 磁盘 | 带宽 |
|---|---|---|---|---|
| 中央服务 + DB + Redis（1 台） | 8 核 | 16 GB | 100 GB SSD | 1 Gbps |
| GS 节点（1 台，同机或单独） | 16 核 | 16 GB | 50 GB SSD | 1 Gbps 上行，每对局约 0.5–2 Mbps |
| 反代 / TLS | 2 核 | 2 GB | — | 1 Gbps |

---

## 3. 本地开发环境搭建

> 适用于开发者本机调试：客户端、GS、中央服务全在 `127.0.0.1`。

### 3.1 拉取代码

```bash
git clone <your-repo-url> openfire && cd openfire
```

### 3.2 安装 Godot 4.7.1

```bash
# Linux 示例；macOS/Windows 请从 https://godotengine.org 下载
mkdir -p .tools
cd .tools
GODOT_V=4.7.1
curl -fLO https://github.com/godotengine/godot-builds/releases/download/${GODOT_V}-stable/Godot_v${GODOT_V}-stable_linux.x86_64.zip
unzip -q Godot_*.zip
mv Godot_v${GODOT_V}-stable_linux.x86_64 godot
chmod +x godot
# 下载 export templates（打包需要；编辑器会提示）
curl -fLO https://github.com/godotengine/godot-builds/releases/download/${GODOT_V}-stable/Godot_v${GODOT_V}-stable_export_templates.tpz
unzip -q Godot_*_export_templates.tpz   # -> ./templates
DEST="$HOME/.local/share/godot/export_templates/${GODOT_V}.stable"
mkdir -p "$DEST"
mv templates/* "$DEST"
cd ..
./.tools/godot --version
```

### 3.3 启动 PostgreSQL 与 Redis

方式 A（Docker，最快）：
```bash
docker run -d --name of-pg  -p 5432:5432 \
  -e POSTGRES_USER=openfire -e POSTGRES_PASSWORD=openfire -e POSTGRES_DB=openfire \
  postgres:16-alpine
docker run -d --name of-redis -p 6379:6379 redis:7-alpine
```

方式 B（本机服务）：
```bash
# Ubuntu
sudo apt-get install -y postgresql redis-server
sudo -u postgres psql -c "CREATE ROLE openfire WITH LOGIN PASSWORD 'openfire';"
sudo -u postgres psql -c "CREATE DATABASE openfire OWNER openfire;"
```

### 3.4 运行中央服务（开发模式）

```bash
cd server
cp .env.example .env        # 开发默认值即可；JWT_SECRET/INTERNAL_TOKEN 可保持（生产必须换）
go mod tidy
go run ./cmd/server
# 预期日志：openfire central server listening on :8080
# 验证：
curl http://127.0.0.1:8080/healthz     # -> {"ok":true}
```

### 3.5 启动客户端并对战

```bash
# 回到仓库根目录
cd ..
# 启动两个客户端（或用编辑器打开）
./.tools/godot --path . res://scenes/main_menu.tscn &   # 客户端 A
./.tools/godot --path . res://scenes/main_menu.tscn &   # 客户端 B
```

在客户端主菜单 → "在线对战 · Online Matchmaking" 面板：
1. A 和 B 分别注册账号（密码 ≥ 6 位）
2. 都点 "在线对战" → 进入排队
3. 匹配到后，中央服务会启动 1 个 GS 进程（在端口 27020–27099）
4. 两客户端自动连入 GS 并开战，打完自动上报

检查：
```bash
curl -H "Authorization: Bearer <jwt>" http://127.0.0.1:8080/api/matches
```
会看到对战记录。

---

## 4. 客户端构建与打包

### 4.1 手动命令行导出

```bash
# 确认 Godot 已安装 export templates
mkdir -p build dist
godot --headless --path . --import                       # 首次导入所有资源
# 全平台
godot --headless --path . --export-release "Linux"   build/openfire.x86_64
godot --headless --path . --export-release "Windows" build/openfire.exe
godot --headless --path . --export-release "macOS"   build/openfire-macos.zip
godot --headless --path . --export-release "Android" build/openfire.apk
# （可选）把 NobodyWho 原生库一起打包进 zip（若有）
cp addons/nobodywho/libnobodywho-*-linux-*.so build/ 2>/dev/null || true
```

### 4.2 使用 GitHub Actions 自动发布（推荐）

仓库已内置 `.github/workflows/release.yml`，**只有手动触发才会发版**（push/PR 只跑 import + smoke 校验）：

1. 推送代码到 `main`。
2. GitHub → Actions → 左侧 "Build & Release" → **Run workflow** → 选：
   - `patch`：0.2.31 → 0.2.32（修 bug）
   - `minor`：0.2.31 → 0.3.0（新特性）
   - `major`：0.2.31 → 1.0.0（大版本）
3. 工作流完成后，在 Releases 页面会生成：
   - `openfire-<v>-linux-x86_64.zip`
   - `openfire-<v>-windows-x64.zip`
   - `openfire-<v>-macos.zip`
   - `openfire-<v>-android-arm64-v8a.apk`（debug 签名）
   - `comfyui-bundle.zip`（可选 AI 包）

### 4.3 Android 发布签名（正式上架）

1. 生成 release keystore：
```bash
keytool -genkeypair -v \
  -keystore ~/.android/release.keystore \
  -storepass <storepass> -alias openfire-release -keypass <keypass> \
  -keyalg RSA -keysize 2048 -validity 36500 -dname "CN=Your Name,O=Your Org,C=CN"
```
2. 修改 `export_presets.cfg` 的 `[preset.3.options]`：
```ini
keystore/release="$HOME/.android/release.keystore"
keystore/release_user="openfire-release"
keystore/release_password="<keypass / storepass 按 Godot 文档填>"
```
3. 用 CI Secret 注入（更安全）—— 在 CI 里将 keystore 进行 base64 保存为 Secret，然后在 workflow 中还原路径并注入环境变量。

---

## 5. 专用游戏服务器（GS）构建

中央服务在每次开一局时 fork 一个 Godot headless 进程；为了**稳定、可重现、可单独发布**，推荐使用导出预设 `Dedicated Server` 生成的单文件二进制，而不是直接用编辑器二进制。

### 5.1 导出 GS

```bash
cd <repo根>
mkdir -p build
godot --headless --path . --import
godot --headless --path . --export-release "Dedicated Server" build/openfire-server
chmod +x build/openfire-server
# 验证：不带任何参数运行，它会因为没环境变量而按 CLIENT 模式启动
# 在有变量时，能正确启动 GS（手动试一次）：
OPENFIRE_GS_MATCH_ID=test1 \
OPENFIRE_GS_MODE=deathmatch \
OPENFIRE_GS_PORT=27020 \
OPENFIRE_CENTRAL_URL=http://127.0.0.1:8080 \
OPENFIRE_INTERNAL_TOKEN=dev-internal-token \
  ./build/openfire-server --headless
# 正常情况：会监听 27020/udp，然后等待 5 分钟超时 → 上报 abandoned 并退出
```

### 5.2 GS 发布到部署机

将 `build/openfire-server` 上传到 GS 运行机，例如：
```bash
scp build/openfire-server deploy@gs01:/opt/openfire/openfire-server
ssh deploy@gs01 chmod +x /opt/openfire/openfire-server
```
然后在中央服务的 `.env` 中设置：
```
OPENFIRE_GODOT_BIN=/opt/openfire/openfire-server
```

> 如果中央服务和 GS 不在同一台机器：需要把中央 `gs.Manager.StartMatch` 的代码替换为"通过 SSH/API 在远程节点启动 GS"；单实例方案里 GS 和中央同机即可。

---

## 6. 中央服务端（Go）部署

### 6.1 二进制 + systemd 方式（推荐）

#### 6.1.1 编译二进制（CI 构建产物或本机）

```bash
cd server
go mod tidy
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/openfire-central ./cmd/server
# CGO_ENABLED=0 即可（无 cgo 依赖）；产物是静态编译单文件
```

#### 6.1.2 上传 + 目录准备

```bash
# 部署机执行
sudo useradd -r -m -d /opt/openfire -s /usr/sbin/nologin openfire
sudo mkdir -p /opt/openfire/server/logs /opt/openfire/server/migrations
sudo chown -R openfire:openfire /opt/openfire
# 本地上传
scp server/dist/openfire-central deploy@central01:/opt/openfire/server/openfire-central
scp server/migrations/001_init.sql deploy@central01:/opt/openfire/server/migrations/001_init.sql
scp build/openfire-server                deploy@central01:/opt/openfire/openfire-server
ssh deploy@central01 sudo chmod +x /opt/openfire/server/openfire-central /opt/openfire/openfire-server
```

#### 6.1.3 生成密钥 `.env`

```bash
ssh deploy@central01 "sudo -u openfire tee /opt/openfire/server/.env >/dev/null <<'EOF'
HTTP_ADDR=127.0.0.1:8080
DB_URL=postgres://openfire:StrongP%21@127.0.0.1:5432/openfire?sslmode=disable
REDIS_URL=redis://127.0.0.1:6379/0
JWT_SECRET=$(openssl rand -hex 32)
INTERNAL_TOKEN=$(openssl rand -hex 32)
GS_HOST=gs.yourgame.com
GS_PORT_MIN=27020
GS_PORT_MAX=27099
OPENFIRE_GODOT_BIN=/opt/openfire/openfire-server
OPENFIRE_CENTRAL_URL=https://api.yourgame.com
MATCHMAKING_TIMEOUT=180s
MIGRATIONS_PATH=migrations/001_init.sql
EOF"
```

#### 6.1.4 systemd 单元文件 `/etc/systemd/system/openfire-central.service`

```ini
[Unit]
Description=OpenFire Central Server
After=network-online.target postgresql.service redis-server.service
Wants=network-online.target

[Service]
Type=simple
User=openfire
WorkingDirectory=/opt/openfire/server
EnvironmentFile=/opt/openfire/server/.env
ExecStart=/opt/openfire/server/openfire-central
Restart=on-failure
RestartSec=3s
# 进程数：单实例即可；GS 是其 subprocess
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal
SyslogIdentifier=openfire-central

# 可选：隔离
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/openfire/server/logs

[Install]
WantedBy=multi-user.target
```

启动 & 检查：
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now openfire-central
sleep 2
curl -s http://127.0.0.1:8080/healthz   # 期望 {"ok":true}
journalctl -u openfire-central -f        # 看日志
```

### 6.2 Docker Compose 方式

若团队偏好容器化，以下示例可直接用：

#### `docker-compose.yml`（放项目根或 `/opt/openfire/`）

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: openfire
      POSTGRES_PASSWORD: openfire
      POSTGRES_DB: openfire
    volumes:
      - pgdata:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    restart: unless-stopped

  central:
    build:
      context: ./server
      dockerfile: Dockerfile          # 下面给一个 Dockerfile 示例
    env_file: ./server/.env
    environment:
      - HTTP_ADDR=:8080
      - DB_URL=postgres://openfire:openfire@db:5432/openfire?sslmode=disable
      - REDIS_URL=redis://redis:6379/0
      - GS_HOST=gs.yourgame.com
      - OPENFIRE_GODOT_BIN=/opt/openfire/openfire-server
      - OPENFIRE_CENTRAL_URL=https://api.yourgame.com
      - MIGRATIONS_PATH=/opt/openfire/server/migrations/001_init.sql
    volumes:
      - ./build/openfire-server:/opt/openfire/openfire-server:ro   # 注意：提前构建 GS 二进制
      - ./server/migrations:/opt/openfire/server/migrations:ro
      - ./server-logs:/opt/openfire/server/logs
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      - db
      - redis
    restart: unless-stopped
    # 重要：需要启动子进程 + 访问宿主机网卡的 GS 端口
    cap_add: [SYS_PTRACE]

volumes:
  pgdata:
```

#### `server/Dockerfile`

```dockerfile
FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/openfire-central ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /opt/openfire/server/logs /opt/openfire/server/migrations && chown -R 65534:65534 /opt/openfire
WORKDIR /opt/openfire/server
COPY --from=builder /out/openfire-central /usr/local/bin/openfire-central
USER 65534
EXPOSE 8080
CMD ["openfire-central"]
```

启动：
```bash
docker compose up -d --build
curl -s http://127.0.0.1:8080/healthz
```

> 提示：如果 GS 和中央分离部署，docker-compose 方案就不能直接 fork GS；需要改成"独立 GS 集群 + 中央 API 分配"模式，对应修改 `internal/gs/manager.go`。

---

## 7. Nginx / Caddy 反向代理与 TLS

### 7.1 推荐：Caddy（自动 HTTPS）

安装 Caddy：`apt install caddy -y` 或官方二进制。

`/etc/caddy/Caddyfile`：

```caddy
api.yourgame.com {
    encode gzip zstd

    # 限流：防止刷注册/登录
    @postapi method POST
    handle @postapi {
        rate_limit {remote.ip} 10r/s
        reverse_proxy 127.0.0.1:8080
    }

    # WebSocket 大厅：需要长连接 + 大 header 缓冲区
    @ws path /api/ws
    handle @ws {
        reverse_proxy 127.0.0.1:8080 {
            header_up Connection Upgrade
            header_up Upgrade websocket
        }
    }

    # 其它 HTTP API
    reverse_proxy 127.0.0.1:8080
}
```

应用：
```bash
sudo systemctl reload caddy
# 首次会自动申请证书；确保 DNS 已解析
curl -s https://api.yourgame.com/healthz
```

### 7.2 Nginx（如果已在运维栈内）

`/etc/nginx/sites-available/openfire.conf`：

```nginx
server {
    listen 443 ssl http2;
    server_name api.yourgame.com;

    ssl_certificate     /etc/letsencrypt/live/api.yourgame.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourgame.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;

    client_max_body_size 1m;

    # WebSocket 大厅（需要支持 Upgrade + 长超时）
    location /api/ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade    $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host       $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

server {
    listen 80;
    server_name api.yourgame.com;
    return 301 https://$host$request_uri;
}
```

启用 + reload：
```bash
sudo ln -s /etc/nginx/sites-available/openfire.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

> ⚠️ TLS 完成后，客户端配置中的 URL 必须切换：
> `central_url=https://api.yourgame.com` 与 `central_ws_url=wss://api.yourgame.com/api/ws`。
>
> 若部署 HTTPS 却仍用 `ws://`，现代浏览器会拦截（混合内容）。

---

## 8. 防火墙与安全

### 8.1 端口开放表

在部署机（假设是一台中央+GS 合并）上：

| 服务 | 协议 | 端口 | 来源 | 作用 |
|---|---|---|---|---|
| HTTP/HTTPS 反代 | TCP | 80, 443 | 0.0.0.0/0 | 客户端 REST + WSS |
| GS ENet 对局 | **UDP** | 27020–27099 | 0.0.0.0/0 | **必须开放 UDP**，客户端直连 |
| Postgres | TCP | 5432 | 127.0.0.1 / 内网 | 仅限本地或内网 |
| Redis | TCP | 6379 | 127.0.0.1 / 内网 | 同上 |
| SSH | TCP | 22 | 你信任的 IP | 管理 |

### 8.2 ufw 示例

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
# 关键：GS 使用 UDP
sudo ufw allow 27020:27099/udp comment "OpenFire GS (ENet)"
sudo ufw enable
sudo ufw status numbered
```

### 8.3 安全加固清单

- [x] 修改 `JWT_SECRET` 与 `INTERNAL_TOKEN`，使用 ≥ 32 字节随机串。
- [x] PostgreSQL 启用 `pg_hba.conf` 限制来源 IP；使用强密码（`%21` 这种 URL 特殊字符记得 encode）。
- [x] Redis 设密码 `requirepass`，并只监听内网。
- [x] `/internal/*` 端点只应内网可达（`middleware.Internal` 已做 token 鉴权，但双重防护更好）。
- [x] Nginx/Caddy 前部署 WAF 或 CDN 层（可选）做 DDoS/CC 防护。
- [x] 对 `/api/auth/register` 和 `/api/auth/login` 加限流（Caddy/nginx 层已演示；或代码层加 golang.org/x/time/rate）。
- [x] 禁止 SSH 密码登录，改用密钥：`PasswordAuthentication no`。

---

## 9. 客户端分发与首次启动配置

### 9.1 玩家下载

从 GitHub Releases / 自建对象存储（S3、七牛、阿里 OSS）分发：
- Windows：解压 `openfire-<v>-windows-x64.zip` → 双击 `openfire.exe`
- Linux：`unzip openfire-<v>-linux-x86_64.zip && ./openfire.x86_64`
- macOS：打开 `openfire-<v>-macos.zip` → 解压出 `.app`，右键 → 打开（跳过首次 Gatekeeper 阻止）
- Android：adb/侧载 APK，或发布到 Google Play/应用商店。

### 9.2 默认 central_url 指向生产

默认值写在 `scripts/autoload/settings.gd`：
```gdscript
var central_url: String = "http://127.0.0.1:8080"
var central_ws_url: String = "ws://127.0.0.1:8080/api/ws"
```
**发版前请改成生产域名**，避免玩家拿到手后要自己手改：

```gdscript
var central_url: String = "https://api.yourgame.com"
var central_ws_url: String = "wss://api.yourgame.com/api/ws"
```

然后再走第 4 章的导出流程重新打包。

### 9.3 让玩家自行切换地址（测试服/私服场景）

如果想让玩家手动切换，可在 Options 菜单添加"服务器地址"配置项（main_menu.gd 里已有类似 Settings UI 模式，照着加两个 LineEdit 并调用 `Settings.central_url=...; Settings.save();` 即可）。

---

## 10. 灰度发布与版本更新

### 10.1 客户端版本

- 版本号存在 `project.godot: config/version = 0.2.31`
- 在登录/注册阶段，服务端可考虑加 `/api/auth/version` 接口校验最小版本；当前代码未实现，建议二次开发时加上，避免老版本 bug 玩家连入。

### 10.2 中央服务滚动升级

使用 systemd：
```bash
# 替换二进制
sudo -u openfire cp /opt/openfire/server/openfire-central /opt/openfire/server/openfire-central.bak
sudo -u openfire cp /tmp/new-bin /opt/openfire/server/openfire-central
sudo systemctl restart openfire-central
journalctl -u openfire-central --since "1 min ago"
```

注意：重启中央服务会**断开所有大厅 WS 连接**、**丢失内存中队列**、**终止正在 fork 的 GS 进程**；GS 进程被终止会把对应对局标记为 abandoned。建议：
1. 匹配高峰外做升级；或
2. 引入 Redis 持久化队列 + 多实例中央 + GS 独立节点，减少单点重启的影响（属于中期扩展）。

### 10.3 数据库迁移

当前 Migrate 方式是"启动时读取整个 SQL 文件执行"。如果增加 `002_xxx.sql`、`003_xxx.sql`：

1. 推荐在 `db.go` 里把 `Migrate` 改成遍历目录；或
2. 手动依次执行：
```bash
psql "$DB_URL" -f migrations/002_add_mmr.sql
psql "$DB_URL" -f migrations/003_add_maps.sql
```
然后重启中央服务。

---

## 11. 监控、日志与备份

### 11.1 日志位置

| 角色 | 日志位置 |
|---|---|
| 中央服务 | systemd journal：`journalctl -u openfire-central -f`；或 Docker `docker logs` |
| 单个 GS 对局 | 中央服务工作目录下 `server/logs/gs-{match_id}.log`（stdout+stderr） |
| 客户端 | Godot 运行日志：各平台 `user://logs/godot.log`；Windows：`%APPDATA%\Godot\app_userdata\OpenFire\logs` |
| Nginx / Caddy 反代访问日志 | `/var/log/nginx/...` 或 `journalctl -u caddy` |

### 11.2 健康检查 & 存活探针

- `/healthz`：用于 load balancer 健康检查。
- 进程级：systemd `Restart=on-failure` 自动拉起。

### 11.3 Prometheus 指标（建议扩展）

当前代码未暴露 Prometheus 指标；建议在 `main.go` 加一个 `promhttp.Handler()` 的 `/metrics` 端点，采集：
- `online_users`：Hub 在线人数
- `queue_length`：各模式排队人数
- `active_games`：GS 进程数
- `http_requests_total / http_request_duration_seconds`（chi middleware）

### 11.4 备份

- PostgreSQL：`pg_dump -d "$DB_URL" | gzip > /backup/of-$(date +%F).sql.gz`，保留 30 天；建议 crontab 或 pgBackRest。
- Redis：开启 AOF（`appendonly yes`），持久化在 `/var/lib/redis/`。
- 对局日志：`server/logs/*.log`，可定期打包到对象存储。

---

## 12. 故障排查（FAQ）

### Q1. 客户端提示"连接大厅失败 / not_connected"
- 先验证中央服务健康：`curl https://api.yourgame.com/healthz`
- 浏览器开发者工具 Network 看是否 WebSocket 升级返回 101；若返回 401 检查 JWT。
- 反代日志看 `/api/ws` 是否正确设置 `Upgrade: websocket`。

### Q2. 匹配很久没反应，玩家一直"匹配中"
- 检查队列总人数：代码中 `QueuedCount()`；可能房间大小 > 同时在线人数。
- 调低 `MATCHMAKING_TIMEOUT` 让玩家更快看到失败。
- 可扩展"人数不足降低房间人数/匹配 AI"等策略。

### Q3. match_found 后客户端无法连接 GS（黑屏/断开）
- **80% 原因：防火墙 UDP 27020–27099 没开放**。在客户端机器上用：
```bash
nc -u gs.yourgame.com 27020
# 或服务器上抓包：
sudo tcpdump -i any udp port 27020
```
- 20%：`GS_HOST` 填成内网 IP / 127.0.0.1，而玩家在外网——改为公网可达的域名/IP。

### Q4. 对局结束后历史战绩没记录
- 查单个对局 GS 日志：`server/logs/gs-{match_id}.log`，看末尾是否有 `POST result`。
- 查中央服务日志，看 `/internal/matches/{id}/result` 是否 401（`X-Internal-Token` 不匹配）。
- 查 PostgreSQL：`SELECT status FROM matches WHERE id='xxx';` 没变成 finished 就是未上报。

### Q5. 启动中央服务报错 "config: GS_PORT_MAX (…) must be greater than GS_PORT_MIN (…)"
- 说明环境变量写反了；修正 `.env`。
- 或 其它报错例如 "JWT_SECRET must not be empty" 也是同样的 `.env` 未生效。

### Q6. 专用服务器 fork 失败 "start godot: executable file not found"
- 改 `OPENFIRE_GODOT_BIN` 为**绝对路径**；并确认文件有 `x` 位。
- 用 `su -s /bin/bash -c '/opt/openfire/openfire-server --version' openfire` 跑一下看是否能启动。

### Q7. Android 版本连不上 WSS（HTTPS 页面下 ws:// 被拦截）
- 客户端包内必须把 `central_ws_url` 改为 `wss://...`，并确保服务端有证书。
- 如果用自签名证书，需要在 Godot 客户端加载时信任证书（更推荐正规 Let's Encrypt）。

### Q8. 数据库 `gen_random_uuid()` 函数不存在
- 说明 PostgreSQL < 13；解决方案：升级 PG ≥ 13，或在迁移文件里手动 `CREATE EXTENSION IF NOT EXISTS pgcrypto;`（然后 UUID 函数变为 `gen_random_uuid()` / `pgcrypto` 有对应实现）。

### Q9. 玩家改密码 / 封号
- 目前没有改密码 / 管理后台接口；需要通过运维手动连数据库：
```sql
-- 重置密码为 bcrypt("NewPass123")，先自己在 Go 里跑一次 HashPassword 拿到 hash
UPDATE users SET password_hash='<bcrypt hash>' WHERE username='badplayer';
-- 封号：在 matches 和 match_players 没外键限制的情况下（实际有外键，要先删）；
-- 或先加一个 banned 字段并在登录返回 401（需改 AuthHandler）
```
建议二次开发时引入管理 API。

---

版本：`OpenFire 部署文档 v1.0`，对应项目版本 `0.2.31`。

**部署 Checklist（上线前最后确认）：**
- [ ] PostgreSQL / Redis 启动并通过健康检查
- [ ] 中央服务 `/healthz` OK，迁移已执行
- [ ] `JWT_SECRET` 与 `INTERNAL_TOKEN` 已替换为随机串
- [ ] GS 二进制已上传，`OPENFIRE_GODOT_BIN` 指向它；手动启动一次 GS 验证
- [ ] 防火墙已开放 UDP 27020–27099
- [ ] Nginx/Caddy HTTPS 已生效，`/api/ws` 可升级到 WSS（101）
- [ ] 客户端已使用生产 URL 打包并导出
- [ ] 至少跑完整流：注册→排队→开局→上报→历史战绩可查
- [ ] 数据库备份任务已配，日志留存 ≥ 7 天
