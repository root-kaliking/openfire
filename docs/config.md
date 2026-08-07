# OpenFire 项目配置文档

> 本文档完整描述 OpenFire 项目（Godot 4 客户端 + Go 中央服务端）的所有可配置项、配置文件位置、默认值、修改方式与生效条件。部署相关请见《部署文档》。

---

## 1. 整体架构与配置分层

OpenFire 的配置按"层"分布，从"离开发者最近"到"离玩家最近"依次为：

| 层 | 适用方 | 存储形式 | 覆盖/修改方式 |
|---|---|---|---|
| ① 代码内常量（编译期） | 客户端 + GS + 服务端 | GDScript `const` / Go 常量 | 改代码重编译/重导包 |
| ② Godot 项目设置 `project.godot` | 客户端 + GS | `project.godot` 编辑器保存 | 编辑器 → Project → Project Settings；或手工编辑 |
| ③ 导出版本预设 `export_presets.cfg` | 打包工具 | `export_presets.cfg` | 编辑器 → Export；或手工编辑 |
| ④ Godot Autoload 默认值 | 客户端 + GS 运行时 | `scripts/autoload/*.gd` | 改 GDScript 默认值或写 `user://settings.cfg` 覆盖 |
| ⑤ 玩家侧 `settings.cfg` | 客户端 | `user://settings.cfg`（用户数据目录） | 菜单 → Options / 在线面板切换地址，或直接编辑 |
| ⑥ 玩家侧 `auth.cfg` | 客户端 | `user://auth.cfg`（用户数据目录） | 由 `Auth` 单例写入，不建议手改 |
| ⑦ 服务端环境变量 / `.env` | 中央服务端 (Go) | `server/.env` 或 OS 环境变量 | 编辑 `.env` 或在 systemd/Docker 中注入 |
| ⑧ 数据库持久化内容 | 服务端 | PostgreSQL `users/matches/match_players` | 注册/对局写入；通过迁移脚本更新 schema |

> 关键原则：**高层覆盖低层**。例如玩家侧 `settings.cfg` ([central] url) 会覆盖 settings.gd 中的 `central_url` 默认值，环境变量 `JWT_SECRET` 会覆盖 `config.go` 的默认值。

---

## 2. Godot 项目级配置：`project.godot`

位置：`/workspace/project.godot`

常用配置：

| 键 | 默认值 | 说明 |
|---|---|---|
| `config/name` | `OpenFire` | 显示名 |
| `config/version` | `0.2.31` | 版本号，主菜单/HUD 显示 |
| `autoload/*` | Auth / Lobby / BootChecker / … | Autoload 单例顺序，新增单例必须在此注册，见下表 |
| `display/window/size/viewport_width` | 1280 | 窗口尺寸 |
| `physics/*` | — | 物理世界参数（一般保持默认） |

已注册的 Autoload 单例（与在线对战有关的加粗）：

| 单例名 | 脚本 | 职责 |
|---|---|---|
| Game | `scripts/autoload/game.gd` | 全局分数、模式、对局结束信号 |
| **Net** | `scripts/autoload/net.gd` | ENet 多人会话管理（host/join/max_players） |
| Missions | `scripts/autoload/missions.gd` | 任务解析与列表 |
| WeaponDB | `scripts/autoload/weapon_db.gd` | 武器数据表 |
| Audio / Music | `scripts/autoload/audio.gd` 等 | 音频总线 |
| Settings | `scripts/autoload/settings.gd` | 所有玩家侧设置（见第 4 章） |
| **Auth** | `scripts/autoload/auth.gd` | 账号 / JWT 客户端 |
| **Lobby** | `scripts/autoload/lobby.gd` | 大厅 WebSocket 与匹配客户端 |
| **BootChecker** | `scripts/autoload/boot_checker.gd` | 启动模式检测：CLIENT / DEDICATED_SERVER |

如需新增 Autoload，请在 `project.godot` 的 `[autoload]` 段追加：

```ini
YourSingleton="*res://scripts/autoload/your_singleton.gd"
```

`*` 前缀表示"先于任何场景加载"（即 Singleton）。

---

## 3. 导出版本预设：`export_presets.cfg`

位置：`/workspace/export_presets.cfg`，共 5 个预设：

