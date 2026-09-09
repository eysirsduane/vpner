# RelayJet API 文档

本文档依据 conf/mapping.json 生成，所有路径、请求字段和响应字段均为 RelayJet 的有效映射名称。

## 公共说明

### 基础地址

本地测试：

```text
http://127.0.0.1:8080
```

RelayJet 接口统一前缀：

```text
/relayjet/jetapi
```

所有客户端请求和响应字段均使用本文件列出的 RelayJet 映射字段名。支付运营商回调使用独立路径和运营商字段，客户端无需调用。

### 响应格式

所有接口统一返回：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {

                       }
}
```

字段说明：

| 字段       | 类型         | 说明                                                                     |
| ---------- | ------------ | ------------------------------------------------------------------------ |
| rely_field_rnumber_fly_suff | number       | `200` 表示成功，`401` 表示登录状态失效，`901` 表示未开通会员或会员已过期，`500` 表示其他业务失败 |
| rely_field_rmessage_fly_suff    | string       | 响应消息                                                                 |
| rely_field_res_data_fly_suff    | object/array | 响应数据                                                                 |

<!-- client-status-codes:start -->
### 客户端业务状态码与处理

所有由客户端业务接口正常返回的响应，HTTP 状态码都固定为 `200`。即使 JSON 中的业务状态码是 `401`、`500` 或 `901`，HTTP 状态码仍然是 `200`。
前端必须读取响应 JSON 中的 `rely_field_rnumber_fly_suff` 判断业务结果，不能只判断 HTTP 状态码。

域名无法访问、Nginx 或网关错误、接口路径不存在、请求超时等未进入正常业务响应的情况，HTTP 状态码可能不是 `200`，应按网络错误处理。

| 业务状态码 | 含义 | 前端处理方式 |
| --- | --- | --- |
| `200` | 请求成功 | 正常处理响应数据。 |
| `401` | 登录状态失效 | 清除旧 Token，重新调用自动登录接口 `/relayjet/jetapi/gst_acc`，并保存返回的 `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff`。 |
| `500` | 系统或业务错误 | 直接展示 `rely_field_rmessage_fly_suff` 返回的错误信息，不要重新自动登录。 |
| `901` | 用户没有有效 VIP | 只会在获取节点相关接口中返回；提示用户开通或续费 VIP，不要刷新 Token。 |

#### 401 处理

Token 过期、Token 无效、用户已被删除或设备不匹配等需要重新登录的情况统一返回业务码 `401`。

前端收到 `401` 后：

1. 清除本地旧 Token。
2. 重新调用 `POST /relayjet/jetapi/gst_acc`。
3. 保存自动登录响应中的 `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff`。

#### 500 处理

前端直接展示 `rely_field_rmessage_fly_suff` 中的错误信息，不需要重新自动登录。

#### 901 处理

`901` 仅用于以下获取节点接口，表示当前用户没有有效 VIP：

- `POST /relayjet/jetapi/tun_ent`
- `POST /relayjet/jetapi/tun_egr`
- `POST /relayjet/jetapi/tun_opts`

前端应提示用户开通或续费 VIP，不需要重新自动登录。
<!-- client-status-codes:end -->

### 请求头

除公开接口外，都需要携带登录 token：

```http
Authorization: Bearer <token>
```

推荐所有接口都携带客户端信息：

| Header            | 必填         | 说明                        | 示例         |
| ----------------- | ------------ | --------------------------- | ------------ |
| Authorization     | 鉴权接口必填 | 登录 token                  | `Bearer xxx` |
| RelayJet-Plat-Name | 建议必填     | 平台，支持 `iphone/android` | `iphone`     |
| RelayJet-Version-Number     | 建议必填     | 展示版本号                  | `1.0.0`      |
| RelayJet-Build-Version        | 版本检测必填 | 数字版本号                  | `100`        |
| RelayJet-Device-Name      | 建议必填     | 设备唯一标识                | `device-001` |
| RelayJet-Time       | 建议必填     | 客户端本地 RFC3339 时间（含偏移量） | `2026-07-11T20:30:15.123+08:00` |
| RelayJet-TimeZone      | 建议必填     | 客户端 IANA 时区名            | `Asia/Shanghai` |
| RelayJet-Stores-Reg | 可选 | 苹果 App Store 地区 | `CHN` |
| RelayJet-Language | 可选 | 当前用户语言；任意 `zh`、`zh-*` 或 `zh_*`（含繁体标识）均显示简体中文，其他非空语言显示英文 | `zh-Hans` |

多语言响应规则：

- 不携带 `RelayJet-Language` 时默认显示简体中文，兼容旧客户端。
- 套餐、弹窗、延迟弹窗、广告、通知、版本内容和线路国家名称在数据库字段为 `{"zh-Hans":"中文","en":"English"}` 时，按本次请求头选择语言。
- 数据库字段仍是普通字符串时原样返回，不会因为英文请求而改变旧内容。
- 双语 JSON 缺少目标语言或目标语言为空时，回退到另一种已有语言。
- 业务错误提示、VIP 提示、邀请奖励文字和本机设备名称等后端固定文字也按本次请求头返回简体中文或英文。

### 用户字段

用户信息结构：

```json
{
  "rely_field_rec_id_fly_suff": 1,
  "rely_field_acc_key_fly_suff": "jwt-token",
  "rely_field_dev_id_fly_suff": "device-001",
  "rely_field_usr_nm_fly_suff": "",
  "rely_field_cat_cd_fly_suff": 1,
  "rely_field_vip_flg_fly_suff": 0,
  "rely_field_vip_exp_tm_fly_suff": "",
  "rely_field_new_usr_flg_fly_suff": 1,
  "rely_field_cli_os_fly_suff": "iphone",
  "rely_field_rel_ver_fly_suff": "1.0.0",
  "rely_field_auth_cnt_fly_suff": 1,
  "rely_field_crt_tm_fly_suff": "2026-07-06 10:00:00",
  "rely_field_last_auth_tm_fly_suff": "2026-07-06 10:00:00",
  "rely_field_pwd_txt_fly_suff": "pass001"
}
```

字段说明：

| 字段         | 说明                                         |
| ------------ | -------------------------------------------- |
| rely_field_rec_id_fly_suff     | 用户 ID                                      |
| rely_field_acc_key_fly_suff  | JWT 登录 token                               |
| rely_field_dev_id_fly_suff    | 设备唯一标识                                 |
| rely_field_usr_nm_fly_suff  | 账号名，游客为空                             |
| rely_field_cat_cd_fly_suff         | `1=游客`，`2=账号用户`                       |
| rely_field_vip_flg_fly_suff    | `1=会员有效`，`0=非会员`                     |
| rely_field_vip_exp_tm_fly_suff  | 会员到期时间，格式 `yyyy-MM-dd HH:mm:ss`，非会员为空 |
| rely_field_new_usr_flg_fly_suff | 是否新用户，`1=首次创建的新用户`，`0=老用户` |
| rely_field_cli_os_fly_suff | 客户端平台，例如 `iphone` 或 `android` |
| rely_field_rel_ver_fly_suff | 客户端版本号 |
| rely_field_auth_cnt_fly_suff   | 登录次数                                     |
| rely_field_crt_tm_fly_suff | 用户创建时间，格式 `yyyy-MM-dd HH:mm:ss` |
| rely_field_last_auth_tm_fly_suff | 本次登录时间，格式 `yyyy-MM-dd HH:mm:ss` |
| rely_field_pwd_txt_fly_suff | 绑定账号的明文密码，游客为空字符串 |

## 公开接口

### 健康检查

```http
GET /relayjet/jetapi/svc_pls
```

响应示例：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {

                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/svc_pls:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/svc_pls:end -->

### 获取静态资源

```http
GET /relayjet/jetapi/med_sto/{filepath}
```

无需登录。后端会从服务运行目录的 `upload` 文件夹读取文件，适合放图片等静态资源。

示例：

```text
/relayjet/jetapi/med_sto/banner.png
/relayjet/jetapi/med_sto/ad/home.png
```

说明：

| 项目     | 说明                                              |
| -------- | ------------------------------------------------- |
| filepath | `upload` 目录下的相对路径                         |
| 返回内容 | 直接返回文件内容，Content-Type 由文件类型自动识别 |
| 不存在   | 返回 HTTP 404                                     |

<!-- endpoint-fields:/relayjet/jetapi/med_sto/*filepath:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `filepath` | `string` | 是 | 资源相对路径 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `HTTP Body` | `binary` | 成功时 | 直接返回文件内容，不使用统一 JSON 响应结构。 |
<!-- endpoint-fields:/relayjet/jetapi/med_sto/*filepath:end -->

### 游客登录

```http
POST /relayjet/jetapi/gst_acc
```

请求体：

```json
{
  "rely_field_dev_id_fly_suff": "device-001",
  "rely_field_cli_os_fly_suff": "iphone",
  "rely_field_rel_ver_fly_suff": "1.0.0",
  "rely_field_mob_nm_fly_suff": "iPhone",
  "rely_field_os_ver_fly_suff": "iOS 17.5",
  "rely_field_mdl_nm_fly_suff": "iPhone 15",
  "rely_field_src_chn_fly_suff": "default"
}
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_rec_id_fly_suff":  1,
                           "rely_field_acc_key_fly_suff":  "jwt-token",
                           "rely_field_dev_id_fly_suff":  "device-001",
                           "rely_field_usr_nm_fly_suff":  "",
                           "rely_field_cat_cd_fly_suff":  1,
                           "rely_field_vip_flg_fly_suff":  0,
                           "rely_field_vip_exp_tm_fly_suff":  "",
                           "rely_field_new_usr_flg_fly_suff":  1,
                           "rely_field_cli_os_fly_suff":  "iphone",
                           "rely_field_rel_ver_fly_suff":  "1.0.0",
                           "rely_field_auth_cnt_fly_suff":  1,
                           "rely_field_crt_tm_fly_suff":  "2026-07-06 10:00:00",
                           "rely_field_last_auth_tm_fly_suff":  "2026-07-06 10:00:00",
                           "rely_field_pwd_txt_fly_suff":  ""
                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/gst_acc:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_dev_id_fly_suff` | `string` | 是 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_cli_os_fly_suff` | `string` | 否 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_rel_ver_fly_suff` | `string` | 否 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_mob_nm_fly_suff` | `string` | 否 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_os_ver_fly_suff` | `string` | 否 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_mdl_nm_fly_suff` | `string` | 否 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_src_chn_fly_suff` | `string` | 否 | 客户端安装或获客来源渠道；未区分时可传 `default`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 用户ID |
| `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` | `string` | 成功时 | JWT登录凭证 |
| `rely_field_res_data_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 设备唯一标识 |
| `rely_field_res_data_fly_suff.rely_field_usr_nm_fly_suff` | `string` | 成功时 | 账号名，游客为空 |
| `rely_field_res_data_fly_suff.rely_field_cat_cd_fly_suff` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `rely_field_res_data_fly_suff.rely_field_vip_flg_fly_suff` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `rely_field_res_data_fly_suff.rely_field_new_usr_flg_fly_suff` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `rely_field_res_data_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台 |
| `rely_field_res_data_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端版本号 |
| `rely_field_res_data_fly_suff.rely_field_auth_cnt_fly_suff` | `integer` | 成功时 | 登录次数 |
| `rely_field_res_data_fly_suff.rely_field_crt_tm_fly_suff` | `string` | 成功时 | 用户创建时间 |
| `rely_field_res_data_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 本次登录时间 |
| `rely_field_res_data_fly_suff.rely_field_pwd_txt_fly_suff` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/relayjet/jetapi/gst_acc:end -->

### 注册账号

```http
POST /relayjet/jetapi/acct_enr
```

需要先调用游客登录拿到 `rely_field_acc_key_fly_suff`，并在请求头携带：

```http
Authorization: Bearer <token>
```

请求体：

```json
{
  "rely_field_dev_id_fly_suff": "device-001",
  "rely_field_usr_nm_fly_suff": "user001",
  "rely_field_pwd_txt_fly_suff": "pass001"
}
```

账号和密码规则：6 到 20 位字母或数字。

响应 `rely_field_res_data_fly_suff` 为用户信息。

<!-- endpoint-fields:/relayjet/jetapi/acct_enr:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_dev_id_fly_suff` | `string` | 是 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_usr_nm_fly_suff` | `string` | 是 | 绑定账号名，规则为 6–20 位字母或数字。 |
| `rely_field_pwd_txt_fly_suff` | `string` | 是 | 账号密码，规则为 6–20 位字母或数字。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 用户ID |
| `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` | `string` | 成功时 | JWT登录凭证 |
| `rely_field_res_data_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 设备唯一标识 |
| `rely_field_res_data_fly_suff.rely_field_usr_nm_fly_suff` | `string` | 成功时 | 账号名，游客为空 |
| `rely_field_res_data_fly_suff.rely_field_cat_cd_fly_suff` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `rely_field_res_data_fly_suff.rely_field_vip_flg_fly_suff` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `rely_field_res_data_fly_suff.rely_field_new_usr_flg_fly_suff` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `rely_field_res_data_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台 |
| `rely_field_res_data_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端版本号 |
| `rely_field_res_data_fly_suff.rely_field_auth_cnt_fly_suff` | `integer` | 成功时 | 登录次数 |
| `rely_field_res_data_fly_suff.rely_field_crt_tm_fly_suff` | `string` | 成功时 | 用户创建时间 |
| `rely_field_res_data_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 本次登录时间 |
| `rely_field_res_data_fly_suff.rely_field_pwd_txt_fly_suff` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/relayjet/jetapi/acct_enr:end -->

### 账号登录

```http
POST /relayjet/jetapi/mbr_acc
```

需要先调用游客登录拿到当前设备的 `rely_field_acc_key_fly_suff`，并在请求头携带：

```http
Authorization: Bearer <token>
```

请求体：

```json
{
  "rely_field_dev_id_fly_suff": "device-001",
  "rely_field_usr_nm_fly_suff": "user001",
  "rely_field_pwd_txt_fly_suff": "pass001",
  "rely_field_cli_os_fly_suff": "iphone",
  "rely_field_rel_ver_fly_suff": "1.0.0",
  "rely_field_mob_nm_fly_suff": "iPhone",
  "rely_field_os_ver_fly_suff": "iOS 17.5",
  "rely_field_mdl_nm_fly_suff": "iPhone 15",
  "rely_field_src_chn_fly_suff": "default"
}
```

响应 `rely_field_res_data_fly_suff` 为用户信息。

<!-- endpoint-fields:/relayjet/jetapi/mbr_acc:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_dev_id_fly_suff` | `string` | 是 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_usr_nm_fly_suff` | `string` | 是 | 绑定账号名，规则为 6–20 位字母或数字。 |
| `rely_field_pwd_txt_fly_suff` | `string` | 是 | 账号密码，规则为 6–20 位字母或数字。 |
| `rely_field_cli_os_fly_suff` | `string` | 否 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_rel_ver_fly_suff` | `string` | 否 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_mob_nm_fly_suff` | `string` | 否 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_os_ver_fly_suff` | `string` | 否 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_mdl_nm_fly_suff` | `string` | 否 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_src_chn_fly_suff` | `string` | 否 | 客户端安装或获客来源渠道；未区分时可传 `default`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 用户ID |
| `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` | `string` | 成功时 | JWT登录凭证 |
| `rely_field_res_data_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 设备唯一标识 |
| `rely_field_res_data_fly_suff.rely_field_usr_nm_fly_suff` | `string` | 成功时 | 账号名，游客为空 |
| `rely_field_res_data_fly_suff.rely_field_cat_cd_fly_suff` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `rely_field_res_data_fly_suff.rely_field_vip_flg_fly_suff` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `rely_field_res_data_fly_suff.rely_field_new_usr_flg_fly_suff` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `rely_field_res_data_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台 |
| `rely_field_res_data_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端版本号 |
| `rely_field_res_data_fly_suff.rely_field_auth_cnt_fly_suff` | `integer` | 成功时 | 登录次数 |
| `rely_field_res_data_fly_suff.rely_field_crt_tm_fly_suff` | `string` | 成功时 | 用户创建时间 |
| `rely_field_res_data_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 本次登录时间 |
| `rely_field_res_data_fly_suff.rely_field_pwd_txt_fly_suff` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/relayjet/jetapi/mbr_acc:end -->

