# Nginx

默认通过宿主机 `80` 端口访问应用，匹配 `/relayjet/` 前缀的请求保留完整路径和查询参数并转发到 `app:8080`。例如 `/relayjet/v1/common/autolog` 转发到 `http://app:8080/relayjet/v1/common/autolog`，不会移除接口前缀。根路径 `/` 返回 200 和 `API is running`，`/nginx-health` 返回 200 和 `ok`，这两个响应仅表示 Nginx 可用，不代表后端应用已就绪。其他路径返回 404。

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
