# Just VPN Origin 原始接口文档

本文档为 Just VPN 母版的原始前端对接文档，所有路由和字段均未做产品映射。

## 公共说明

### 基础地址

本地测试：

```text
http://127.0.0.1:8080
```

原始接口统一前缀：

```text
/api/v1
```

所有客户端请求和响应字段均使用本文件列出的原始字段名。支付运营商回调使用独立路径和运营商字段，客户端无需调用。

### 响应格式

所有接口统一返回：

```json
{
  "code": 200,
  "msg": "success",
  "result": {}
}
```

字段说明：

| 字段       | 类型         | 说明                                                                     |
| ---------- | ------------ | ------------------------------------------------------------------------ |
| code | number       | `200` 表示成功，`901` 表示未开通会员或会员已过期，`500` 表示其他业务失败 |
| msg    | string       | 响应消息                                                                 |
| result    | object/array | 响应数据                                                                 |

### 请求头

除公开接口外，都需要携带登录 token：

```http
Authorization: Bearer <token>
```

推荐所有接口都携带客户端信息：

| Header            | 必填         | 说明                        | 示例         |
| ----------------- | ------------ | --------------------------- | ------------ |
| Authorization     | 鉴权接口必填 | 登录 token                  | `Bearer xxx` |
| X-Platform | 建议必填     | 平台，支持 `iphone/android` | `iphone`     |
| X-Version     | 建议必填     | 展示版本号                  | `1.0.0`      |
| X-Build        | 版本检测必填 | 数字版本号                  | `100`        |
| X-Device-No      | 建议必填     | 设备唯一标识                | `device-001` |
| X-Client-Time       | 建议必填     | 客户端本地 RFC3339 时间（含偏移量） | `2026-07-11T20:30:15.123+08:00` |
| X-Time-Zone      | 建议必填     | 客户端 IANA 时区名            | `Asia/Shanghai` |

### 用户字段

用户信息结构：

```json
{
  "id": 1,
  "token": "jwt-token",
  "device_no": "device-001",
  "username": "",
  "type": 1,
  "is_vip": 0,
  "vip_time": "",
  "is_new_user": 1,
  "platform": "iphone",
  "version": "1.0.0",
  "login_times": 1,
  "create_time": "2026-07-06 10:00:00",
  "last_login_time": "2026-07-06 10:00:00",
  "password": "pass001"
}
```

字段说明：

| 字段         | 说明                                         |
| ------------ | -------------------------------------------- |
| id     | 用户 ID                                      |
| token  | JWT 登录 token                               |
| device_no    | 设备唯一标识                                 |
| username  | 账号名，游客为空                             |
| type         | `1=游客`，`2=账号用户`                       |
| is_vip    | `1=会员有效`，`0=非会员`                     |
| vip_time  | 会员到期时间，格式 `yyyy-MM-dd HH:mm:ss`，非会员为空 |
| is_new_user | 是否新用户，`1=首次创建的新用户`，`0=老用户` |
| platform | 客户端平台，例如 `iphone` 或 `android` |
| version | 客户端版本号 |
| login_times   | 登录次数                                     |
| create_time | 用户创建时间，格式 `yyyy-MM-dd HH:mm:ss` |
| last_login_time | 本次登录时间，格式 `yyyy-MM-dd HH:mm:ss` |
| password | 绑定账号的明文密码，游客为空字符串 |

## 公开接口

### 健康检查

```http
GET /api/v1/health
```

响应示例：

```json
{
  "code": 200,
  "msg": "success",
  "result": {}
}
```

### 获取静态资源

```http
GET /api/v1/upload/{filepath}
```

无需登录。后端会从服务运行目录的 `upload` 文件夹读取文件，适合放图片等静态资源。

示例：

```text
/api/v1/upload/banner.png
/api/v1/upload/ad/home.png
```

说明：

| 项目     | 说明                                              |
| -------- | ------------------------------------------------- |
| filepath | `upload` 目录下的相对路径                         |
| 返回内容 | 直接返回文件内容，Content-Type 由文件类型自动识别 |
| 不存在   | 返回 HTTP 404                                     |

### 游客登录

```http
POST /api/v1/auto_login
```

请求体：