## 用户接口

### 退出登录

```http
POST /relayjet/jetapi/sess_lve
```

当前设备退出账号，设备恢复为游客状态，账号会员保留。

响应 `rely_field_res_data_fly_suff` 为用户信息。

<!-- endpoint-fields:/relayjet/jetapi/sess_lve:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 用户ID |
| `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` | `string` | 成功时 | JWT登录凭证 |
| `rely_field_res_data_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 设备唯一标识 |
| `rely_field_res_data_fly_suff.rely_field_usr_nm_fly_suff` | `string` | 成功时 | 账号名，游客为空 |
| `rely_field_res_data_fly_suff.rely_field_cat_cd_fly_suff` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `rely_field_res_data_fly_suff.rely_field_vip_flg_fly_suff` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `rely_field_res_data_fly_suff.rely_field_new_usr_flg_fly_suff` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `rely_field_res_data_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台 |
| `rely_field_res_data_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端版本号 |
| `rely_field_res_data_fly_suff.rely_field_auth_cnt_fly_suff` | `integer` | 成功时 | 登录次数 |
| `rely_field_res_data_fly_suff.rely_field_crt_tm_fly_suff` | `string` | 成功时 | 用户创建时间 |
| `rely_field_res_data_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 本次登录时间 |
| `rely_field_res_data_fly_suff.rely_field_pwd_txt_fly_suff` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/relayjet/jetapi/sess_lve:end -->

### 账号注销

```http
POST /relayjet/jetapi/acct_ret
```

账号注销接口。后端通过配置表 `account.real_logoff_versions` 控制真实注销版本，配置值为英文逗号分隔的版本号。

如果当前请求头 `RelayJet-Version-Number` 命中该配置：

- 当前设备未登录账号时，删除当前设备用户
- 当前设备已登录账号时，删除账号表记录，并删除该账号已登录的所有设备用户
- 删除后旧 token 立即失效，当前设备会直接创建一个新的游客用户，不赠送新用户会员
- 接口会直接返回新游客用户的登录 token，客户端保存该 token 后即可继续调用鉴权接口，无需再次自动登录

真实注销成功响应只返回新 token：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_acc_key_fly_suff":  "new-jwt-token"
                       }
}
```

如果版本未命中配置，则返回成功和空对象，不删除数据，当前 token 继续有效。

<!-- endpoint-fields:/relayjet/jetapi/acct_ret:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` | `string` | 成功时 | JWT 登录凭证，后续鉴权接口通过 Bearer Token 携带。 |
<!-- endpoint-fields:/relayjet/jetapi/acct_ret:end -->

### 修改密码

```http
POST /relayjet/jetapi/sec_rnw
```

请求体：

```json
{
  "rely_field_new_pwd_fly_suff": "pass002"
}
```

当前登录账号无需提交旧密码；新密码不能与当前密码相同。

<!-- endpoint-fields:/relayjet/jetapi/sec_rnw:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_new_pwd_fly_suff` | `string` | 是 | 需要设置的新密码，规则为 6–20 位字母或数字。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/sec_rnw:end -->

### 获取设备列表

```http
POST /relayjet/jetapi/dev_prof
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_loc_dev_fly_suff":  {
                                                 "rely_field_rec_id_fly_suff":  1,
                                                 "rely_field_dev_id_fly_suff":  "device-001",
                                                 "rely_field_mob_nm_fly_suff":  "本机设备",
                                                 "rely_field_dev_ver_fly_suff":  "iOS 17.5",
                                                 "rely_field_mdl_nm_fly_suff":  "iPhone 15",
                                                 "rely_field_cli_os_fly_suff":  "iphone",
                                                 "rely_field_rel_ver_fly_suff":  "1.0.0",
                                                 "rely_field_last_auth_tm_fly_suff":  "2026-07-06 10:00:00"
                                             },
                           "rely_field_pair_dev_fly_suff":  {

                                              },
                           "rely_field_dev_lst_fly_suff":  {

                                              }
                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/dev_prof:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff` | `object` | 成功时 | 当前发起请求的本机设备信息。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 设备记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_mob_nm_fly_suff` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_dev_ver_fly_suff` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_mdl_nm_fly_suff` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff` | `object` | 成功时 | 账号绑定的另一台设备；不存在时返回空对象。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 设备记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_mob_nm_fly_suff` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_dev_ver_fly_suff` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_mdl_nm_fly_suff` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff` | `array<object>` | 成功时 | 当前账号绑定的设备列表。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 设备记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_dev_id_fly_suff` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_mob_nm_fly_suff` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_dev_ver_fly_suff` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_mdl_nm_fly_suff` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
<!-- endpoint-fields:/relayjet/jetapi/dev_prof:end -->

### 移除设备

```http
POST /relayjet/jetapi/dev_rel
```

请求体：

```json
{
  "rely_field_rec_id_fly_suff": 2
}
```

不能移除当前设备。

<!-- endpoint-fields:/relayjet/jetapi/dev_rel:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rec_id_fly_suff` | `integer` | 是 | 设备记录 ID。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff` | `object` | 成功时 | 当前发起请求的本机设备信息。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 设备记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_mob_nm_fly_suff` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_dev_ver_fly_suff` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_mdl_nm_fly_suff` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_res_data_fly_suff.rely_field_loc_dev_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff` | `object` | 成功时 | 账号绑定的另一台设备；不存在时返回空对象。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 设备记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_mob_nm_fly_suff` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_dev_ver_fly_suff` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_mdl_nm_fly_suff` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_res_data_fly_suff.rely_field_pair_dev_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff` | `array<object>` | 成功时 | 当前账号绑定的设备列表。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 设备记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_dev_id_fly_suff` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_mob_nm_fly_suff` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_dev_ver_fly_suff` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_mdl_nm_fly_suff` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `rely_field_res_data_fly_suff.rely_field_dev_lst_fly_suff[].rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
<!-- endpoint-fields:/relayjet/jetapi/dev_rel:end -->