| 预设名 | 平台 | 角色 | 主要配置 |
|---|---|---|---|
| Linux | Linux x86_64 | 客户端 | embed_pck=true，s3tc_bptc 纹理，排除 tests/audio |
| Windows | Windows x86_64 | 客户端 | 同上，带产品名/图标 |
| macOS | macOS Universal | 客户端 | S3TC + ETC2 纹理，bundle ID = `com.openfire.game` |
| Android | Android arm64-v8a | 客户端 | gradle 构建，internet/vibrate 权限，`debug.keystore` 签名 |
| **Dedicated Server** | Linux x86_64 | **专用服务器（GS）** | `dedicated_server=true`，features=`dedicated`，排除 audio/nobodywho，无纹理压缩 |

### 3.1 Dedicated Server 预设关键参数

```ini
[preset.4]
name="Dedicated Server"
dedicated_server=true          # Godot server template 构建（无渲染器/音频）
custom_features="dedicated"    # 代码里可 Engine.has_feature("dedicated") 判断
exclude_filter="tests/*,assets/kenney/audio/*,addons/nobodywho/*"  # 减小产物体积
export_path="build/openfire-server"
[preset.4.options]
binary_format/embed_pck=true   # 单文件部署
```

**编译产物**：`build/openfire-server`（可执行），由中央服务器通过 `$OPENFIRE_GODOT_BIN` 调用运行；或直接运行 `build/openfire-server --headless --path /workspace res://scenes/dedicated_server.tscn`。

### 3.2 Android 预设说明

- 调试签名密码：`android` / alias=`androiddebugkey`
- 发布签名需要：替换 `keystore/release` / `release_user` / `release_password`，否则 Release APK 无法安装。
- 包名：`com.openfire.game`，版本号随 `project.godot/config/version`。

---

## 4. 玩家侧运行时配置：`Settings` 单例

位置：`scripts/autoload/settings.gd`；持久化文件：`user://settings.cfg`（Godot 跨平台用户数据目录）。

Godot 各平台 `user://` 绝对路径：

| 平台 | 默认位置 |
|---|---|
| Linux | `~/.local/share/godot/app_userdata/OpenFire/settings.cfg` |
| macOS | `~/Library/Application Support/Godot/app_userdata/OpenFire/settings.cfg` |
| Windows | `%APPDATA%\Godot\app_userdata\OpenFire\settings.cfg` |
| Android | `/sdcard/Android/data/com.openfire.game/files/`（或内部存储私有区） |

### 4.1 玩家通用设置

| 键 | 默认值 | 说明 | 范围 |
|---|---|---|---|
| `mouse_sensitivity` | `1.0` | 鼠标灵敏度 | 0.2–3.0 |
| `master_volume` | `0.8` | 主音量 | 0.0–1.0 |
| `fov` | `75.0` | 第一人称 FOV | 60–110 |
| `quality` | `2` | 画质档位（0=Low, 1=Med, 2=High） | 0–2 |
| `debug_mode` | `false` | 单人局调试/作弊菜单 `[0]` 开关 | bool |

对应配置文件片段：

```ini
[general]
mouse_sensitivity=1.0
master_volume=0.8
fov=75.0
quality=2
debug_mode=false
```

### 4.2 本地 LLM（Survival 剧情）设置

| 键 | 默认值 |
|---|---|
| `llm_endpoint` | `http://localhost:1234/v1/chat/completions` |
| `llm_model` | `local-model` |
| `llm_api_key` | `""` |
| `llm_model_url` | HuggingFace `Qwen2.5-1.5B-Instruct-Q4_K_M.gguf` |
| `llm_model_file` | `Qwen2.5-1.5B-Instruct-Q4_K_M.gguf` |

### 4.3 ComfyUI 文本→3D/AI 设置

| 键 | 默认值 | 说明 |
|---|---|---|
| `comfyui_endpoint` | `http://127.0.0.1:8188` | 运行中 ComfyUI 地址 |
| `comfyui_checkpoint` | `v1-5-pruned-emaonly.safetensors` | 主模型 |
| `comfyui_exec` | `""` | 可选：游戏启动 ComfyUI 的命令 |
| `comfyui_args` | `--listen 127.0.0.1 --port 8188` | |
| `comfyui_model_url` | HuggingFace SD1.5 safetensors | 自动下载源 |
| `comfyui_model_file` | `v1-5-pruned-emaonly.safetensors` | |
| `comfyui_bundle_url` | `https://github.com/.../releases/.../comfyui-bundle.zip` | 若本机无 ComfyUI 则下载解压 |
| `comfyui_model_size` | `4265146304` (≈4 GB) | 用于显示下载百分比 |