```json
{
  "device_no": "device-001",
  "platform": "iphone",
  "version": "1.0.0",
  "mobile_name": "iPhone",
  "mobile_version": "iOS 17.5",
  "mobile_model_name": "iPhone 15",
  "source_channel": "default"
}
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "id": 1,
    "token": "jwt-token",
    "device_no": "device-001",
    "username": "",
    "type": 1,
    "is_vip": 0,
    "vip_time": "",
    "is_new_user": 1,
    "platform": "iphone",
    "version": "1.0.0",
    "login_times": 1,
    "create_time": "2026-07-06 10:00:00",
    "last_login_time": "2026-07-06 10:00:00",
    "password": ""
  }
}
```

### 注册账号

```http
POST /api/v1/register
```

需要先调用游客登录拿到 `token`，并在请求头携带：

```http
Authorization: Bearer <token>
```

请求体：

```json
{
  "device_no": "device-001",
  "username": "user001",
  "password": "pass001"
}
```

账号和密码规则：6 到 20 位字母或数字。

响应 `result` 为用户信息。

### 账号登录

```http
POST /api/v1/login
```

需要先调用游客登录拿到当前设备的 `token`，并在请求头携带：

```http
Authorization: Bearer <token>
```

请求体：

```json
{
  "device_no": "device-001",
  "username": "user001",
  "password": "pass001",
  "platform": "iphone",
  "version": "1.0.0",
  "mobile_name": "iPhone",
  "mobile_version": "iOS 17.5",
  "mobile_model_name": "iPhone 15",
  "source_channel": "default"
}
```

响应 `result` 为用户信息。

## 用户接口

### 退出登录

```http
POST /api/v1/logout
```

当前设备退出账号，设备恢复为游客状态，账号会员保留。

响应 `result` 为用户信息。

### 账号注销

```http
POST /api/v1/logoff
```

账号注销接口。后端通过配置表 `account.real_logoff_versions` 控制真实注销版本，配置值为英文逗号分隔的版本号。

如果当前请求头 `X-Version` 命中该配置：

- 当前设备未登录账号时，删除当前设备用户
- 当前设备已登录账号时，删除账号表记录，并删除该账号已登录的所有设备用户
- 删除后旧 token 立即失效，当前设备会直接创建一个新的游客用户，不赠送新用户会员
- 接口会直接返回新游客用户的登录 token，客户端保存该 token 后即可继续调用鉴权接口，无需再次自动登录

真实注销成功响应只返回新 token：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "token": "new-jwt-token"
  }
}
```

如果版本未命中配置，则返回成功和空对象，不删除数据，当前 token 继续有效。

### 修改密码

```http
POST /api/v1/change_password
```

请求体：

```json
{
  "old_password": "pass001",
  "new_password": "pass002"
}
```

### 获取设备列表

```http
POST /api/v1/device_info
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "local_device": {
      "id": 1,
      "device_no": "device-001",
      "mobile_name": "本机设备",
      "mobile_version": "iOS 17.5",
      "mobile_model_name": "iPhone 15",
      "platform": "iphone",
      "version": "1.0.0",
      "last_login_time": "2026-07-06 10:00:00"
    },
    "two_device": {},
    "devices": []
  }
}
```

### 移除设备

```http
POST /api/v1/device_logout
```

请求体：

```json
{
  "id": 2
}
```

不能移除当前设备。

### 获取邀请码

```http
POST /api/v1/draw_invite_code
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "invite_code": "A8K29QXZ"
  }
}
```

### 填写邀请码

```http
POST /api/v1/invite_code
```

请求体：

```json
{
  "invite_code": "A8K29QXZ"
}
```

### 邀请详情

```http
POST /api/v1/invite_detail
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "invite_code": "A8K29QXZ",
    "invite_valid": true,
    "share_count": 3,
    "reward_time": 10800,
    "per_invite_seconds": 3600,
    "per_invite_text": "1小时会员",
    "milestones": [
      {
        "count": 24,
        "reward_seconds": 2592000,
        "reward_text": "1个月会员"
      }
    ]
  }
}
```

### 用户信息

```http
GET /api/v1/user_info
```

响应 `result` 为用户信息。

## 系统接口

### 初始化配置

```http
POST /api/v1/init
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "agreements": {
      "user_agreement": "",
      "privacy_agreement": ""
    },
    "share": {
      "qrcode": "",
      "links": []
    },
    "invite": {
      "valid_hours": 72,
      "per_invite_seconds": 3600,
      "per_invite_text": "1小时会员",
      "max_reward_days": 300,
      "milestones": [
        {
          "count": 24,
          "reward_seconds": 2592000,
          "reward_text": "1个月会员"
        }
      ]
    },
    "app": {
      "website": "",
      "new_user_free_seconds": 0
    },
    "tools": {
      "customer_service_enabled": "off",
      "customer_service_url": "",
      "clean_memory_enabled": "off",
      "message_enabled": "on"
    },
    "proxy": {
      "skip_domains": []
    }
  }
}
```

`tools` 字段说明：

| 字段           | 说明                                                                                             |
| -------------- | ------------------------------------------------------------------------------------------------ |
| customer_service_enabled | 在线客服入口开关，`on=展示`，`off=隐藏`                                                          |
| customer_service_url     | 在线客服地址，仅 `customer_service_enabled=on` 时使用；后端会把配置模板中的 `#ID` 替换为当前用户 ID 后返回 |
| clean_memory_enabled | 清理内存入口开关，`on=展示`，`off=隐藏`                                                          |
| message_enabled | 消息通知入口开关，`on=展示`，`off=隐藏`                                                          |