### 获取邀请码

```http
POST /relayjet/jetapi/ref_iss
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_ref_cd_fly_suff":  "A8K29QXZ"
                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/ref_iss:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_ref_cd_fly_suff` | `string` | 成功时 | 当前用户自己的长期有效邀请码；不存在时由后端自动生成。 |
<!-- endpoint-fields:/relayjet/jetapi/ref_iss:end -->

### 填写邀请码

```http
POST /relayjet/jetapi/ref_apl
```

请求体：

```json
{
  "rely_field_ref_cd_fly_suff": "A8K29QXZ"
}
```

<!-- endpoint-fields:/relayjet/jetapi/ref_apl:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_ref_cd_fly_suff` | `string` | 是 | 需要绑定的其他用户邀请码；后端会转为大写后校验。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/ref_apl:end -->

### 邀请详情

```http
POST /relayjet/jetapi/ref_dtl
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_ref_cd_fly_suff":  "A8K29QXZ",
                           "rely_field_inv_valid_flg_fly_suff":  true,
                           "rely_field_shr_cnt_fly_suff":  3,
                           "rely_field_bns_tm_fly_suff":  10800,
                           "rely_field_inv_bns_sec_fly_suff":  3600,
                           "rely_field_inv_bns_txt_fly_suff":  "1小时会员",
                           "rely_field_stage_lst_fly_suff":  {
                                                  "rely_field_hit_cnt_fly_suff":  24,
                                                  "rely_field_bns_sec_fly_suff":  2592000,
                                                  "rely_field_bns_txt_fly_suff":  "1个月会员"
                                              }
                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/ref_dtl:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_ref_cd_fly_suff` | `string` | 成功时 | 当前用户自己的邀请码。 |
| `rely_field_res_data_fly_suff.rely_field_inv_valid_flg_fly_suff` | `boolean` | 成功时 | 当前用户是否仍可填写他人邀请码：未绑定邀请人且仍在允许填写的注册时间范围内时为 `true`。 |
| `rely_field_res_data_fly_suff.rely_field_shr_cnt_fly_suff` | `integer` | 成功时 | 用户当前累计成功邀请人数。 |
| `rely_field_res_data_fly_suff.rely_field_bns_tm_fly_suff` | `integer` | 成功时 | 用户累计获得的邀请奖励时长，单位为秒。 |
| `rely_field_res_data_fly_suff.rely_field_inv_bns_sec_fly_suff` | `integer` | 成功时 | 每成功邀请一名用户奖励的会员秒数。 |
| `rely_field_res_data_fly_suff.rely_field_inv_bns_txt_fly_suff` | `string` | 成功时 | 每次邀请奖励时长的展示文案。 |
| `rely_field_res_data_fly_suff.rely_field_stage_lst_fly_suff` | `array<object>` | 成功时 | 邀请人数阶梯奖励列表。 |
| `rely_field_res_data_fly_suff.rely_field_stage_lst_fly_suff[].rely_field_hit_cnt_fly_suff` | `integer` | 成功时 | 获得本阶梯奖励所需的累计成功邀请人数。 |
| `rely_field_res_data_fly_suff.rely_field_stage_lst_fly_suff[].rely_field_bns_sec_fly_suff` | `integer` | 成功时 | 达到当前条件后奖励的会员秒数。 |
| `rely_field_res_data_fly_suff.rely_field_stage_lst_fly_suff[].rely_field_bns_txt_fly_suff` | `string` | 成功时 | 当前奖励时长的展示文案。 |
<!-- endpoint-fields:/relayjet/jetapi/ref_dtl:end -->

### 用户信息

```http
GET /relayjet/jetapi/mbr_prof
```

响应 `rely_field_res_data_fly_suff` 为用户信息。

<!-- endpoint-fields:/relayjet/jetapi/mbr_prof:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 用户ID |
| `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` | `string` | 成功时 | JWT登录凭证 |
| `rely_field_res_data_fly_suff.rely_field_dev_id_fly_suff` | `string` | 成功时 | 设备唯一标识 |
| `rely_field_res_data_fly_suff.rely_field_usr_nm_fly_suff` | `string` | 成功时 | 账号名，游客为空 |
| `rely_field_res_data_fly_suff.rely_field_cat_cd_fly_suff` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `rely_field_res_data_fly_suff.rely_field_vip_flg_fly_suff` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `rely_field_res_data_fly_suff.rely_field_new_usr_flg_fly_suff` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `rely_field_res_data_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 客户端平台 |
| `rely_field_res_data_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 客户端版本号 |
| `rely_field_res_data_fly_suff.rely_field_auth_cnt_fly_suff` | `integer` | 成功时 | 登录次数 |
| `rely_field_res_data_fly_suff.rely_field_crt_tm_fly_suff` | `string` | 成功时 | 用户创建时间 |
| `rely_field_res_data_fly_suff.rely_field_last_auth_tm_fly_suff` | `string` | 成功时 | 本次登录时间 |
| `rely_field_res_data_fly_suff.rely_field_pwd_txt_fly_suff` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/relayjet/jetapi/mbr_prof:end -->

## 系统接口

### 初始化配置

```http
POST /relayjet/jetapi/stp_opts
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_terms_set_fly_suff":  {
                                                "rely_field_usr_terms_fly_suff":  "",
                                                "rely_field_priv_terms_fly_suff":  ""
                                            },
                           "rely_field_shr_meta_fly_suff":  {
                                                     "rely_field_qr_img_fly_suff":  "",
                                                     "rely_field_url_lst_fly_suff":  [

                                                                        ]
                                                 },
                           "rely_field_inv_meta_fly_suff":  {
                                                   "rely_field_valid_hr_fly_suff":  72,
                                                   "rely_field_inv_bns_sec_fly_suff":  3600,
                                                   "rely_field_inv_bns_txt_fly_suff":  "1小时会员",
                                                   "rely_field_max_bns_day_fly_suff":  300,
                                                   "rely_field_stage_lst_fly_suff":  {
                                                                          "rely_field_hit_cnt_fly_suff":  24,
                                                                          "rely_field_bns_sec_fly_suff":  2592000,
                                                                          "rely_field_bns_txt_fly_suff":  "1个月会员"
                                                                      }
                                               },
                           "rely_field_app_meta_fly_suff":  {
                                                 "rely_field_web_uri_fly_suff":  "",
                                                 "rely_field_new_free_sec_fly_suff":  0
                                             },
                           "rely_field_util_set_fly_suff":  {
                                                    "rely_field_svc_en_fly_suff":  "off",
                                                    "rely_field_svc_uri_fly_suff":  "",
                                                    "rely_field_mem_clr_flg_fly_suff":  "off",
                                                    "rely_field_msg_en_fly_suff":  "on"
                                                },
                           "rely_field_prx_meta_fly_suff":  {
                                                    "rely_field_skip_doms_fly_suff":  [

                                                                             ]
                                                }
                       }
}
```

`rely_field_util_set_fly_suff` 字段说明：

| 字段           | 说明                                                                                             |
| -------------- | ------------------------------------------------------------------------------------------------ |
| `rely_field_svc_en_fly_suff` | 在线客服入口开关，`on=展示`，`off=隐藏`                                                          |
| `rely_field_svc_uri_fly_suff`     | 在线客服地址，仅 `rely_field_svc_en_fly_suff=on` 时使用；后端会把配置模板中的 `#ID` 替换为当前用户 ID 后返回 |
| `rely_field_mem_clr_flg_fly_suff` | 清理内存入口开关，`on=展示`，`off=隐藏`                                                          |
| `rely_field_msg_en_fly_suff` | 消息通知入口开关，`on=展示`，`off=隐藏`                                                          |