### 4.4 中央服务器（在线对战）设置 ⭐

| 键 | 默认值 | 说明 | 对应配置节 |
|---|---|---|---|
| `central_url` | `http://127.0.0.1:8080` | 中央服务端 HTTP 基础地址，用于 REST（注册/登录/历史） | `[central] url=` |
| `central_ws_url` | `ws://127.0.0.1:8080/api/ws` | 大厅 WebSocket 升级地址，`?token=` 由代码追加 | `[central] ws_url=` |
| `central_internal_token` | `""` | **GS 专用**。客户端忽略，仅由 `OPENFIRE_INTERNAL_TOKEN` 环境变量读取 | 不用 settings.cfg |

典型**生产环境**玩家配置（写 `user://settings.cfg`）：

```ini
[central]
url="https://api.yourgame.com"
ws_url="wss://api.yourgame.com/api/ws"
```

> 客户端会把上面两个 URL 拼到所有请求：
> - `POST {central_url}/api/auth/register`
> - `POST {central_url}/api/auth/login`
> - `GET  {central_ws_url}?token=<jwt>`（实际是 `ws(s)://.../api/ws?token=...`）
> - `GET  {central_url}/api/matches` （历史战绩，Header: Bearer jwt）

加载与保存函数：`load_settings()` 和 `save()` 分别在 `Settings._ready` 和 UI 改动时调用。

---

## 5. 玩家侧鉴权态缓存：`auth.cfg`

位置：`user://auth.cfg`（与 `settings.cfg` 同目录）；由 `scripts/autoload/auth.gd` 管理。

**格式**：

```ini
[auth]
token="<HS256 JWT，有效期 7 天>"
user_id="<UUID>"
username="<玩家昵称>"
```

**何时写入**：注册成功、登录成功时；`_save_session()`。
**何时清除**：点击"退出登录"→ `logout()` → `_clear_local_state()` 删除本地文件。
**何时恢复**：进程启动 `_ready()` → `try_restore_session()` 读入内存（目前不做远程校验，过期后 REST/WS 会失败并被 UI 提示）。

> ⚠️ 安全提醒：该文件明文保存 JWT，等价于"会话 cookie"。多用户共享系统请注意目录权限。

---

## 6. 大厅与匹配：`Lobby` 单例配置

位置：`scripts/autoload/lobby.gd`。常量参数：

| 常量 | 默认值 | 说明 |
|---|---|---|
| `MAX_RETRIES` | `5` | WebSocket 自动重连次数 |
| `RECONNECT_DELAY` | `3.0` 秒 | 每次重连间隔 |
| `READ_DEADLINE` | 60 秒（与服务端 ping 协调） | 空闲超时后重连 |

连接 URL 构造逻辑（`_open_socket()`）：
```
full_url = Settings.central_ws_url + sep + "token=" + Auth.get_token().uri_encode()
```
sep 取 `?` 或 `&`，取决于 URL 是否已有 query string。

信号（main_menu.gd 订阅）：
- `connected / disconnected(reason) / queued / match_found(match_id,gs_host,gs_port,players,mode) / match_canceled / ws_error(msg)`

---

## 7. 专用游戏服务器（GS）配置

专用服务器由中央服务端通过 `exec.Command` 拉起，**所有配置通过环境变量传递**，不读 `settings.cfg`。

对应代码：`scripts/autoload/boot_checker.gd` + `scripts/dedicated_server.gd`；对应场景：`scenes/dedicated_server.tscn`。

### 7.1 BootChecker 启动模式判定

BootChecker 顺序：
1. 如果检测到环境变量 `OPENFIRE_GS_MATCH_ID` 非空 → **模式 = DEDICATED_SERVER**，切换到 `res://scenes/dedicated_server.tscn`。
2. 否则 → **模式 = CLIENT**，进入 `main_menu.tscn`。

