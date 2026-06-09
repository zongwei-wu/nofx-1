# NOFX 生产部署记录 (ttai.me)

> 部署日期：2026-06-09 | 分支：`dev-cursor` | 服务器：Alibaba Cloud Linux 3

## 环境信息

| 项目 | 版本/值 |
|------|---------|
| OS | Alibaba Cloud Linux 3 (Anolis) |
| Go | 1.23.4 |
| Node.js | v22.22.3 |
| npm | 10.9.8 |
| PM2 | 最新 stable |
| nginx | 1.20.1 |
| 公网 IP | 47.242.54.28 |
| 域名 | www.ttai.me |

## 1. 安装依赖

```bash
# Go
GO_VERSION=1.23.4
curl -sLO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
export PATH=$PATH:/usr/local/go/bin

# PM2
sudo npm install -g pm2

# certbot (SSL)
sudo yum install -y certbot python3-certbot-nginx
```

## 2. 克隆项目

```bash
git clone https://github.com/zongwei-wu/nofx-1.git
cd nofx-1
git checkout dev-cursor
```

## 3. 配置文件

```bash
cp .env.example .env
cp config.json.example config.json
```

生成加密密钥并写入 `.env`：
```bash
DATA_ENCRYPTION_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
echo "DATA_ENCRYPTION_KEY=${DATA_ENCRYPTION_KEY}" >> .env
echo "JWT_SECRET=${JWT_SECRET}" >> .env
chmod 600 .env
```

生成 RSA 密钥对（首次需要）：
```bash
mkdir -p secrets
openssl genrsa -out secrets/rsa_key 2048
openssl rsa -in secrets/rsa_key -pubout -out secrets/rsa_key.pub
chmod 600 secrets/rsa_key
```

## 4. Bug 修复（dev-cursor 分支必须）

### 4.1 后端：路由重复注册 panic

**文件**：`api/server.go:171-176`

`hermesGroup` 中重复注册了 `/exchanges`、`/models` 路由（已在 `aiTrader` 组中注册），导致 `panic: handlers are already registered`。

**修复**：删除 `hermesGroup` 中的重复路由。

### 4.2 前端：npm install 失败

**文件**：`web/package.json:14`

`"prepare": "husky"` 在没有全局 husky 时会导致 `npm install` 报错退出。

**修复**：改为 `"prepare": "husky || true"`

## 5. 编译 & 安装

```bash
# 编译后端
export PATH=$PATH:/usr/local/go/bin
go build -o nofx .

# 安装前端依赖（需先做完 4.2 的修复）
cd web
npm install
cd ..
```

### 5.1 前端依赖补充（解决 npm 遗漏问题）

npm 偶发不会将以下包写入 `node_modules`，需全局安装后手动复制：

```bash
sudo npm install -g vite@6.0.7 @vitejs/plugin-react@4.3.4 tailwindcss@3.4.17 postcss@8.4.49 autoprefixer@10.4.20

cp -r /usr/lib/node_modules/vite web/node_modules/vite
cp -r /usr/lib/node_modules/@vitejs web/node_modules/@vitejs
cp -r /usr/lib/node_modules/tailwindcss web/node_modules/tailwindcss
cp -r /usr/lib/node_modules/postcss web/node_modules/postcss
cp -r /usr/lib/node_modules/autoprefixer web/node_modules/autoprefixer
```

## 6. PM2 启动

使用全局 vite 而非项目 node_modules 中的（避免 npm 反复清空）：

**`pm2.config.js` 关键配置**：

```javascript
// 后端
{ name: 'nofx-backend', script: './nofx', cwd: __dirname,
  env: { NODE_ENV: 'production', ADMIN_PASSWORD: 'Admin123!' } }

// 前端 — 使用全局 vite
{ name: 'nofx-frontend', script: '/usr/bin/vite',
  args: '--host 0.0.0.0 --port 3000', cwd: path.join(__dirname, 'web') }
```

```bash
chmod +x nofx
./pm2.sh start
```

## 7. Nginx 配置

**文件**：`/etc/nginx/conf.d/ttai.conf`

```nginx
server {
    server_name www.ttai.me ttai.me;
    listen 443 ssl;

    # 管理后台 (Ant Design Pro)
    location /admin {
        alias /home/admin/nofx-1/admin-web/dist;
        index index.html;
        try_files $uri $uri/ /admin/index.html;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # 前端 Vite
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}

# HTTP → HTTPS 跳转
server {
    listen 80;
    server_name www.ttai.me ttai.me;
    return 301 https://$host$request_uri;
}
```

```bash
sudo nginx -t && sudo nginx -s reload
```

### 权限问题

如果 nginx 返回 500 + Permission denied，需给路径加执行权限：

```bash
chmod +x /home/admin /home/admin/nofx-1 /home/admin/nofx-1/admin-web
chmod +rx /home/admin/nofx-1/admin-web/dist
```

## 8. SSL 证书

```bash
sudo certbot --nginx -d www.ttai.me -d ttai.me
```

自动续期已配置（certbot timer）。

## 9. 管理后台

```bash
cd admin-web
npm install
npm run build
```

构建产物：`admin-web/dist/`

## 10. 管理员账号

系统自动创建管理员用户。通过环境变量 `ADMIN_PASSWORD` 设置密码：

```bash
# 在 pm2.config.js 中配置，或直接设置数据库
sqlite3 config.db "UPDATE users SET email='admin@ttai.me' WHERE id='admin'"
```

## 11. 部署架构总览

```
                     ┌──────────────────┐
                     │   Cloudflare /   │
                     │   DNS (ttai.me)  │
                     └────────┬─────────┘
                              │ 47.242.54.28:443
                     ┌────────▼─────────┐
                     │  nginx (1.20.1)  │
                     │  SSL termination │
                     └──┬──────┬─────┬──┘
                /       │      │      \ /admin
           ┌────────────▼┐  ┌──▼──┐  ┌──────────┐
           │ Vite :3000  │  │:8080│  │ 静态文件  │
           │ (前端 SPA)   │  │ API │  │ admin-web │
           └─────────────┘  └─────┘  └──────────┘
                   │            │
           ┌───────▼────────────▼───────┐
           │         PM2 管理           │
           │  nofx-backend | nofx-frontend │
           └────────────────────────────┘
```

## 12. 日常运维

```bash
cd /home/admin/nofx-1

# 查看状态
./pm2.sh status

# 查看日志
./pm2.sh logs backend
./pm2.sh logs frontend

# 重启
./pm2.sh restart

# 重新编译后端
./pm2.sh rebuild

# SSL 续期检查
sudo certbot renew --dry-run
```