<!-- endpoint-fields:/relayjet/jetapi/stp_opts:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 响应状态码 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 响应消息 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 初始化配置 |
| `rely_field_res_data_fly_suff.rely_field_terms_set_fly_suff` | `object` | 成功时 | 协议配置 |
| `rely_field_res_data_fly_suff.rely_field_terms_set_fly_suff.rely_field_usr_terms_fly_suff` | `string` | 成功时 | 用户服务协议地址 |
| `rely_field_res_data_fly_suff.rely_field_terms_set_fly_suff.rely_field_priv_terms_fly_suff` | `string` | 成功时 | 隐私协议地址 |
| `rely_field_res_data_fly_suff.rely_field_shr_meta_fly_suff` | `object` | 成功时 | 分享配置 |
| `rely_field_res_data_fly_suff.rely_field_shr_meta_fly_suff.rely_field_qr_img_fly_suff` | `string` | 成功时 | 分享二维码图片地址 |
| `rely_field_res_data_fly_suff.rely_field_shr_meta_fly_suff.rely_field_url_lst_fly_suff` | `array<string>` | 成功时 | 分享链接列表 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff` | `object` | 成功时 | 邀请配置 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_valid_hr_fly_suff` | `integer` | 成功时 | 新用户注册后可填写邀请码的小时数 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_inv_bns_sec_fly_suff` | `integer` | 成功时 | 每邀请一个用户奖励的会员秒数 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_inv_bns_txt_fly_suff` | `string` | 成功时 | 每邀请一个用户奖励的会员时长文案 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_max_bns_day_fly_suff` | `integer` | 成功时 | 邀请累计最大奖励天数 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_stage_lst_fly_suff` | `array<object>` | 成功时 | 邀请人数阶梯奖励列表 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_stage_lst_fly_suff[].rely_field_hit_cnt_fly_suff` | `integer` | 成功时 | 达到邀请人数 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_stage_lst_fly_suff[].rely_field_bns_sec_fly_suff` | `integer` | 成功时 | 额外奖励会员秒数 |
| `rely_field_res_data_fly_suff.rely_field_inv_meta_fly_suff.rely_field_stage_lst_fly_suff[].rely_field_bns_txt_fly_suff` | `string` | 成功时 | 额外奖励会员时长文案 |
| `rely_field_res_data_fly_suff.rely_field_app_meta_fly_suff` | `object` | 成功时 | 应用配置 |
| `rely_field_res_data_fly_suff.rely_field_app_meta_fly_suff.rely_field_web_uri_fly_suff` | `string` | 成功时 | 官网地址 |
| `rely_field_res_data_fly_suff.rely_field_app_meta_fly_suff.rely_field_new_free_sec_fly_suff` | `integer` | 成功时 | 新用户默认赠送会员秒数 |
| `rely_field_res_data_fly_suff.rely_field_util_set_fly_suff` | `object` | 成功时 | 工具入口配置 |
| `rely_field_res_data_fly_suff.rely_field_util_set_fly_suff.rely_field_svc_en_fly_suff` | `string` | 成功时 | 在线客服入口开关(on=开启,off=关闭) |
| `rely_field_res_data_fly_suff.rely_field_util_set_fly_suff.rely_field_svc_uri_fly_suff` | `string` | 成功时 | 在线客服地址，配置模板中的#ID会替换为当前用户ID |
| `rely_field_res_data_fly_suff.rely_field_util_set_fly_suff.rely_field_mem_clr_flg_fly_suff` | `string` | 成功时 | 清理内存入口开关(on=开启,off=关闭) |
| `rely_field_res_data_fly_suff.rely_field_util_set_fly_suff.rely_field_msg_en_fly_suff` | `string` | 成功时 | 消息通知入口开关(on=开启,off=关闭) |
| `rely_field_res_data_fly_suff.rely_field_prx_meta_fly_suff` | `object` | 成功时 | 代理配置 |
| `rely_field_res_data_fly_suff.rely_field_prx_meta_fly_suff.rely_field_skip_doms_fly_suff` | `array<string>` | 成功时 | 不走代理的域名列表 |
<!-- endpoint-fields:/relayjet/jetapi/stp_opts:end -->

### 版本检测

```http
POST /relayjet/jetapi/ed_chk
```

必填请求头：

```http
RelayJet-Plat-Name: iphone
RelayJet-Version-Number: 1.0.0
RelayJet-Build-Version: 100
```

无更新：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_upd_flg_fly_suff":  false
                       }
}
```

有更新：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_upd_flg_fly_suff":  true,
                           "rely_field_disp_nm_fly_suff":  "iOS 1.0.2",
                           "rely_field_cli_os_fly_suff":  "iphone",
                           "rely_field_rel_ver_fly_suff":  "1.0.2",
                           "rely_field_bld_rev_fly_suff":  102,
                           "rely_field_req_flg_fly_suff":  false,
                           "rely_field_dl_uri_fly_suff":  "https://example.com/app",
                           "rely_field_file_sz_fly_suff":  "23.5MB",
                           "rely_field_body_txt_fly_suff":  "修复已知问题"
                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/ed_chk:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 响应状态码 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 响应消息 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 版本检测结果 |
| `rely_field_res_data_fly_suff.rely_field_upd_flg_fly_suff` | `boolean` | 成功时 | 是否有新版本 |
| `rely_field_res_data_fly_suff.rely_field_disp_nm_fly_suff` | `string` | 成功时 | 版本名称 |
| `rely_field_res_data_fly_suff.rely_field_cli_os_fly_suff` | `string` | 成功时 | 设备平台(iphone=苹果,android=安卓) |
| `rely_field_res_data_fly_suff.rely_field_rel_ver_fly_suff` | `string` | 成功时 | 展示版本号 |
| `rely_field_res_data_fly_suff.rely_field_bld_rev_fly_suff` | `integer` | 成功时 | 数字版本号 |
| `rely_field_res_data_fly_suff.rely_field_req_flg_fly_suff` | `boolean` | 成功时 | 是否强制更新 |
| `rely_field_res_data_fly_suff.rely_field_dl_uri_fly_suff` | `string` | 成功时 | 下载地址 |
| `rely_field_res_data_fly_suff.rely_field_file_sz_fly_suff` | `string` | 成功时 | 安装包大小 |
| `rely_field_res_data_fly_suff.rely_field_body_txt_fly_suff` | `string` | 成功时 | 更新内容 |
<!-- endpoint-fields:/relayjet/jetapi/ed_chk:end -->

### 广告位列表

```http
GET /relayjet/jetapi/spn_shw
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_rec_id_fly_suff":  1,
                           "rely_field_slot_pos_fly_suff":  "banner",
                           "rely_field_head_txt_fly_suff":  "会员优惠",
                           "rely_field_body_txt_fly_suff":  "限时开通会员享优惠",
                           "rely_field_img_uri_fly_suff":  "https://example.com/advert.png",
                           "rely_field_lnk_uri_fly_suff":  "https://example.com",
                           "rely_field_disp_cnt_fly_suff":  1,
                           "rely_field_max_disp_cnt_fly_suff":  3,
                           "rely_field_sort_idx_fly_suff":  100
                       }
}
```

`rely_field_slot_pos_fly_suff` 固定只有三个值：

| `rely_field_slot_pos_fly_suff`      | 含义         |
| ------------ | ------------ |
| `splash`     | 启动页广告   |
| `home_popup` | 首页弹窗广告 |
| `banner`     | banner 广告  |

<!-- endpoint-fields:/relayjet/jetapi/spn_shw:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 广告ID |
| `rely_field_res_data_fly_suff[].rely_field_slot_pos_fly_suff` | `string` | 成功时 | 广告位标识(splash=启动页广告,home_popup=首页弹窗广告,banner=banner广告) |
| `rely_field_res_data_fly_suff[].rely_field_head_txt_fly_suff` | `string` | 成功时 | 广告标题 |
| `rely_field_res_data_fly_suff[].rely_field_body_txt_fly_suff` | `string` | 成功时 | 广告内容 |
| `rely_field_res_data_fly_suff[].rely_field_img_uri_fly_suff` | `string` | 成功时 | 广告图片地址 |
| `rely_field_res_data_fly_suff[].rely_field_lnk_uri_fly_suff` | `string` | 成功时 | 广告点击后使用浏览器打开的链接地址 |
| `rely_field_res_data_fly_suff[].rely_field_disp_cnt_fly_suff` | `integer` | 成功时 | 当前用户已展示次数 |
| `rely_field_res_data_fly_suff[].rely_field_max_disp_cnt_fly_suff` | `integer` | 成功时 | 每个用户最大展示次数，0表示不限次数 |
| `rely_field_res_data_fly_suff[].rely_field_sort_idx_fly_suff` | `integer` | 成功时 | 排序值，越大越优先 |
<!-- endpoint-fields:/relayjet/jetapi/spn_shw:end -->

### 全量通知

```http
GET /relayjet/jetapi/msg_brd
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_sys_box_fly_suff":  {
                                                 "rely_field_item_lst_fly_suff":  {
                                                                        "rely_field_rec_id_fly_suff":  1,
                                                                        "rely_field_head_txt_fly_suff":  "系统维护通知",
                                                                        "rely_field_body_txt_fly_suff":  "今晚维护",
                                                                        "rely_field_read_flg_fly_suff":  0,
                                                                        "rely_field_crt_tm_fly_suff":  "2026-07-06 10:00:00"
                                                                    },
                                                 "rely_field_tot_cnt_fly_suff":  1,
                                                 "rely_field_unread_cnt_fly_suff":  1
                                             },
                           "rely_field_usr_box_fly_suff":  {
                                                 "rely_field_item_lst_fly_suff":  {

                                                                    },
                                                 "rely_field_tot_cnt_fly_suff":  0,
                                                 "rely_field_unread_cnt_fly_suff":  0
                                             }
                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/msg_brd:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff` | `object` | 成功时 | 系统通知分组。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 通知记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_head_txt_fly_suff` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_body_txt_fly_suff` | `string` | 成功时 | 当前记录的正文内容。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_read_flg_fly_suff` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_crt_tm_fly_suff` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_tot_cnt_fly_suff` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_unread_cnt_fly_suff` | `integer` | 成功时 | 当前未读通知总数。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff` | `object` | 成功时 | 当前用户的个人通知分组。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 通知记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_head_txt_fly_suff` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_body_txt_fly_suff` | `string` | 成功时 | 当前记录的正文内容。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_read_flg_fly_suff` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_crt_tm_fly_suff` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_tot_cnt_fly_suff` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_unread_cnt_fly_suff` | `integer` | 成功时 | 当前未读通知总数。 |
<!-- endpoint-fields:/relayjet/jetapi/msg_brd:end -->

### 未读通知

```http
GET /relayjet/jetapi/unrd_ntc
```

响应结构同全量通知，只返回未读消息。

<!-- endpoint-fields:/relayjet/jetapi/unrd_ntc:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff` | `object` | 成功时 | 系统通知分组。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 通知记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_head_txt_fly_suff` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_body_txt_fly_suff` | `string` | 成功时 | 当前记录的正文内容。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_read_flg_fly_suff` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_crt_tm_fly_suff` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_tot_cnt_fly_suff` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `rely_field_res_data_fly_suff.rely_field_sys_box_fly_suff.rely_field_unread_cnt_fly_suff` | `integer` | 成功时 | 当前未读通知总数。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff` | `object` | 成功时 | 当前用户的个人通知分组。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 通知记录 ID。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_head_txt_fly_suff` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_body_txt_fly_suff` | `string` | 成功时 | 当前记录的正文内容。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_read_flg_fly_suff` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_item_lst_fly_suff[].rely_field_crt_tm_fly_suff` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_tot_cnt_fly_suff` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `rely_field_res_data_fly_suff.rely_field_usr_box_fly_suff.rely_field_unread_cnt_fly_suff` | `integer` | 成功时 | 当前未读通知总数。 |
<!-- endpoint-fields:/relayjet/jetapi/unrd_ntc:end -->

