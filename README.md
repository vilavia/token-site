# Token Site

AI Token 转售平台，基于 [sub2api](https://github.com/Wei-Shaw/sub2api) 二次开发。

## 上游版本追踪

| 项目 | 值 |
|------|-----|
| 上游仓库 | https://github.com/Wei-Shaw/sub2api |
| Fork 版本 | v0.1.104 |
| Fork commit | `94bba415b1e5b3f8ed36a49ac818d2443074333e` |
| **当前同步到** | **`a225a241d76b2c4f2771e28ed1fad02438505eb4`** (2026-03-20) |

> 每次同步上游更新后，更新上面的"当前同步到"行。

## 相比原项目的改动

- **易支付集成** — 后台可配置易支付参数（PID/密钥/回调URL/汇率），用户充值页面支持预设金额和自定义金额，支持支付宝/微信
- **按量付费** — 移除了订阅模式，所有用户按 token 使用量扣费（美元）
- **在线聊天** — 类似 ChatGPT 的网页聊天，对话记录保存到数据库，支持多会话切换，每个会话显示使用的模型
- **模型目录** — 从后台账号配置动态读取可用模型，按版本号排序，显示我们的价格和官方价格
- **后台管理增强** — 支付设置页、订单管理页、用户管理里可查看订单、充值金额上下限/预设金额配置、在线对话开关

## 部署

### 前置要求

- Linux 服务器（amd64 或 arm64）
- Docker + Docker Compose
- PostgreSQL（15+）
- Redis（6+）

### 一键安装

```bash
curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash
```

脚本会自动拉取 Docker 镜像（`ghcr.io/vilavia/token-site:latest`，Alpine 基础，约30MB）并启动。

首次打开 `http://服务器IP:8080` 进入引导页面，配置数据库连接和管理员账号。数据库表会在首次启动时自动创建。

### 手动安装

```bash
mkdir token-site && cd token-site

# 下载 compose 文件（只有应用容器，不含数据库）
curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/deploy/docker-compose.standalone.yml -o docker-compose.yml
curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/deploy/.env.example -o .env

# 修改镜像名
sed -i 's|weishaw/sub2api:latest|ghcr.io/vilavia/token-site:latest|' docker-compose.yml

# 生成密钥
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(openssl rand -hex 32)/" .env
sed -i "s/^TOTP_ENCRYPTION_KEY=.*/TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)/" .env

# 启动
docker compose up -d
```

### 初始配置（在网页后台操作）

1. **设置 → Payment** — 配置易支付参数
2. **账号管理** — 添加 AI 供应商账号
3. **分组管理** — 创建分组，设置计费倍率

## 反向代理（生产环境推荐）

用 Nginx 做反向代理并配置 HTTPS：

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE/WebSocket 支持（聊天需要）
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 300s;
    }
}
```

配置好后将易支付的回调URL改成 `https://your-domain.com/api/v1/payment/notify`。

## 常用命令

```bash
# 查看日志
docker compose -f docker-compose.local.yml logs -f sub2api

# 重启
docker compose -f docker-compose.local.yml restart sub2api

# 停止
docker compose -f docker-compose.local.yml down

# 更新（从源码构建时）
git pull
docker compose -f docker-compose.dev.yml up --build -d
```

## 同步上游更新

```bash
# 获取上游最新代码
git fetch upstream

# 查看有哪些新commit
git log upstream/main --oneline -10

# 合并（合并前确保本地修改已commit）
git merge upstream/main

# 如有冲突，手动解决后 git add + git commit
# 重新构建部署
docker compose -f docker-compose.dev.yml up --build -d

# 推送到自己的仓库
git push origin main

# 更新本文件的"当前同步到"版本号
```

## 文件结构

```
deploy/
  .env.example       # 环境变量模板
  .env                # 实际环境变量（不入库，含密码）
  docker-compose.local.yml   # 生产部署（官方镜像）
  docker-compose.dev.yml     # 开发部署（源码构建）
  data/               # 应用数据（不入库）
  postgres_data/       # 数据库数据（不入库）
  redis_data/          # Redis数据（不入库）

backend/
  cmd/server/          # 程序入口
  internal/
    config/            # 配置加载
    handler/           # HTTP处理器
    handler/admin/     # 管理后台API
    service/           # 业务逻辑
    server/routes/     # 路由注册
  ent/                 # ORM生成代码
  migrations/          # SQL迁移脚本

frontend/
  src/
    views/user/        # 用户页面（聊天/模型/充值）
    views/admin/       # 管理页面（订单/设置）
    components/        # UI组件
    stores/            # Pinia状态管理
    api/               # API调用封装
    i18n/              # 中英文翻译
```
