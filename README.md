# Cee - 一起看 / 一起听

一个轻量级、低依赖部署的多人同步播放工具。单 Go 二进制 + 内嵌 Vue 前端，支持本地文件临时托管与公网直链，局域网与公网双场景部署。

## 快速开始

```bash
# 1. 构建前端
cd web && pnpm install && pnpm build

# 2. 构建二进制
cd .. && go build -o watch-together .

# 3. 运行
./watch-together

# 打开 http://localhost:8080
```

## 用法

1. 打开首页，点击"创建房间"或输入房间码加入已有房间
2. 进入房间后，可以：
   - **直链播放**：粘贴 mp4/webm/m3u8 视频直链，所有成员同步观看
   - **上传文件**：选择本地 .mp4/.webm 文件分块上传，成员自动同步
   - **聊天**：发送消息，自动渲染 URL 链接
3. 任意成员均可控制播放/暂停/跳转

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--listen` | `:8080` | 监听地址 |
| `--upload-dir` | `./uploads` | 上传文件存储目录 |
| `--max-room-uploads` | `8GB` | 单房间上传总配额 |
| `--min-free-disk` | `5GB` | 磁盘最低剩余空间 |
| `--upload-idle-timeout` | `30s` | 上传方断线自动取消等待时间 |
| `--base-url` | `""` | 公网部署时设置，影响 join URL |
| `--trust-proxy` | `false` | 反代后开启，从 X-Forwarded-* 取 IP |

## 开发模式

```bash
# 终端 1: 启动 Vite 开发服务器
cd web && pnpm dev

# 终端 2: 启动 Go 后端 (自动反代到 Vite)
go run -tags dev .
```

## 公网部署 (nginx + Let's Encrypt)

参考 [nginx 配置模板](https://github.com/cee/watch-together#nginx)。

## 技术栈

- **后端**: Go 1.22+, Echo v4, Melody (WebSocket)
- **前端**: Vue 3 + Composition API, Naive UI, Pinia, Vite, hls.js
- **上传**: 分块上传 + 服务端顺序拼接

## 许可

Apache License 2.0