### 获取弹窗

```http
GET /relayjet/jetapi/ovl_crd
```

无弹窗时：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {

                       }
}
```

有弹窗时：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_rec_id_fly_suff":  1,
                           "rely_field_head_txt_fly_suff":  "会员活动",
                           "rely_field_body_txt_fly_suff":  "限时开通会员享优惠",
                           "rely_field_img_uri_fly_suff":  [
                                                   "https://example.com/popup.png"
                                               ],
                           "rely_field_nav_cat_fly_suff":  "internal",
                           "rely_field_nav_dest_fly_suff":  "purchase",
                           "rely_field_close_flg_fly_suff":  1,
                           "rely_field_disp_cnt_fly_suff":  1,
                           "rely_field_max_disp_cnt_fly_suff":  3
                       }
}
```

后台配置 `rely_field_img_uri_fly_suff` 可填写单个 URL、英文逗号分隔的多个 URL，或 JSON 字符串数组，接口始终返回 JSON URL 数组。`rely_field_close_flg_fly_suff` 取值：`1=可关闭`、`0=不可关闭`。

`rely_field_nav_cat_fly_suff` 取值：

| 值       | 说明       |
| -------- | ---------- |
| none     | 不跳转     |
| internal | App 内跳转 |
| external | 浏览器打开 |

<!-- endpoint-fields:/relayjet/jetapi/ovl_crd:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 弹窗ID |
| `rely_field_res_data_fly_suff.rely_field_head_txt_fly_suff` | `string` | 成功时 | 弹窗标题 |
| `rely_field_res_data_fly_suff.rely_field_body_txt_fly_suff` | `string` | 成功时 | 弹窗内容 |
| `rely_field_res_data_fly_suff.rely_field_img_uri_fly_suff` | `array<string>` | 成功时 | 弹窗图片地址数组 |
| `rely_field_res_data_fly_suff.rely_field_nav_cat_fly_suff` | `string` | 成功时 | 跳转方式(none=不跳转,internal=内部跳转,external=外部浏览器) |
| `rely_field_res_data_fly_suff.rely_field_nav_dest_fly_suff` | `string` | 成功时 | 跳转目标，内部跳转填业务code，外部跳转填URL |
| `rely_field_res_data_fly_suff.rely_field_close_flg_fly_suff` | `integer` | 成功时 | 是否可关闭(0=不可关闭,1=可关闭) |
| `rely_field_res_data_fly_suff.rely_field_disp_cnt_fly_suff` | `integer` | 成功时 | 当前用户已展示次数 |
| `rely_field_res_data_fly_suff.rely_field_max_disp_cnt_fly_suff` | `integer` | 成功时 | 每个用户最大展示次数，0表示不限次数 |
<!-- endpoint-fields:/relayjet/jetapi/ovl_crd:end -->

### 延迟迁移弹窗

```http
GET /relayjet/jetapi/dfr_crd
```

客户端成功请求后保存到本地，当连续断网达到 `rely_field_wait_day_fly_suff` 天后展示给用户，用于引导用户转移到新软件。

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_rec_id_fly_suff":  1,
                           "rely_field_head_txt_fly_suff":  "服务迁移提醒",
                           "rely_field_body_txt_fly_suff":  "如果当前软件长时间无法连接，请使用转移码前往新软件兑换会员权益",
                           "rely_field_img_uri_fly_suff":  [
                                                   "/relayjet/jetapi/med_sto/pop.png"
                                               ],
                           "rely_field_lnk_uri_fly_suff":  "https://example.com/download",
                           "rely_field_close_flg_fly_suff":  1,
                           "rely_field_wait_day_fly_suff":  3,
                           "rely_field_xfer_cd_fly_suff":  "origin8f3k9q"
                       }
}
```

延迟弹窗的 `rely_field_img_uri_fly_suff` 规则与普通弹窗一致，始终返回 JSON URL 数组。`rely_field_close_flg_fly_suff` 取值：`1=可关闭`、`0=不可关闭`。

无可用配置时：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {

                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/dfr_crd:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff` | `integer` | 成功时 | 弹窗ID |
| `rely_field_res_data_fly_suff.rely_field_head_txt_fly_suff` | `string` | 成功时 | 弹窗标题 |
| `rely_field_res_data_fly_suff.rely_field_body_txt_fly_suff` | `string` | 成功时 | 弹窗内容 |
| `rely_field_res_data_fly_suff.rely_field_img_uri_fly_suff` | `array<string>` | 成功时 | 弹窗图片地址数组 |
| `rely_field_res_data_fly_suff.rely_field_lnk_uri_fly_suff` | `string` | 成功时 | 弹窗跳转链接 |
| `rely_field_res_data_fly_suff.rely_field_close_flg_fly_suff` | `integer` | 成功时 | 是否可关闭(0=不可关闭,1=可关闭) |
| `rely_field_res_data_fly_suff.rely_field_wait_day_fly_suff` | `integer` | 成功时 | 断网后延迟展示天数 |
| `rely_field_res_data_fly_suff.rely_field_xfer_cd_fly_suff` | `string` | 成功时 | 当前用户转移码 |
<!-- endpoint-fields:/relayjet/jetapi/dfr_crd:end -->

### 标记通知已读

```http
POST /relayjet/jetapi/ntc_ack
```

请求体：

```json
{
  "rely_field_sys_note_ids_fly_suff": [1, 2],
  "rely_field_usr_note_ids_fly_suff": [3, 4]
}
```

<!-- endpoint-fields:/relayjet/jetapi/ntc_ack:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_sys_note_ids_fly_suff` | `array<integer>` | 否 | 需要标记为已读的系统通知 ID 数组。 |
| `rely_field_usr_note_ids_fly_suff` | `array<integer>` | 否 | 需要标记为已读的个人通知 ID 数组。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/ntc_ack:end -->

## 上报接口

### 错误上报

```http
POST /relayjet/jetapi/iss_rpt
```

请求体：

```json
{
  "rely_field_msg_txt_fly_suff": "vpn connect timeout",
  "rely_field_os_ver_fly_suff": "iOS 17.5",
  "rely_field_mdl_nm_fly_suff": "iPhone 15",
  "rely_field_err_cat_fly_suff": "vpn"
}
```

`rely_field_err_cat_fly_suff` 可选：

| 值  | 说明     |
| --- | -------- |
| app | App 错误 |
| vpn | VPN 错误 |

<!-- endpoint-fields:/relayjet/jetapi/iss_rpt:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_msg_txt_fly_suff` | `string` | 是 | 客户端需要上报的错误详情，例如 VPN 连接超时或配置解析失败。 |
| `rely_field_os_ver_fly_suff` | `string` | 是 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `rely_field_mdl_nm_fly_suff` | `string` | 是 | 客户端设备型号，例如 `iPhone 15`。 |
| `rely_field_err_cat_fly_suff` | `string` | 否 | 错误分类：`rely_field_app_meta_fly_suff` 表示 App 错误，`vpn` 表示 VPN 错误；未传或空值时默认 `rely_field_app_meta_fly_suff`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/iss_rpt:end -->

## 套餐接口

### 套餐列表

```http
GET /relayjet/jetapi/ofr_cat
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_rec_id_fly_suff":  1,
                           "rely_field_apl_id_fly_suff":  "vip_month",
                           "rely_field_disp_nm_fly_suff":  "月度会员",
                           "rely_field_sub_nm_fly_suff":  "连续30天高速线路",
                           "rely_field_sel_flg_fly_suff":  1,
                           "rely_field_badge_pos_fly_suff":  "推荐",
                           "rely_field_pay_amt_fly_suff":  1990,
                           "rely_field_amt_txt_fly_suff":  "¥19.9",
                           "rely_field_orig_amt_fly_suff":  "¥29.9",
                           "rely_field_val_num_fly_suff":  "畅享全部会员线路",
                           "rely_field_note_txt_fly_suff":  "适合短期使用",
                           "rely_field_dur_day_fly_suff":  30
                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/ofr_cat:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 套餐ID |
| `rely_field_res_data_fly_suff[].rely_field_apl_id_fly_suff` | `string` | 成功时 | 苹果内购产品ID |
| `rely_field_res_data_fly_suff[].rely_field_disp_nm_fly_suff` | `string` | 成功时 | 套餐名称 |
| `rely_field_res_data_fly_suff[].rely_field_sub_nm_fly_suff` | `string` | 成功时 | 套餐副标题 |
| `rely_field_res_data_fly_suff[].rely_field_sel_flg_fly_suff` | `integer` | 成功时 | 是否默认选中(1=选中,0=未选中) |
| `rely_field_res_data_fly_suff[].rely_field_badge_pos_fly_suff` | `string` | 成功时 | 角标文案 |
| `rely_field_res_data_fly_suff[].rely_field_pay_amt_fly_suff` | `integer` | 成功时 | 真实价格，单位分 |
| `rely_field_res_data_fly_suff[].rely_field_amt_txt_fly_suff` | `string` | 成功时 | 展示价格文案 |
| `rely_field_res_data_fly_suff[].rely_field_orig_amt_fly_suff` | `string` | 成功时 | 原价文案 |
| `rely_field_res_data_fly_suff[].rely_field_val_num_fly_suff` | `string` | 成功时 | 底部描述内容 |
| `rely_field_res_data_fly_suff[].rely_field_note_txt_fly_suff` | `string` | 成功时 | 套餐描述 |
| `rely_field_res_data_fly_suff[].rely_field_dur_day_fly_suff` | `integer` | 成功时 | 套餐天数 |
<!-- endpoint-fields:/relayjet/jetapi/ofr_cat:end -->

## 支付接口

### 获取公共 Apple ID

```http
GET /relayjet/jetapi/shr_apl
```

无需登录。用于获取可用的公共 Apple ID 列表，每个 IP 24 小时内返回同一组账号。

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_rec_id_fly_suff":  1,
                           "rely_field_app_key_fly_suff":  "apple@example.com",
                           "rely_field_pwd_txt_fly_suff":  "apple-password"
                       }
}
```

字段说明：

| 字段          | 说明          |
| ------------- | ------------- |
| `rely_field_rec_id_fly_suff`      | 记录 ID       |
| `rely_field_app_key_fly_suff`  | Apple ID 账号 |
| `rely_field_pwd_txt_fly_suff` | Apple ID 密码 |

