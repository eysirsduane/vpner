# Just VPN Origin

`just-vpn-origin` 是所有新 VPN 产品的母版。它保留完整的用户、会员、支付、线路、通知、弹窗、统计、节点调度和内部查询能力，但不包含任何产品 ID、域名、产品映射、线上数据库或线上密钥。

## 原始客户端协议

- 原始 API 前缀固定为 `/api/v1`
- 原始请求和响应字段使用代码中的语义名称，例如 `device_no`、`vip_time`、`result`
- 默认请求头为 `X-Platform`、`X-Version`、`X-Build`、`X-Device-No`、`X-Client-Time`、`X-Time-Zone`
- Swagger 仅在 `RunMode = dev` 时开放，地址为 `/api/v1/docs/`
- 原始 Swagger 和完整字段说明见 [docs/API.md](docs/API.md)

`conf/mapping.json` 保持原始路由和字段的一一对应关系，默认不做任何前后缀或协议改写。它不是产品映射文件，不能直接修改成某个产品的混淆协议。

## 创建新产品

1. 复制本目录为 `just-vpn-id<产品ID>`，并初始化一个新的 Git 仓库
2. 更新 `conf/app.ini` 与 `conf/app.local.ini` 中的 `Product`、数据库名、JWT 密钥及运行环境
3. 复制 `conf/mapping.json` 为 `conf/mapping-id<产品ID>.json`，在新文件内配置路由、请求头、请求字段、响应字段及前后缀
4. 将 `MappingFile` 指向新产品映射文件，并按产品创建规范更新 Swagger 路径、转移码前缀、日活产品编码和 Docker Compose 容器名
5. 生成并核对映射后的 Swagger 与前端接口文档，再部署到独立服务器

母版中的业务控制器、模型和路由均以原始 API 为准。产品差异应优先放进映射 JSON，避免直接改业务字段或控制器。

## 本地运行

本地依赖由 Docker Compose 提供：

```sh
docker compose up -d
cp conf/app.local.example.ini conf/app.local.ini
go run .
```

默认端口：

- API: `8080`
- MySQL: `13306`
- Redis: `16379`

容器内应用使用 `conf/app.ini`，数据库地址为 `mysql:3306`。本机直接运行 Go 服务时，使用 `conf/app.local.ini` 覆盖为 `127.0.0.1:13306` 和 `127.0.0.1:16379`。

## 目录

```text
conf/       运行配置、IP 库和原始映射模板
controller/ HTTP 接口业务
middleware/ 鉴权、客户端信息和跨域
model/      GORM 模型、缓存与数据库逻辑
pkg/        JWT、Redis、支付、映射和 Swagger 工具
router/     原始 API 路由
task/       定时任务
docs/       原始 API 文档与 Swagger 生成文件
```

## 配置安全

仓库中的 `conf/app.ini` 和 Compose 密码仅用于本地母版运行。新产品部署前必须替换 JWT、MySQL、Redis、支付和内部接口密钥，生产配置不要写入 Git。
