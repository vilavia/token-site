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

## Linux 部署

### 前置要求

- Docker 和 Docker Compose
- 一个域名（可选，本地测试不需要）

### 1. 克隆代码

```bash
git clone https://github.com/vilavia/token-site.git
cd token-site
```

### 2. 配置环境变量

```bash
cd deploy
cp .env.example .env
nano .env
```

必须修改的配置：

```env
# 数据库密码（改成强密码）
POSTGRES_PASSWORD=你的强密码

# 管理员
ADMIN_EMAIL=admin@你的域名.com
ADMIN_PASSWORD=你的管理员密码

# JWT密钥（必须改，用下面的命令生成）
# openssl rand -hex 32
JWT_SECRET=生成的随机hex

# TOTP密钥（必须改）
# openssl rand -hex 32
TOTP_ENCRYPTION_KEY=生成的随机hex

# 易支付（可在后台网页设置里改，这里可以先不填）
EPAY_ENABLED=false
```

### 3. 创建数据目录

```bash
mkdir -p data postgres_data redis_data
```

### 4. 启动服务

**生产环境（使用官方镜像 + 本地数据目录）：**

```bash
docker compose -f docker-compose.local.yml up -d
```

**开发环境（从源码构建）：**

```bash
docker compose -f docker-compose.dev.yml up --build -d
```

### 5. 运行数据库迁移

首次部署需要运行（只需执行一次）：

```bash
# 支付订单表
docker exec sub2api-postgres psql -U sub2api -d sub2api -c "
CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    order_no VARCHAR(64) NOT NULL UNIQUE,
    trade_no VARCHAR(64),
    amount_usd DECIMAL(20,8) NOT NULL,
    amount_rmb DECIMAL(20,2) NOT NULL,
    exchange_rate DECIMAL(10,4) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    pay_type VARCHAR(32),
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payment_orders_user_id ON payment_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_status ON payment_orders(status);
"

# 聊天记录表
docker exec sub2api-postgres psql -U sub2api -d sub2api -c "
CREATE TABLE IF NOT EXISTS chat_conversations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL DEFAULT 'New Chat',
    model VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_id ON chat_conversations(user_id);

CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);
"
```

> 如果用 docker-compose.dev.yml，容器名是 `sub2api-postgres-dev`

### 6. 访问

- 网站：`http://你的IP:8080`
- 管理后台：登录后左侧菜单切换到管理视图

### 7. 初始配置（在网页后台操作）

1. **设置 → Payment** — 配置易支付参数（API地址、商户ID、密钥、回调URL、汇率）
2. **设置 → Payment** — 设置充值金额上下限和预设金额
3. **账号管理** — 添加 AI 供应商账号（OpenAI API Key 或 OAuth）
4. **分组管理** — 创建分组，设置计费倍率，将账号加入分组

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