<!-- endpoint-fields:/relayjet/jetapi/shr_apl:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff[].rely_field_rec_id_fly_suff` | `integer` | 成功时 | 记录ID |
| `rely_field_res_data_fly_suff[].rely_field_app_key_fly_suff` | `string` | 成功时 | Apple ID账号 |
| `rely_field_res_data_fly_suff[].rely_field_pwd_txt_fly_suff` | `string` | 成功时 | Apple ID密码 |
<!-- endpoint-fields:/relayjet/jetapi/shr_apl:end -->

### 发起支付

```http
POST /relayjet/jetapi/chk_bgn
```

请求体：

```json
{
  "rely_field_pkg_id_fly_suff": 1
}
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_cat_cd_fly_suff":  "apple_iap",
                           "rely_field_dest_addr_fly_suff":  "com.lamp.app.premium.7days",
                           "rely_field_app_acct_tkn_fly_suff":  "C56A4180-65AA-42EC-A945-5FD21DEC0538"
                       }
}
```

或：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_cat_cd_fly_suff":  "h5",
                           "rely_field_dest_addr_fly_suff":  "https://pay.example.com/order/xxx",
                           "rely_field_app_acct_tkn_fly_suff":  ""
                       }
}
```

字段说明：

| 字段            | 说明                                                                                                      |
| --------------- | --------------------------------------------------------------------------------------------------------- |
| `rely_field_cat_cd_fly_suff`      | 支付方式，`apple_iap=苹果内购`，`h5=三方 H5 支付`                                                         |
| `rely_field_dest_addr_fly_suff`       | `apple_iap` 时返回 Apple 商品 ID，`h5` 时返回浏览器跳转支付地址                                           |
| `rely_field_app_acct_tkn_fly_suff` | `apple_iap` 时返回该订单对应的 UUID，前端调起 StoreKit 时必须原样作为 `appAccountToken` 携带；`h5` 时为空 |

苹果内购注意事项：

| 事项      | 说明                                                                           |
| --------- | ------------------------------------------------------------------------------ |
| 商品 ID   | 使用 `rely_field_dest_addr_fly_suff` 调起内购                                                      |
| 订单 UUID | 使用 `rely_field_app_acct_tkn_fly_suff` 调起内购，必须传给 StoreKit 的 `appAccountToken`        |
| 验单      | 支付完成后调用 `POST /relayjet/jetapi/apl_cfm`，只提交 `rely_field_txn_id_fly_suff` |
| 匹配订单  | 后端通过苹果交易中的 `appAccountToken` 找到本次发起支付创建的订单              |

判断规则：

| 规则        | 说明                                                                    |
| ----------- | ----------------------------------------------------------------------- |
| 客户端时区  | 请求头 `RelayJet-TimeZone` 不是中国大陆时区或未传时，直接返回 `apple_iap`；该规则优先级最高 |
| 仅内购地区  | 用户当前 IP 地区命中配置时返回 `apple_iap`                              |
| 仅内购版本  | 请求头 `RelayJet-Version-Number` 命中配置时返回 `apple_iap`                       |
| H5 开放金额 | 今日苹果内购收款额度未达到配置金额时返回 `apple_iap`，达到后可返回 `h5` |
| 默认        | 不命中以上限制时返回 `h5`                                               |

中国大陆时区包括 `Asia/Shanghai`、`Asia/Chongqing`、`Asia/Harbin`、`Asia/Urumqi` 等中国大陆 IANA 时区；`Asia/Hong_Kong`、`Asia/Macau`、`Asia/Taipei` 等不属于中国大陆时区。

当支付配置开启“已成功三方支付用户直接 H5”后，中国大陆时区内已存在成功三方支付订单的用户会直接返回 `h5`，该规则优先于地区、版本和今日苹果内购金额限制。

<!-- endpoint-fields:/relayjet/jetapi/chk_bgn:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_pkg_id_fly_suff` | `integer` | 是 | 套餐记录 ID，必须使用套餐列表接口返回且当前已上架的有效 ID。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_cat_cd_fly_suff` | `string` | 成功时 | 支付方式(apple_iap=苹果内购,h5=H5支付) |
| `rely_field_res_data_fly_suff.rely_field_dest_addr_fly_suff` | `string` | 成功时 | 支付目标，内购返回apple_id，H5返回支付页面地址 |
| `rely_field_res_data_fly_suff.rely_field_app_acct_tkn_fly_suff` | `string` | 成功时 | 苹果内购订单标识，内购时前端作为appAccountToken传给StoreKit |
<!-- endpoint-fields:/relayjet/jetapi/chk_bgn:end -->

### 苹果订单验证

```http
POST /relayjet/jetapi/apl_cfm
```

客户端完成苹果内购后提交苹果交易号。后端会通过苹果交易信息中的商品 ID 和 `appAccountToken` 匹配发起支付时创建的订单，验证成功后发放会员权益。

请求体：

```json
{
  "rely_field_txn_id_fly_suff": "1000000000000000"
}
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_ord_id_fly_suff":  "20260707120000123456",
                           "rely_field_pay_st_fly_suff":  3,
                           "rely_field_pay_mth_fly_suff":  "apple_iap",
                           "rely_field_txn_id_fly_suff":  "1000000000000000",
                           "rely_field_vip_exp_tm_fly_suff":  "2026-08-06 12:00:00"
                       }
}
```

字段说明：

| 字段             | 说明                             |
| ---------------- | -------------------------------- |
| `rely_field_ord_id_fly_suff`       | 后端订单号                       |
| `rely_field_pay_st_fly_suff`     | 支付状态，`1=未支付`，`3=已支付` |
| `rely_field_pay_mth_fly_suff`       | 支付方式                         |
| `rely_field_txn_id_fly_suff` | 苹果交易号                       |
| `rely_field_vip_exp_tm_fly_suff`      | 会员到期时间                     |

<!-- endpoint-fields:/relayjet/jetapi/apl_cfm:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_txn_id_fly_suff` | `string` | 是 | Apple 支付成功后返回的交易号，不能为空。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_ord_id_fly_suff` | `string` | 成功时 | 订单号 |
| `rely_field_res_data_fly_suff.rely_field_pay_st_fly_suff` | `integer` | 成功时 | 支付状态(1=未支付,3=已支付) |
| `rely_field_res_data_fly_suff.rely_field_pay_mth_fly_suff` | `string` | 成功时 | 支付方式 |
| `rely_field_res_data_fly_suff.rely_field_txn_id_fly_suff` | `string` | 成功时 | 苹果交易号 |
| `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff` | `string` | 成功时 | 会员到期时间 |
<!-- endpoint-fields:/relayjet/jetapi/apl_cfm:end -->

## VPN 接口

### 线路区域列表

```http
POST /relayjet/jetapi/rgn_cat
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  [{
                           "rely_field_ctry_nm_fly_suff":  "香港",
                           "rely_field_reg_cd_fly_suff":  "HK",
                           "rely_field_min_conn_sec_fly_suff":  100,
                           "rely_field_max_conn_sec_fly_suff":  300
                       }]
}
```

<!-- endpoint-fields:/relayjet/jetapi/rgn_cat:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff[].rely_field_ctry_nm_fly_suff` | `string` | 成功时 | 国家或地区的展示名称。 |
| `rely_field_res_data_fly_suff[].rely_field_reg_cd_fly_suff` | `string` | 成功时 | 线路代码，例如 `HK` 或 `AUTO`。 |
| `rely_field_res_data_fly_suff[].rely_field_min_conn_sec_fly_suff` | `integer` | 成功时 | 线路建议的最小连接耗时阈值，单位为毫秒。 |
| `rely_field_res_data_fly_suff[].rely_field_max_conn_sec_fly_suff` | `integer` | 成功时 | 线路建议的最大连接耗时阈值，单位为毫秒。 |
<!-- endpoint-fields:/relayjet/jetapi/rgn_cat:end -->

### 节点返回模式

母版当前默认实现三种节点返回方式：

| 接口 | 返回内容 | 状态 |
| --- | --- | --- |
| `POST /relayjet/jetapi/tun_ent` | 加密节点 URL | 母版和现有 VPN 项目已实现 |
| `POST /relayjet/jetapi/tun_opts` | 加密的完整 JSON 配置 | 母版和大部分项目已实现，少数老项目尚无 |
| `POST /relayjet/jetapi/tun_egr` | 分别加密的节点 URL 和 `rely_field_exit_lst_fly_suff` JSON | 已实现 |

三种方式的节点选择、会员校验、审核节点和临时节点记录语义必须一致。前端只能调用当前产品文档和 Swagger 中实际存在的接口。

### 获取节点

```http
POST /relayjet/jetapi/tun_ent
```

请求体：

```json
{
  "rely_field_rte_cd_fly_suff": "HK"
}
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_lnk_uri_fly_suff":  "x/k5A0v9kiJjL0r3m6X9dA=="
                       }
}
```

说明：

| 规则     | 说明                                                                          |
| -------- | ----------------------------------------------------------------------------- |
| 节点链接 | `rely_field_lnk_uri_fly_suff` 是加密后的节点连接数据，所有节点类型都按字符串返回               |
| 解密流程 | 客户端先对返回字符串做标准 Base64 解码得到 AES 密文，再以 AES-CBC/PKCS7 解密得到内部 Base64 字符串，最后再次标准 Base64 解码得到原始节点内容 |
| 加密 key | 后端配置提供，前端使用项目约定的节点解密 key                                  |
| 免费线路 | `FREE` 或 `FREE_` 开头的线路不校验会员                                        |
| 会员线路 | 需要会员未过期，未开通会员或会员已过期时返回 `rely_field_rnumber_fly_suff=901`                 |
| 连接确认 | 成功拿到节点后，需要连接成功再调用在线接口                                    |

会员过期响应：

```json
{
    "rely_field_rnumber_fly_suff":  901,
    "rely_field_rmessage_fly_suff":  "未开通会员或会员已过期，请开通后使用",
    "rely_field_res_data_fly_suff":  {

                       }
}
```

<!-- endpoint-fields:/relayjet/jetapi/tun_ent:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rte_cd_fly_suff` | `string` | 是 | 线路代码，例如 `HK` 或 `AUTO`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_lnk_uri_fly_suff` | `string` | 成功时 | 加密节点 URL；先对该字符串做标准 Base64 解码得到 AES 密文，再执行 AES-CBC/PKCS7 解密，最后对解密所得内部 Base64 字符串再次解码得到原始节点 URL。 |
<!-- endpoint-fields:/relayjet/jetapi/tun_ent:end -->

### 获取节点 URL 和 outbounds

```http
POST /relayjet/jetapi/tun_egr
```

请求体：

```json
{
  "rely_field_rte_cd_fly_suff": "HK"
}
```

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_lnk_uri_fly_suff":  "x/k5A0v9kiJjL0r3m6X9dA==",
                           "rely_field_exit_lst_fly_suff":  "x/k5A0v9kiJjL0r3m6X9dA=="
                       }
}
```

