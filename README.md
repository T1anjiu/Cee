# Cee — 一起看

**多人同步播放工具** · 单二进制部署 · 直链与上传双模式

```
功能: 创建房间 → 粘贴直链/上传文件 → 邀请好友 → 同步观看
场景: 局域网派对 / 异地同步观赏 / 小圈子分享
```

---

## 快速开始

```bash
# 下载后在终端直接运行
./watch-together

# 浏览器打开 http://localhost:8080
# 点击"创建房间"，把链接分享给好友即可
```

> 无需 Node.js、无需数据库、无需任何配置。Windows 双击 `watch-together.exe`，macOS/Linux 执行对应二进制。

---

## 使用说明

### 创建房间

| 步骤 | 操作 |
|------|------|
| 1 | 打开首页，点击 **创建房间** |
| 2 | 浏览器自动跳转到房间页，输入昵称加入 |
| 3 | 复制地址栏 URL 或房间码发给好友 |

### 加入房间

| 方式 | 操作 |
|------|------|
| 点击链接 | 直接打开好友分享的房间链接 |
| 输入房间码 | 在首页输入 6 位房间码（不区分大小写，`O`→`0` 自动纠正） |

### 播放视频

**直链播放** — 粘贴视频直链即可同步观看

```
支持格式: .mp4 / .webm / .m3u8（HLS 受跨域限制，大概率失败）
示例: https://example.com/movie.mp4
```

操作: 房间内点 **直链** Tab → 输入 URL → 提交 → 全员同步播放

---

**上传文件** — 房主上传本地视频，全员同步观看

```
支持格式: .mp4 / .webm
限制: 单文件 ≤ 4GB，单房间配额 8GB
```

操作: 房间内点 **上传** Tab → 选择文件 → 自动分块上传 → 完成后全员同步播放

### 播放控制

任意成员均可控制：

| 操作 | 说明 |
|------|------|
| 播放 / 暂停 | 点击中央播放按钮 |
| 快进 / 快退 | ±10 秒步进 |
| 进度条拖拽 | 点击进度条任意位置跳转 |
| 同步机制 | 服务端权威时钟 + 客户端 `playbackRate` 软同步，偏差 ≥1s 时硬 seek |

### 聊天

房间内所有成员可实时聊天，URL 自动识别为可点击链接。

---

## 命令行参数

```bash
./watch-together \
  --listen :8080 \
  --upload-dir ./uploads \
  --max-upload-size 4GB \
  --max-room-uploads 8GB \
  --min-free-disk 5GB \
  --upload-idle-timeout 30s \
  --base-url "" \
  --trust-proxy false
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--listen` | `:8080` | 监听地址 |
| `--upload-dir` | `./uploads` | 上传文件存储目录 |
| `--max-upload-size` | `4GB` | 单文件大小上限 |
| `--max-room-uploads` | `8GB` | 单房间上传总配额 |
| `--min-free-disk` | `5GB` | 磁盘最低剩余空间，低于此值拒绝新上传 |
| `--upload-idle-timeout` | `30s` | 上传方断线后自动取消的等待时间 |
| `--base-url` | `""` | 公网部署时设置，影响生成的 join URL |
| `--trust-proxy` | `false` | 反代后开启，从 `X-Forwarded-*` 取客户端 IP |

---

## 部署

### 局域网

```bash
./watch-together
# 访问 http://你的局域网IP:8080
# 确保防火墙放行 8080 端口
```

### 公网 (nginx + Let's Encrypt)

```nginx
server {
    listen 443 ssl http2;
    server_name watch.example.com;

    client_max_body_size 4G;
    proxy_request_buffering off;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
./watch-together --trust-proxy=true --base-url=https://watch.example.com
```

> 公网上传场景注意流量：4 人观看 1080p 一小时约 9GB 出向流量，建议使用不限流量 VPS。

---

## 开发

```bash
# 终端 1: Vite 开发服务器 (热更新)
cd web && pnpm install && pnpm dev

# 终端 2: Go 后端 (自动反代到 Vite)
go run -tags dev .
```

### 构建

```bash
cd web && pnpm build        # 构建前端 → web/dist/
cd .. && go build -o watch-together .  # 内嵌前端 → 单二进制
```

### 运行测试

```bash
go test ./...                              # 后端单元测试
go test ./internal/upload/... -tags integration  # 后端集成测试
pnpm -C web test                           # 前端测试
```

---

## 技术栈

| 层 | 选型 |
|------|------|
| 后端语言 | Go 1.22+ |
| Web 框架 | Echo v4 |
| WebSocket | Melody (Session + 房间广播) |
| 前端框架 | Vue 3 Composition API |
| 状态管理 | Pinia |
| UI 库 | Naive UI (暗色主题) |
| 构建工具 | Vite |
| 流媒体 | hls.js |
| 上传策略 | 分块上传 + 服务端顺序拼接 |

### 项目结构

```
watch-together/
├── main.go                  # 入口, embed web/dist
├── static_dev.go            # 开发模式 Vite 反代
├── static_prod.go           # 生产模式 embed 静态文件
├── internal/
│   ├── room/                # 房间管理器 + 状态机
│   │   ├── manager.go       # 房间 CRUD + GC
│   │   ├── room.go          # Room / Member 模型
│   │   ├── player.go        # 三态 PlayerState 状态机
│   │   └── player_test.go   # 状态机测试
│   ├── upload/               # 分块上传管理器
│   │   ├── manager.go       # 上传任务 + 拼接 + GC
│   │   ├── disk_unix.go     # Unix 磁盘空间查询
│   │   └── disk_windows.go  # Windows 磁盘空间查询
│   ├── ws/hub.go            # WebSocket 消息路由 + 限流
│   ├── handler/             # HTTP 处理器
│   │   ├── room.go          # 房间 API
│   │   └── upload.go        # 上传 API + 媒体流
│   ├── model/message.go     # WS 消息类型定义
│   └── util/id.go           # 房间码/UUID/token 生成
├── web/                     # 前端
│   └── src/
│       ├── views/           # Home, Room
│       ├── components/      # PlayerArea, ChatPanel, MediaInput 等
│       ├── composables/     # useWebSocket, usePlayer, useUploader 等
│       ├── stores/          # Pinia store
│       └── types/           # TypeScript 类型定义
└── uploads/                 # 上传文件存储目录
```

---

## 设计要点

- **服务端权威**: 所有播放状态以服务端为准，客户端推算实时位置
- **软同步优先**: 通过 `playbackRate` 微调追赶，差距 ≥1s 才触发 seek
- **三态状态机**: `Playing` / `UserPaused` / `BufferingPaused`，避免频繁暂停抖动
- **无用户系统**: 匿名昵称 + sessionStorage token，重启即清空
- **自动清理**: 房间空置 5 分钟销毁，上传文件 24h 强制回收

---

## License

Apache License 2.0
