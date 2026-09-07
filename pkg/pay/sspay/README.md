# SS 支付

协议文档：https://ssvip.biz/doc_old.html

服务启动时 `InitPayConfigs` 会补齐缺失配置，不覆盖已有值。在 `pay_config` 中设置：

| code | value |
| --- | --- |
| `third_pay_provider` | `sspay`（默认 `xxpay`） |
| `sspay.api_url` | `https://ssvip.biz`，不带 `/mapi.php` |
| `sspay.pid` | 商户 ID |
| `sspay.key` | 商户 MD5 密钥 |
| `sspay.notify_url` | 公网 GET 回调地址，路径 `/api/v1/pay/ss_callback`；如增加路由映射，使用映射后的地址 |
| `sspay.return_url` | 支付完成后跳转的页面地址 |
| `sspay.callback_ips` | 可选，回调 IP 白名单，英文逗号分隔 |

沿用现有苹果内购分流规则，仅在允许三方支付时选择上述渠道。SS 订单内部标记为 `sspay`，客户端响应仍为 `type=h5`，`target` 为支付地址。已支付 SS 订单也计入三方支付历史。

表单参数使用字符串；签名排除 `sign`、`sign_type` 和空字符串，按参数名排序，用未 URL 编码的值拼接 `k=v`，以 `&` 连接后直接追加密钥，计算小写 MD5。表单传输时仍正常 URL 编码；回调验签保留解码后的原始值。

下单返回 `payurl` 时直接使用；返回 `urlscheme` 时保留完整协议链接，客户端需允许对应支付应用跳转。返回 `qrcode` 时，使用相同商户订单号构造平台 `submit.php` 页面支付地址，由平台展示收银台，避免将二维码内容当成 HTTP 地址打开。此回退需要在商户联调时验证平台对同一商户订单号的处理。

本地测试使用模拟 HTTP 服务，不会创建真实支付订单。正式启用前需使用商户配置验证实际下单、回调及客户端跳转。