| 字段 | 类型 | 必填/必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rte_cd_fly_suff` | string | 必填 | 线路代码，与其他两种节点接口一致 |
| `rely_field_lnk_uri_fly_suff` | string | 必返 | 与 `/relayjet/jetapi/tun_ent` 相同的加密节点 URL；先做标准 Base64 解码，再执行 AES-CBC/PKCS7 解密，最后再次标准 Base64 解码 |
| `rely_field_exit_lst_fly_suff` | string | 必返 | 独立加密的 JSON 数组，必须与 `rely_field_lnk_uri_fly_suff` 分别解密；解密方式相同，解密后与 `/relayjet/jetapi/tun_opts` 完整配置中的 `rely_field_exit_lst_fly_suff` 完全一致 |

<!-- endpoint-fields:/relayjet/jetapi/tun_egr:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rte_cd_fly_suff` | `string` | 是 | 线路代码，例如 `HK` 或 `AUTO`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_lnk_uri_fly_suff` | `string` | 成功时 | 独立加密的节点 URL；完整解密流程为外层标准 Base64 解码、AES-CBC/PKCS7 解密、内部标准 Base64 解码。 |
| `rely_field_res_data_fly_suff.rely_field_exit_lst_fly_suff` | `string` | 成功时 | 独立加密的 outbounds JSON 数组；必须单独完成与 `rely_field_res_data_fly_suff.rely_field_lnk_uri_fly_suff` 相同的完整解密流程。 |
<!-- endpoint-fields:/relayjet/jetapi/tun_egr:end -->

### 获取 JSON 节点配置

```http
POST /relayjet/jetapi/tun_opts
```

请求体：

```json
{
  "rely_field_rte_cd_fly_suff": "HK",
  "rely_field_cfg_cat_fly_suff": "fast"
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `rely_field_rte_cd_fly_suff` | 线路代码 |
| `rely_field_cfg_cat_fly_suff` | 连接模式，`fast` 或 `极速` 表示极速模式，`global` 或 `全局` 表示全局模式 |

响应：

```json
{
    "rely_field_rnumber_fly_suff":  200,
    "rely_field_rmessage_fly_suff":  "success",
    "rely_field_res_data_fly_suff":  {
                           "rely_field_cfg_data_fly_suff":  "x/k5A0v9kiJjL0r3m6X9dA=="
                       }
}
```

`rely_field_cfg_data_fly_suff` 是完整 JSON 节点配置的加密结果，解密流程与 `rely_field_lnk_uri_fly_suff` 相同：先对返回字符串做标准 Base64 解码得到 AES 密文，再以 AES-CBC/PKCS7 解密得到内部 Base64 字符串，最后再次标准 Base64 解码得到 JSON 配置。该接口与 `/relayjet/jetapi/tun_ent` 使用相同的线路选择、会员校验和临时节点记录逻辑。

<!-- endpoint-fields:/relayjet/jetapi/tun_opts:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rte_cd_fly_suff` | `string` | 是 | 线路代码 |
| `rely_field_cfg_cat_fly_suff` | `string` | 是 | 连接模式(fast=极速,global=全局) |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_cfg_data_fly_suff` | `string` | 成功时 | 加密的完整 JSON 节点配置；完整解密流程为外层标准 Base64 解码、AES-CBC/PKCS7 解密、内部标准 Base64 解码。 |
<!-- endpoint-fields:/relayjet/jetapi/tun_opts:end -->

### 确认已连接

```http
POST /relayjet/jetapi/sess_opn
```

客户端成功建立 VPN 后调用。

<!-- endpoint-fields:/relayjet/jetapi/sess_opn:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/sess_opn:end -->

### 心跳

```http
POST /relayjet/jetapi/sess_pls
```

连接中定时调用，用于刷新在线状态。

客户端应每 30 秒调用一次。服务端在线状态记录有效期为 90 秒；记录过期或不存在时，只要会员仍有效，心跳仍返回继续连接，并重新等待后续连接状态上报。只有会员失效时才返回断开指令。

响应 `rely_field_res_data_fly_suff.rely_field_conn_st_fly_suff` 表示连接指令：`1` 继续连接，`0` 断开连接。仅会员过期时返回 `0`；当前没有连接记录但会员有效时仍返回 `1`，避免连接成功上报失败导致客户端被心跳断开。

<!-- endpoint-fields:/relayjet/jetapi/sess_pls:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `rely_field_res_data_fly_suff.rely_field_conn_st_fly_suff` | `integer` | 成功时 | 连接状态(1=继续连接,0=断开连接) |
<!-- endpoint-fields:/relayjet/jetapi/sess_pls:end -->

### 断开连接

```http
POST /relayjet/jetapi/sess_cls
```

主动断开 VPN 时调用。

<!-- endpoint-fields:/relayjet/jetapi/sess_cls:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/sess_cls:end -->

### 流量上报

```http
POST /relayjet/jetapi/trf_rpt
```

请求体：

```json
{
  "rely_field_net_bytes_fly_suff": 1024
}
```

`rely_field_net_bytes_fly_suff` 单位为 K，必须大于 0。

<!-- endpoint-fields:/relayjet/jetapi/trf_rpt:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `rely_field_net_bytes_fly_suff` | `integer` | 是 | 本次上报的 VPN 使用流量，单位为 K，必须大于 0。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `rely_field_rnumber_fly_suff` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `rely_field_rmessage_fly_suff` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `rely_field_res_data_fly_suff` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/relayjet/jetapi/trf_rpt:end -->

<!-- detailed-client-flows:start -->
## 推荐调用流程

本节描述前端实际接入顺序、状态保存和异常分支。字段名和路径均为当前产品的真实映射值，可直接用于客户端实现。

### 所有流程共同遵守的请求规则

每次请求都应携带当前平台和版本环境：`RelayJet-Plat-Name`、`RelayJet-Version-Number`、`RelayJet-Build-Version`、`RelayJet-Device-Name`、`RelayJet-Time`、`RelayJet-TimeZone`。可选扩展请求头为：`RelayJet-Stores-Reg`（App Store 三位地区代码，例如 `CHN`）、`RelayJet-Language`（中文可传 `zh-Hans`、`zh-Hant`、`zh-CN`、`zh-TW` 等 `zh` 标识，其他语言按英文处理）。可选头未携带时不能阻止正常请求。

除自动登录等公开接口外，鉴权接口必须携带 `Authorization: Bearer <token>`。前端收到响应后先读取 `rely_field_rnumber_fly_suff`：`200` 才处理数据；`401` 清除旧 Token 并重新自动登录；`500` 展示 `rely_field_rmessage_fly_suff`；获取节点接口的 `901` 只表示没有有效 VIP，应进入购买或续费页面，不能刷新 Token。网络超时、DNS、Nginx `502` 等没有形成正常业务 JSON 的情况按网络错误处理。

### 应用首次启动与日常启动

| 阶段 | 触发条件与请求 | 成功后前端处理 | 异常与注意事项 |
| --- | --- | --- | --- |
| 建立登录态 | 应用首次安装、本地没有 Token，或鉴权接口返回业务码 `401` 时，调用 `POST /relayjet/jetapi/gst_acc`。请求体至少传 `rely_field_dev_id_fly_suff`；平台、版本、设备名称、系统版本、设备型号和来源渠道按该接口字段表传递。 | 保存 `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff`，并缓存 `rely_field_res_data_fly_suff.rely_field_rec_id_fly_suff`、`rely_field_res_data_fly_suff.rely_field_vip_flg_fly_suff`、`rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff`。同一安装应始终使用稳定的设备号，避免重复创建游客。 | 自动登录本身失败时展示业务提示或网络重试；不要拿已经收到 `401` 的旧 Token 循环重试原接口。 |
| 获取运营配置 | 已取得 Token 后调用 `POST /relayjet/jetapi/stp_opts`。 | 缓存协议、分享、邀请、官网、试用时长、客服入口、工具开关和代理白名单。后端配置可热更新，因此每次冷启动应重新获取。 | 单个可选配置为空时使用客户端默认值，不应把空配置当成登录失败。 |
| 检查版本 | 调用 `POST /relayjet/jetapi/ed_chk`，构建号从请求头 `RelayJet-Build-Version` 读取。 | 根据返回的是否更新、是否强制、版本号、下载地址、大小和更新内容决定是否展示更新界面。 | 没有匹配的更新版本时返回空结果属于正常情况；强制更新时应阻止继续进入主界面。 |
| 获取首页内容 | 按页面需要调用 `GET /relayjet/jetapi/ovl_crd`、`GET /relayjet/jetapi/dfr_crd`、`GET /relayjet/jetapi/spn_shw` 和 `GET /relayjet/jetapi/unrd_ntc`。 | 普通弹窗按后端返回直接展示；延迟弹窗按返回延迟控制；未读数用于消息角标；广告按启用状态和展示位置渲染。 | 返回空对象或空数组通常表示当前没有可展示内容，不应提示系统错误。弹窗展示次数和新用户、地区限制由后端判断。 |

日常启动如果本地已有 Token，可以先直接调用鉴权接口；只有收到业务码 `401` 时才重新自动登录并替换 Token。不要仅因为应用重启就丢弃仍有效的 Token。

### VPN 线路选择、连接、心跳与断开

#### 获取线路并选择节点返回格式

进入线路页面后调用 `POST /relayjet/jetapi/rgn_cat`。使用 `rely_field_res_data_fly_suff[].rely_field_ctry_nm_fly_suff` 展示线路名称，提交获取节点请求时必须原样使用同一项的 `rely_field_res_data_fly_suff[].rely_field_reg_cd_fly_suff`，例如 `HK` 或 `AUTO`，不能根据展示文案自行拼接代码。

客户端应根据自身实现固定选择下面一种节点接口；一次正常连接不需要把三种接口全部调用一遍。三种接口共用会员校验、审核版本、试用节点和负载均衡逻辑，区别只在返回格式。

| 客户端能力 | 请求接口与请求体 | 成功响应的使用方式 |
| --- | --- | --- |
| 客户端自行拼接完整配置 | 调用 `POST /relayjet/jetapi/tun_ent`，请求体传 `rely_field_rte_cd_fly_suff`。 | 解密 `rely_field_res_data_fly_suff.rely_field_lnk_uri_fly_suff` 得到原始节点 URL，再由客户端生成完整配置。 |
| 客户端只拼接配置其余部分 | 调用 `POST /relayjet/jetapi/tun_egr`，请求体传 `rely_field_rte_cd_fly_suff`。 | 分别解密 `rely_field_res_data_fly_suff.rely_field_lnk_uri_fly_suff` 和 `rely_field_res_data_fly_suff.rely_field_exit_lst_fly_suff`；后者是可直接嵌入配置的 `rely_field_exit_lst_fly_suff` JSON 数组。 |
| 客户端直接使用完整 JSON | 调用 `POST /relayjet/jetapi/tun_opts`，请求体传 `rely_field_rte_cd_fly_suff` 和 `rely_field_cfg_cat_fly_suff`；连接模式使用 `fast`/`极速` 或 `global`/`全局`。 | 解密 `rely_field_res_data_fly_suff.rely_field_cfg_data_fly_suff` 得到完整 JSON 配置并交给连接内核。 |

所有加密节点数据都使用当前产品约定的节点密钥和既有 AES-CBC/PKCS7 + Base64 流程。前端不得把密钥、解密后的节点 URL 或完整配置写入用户可见日志。业务码 `901` 时停止连接并进入会员购买提示；业务码 `401` 时先自动登录再重新获取节点；业务码 `500` 时展示服务端提示。

#### 建立连接后的状态闭环

| 连接状态 | 前端必须调用 | 服务端响应与前端动作 |
| --- | --- | --- |
| VPN 内核尚未确认连通 | 暂时不要调用 `POST /relayjet/jetapi/sess_opn`。如果内核直接失败，可按产品交互调用 `POST /relayjet/jetapi/iss_rpt` 上报错误信息。 | 获取到节点不等于连接成功，不能提前产生在线记录。 |
| VPN 内核已确认连通 | 立即调用 `POST /relayjet/jetapi/sess_opn`。 | 业务码 `200` 后开始心跳；如果该上报因瞬时网络问题失败，只要会员仍有效，后续心跳没有在线记录时也会返回继续连接。 |
| VPN 正在连接 | 每 30 秒调用 `POST /relayjet/jetapi/sess_pls`。 | `rely_field_res_data_fly_suff.rely_field_conn_st_fly_suff=1` 时保持连接；`rely_field_res_data_fly_suff.rely_field_conn_st_fly_suff=0` 时立即让内核断开。当前逻辑只有会员失效才要求断开。 |
| 用户主动断开或内核结束 | 调用 `POST /relayjet/jetapi/sess_cls`，并停止心跳定时器。 | 即使断开上报失败，也必须先清理客户端本地连接状态，后续可在有网络时记录错误。 |
| 产生可统计流量 | 在适当的周期或断开前调用 `POST /relayjet/jetapi/trf_rpt`，请求体 `rely_field_net_bytes_fly_suff` 传本次增量流量。 | 单位为 K，值必须大于 `0`；不要重复累计上报同一段流量。 |

### 套餐购买、苹果验单与会员状态刷新

| 阶段 | 请求与判断 | 前端处理 |
| --- | --- | --- |
| 展示套餐 | 调用 `GET /relayjet/jetapi/ofr_cat`。 | 使用返回数组渲染套餐；用户选择后保存该项 `rely_field_res_data_fly_suff[].rely_field_rec_id_fly_suff`。发起支付时只能提交接口实际返回且当前上架的套餐 ID，不能把数组下标当套餐 ID。 |
| 创建订单并决定支付渠道 | 调用 `POST /relayjet/jetapi/chk_bgn`，请求体 `rely_field_pkg_id_fly_suff` 传选中的套餐 ID。支付渠道由后端根据时区、IP 地区、版本和支付配置决定。 | 读取 `rely_field_res_data_fly_suff.rely_field_cat_cd_fly_suff`，不要由客户端自行判断走苹果还是 H5。 |
| 返回 `apple_iap` | `rely_field_res_data_fly_suff.rely_field_dest_addr_fly_suff` 是 Apple 商品 ID，`rely_field_res_data_fly_suff.rely_field_app_acct_tkn_fly_suff` 是本次后端订单 UUID。 | 使用商品 ID 调起 StoreKit，并把订单 UUID 原样作为 `appAccountToken`；支付成功取得苹果交易号后调用 `POST /relayjet/jetapi/apl_cfm`，请求体传 `rely_field_txn_id_fly_suff`。验单业务码 `200` 后，以 `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff` 更新界面，并再调用用户信息接口校准会员状态。 |
| 返回 `h5` | `rely_field_res_data_fly_suff.rely_field_dest_addr_fly_suff` 是三方支付页面 URL，`rely_field_res_data_fly_suff.rely_field_app_acct_tkn_fly_suff` 为空。 | 用系统浏览器或约定 WebView 打开 URL。支付成功由三方服务端回调后端发放会员，客户端不要调用支付回调接口；用户返回 App 后调用 `GET /relayjet/jetapi/mbr_prof` 刷新 `rely_field_res_data_fly_suff.rely_field_vip_flg_fly_suff` 和 `rely_field_res_data_fly_suff.rely_field_vip_exp_tm_fly_suff`。 |
| 支付失败或取消 | 苹果取消、验单失败、H5 页面关闭或业务码 `500`。 | 保持原会员状态；`500` 展示 `rely_field_rmessage_fly_suff`。允许用户重新发起支付，但不要复用上一笔的 `rely_field_res_data_fly_suff.rely_field_app_acct_tkn_fly_suff` 或苹果交易号。 |

苹果和三方支付回调是服务端对服务端接口，不属于客户端接入范围。客户端只调用发起支付与苹果验单两个接口。

### 游客、注册、账号登录、设备管理与注销

| 用户动作 | 调用方式 | Token 与本地状态处理 |
| --- | --- | --- |
| 游客首次进入 | 调用 `POST /relayjet/jetapi/gst_acc`，设备号使用稳定值。 | 保存 `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff`。后端可能创建新游客，也可能找到已有设备用户；前端不需要区分创建接口。 |
| 游客注册账号 | 在已有游客 Token 下调用 `POST /relayjet/jetapi/acct_enr`，提交 `rely_field_dev_id_fly_suff`、`rely_field_usr_nm_fly_suff`、`rely_field_pwd_txt_fly_suff`。账号和密码均为 6–20 位字母或数字。 | 业务码 `200` 后必须用响应中的 `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` 覆盖本地 Token，并刷新用户类型、账号名和会员状态。 |
| 已有账号登录当前设备 | 调用 `POST /relayjet/jetapi/mbr_acc`，至少提交 `rely_field_dev_id_fly_suff`、`rely_field_usr_nm_fly_suff`、`rely_field_pwd_txt_fly_suff`，其余设备环境字段按接口字段表传递。 | 业务码 `200` 后用 `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff` 覆盖本地 Token。账号不存在、密码错误、设备数达到上限等返回 `500`，直接展示提示。 |
| 刷新个人资料 | 调用 `GET /relayjet/jetapi/mbr_prof`。 | 使用服务端返回覆盖会员状态和到期时间；支持 App Store 地区或语言记录的产品也会在此接口按非空有效请求头更新用户资料。 |
| 查看或移除其他设备 | 调用 `POST /relayjet/jetapi/dev_prof` 获取本机及其他设备；移除设备时调用 `POST /relayjet/jetapi/dev_rel`，用 `rely_field_rec_id_fly_suff` 提交设备列表返回的记录 ID。 | 不能提交用户 ID 或设备号代替设备记录 ID。移除操作会把目标设备恢复成游客并清除该设备会员；目标设备的原 Token 仍可继续鉴权，但后续读取到的是游客状态。 |
| 当前设备退出账号 | 调用 `POST /relayjet/jetapi/sess_lve`。 | 当前设备恢复游客状态并清空本设备会员，账号会员仍保留；保存接口返回的新用户信息和 Token。不要把“退出账号”当成删除账号。 |
| 修改密码 | 已登录账号调用 `POST /relayjet/jetapi/sec_rnw`，只提交 `rely_field_new_pwd_fly_suff`，无需旧密码。 | 新密码必须为 6–20 位字母或数字且不能与当前密码相同；成功后 Token 不需要更换。 |
| 注销账号 | 调用 `POST /relayjet/jetapi/acct_ret`。是否真实注销由请求头 `RelayJet-Version-Number` 是否命中服务端配置决定。 | 真实注销会删除账号或设备用户并创建新游客，必须保存 `rely_field_res_data_fly_suff.rely_field_acc_key_fly_suff`；未命中真实注销版本时返回成功空对象，原 Token 继续有效。前端必须兼容这两种成功响应。 |

### 通知读取、邀请和客户端错误上报

通知页面先调用 `GET /relayjet/jetapi/msg_brd` 获取系统与个人通知；用户打开通知后调用 `POST /relayjet/jetapi/ntc_ack`，分别用 `rely_field_sys_note_ids_fly_suff` 和 `rely_field_usr_note_ids_fly_suff` 提交系统通知 ID 数组与个人通知 ID 数组。读取成功后再更新本地已读状态和角标，避免网络失败时界面与服务端不一致。

邀请页面调用 `POST /relayjet/jetapi/ref_iss` 获取自己的邀请码，输入他人邀请码时调用 `POST /relayjet/jetapi/ref_apl` 并将输入值放入 `rely_field_ref_cd_fly_suff`；邀请记录和奖励进度从 `POST /relayjet/jetapi/ref_dtl` 获取。当前用户是否仍可填写他人邀请码及奖励是否到账以接口返回为准，不由客户端本地计算。

VPN 内核、配置解析或其他可诊断错误可调用 `POST /relayjet/jetapi/iss_rpt` 上报，用 `rely_field_err_cat_fly_suff` 标识错误分类，用 `rely_field_msg_txt_fly_suff` 提交可诊断信息，并同时提交 `rely_field_os_ver_fly_suff` 操作系统版本和 `rely_field_mdl_nm_fly_suff` 设备型号。上报内容不得包含登录密码、支付密钥、完整 Token、节点解密密钥或其他敏感信息；错误上报失败也不能阻塞用户退出连接或继续使用 App。
<!-- detailed-client-flows:end -->