BootChecker 对外暴露的变量（供 `dedicated_server.gd` 读取）：

| 变量 | 来源 | 含义 |
|---|---|---|
| `gs_mode` | `OPENFIRE_GS_MODE` | 模式字符串：deathmatch / team_dm / domination / battle_royale / coop / adventure |
| `gs_match_id` | `OPENFIRE_GS_MATCH_ID` | 对局 UUID，用于向中央服务器上报 |
| `gs_port` | `OPENFIRE_GS_PORT` | ENet UDP 监听端口（默认 27015，被 Net.host_game 使用） |
| `central_url` | `OPENFIRE_CENTRAL_URL` | 回传结果的 HTTP 基础地址 |
| `internal_token` | `OPENFIRE_INTERNAL_TOKEN` | 访问 `/internal/*` 的令牌（X-Internal-Token） |

### 7.2 Dedicated Server 行为与可调参数

流程：
1. `_ready`：读取模式 → `_apply_mode_config(mode)` 写入 `Game.config`
2. 下一帧：`Net.host_game(BootChecker.gs_port)`；端口失败 → `_report_abandoned_and_quit()`
3. `_fetch_roster()`：调用 `GET {central_url}/internal/matches/{id}`（若失败则回退到"连接 1+ 玩家且 15s 后开赛"）
4. `_process`：满足开赛条件（全员到位 或 回退计时）→ `Net.start_match()`，0.2s 后 `change_scene_to_file(world.tscn)`
5. `Game.match_over` → `_post_result(players_payload)` 上报结果 → 5s 后 quit
6. 保护：300 秒无任何玩家连接 → abandon + quit；收到 `NOTIFICATION_WM_CLOSE_REQUEST` → abandon + quit

`dedicated_server.gd` 关键常量：

| 常量 | 值 | 说明 |
|---|---|---|
| `WORLD_SCENE` | `res://scenes/world.tscn` | 对局世界主场景 |
| `RESULT_PATH_FMT` | `%s/internal/matches/%s/result` | 上报 URL 模板 |
| `_MODE_MAP` | 6 种模式 ↔ Game.Mode enum | 后台模式字符串映射 |
| 5 分钟无人超时 | `300.0` | 硬安全上限 |
| 无名单回退开赛宽限 | `15.0` 秒 | 玩家至少 1 人且超过宽限开始 |

---

## 8. 中央服务端（Go）配置

位置：`server/internal/config/config.go`；所有值从环境变量加载（无配置文件）；可选 `.env` 文件（`github.com/joho/godotenv`）。

配置加载函数：`config.Load()`，启动后返回 `*Config`，包含：

### 8.1 完整配置项表

| 环境变量 | 默认值 | 类型 | 含义 | 校验 |
|---|---|---|---|---|
| `HTTP_ADDR` | `:8080` | string | 监听地址 | — |
| `DB_URL` | `postgres://openfire:openfire@localhost:5432/openfire?sslmode=disable` | pq URL | Postgres 连接串 | 启动时 ping |
| `REDIS_URL` | `redis://localhost:6379/0` | go-redis URL | Redis 客户端（当前未强用，预留给分布式） | ParseURL 失败报错 |
| `JWT_SECRET` | `dev-secret-change-me` | string | HS256 签名密钥 | 非空；生产必须替换 |
| `JWTTTL` | 7 \* 24h | 时长 | Token 有效期 | — |
| `INTERNAL_TOKEN` | `dev-internal-token` | string | GS ↔ 中央通信令牌 | 非空；生产必须替换 |
| `GS_HOST` | `127.0.0.1` | string | 广播给玩家的 GS 主机地址/域名（匹配监听网卡） | — |
| `GS_PORT_MIN` | `27020` | int | GS 端口池下界 | GS_PORT_MAX > MIN |
| `GS_PORT_MAX` | `27099` | int | GS 端口池上界 | |
| `OPENFIRE_GODOT_BIN` 或 `GODOT_BIN`（后者回退）或默认 | `godot` | 路径 | 启动 GS 用的 Godot 二进制 | 必须在 PATH 或使用绝对路径 |
| `OPENFIRE_CENTRAL_URL` | `http://127.0.0.1:8080` | URL | 注入 GS 进程的回传 URL | — |
| `MATCHMAKING_TIMEOUT` | `120s` | 时长 | 排队超时（超时后自动踢出并通知） | — |
| `MIGRATIONS_PATH` | `migrations/001_init.sql` | 相对 server/ 的路径 | 启动时自动执行 | 读取失败即 Fatal |

