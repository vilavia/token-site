# Token Site

AI Token 转售平台，基于 [sub2api](https://github.com/Wei-Shaw/sub2api) 二次开发。

## 上游版本

| 项目 | 值 |
|------|-----|
| 上游仓库 | https://github.com/Wei-Shaw/sub2api |
| Fork 基线 | `94bba415` (v0.1.104) |
| **当前同步到** | **`a225a241`** (2026-03-20) |

## 新增功能

- **易支付** — 后台可配置支付参数，支持支付宝/微信，充值金额上下限和预设金额可配
- **按量付费** — 移除订阅模式，按 token 用量扣费（美元）
- **在线聊天** — 网页版多会话聊天，对话记录云端保存，支持后台开关
- **模型目录** — 从账号配置动态读取，按版本号排序，显示我们的价格和官方价格
- **管理增强** — 订单管理页、支付设置、用户订单查看、对话开关

## 安装

前置要求：Linux (amd64/arm64) + PostgreSQL + Redis

```bash
curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash
```

首次打开 `http://服务器IP:8080` 进入引导页面，配置数据库连接和管理员账号。

## 常用命令

```bash
systemctl status sub2api      # 查看状态
journalctl -u sub2api -f      # 查看日志
systemctl restart sub2api     # 重启
systemctl stop sub2api        # 停止
```

## 升级

```bash
curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash -s upgrade
```

## 卸载

```bash
curl -sSL https://raw.githubusercontent.com/vilavia/token-site/main/install.sh | bash -s uninstall
```

## 初始配置

引导完成后进入管理后台：

1. **账号管理** — 添加 AI 供应商账号（OpenAI 等）
2. **分组管理** — 创建分组，设置计费倍率
3. **设置 → Payment** — 配置易支付参数和充值选项

## Nginx 反向代理（可选）

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
        proxy_buffering off;
        proxy_read_timeout 300s;
    }
}
```

## 同步上游更新

```bash
git fetch upstream
git merge upstream/main
# 重新编译并发布 Release，然后在服务器上 upgrade
```

## 项目结构

```
install.sh              # 一键安装脚本
deploy/
  config.example.yaml   # 配置模板
backend/
  cmd/server/           # 程序入口
  internal/             # 业务代码
  ent/                  # ORM
  migrations/           # 数据库迁移（自动执行）
frontend/
  src/views/            # 页面
  src/components/       # 组件
  src/stores/           # 状态管理
  src/api/              # API 调用
```

## License

[MIT](LICENSE)
