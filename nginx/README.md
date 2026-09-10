# Nginx

默认通过宿主机 `80` 端口访问应用，匹配 `/newsconns/` 前缀的请求保留完整路径和查询参数并转发到 `app:8080`。例如 `/newsconns/newapis/anon_lgn` 转发到 `http://app:8080/newsconns/newapis/anon_lgn`，不会移除接口前缀。根路径 `/` 返回 `/usr/share/nginx/html/index.html` 静态首页，`/nginx-health` 返回 200 和 `ok`，这两个响应仅表示 Nginx 可用，不代表后端应用已就绪。其余请求从该静态目录读取文件，不存在的文件返回 404。

静态目录使用 `location /`，以便首页内部跳转到 `/index.html` 后仍使用相同的 `root`。仅在 `location = /` 中设置 `root` 会导致跳转后的请求使用其他配置中的根目录。

本地入口为 `http://localhost/`；应用健康检查为 `http://localhost/newsconns/newapis/sys_ping`；开发模式下的 Swagger 文档为 `http://localhost/newsconns/v1/documents/`。

启动：

```sh
docker compose up -d --build
docker compose exec nginx nginx -t
```

`/nginx-health` 仅检查 nginx 是否可用，不代表应用或数据库已就绪。nginx 在应用容器启动后启动，应用尚未就绪时可能暂时返回 502。

启用 HTTPS：

1. 将域名证书完整链保存为 `nginx/ssl/fullchain.pem`，私钥保存为 `nginx/ssl/privkey.pem`（该目录内证书文件已被 Git 忽略）。
2. 修改 `conf.d/default.conf` 中的 `server_name` 为实际域名，并取消 TLS 配置行的注释。
3. 取消 `docker-compose.yml` 中 nginx 的 `443:443` 端口映射注释。
4. 执行 `docker compose up -d nginx`。此配置同时保留 HTTP 和 HTTPS 访问。

修改配置或更新证书后，检查并重新加载：

```sh
docker compose exec nginx nginx -t
docker compose exec nginx nginx -s reload
```