### 8.2 `.env` 示例（对应 `server/.env.example`）

```sh
# ===== 生产必须替换 =====
JWT_SECRET=$(openssl rand -hex 32)
INTERNAL_TOKEN=$(openssl rand -hex 32)

# ===== 部署相关 =====
HTTP_ADDR=:8080
DB_URL=postgres://openfire:StrongPass%21@db.local:5432/openfire?sslmode=require
REDIS_URL=redis://:pass@redis.local:6379/0

# ===== 对战端暴露的地址（玩家客户端可达） =====
GS_HOST=gs.yourgame.com
GS_PORT_MIN=27020
GS_PORT_MAX=27099

# ===== GS 进程 =====
OPENFIRE_GODOT_BIN=/opt/openfire/openfire-server
OPENFIRE_CENTRAL_URL=https://api.yourgame.com
MATCHMAKING_TIMEOUT=120s
MIGRATIONS_PATH=migrations/001_init.sql
```

### 8.3 启动期自动行为

1. 加载 `.env`（不存在不报错）
2. 打开 PG 连接池：`maxOpen=20 / maxIdle=5 / maxLifetime=1h`
3. 打开 Redis 客户端
4. 读取 `MIGRATIONS_PATH` 文件 → `Exec(string(sql))` 全量执行（幂等：全部 `IF NOT EXISTS`）
5. 构建：Hub → GS Manager → Queue，循环依赖通过 `SetHandler(queue)` 解耦
6. 注册 chi 路由 + 启动 HTTP Server
7. 阻塞到 SIGINT / SIGTERM → `srv.Shutdown(10s)` → 返回 0

### 8.4 数据库 Schema（迁移脚本 `server/migrations/001_init.sql`）

```sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,           -- bcrypt hash
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mode TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','in_progress','finished','abandoned')),
    gs_host TEXT NOT NULL DEFAULT '',
    gs_port INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_created_at ON matches(created_at DESC);

CREATE TABLE IF NOT EXISTS match_players (
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username TEXT NOT NULL,
    kills INTEGER NOT NULL DEFAULT 0,
    deaths INTEGER NOT NULL DEFAULT 0,
    placement INTEGER,
    team INTEGER,
    PRIMARY KEY (match_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_match_players_user_id ON match_players(user_id);
```

如需新增字段（例如段位、MMR），推荐以**新的 `migrations/NNN_xxx.sql` 文件**方式追加，并更新 `MIGRATIONS_PATH` 或在启动逻辑里按序执行。

---

## 9. 代码内常量（无需配置文件的"设计"设定）

### 9.1 Net（ENet）— `scripts/autoload/net.gd`

| 常量 | 值 | 说明 |
|---|---|---|
| `DEFAULT_PORT` | `27015` | 默认 UDP 端口 |
| `MAX_PLAYERS` | `16` | 上限（为 battle_royale 调大；小模式留空槽位不影响） |

`host_game(port, max_players=MAX_PLAYERS)` 可按模式调整；被专用服务器默认调用 `BootChecker.gs_port`。

### 9.2 Matchmaking — `server/internal/models/match.go`

| 模式字符串 | 房间人数 | 对应 Game.Mode |
|---|---|---|
| `deathmatch` | 8 | DEATHMATCH |
| `team_dm` | 8 | TEAM_DEATHMATCH |
| `domination` | 8 | DOMINATION |
| `battle_royale` | 16 | BATTLE_ROYALE |
| `coop` | 4 | COOP |
| `adventure` | 4 | ADVENTURE |

未列出的模式字符串 → `KnownMode()` 返回 false，会被 HTTP 400/WS error。

### 9.3 账号规则 — handlers + 客户端协同

| 参数 | 校验（服务端强校验） |
|---|---|
| 用户名长度 | 1–32 字符 |
| 密码长度 | 6–128 字符 |
| 密码哈希 | bcrypt.DefaultCost ≈ 10 轮 |
| 用户名冲突 | pq SQLSTATE 23505 → HTTP 409 "username already taken" |
| JWT 算法 | HS256；Claims 含 `user_id`、`username`、过期 |

