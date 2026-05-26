# Hero3 服务器部署文档

本文记录 Hero3 当前线上服务器的真实部署结构，以及以后从本地发布到服务器的标准流程。

> 注意：本文不记录数据库密码、SSH 密码、证书私钥等敏感信息。服务器上的敏感配置放在 `/etc/hero3/hero3.env`。

## 当前线上结构

线上入口：

- 玩家站：`https://hero3.ccoos.cn/`
- GM 后台：`https://hero3.ccoos.cn/admin/`
- 健康检查：`https://hero3.ccoos.cn/healthz`

服务器路径：

```text
/opt/hero3/go/bin/hero3-server     # Go 后端可执行文件
/opt/hero3/go/config/              # 后端生产配置，如 balance/factions/units
/var/www/hero3/web/                # 玩家前端静态文件
/var/www/hero3/admin/              # GM 后台静态文件
/var/www/hero3/backups/            # 前端静态文件备份
/opt/hero3/backups/                # 后端文件备份
/etc/hero3/hero3.env               # 后端环境变量
/etc/systemd/system/hero3.service  # systemd 服务
/etc/nginx/conf.d/hero3.ccoos.cn.conf # Nginx 站点配置
```

当前运行方式：

```text
Nginx 80/443
  /         -> /var/www/hero3/web/
  /admin/   -> /var/www/hero3/admin/
  /api/     -> http://127.0.0.1:8081
  /healthz  -> http://127.0.0.1:8081

hero3.service
  WorkingDirectory=/opt/hero3/go
  ExecStart=/opt/hero3/go/bin/hero3-server
  EnvironmentFile=/etc/hero3/hero3.env
```

后端监听端口：

```text
HERO3_PORT=8081
```

数据库：

```text
MySQL/MariaDB: 127.0.0.1:3306
Database: hero3
```

IP 根路径说明：

```text
http://124.223.111.163/
```

这个 IP 根路径当前不是 Hero3，而是 Nginx 中另一个 `ocr-lab-report` 服务。Hero3 通过域名 `hero3.ccoos.cn` 访问。

## 首次部署准备

服务器需要具备：

- Git
- Go
- Node.js
- pnpm
- MySQL/MariaDB
- Nginx
- systemd

数据库示例：

```sql
CREATE DATABASE hero3 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'hero3_user'@'127.0.0.1' IDENTIFIED BY '替换为强密码';
GRANT ALL PRIVILEGES ON hero3.* TO 'hero3_user'@'127.0.0.1';
FLUSH PRIVILEGES;
```

环境变量文件示例：

```bash
sudo mkdir -p /etc/hero3
sudo nano /etc/hero3/hero3.env
```

内容参考：

```env
HERO3_ENV=production
HERO3_PORT=8081
HERO3_VERSION=0.1.0
HERO3_LOG_LEVEL=info
HERO3_ALLOWED_ORIGINS=https://hero3.ccoos.cn,http://hero3.ccoos.cn
HERO3_DATABASE_DSN=hero3_user:替换为强密码@tcp(127.0.0.1:3306)/hero3?parseTime=true&charset=utf8mb4&loc=UTC
HERO3_BALANCE_PATH=/opt/hero3/go/config/balance.json
HERO3_FACTIONS_PATH=/opt/hero3/go/config/factions.json
HERO3_UNITS_DIR=/opt/hero3/go/config/units
```

## 本地构建（Windows PowerShell）

当前开发机是 Windows、服务器是 Ubuntu，因此 Go 后端必须构建为 Linux 可执行文件，不能直接上传默认的 Windows 构建产物。

在本地仓库根目录执行：

```powershell
Set-Location D:\Document\Game\Hero3
git pull origin main
```

构建 Linux 后端：

```powershell
Push-Location .\go
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -o bin/hero3-server ./cmd/server
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
Pop-Location
```

构建玩家前端和 GM 后台：

```powershell
Push-Location .\web
pnpm install
pnpm build
Pop-Location

Push-Location .\admin
pnpm install
pnpm build
Pop-Location
```

## 发布到服务器（Windows PowerShell）

先设置服务器和本次发版时间戳：

```powershell
$SERVER = "root@124.223.111.163"
$STAMP = Get-Date -Format "yyyyMMddHHmmss"
```

先在服务器上备份旧版本：

```powershell
ssh $SERVER "mkdir -p /opt/hero3/backups /var/www/hero3/backups; cp -a /opt/hero3/go/bin/hero3-server /opt/hero3/backups/hero3-server.$STAMP; tar -czf /var/www/hero3/backups/web.$STAMP.tar.gz -C /var/www/hero3 web; tar -czf /var/www/hero3/backups/admin.$STAMP.tar.gz -C /var/www/hero3 admin"
```