### 版本检测

```http
POST /api/v1/version
```

必填请求头：

```http
X-Platform: iphone
X-Version: 1.0.0
X-Build: 100
```

无更新：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "has_update": false
  }
}
```

有更新：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "has_update": true,
    "name": "iOS 1.0.2",
    "platform": "iphone",
    "version": "1.0.2",
    "build": 102,
    "force": false,
    "url": "https://example.com/app",
    "size": "23.5MB",
    "content": "修复已知问题"
  }
}
```

### 广告位列表

```http
GET /api/v1/advert
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": [
    {
      "id": 1,
      "position": "banner",
      "title": "会员优惠",
      "content": "限时开通会员享优惠",
      "image_url": "https://example.com/advert.png",
      "link_url": "https://example.com",
      "show_times": 1,
      "max_show_times": 3,
      "sorter": 100
    }
  ]
}
```

`position` 固定只有三个值：

| position      | 含义         |
| ------------ | ------------ |
| `splash`     | 启动页广告   |
| `home_popup` | 首页弹窗广告 |
| `banner`     | banner 广告  |

### 全量通知

```http
GET /api/v1/notice
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "system": {
      "list": [
        {
          "id": 1,
          "title": "系统维护通知",
          "content": "今晚维护",
          "is_read": 0,
          "create_time": "2026-07-06 10:00:00"
        }
      ],
      "total": 1,
      "unread_count": 1
    },
    "user": {
      "list": [],
      "total": 0,
      "unread_count": 0
    }
  }
}
```

### 未读通知

```http
GET /api/v1/notice/unread
```

响应结构同全量通知，只返回未读消息。

### 获取弹窗

```http
GET /api/v1/popup
```

无弹窗时：

```json
{
  "code": 200,
  "msg": "success",
  "result": {}
}
```

有弹窗时：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "id": 1,
    "title": "会员活动",
    "content": "限时开通会员享优惠",
    "image_url": [
      "https://example.com/popup.png"
    ],
    "jump_type": "internal",
    "jump_target": "purchase",
    "can_close": 1,
    "show_times": 1,
    "max_show_times": 3
  }
}
```

后台配置 `image_url` 可填写单个 URL、英文逗号分隔的多个 URL，或 JSON 字符串数组，接口始终返回 JSON URL 数组。`can_close` 取值：`1=可关闭`、`0=不可关闭`。

`jump_type` 取值：

| 值       | 说明       |
| -------- | ---------- |
| none     | 不跳转     |
| internal | App 内跳转 |
| external | 浏览器打开 |

### 延迟迁移弹窗

```http
GET /api/v1/delayed_popup
```

客户端成功请求后保存到本地，当连续断网达到 `delay_days` 天后展示给用户，用于引导用户转移到新软件。

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "id": 1,
    "title": "服务迁移提醒",
    "content": "如果当前软件长时间无法连接，请使用转移码前往新软件兑换会员权益",
    "image_url": [
      "/api/v1/upload/pop.png"
    ],
    "link_url": "https://example.com/download",
    "can_close": 1,
    "delay_days": 3,
    "transfer_code": "origin8f3k9q"
  }
}
```

延迟弹窗的 `image_url` 规则与普通弹窗一致，始终返回 JSON URL 数组。`can_close` 取值：`1=可关闭`、`0=不可关闭`。

无可用配置时：

```json
{
  "code": 200,
  "msg": "success",
  "result": {}
}
```

### 标记通知已读

```http
POST /api/v1/read_notice
```

请求体：

```json
{
  "system_ids": [1, 2],
  "user_ids": [3, 4]
}
```

## 上报接口

### 错误上报

```http
POST /api/v1/error
```

请求体：