### 9.4 CI / 发布流程变量（非配置文件，见 `release.yml`）

| 变量 | 值 | 说明 |
|---|---|---|
| `GODOT_VERSION` / `GODOT_RELEASE` | `4.7.1` / `stable` | CI 下载的 Godot 版本 |
| 发布触发 | `workflow_dispatch` + bump 选项（patch/minor/major） | push/PR 只跑 check 不发版 |
| `NOBODYWHO_URL` | Repo Actions Variables | 可选：将 NobodyWho 原生库打包进客户端 |

---

## 10. 常见自定义配置清单

### ✅ 场景：从本地开发切换到线上中央服务器

玩家侧 `user://settings.cfg`：
```ini
[central]
url="https://api.yourgame.com"
ws_url="wss://api.yourgame.com/api/ws"
```

服务端 `server/.env`：
```sh
JWT_SECRET=256-bit-random
INTERNAL_TOKEN=256-bit-random
DB_URL=postgres://...?sslmode=require
REDIS_URL=redis://...
GS_HOST=gs.yourgame.com
OPENFIRE_GODOT_BIN=/opt/openfire/openfire-server
OPENFIRE_CENTRAL_URL=https://api.yourgame.com
MATCHMAKING_TIMEOUT=180s
```

### ✅ 场景：把 MAX_PLAYERS 改回 8（关闭 battle_royale）

- 修改 `scripts/autoload/net.gd` 的 `MAX_PLAYERS = 8`
- 同步删除/修改 `models/match.go` 的 `battle_royale = 16`（保持一致）
- 重导出客户端与 GS 二进制

### ✅ 场景：修改账号密码长度/规则

服务端：`server/handlers/auth_handler.go`
```go
if username == "" || len(username) > 32 { /*...*/ }
if len(req.Password) < 6 || len(req.Password) > 128 { /*...*/ }
```
客户端同步：`scripts/ui/main_menu.gd` `_on_login_pressed / _on_register_pressed` 已有基础非空检查，可加 UI 层长度提示。

### ✅ 场景：扩展一种新的模式 "ctf"（夺旗）

1. 服务端：`models/match.go` → `KnownMode()` 增加 `"ctf"`；`ModePlayers()` 返回例如 `8`。
2. 客户端 GS 侧：`dedicated_server.gd` `_MODE_MAP` 增加 `"ctf": Game.Mode.CTF`（需要在 `game.gd` 加 enum 并写游戏规则）。
3. 客户端菜单：`main_menu.gd` `_selected_online_mode()` 增加映射。
4. 重新发布并重启中央服务端。

---

## 11. 配置清单（上线前核查表）

| # | 项目 | 目标 |
|---|---|---|
| 1 | `JWT_SECRET`、`INTERNAL_TOKEN` | 使用 `openssl rand -hex 32` 随机生成，不使用默认值 |
| 2 | Postgres 账户权限 | 非超级用户；只允许本地或内网访问；考虑 sslmode=require |
| 3 | `GS_HOST` | 玩家机器**可直接 UDP 连接**的公网 IP/域名；注意防火墙开放 `GS_PORT_MIN–MAX` UDP 段 |
| 4 | `OPENFIRE_GODOT_BIN` | 使用绝对路径，且权限为可执行（`chmod +x`） |
| 5 | `HTTP_ADDR` | 建议配 `127.0.0.1:8080` + 前置 Nginx/Caddy 做 TLS 与 WSS 升级 |
| 6 | 客户端 `central_url/ws_url` | 指向正式域名；ws:// → wss://（HTTPS 站需 WSS，浏览器会拒绝 ws:// 混用） |
| 7 | `JWTTTL` | 默认 7 天。需要更强安全 → 1 天 + 自动刷新（暂未实现 Refresh token，需二次开发） |
| 8 | `MIGRATIONS_PATH` | 指向真实存在的迁移文件目录（若拆分为多文件，需要把 Migrate 改造为目录遍历） |

---

版本：`OpenFire 配置文档 v1.0`，对应项目版本 `0.2.31`。