上传临时发布文件：

```powershell
ssh $SERVER "mkdir -p /tmp/hero3-web.$STAMP /tmp/hero3-admin.$STAMP"
scp .\go\bin\hero3-server "${SERVER}:/tmp/hero3-server.$STAMP"
scp -r .\go\config "${SERVER}:/opt/hero3/go/"
scp -r .\web\dist\* "${SERVER}:/tmp/hero3-web.$STAMP/"
scp -r .\admin\dist\* "${SERVER}:/tmp/hero3-admin.$STAMP/"
```

替换后端文件、覆盖静态资源并重启服务：

```powershell
ssh $SERVER "chmod +x /tmp/hero3-server.$STAMP; mv -f /tmp/hero3-server.$STAMP /opt/hero3/go/bin/hero3-server; cp -a /tmp/hero3-web.$STAMP/. /var/www/hero3/web/; cp -a /tmp/hero3-admin.$STAMP/. /var/www/hero3/admin/; rm -rf /tmp/hero3-web.$STAMP /tmp/hero3-admin.$STAMP; systemctl restart hero3; nginx -t && systemctl reload nginx"
```

静态资源使用覆盖发布，不在上线过程中清空目录，避免用户刷新时遇到空白站点。Vite 生成的旧 hash 资源会保留，可在确认版本稳定且已有备份后定期清理。

## 验证发布

检查后端服务：

```bash
ssh root@124.223.111.163 'systemctl status hero3 --no-pager'
```

检查端口：

```bash
ssh root@124.223.111.163 'ss -lntp | grep -E "8081|80|443|3306"'
```

检查健康接口：

```bash
curl https://hero3.ccoos.cn/healthz
```

检查页面：

```bash
curl -I https://hero3.ccoos.cn/
curl -I https://hero3.ccoos.cn/admin/
```

预期：

- `/healthz` 返回 `status:"ok"`
- 玩家站返回 `200`
- GM 后台返回 `200`

## 常用运维命令

查看服务状态：

```bash
systemctl status hero3 --no-pager
```

查看日志：

```bash
journalctl -u hero3 -f
```

重启后端：

```bash
systemctl restart hero3
```

重载 Nginx：

```bash
nginx -t && systemctl reload nginx
```

查看 Nginx 访问日志：

```bash
tail -f /var/log/nginx/hero3.access.log
```

查看 Nginx 错误日志：

```bash
tail -f /var/log/nginx/hero3.error.log
```

## 回滚

回滚后端：

```bash
ls -lh /opt/hero3/backups/
cp /opt/hero3/backups/hero3-server.时间戳 /opt/hero3/go/bin/hero3-server
chmod +x /opt/hero3/go/bin/hero3-server
systemctl restart hero3
```

回滚玩家前端：

```bash
ls -lh /var/www/hero3/backups/
rm -rf /var/www/hero3/web
tar -xzf /var/www/hero3/backups/web.时间戳.tar.gz -C /var/www/hero3
```

回滚 GM 后台：

```bash
rm -rf /var/www/hero3/admin
tar -xzf /var/www/hero3/backups/admin.时间戳.tar.gz -C /var/www/hero3
```

## 安全注意

- GM 后台 `/admin/` 当前通过公网域名可访问。后续上线封号、补资源、删档、发公告等功能后，建议加 Nginx Basic Auth、IP 白名单或单独后台域名。
- 不要把 `/etc/hero3/hero3.env`、数据库密码、SSH 密码、证书私钥提交到 Git。
- MySQL 如果没有必要，不建议长期开放公网 `3306`。
- 每次发版前先确认本地 `git status` 干净，避免把半成品部署上去。

## 排查清单

如果页面打不开：

```bash
systemctl status nginx --no-pager
nginx -t
tail -n 100 /var/log/nginx/hero3.error.log
```

如果 API 不通：

```bash
systemctl status hero3 --no-pager
journalctl -u hero3 -n 100 --no-pager
curl http://127.0.0.1:8081/healthz
```

如果数据库异常：

```bash
systemctl status mysql --no-pager
mysql -u hero3_user -p -h 127.0.0.1 hero3
```

如果前端页面 200 但功能异常：

- 检查浏览器控制台是否有 `/api/` 请求失败。
- 检查 Nginx 是否仍把 `/api/` 反代到 `127.0.0.1:8081`。
- 检查 `HERO3_ALLOWED_ORIGINS` 是否包含当前访问域名。