```json
{
  "msg": "vpn connect timeout",
  "mobile_version": "iOS 17.5",
  "mobile_model_name": "iPhone 15",
  "msg_type": "vpn"
}
```

`msg_type` 可选：

| 值  | 说明     |
| --- | -------- |
| app | App 错误 |
| vpn | VPN 错误 |

## 套餐接口

### 套餐列表

```http
GET /api/v1/packages
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": [
    {
      "id": 1,
      "apple_id": "vip_month",
      "name": "月度会员",
      "sub_name": "连续30天高速线路",
      "selected": 1,
      "corner": "推荐",
      "price": 1990,
      "price_text": "¥19.9",
      "origin_price": "¥29.9",
      "value": "畅享全部会员线路",
      "remark": "适合短期使用",
      "day": 30
    }
  ]
}
```

## 支付接口

### 获取公共 Apple ID

```http
GET /api/v1/public_apple_id
```

无需登录。用于获取可用的公共 Apple ID 列表，每个 IP 24 小时内返回同一组账号。

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": [
    {
      "id": 1,
      "appid": "apple@example.com",
      "pwd": "apple-password"
    }
  ]
}
```

字段说明：

| 字段          | 说明          |
| ------------- | ------------- |
| id      | 记录 ID       |
| appid  | Apple ID 账号 |
| pwd | Apple ID 密码 |

### 发起支付

```http
POST /api/v1/pay/launch
```

请求体：

```json
{
  "package_id": 1
}
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "pay_type": "apple_iap",
    "target": "com.lamp.app.premium.7days",
    "app_account_token": "C56A4180-65AA-42EC-A945-5FD21DEC0538"
  }
}
```

或：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "pay_type": "h5",
    "target": "https://pay.example.com/order/xxx",
    "app_account_token": ""
  }
}
```

字段说明：

| 字段            | 说明                                                                                                      |
| --------------- | --------------------------------------------------------------------------------------------------------- |
| pay_type      | 支付方式，`apple_iap=苹果内购`，`h5=三方 H5 支付`                                                         |
| target       | `apple_iap` 时返回 Apple 商品 ID，`h5` 时返回浏览器跳转支付地址                                           |
| app_account_token | `apple_iap` 时返回该订单对应的 UUID，前端调起 StoreKit 时必须原样作为 `appAccountToken` 携带；`h5` 时为空 |

苹果内购注意事项：

| 事项      | 说明                                                                           |
| --------- | ------------------------------------------------------------------------------ |
| 商品 ID   | 使用 `target` 调起内购                                                      |
| 订单 UUID | 使用 `app_account_token` 调起内购，必须传给 StoreKit 的 `appAccountToken`        |
| 验单      | 支付完成后调用 `POST /api/v1/pay/apple_verify`，只提交 `transaction_id` |
| 匹配订单  | 后端通过苹果交易中的 `appAccountToken` 找到本次发起支付创建的订单              |

判断规则：

| 规则        | 说明                                                                    |
| ----------- | ----------------------------------------------------------------------- |
| 客户端时区  | 请求头 `X-Time-Zone` 不是中国大陆时区或未传时，直接返回 `apple_iap`；该规则优先级最高 |
| 仅内购地区  | 用户当前 IP 地区命中配置时返回 `apple_iap`                              |
| 仅内购版本  | 请求头 `X-Version` 命中配置时返回 `apple_iap`                       |
| H5 开放金额 | 今日苹果内购收款额度未达到配置金额时返回 `apple_iap`，达到后可返回 `h5` |
| 默认        | 不命中以上限制时返回 `h5`                                               |

中国大陆时区包括 `Asia/Shanghai`、`Asia/Chongqing`、`Asia/Harbin`、`Asia/Urumqi` 等中国大陆 IANA 时区；`Asia/Hong_Kong`、`Asia/Macau`、`Asia/Taipei` 等不属于中国大陆时区。

当支付配置开启“已成功三方支付用户直接 H5”后，中国大陆时区内已存在成功三方支付订单的用户会直接返回 `h5`，该规则优先于地区、版本和今日苹果内购金额限制。

### 苹果订单验证

```http
POST /api/v1/pay/apple_verify
```

客户端完成苹果内购后提交苹果交易号。后端会通过苹果交易信息中的商品 ID 和 `appAccountToken` 匹配发起支付时创建的订单，验证成功后发放会员权益。

请求体：

```json
{
  "transaction_id": "1000000000000000"
}
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "order_no": "20260707120000123456",
    "pay_status": 3,
    "pay_type": "apple_iap",
    "transaction_id": "1000000000000000",
    "vip_time": "2026-08-06 12:00:00"
  }
}
```

