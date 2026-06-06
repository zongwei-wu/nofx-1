# NOFX 管理后台

基于 Ant Design Pro（Umi Max + ProComponents）的运营后台，部署在 `/admin` 子路径。

## 本地开发

1. 确保后端已启动（默认 `http://localhost:8080`）
2. 设置管理员密码（首次）：

```bash
export ADMIN_PASSWORD=your-secure-password
# 重启后端，admin 账号密码将被写入数据库
```

3. 启动管理端：

```bash
cd admin-web
npm install
npm run dev
```

开发服务器默认 `http://localhost:8000`，访问 `http://localhost:8000/admin/`。

API 请求通过 Umi 代理转发到 `http://localhost:8080`。

## 管理员登录

- 邮箱：`admin@localhost`
- 密码：通过环境变量 `ADMIN_PASSWORD` 在首次启动时设置

管理员账号跳过 OTP，直接使用 `POST /api/login` 登录。

## 功能模块

| 菜单 | 路径 | 说明 |
|------|------|------|
| 用户管理 | `/user/list` | 分页查询全站用户 |
| 跟单记录 | `/copy-trade/records` | 跨用户跟单记录查询 |
| 系统配置 | `/system/config` | 注册开关等运营配置 |

## 生产构建

```bash
npm run build
```

产物输出到 `dist/`，由 Nginx 挂载到 `/admin/`。