字段说明：

| 字段             | 说明                             |
| ---------------- | -------------------------------- |
| order_no       | 后端订单号                       |
| pay_status     | 支付状态，`1=未支付`，`3=已支付` |
| pay_type       | 支付方式                         |
| transaction_id | 苹果交易号                       |
| vip_time      | 会员到期时间                     |

## VPN 接口

### 线路区域列表

```http
POST /api/v1/lines_list
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": [
    {
      "country": "香港",
      "code": "HK",
      "min_conn_time": 100,
      "max_conn_time": 300,
      "img_url": "https://example.com/hk.png"
    }
  ]
}
```

### 获取节点

```http
POST /api/v1/node
```

请求体：

```json
{
  "code": "HK"
}
```

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "link_url": "x/k5A0v9kiJjL0r3m6X9dA=="
  }
}
```

说明：

| 规则     | 说明                                                                          |
| -------- | ----------------------------------------------------------------------------- |
| 节点链接 | `link_url` 是加密后的节点连接数据，所有节点类型都按字符串返回               |
| 解密流程 | 客户端先 AES-CBC/PKCS7 解密得到 base64 字符串，再 base64 解码得到原始节点内容 |
| 加密 key | 后端配置提供，前端使用项目约定的节点解密 key                                  |
| 免费线路 | `FREE` 或 `FREE_` 开头的线路不校验会员                                        |
| 会员线路 | 需要会员未过期，未开通会员或会员已过期时返回 `code=901`                 |
| 连接确认 | 成功拿到节点后，需要连接成功再调用在线接口                                    |

会员过期响应：

```json
{
  "code": 901,
  "msg": "未开通会员或会员已过期，请开通后使用",
  "result": {}
}
```

### 获取 JSON 节点配置

```http
POST /api/v1/node_config
```

请求体：

```json
{
  "code": "HK",
  "type": "fast"
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| code | 线路代码 |
| type | 连接模式，`fast` 或 `极速` 表示极速模式，`global` 或 `全局` 表示全局模式 |

响应：

```json
{
  "code": 200,
  "msg": "success",
  "result": {
    "config": "x/k5A0v9kiJjL0r3m6X9dA=="
  }
}
```

`config` 是完整 JSON 节点配置的加密结果，解密流程与 `link_url` 相同：先 AES-CBC/PKCS7 解密，再 base64 解码得到 JSON 配置。该接口与 `node` 使用相同的线路选择、会员校验和临时节点记录逻辑。

### 确认已连接

```http
POST /api/v1/connected
```

客户端成功建立 VPN 后调用。

### 心跳

```http
POST /api/v1/heartbeat
```

连接中定时调用，用于刷新在线状态。

客户端应每 30 秒调用一次。服务端在线状态有效期为 90 秒，90 秒内没有收到下一次心跳时，后续心跳会返回断开指令。

响应 `result.connect_status` 表示连接指令：`1` 继续连接，`0` 断开连接。会员过期或当前没有连接记录时会返回 `0`，客户端收到 `0` 后应立即断开 VPN。

### 断开连接

```http
POST /api/v1/disconnect
```

主动断开 VPN 时调用。

### 流量上报

```http
POST /api/v1/vpn_flow
```

请求体：

```json
{
  "flow": 1024
}
```

`flow` 单位为 K，必须大于 0。

## 推荐调用流程

### 启动流程

1. 调用 `POST /api/v1/auto_login`
2. 保存 `result.token`
3. 调用 `POST /api/v1/init`
4. 调用 `POST /api/v1/version`
5. 调用 `GET /api/v1/popup`
6. 调用 `GET /api/v1/advert`
7. 调用 `GET /api/v1/notice/unread`

### VPN 连接流程

1. 调用 `POST /api/v1/lines_list`
2. 用户选择线路
3. 根据客户端实现调用 `POST /api/v1/node` 或 `POST /api/v1/node_config`
4. 客户端建立 VPN 连接
5. 连接成功后调用 `POST /api/v1/connected`
6. 连接中定时调用 `POST /api/v1/heartbeat`
7. 断开时调用 `POST /api/v1/disconnect`
8. 适当时机调用 `POST /api/v1/vpn_flow`

### 账号流程

1. 游客进入调用 `POST /api/v1/auto_login`
2. 注册调用 `POST /api/v1/register`
3. 登录调用 `POST /api/v1/login`
4. 退出调用 `POST /api/v1/logout`
5. 注销入口调用 `POST /api/v1/logoff`
