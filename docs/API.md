# Uppiaing API 文档

本文档依据 conf/mapping.json 生成，所有路径、请求字段和响应字段均为 Uppiaing 的有效映射名称。

文档中的请求和响应字段已包含 `field_affix` 配置的前缀 `upp_` 和后缀 `_upld`，客户端直接使用文档中的完整字段名，无需重复拼接。例如映射值 `reccd1`、`remms1`、`out_dat` 对应 `upp_reccd1_upld`、`upp_remms1_upld`、`upp_out_dat_upld`。请求头名称和接口路径不添加字段前后缀。

## 幸运转盘

- 请求：`POST /uppiaing/maycallers_v/whl_run`（原始路由：`/api/v1/lucky_wheel_play`）。
- 鉴权：`Authorization: Bearer <token>`，无需请求参数或请求体。
- 满足任意一个条件即可参与：`lucky_wheel.general_enabled` 为 `on`；或 `lucky_wheel.new_user_enabled` 为 `on` 且当前用户 `type=1`。仅字符串 `on` 视为开启，配置缺失或读取失败按关闭处理。不再限制注册时长。
- 使用 Redis `SET NX` 原子占用当天参与次数，键为 `lucky_wheel:played:YYYYMMDD:<user_id>`，日期按北京时间（Asia/Shanghai）计算，TTL 为距离次日零点的剩余时间。已有标记则拒绝参与；次日仍须满足开关与用户类型条件。Redis 不可用时返回错误，不继续抽奖。
- 不再查询数据库中的参与记录。抽奖或记录写入失败时，通过比较请求标识的 Lua 脚本释放本次占位；写入成功后保留标记至过期。Redis 标记被清空或淘汰后可能重复参与；旧数据库记录不会自动回填 Redis。进程中断或占位释放失败时，标记可能保留至零点。
- 将 `lucky_wheel` 表中全部未删除奖品缓存到 Redis（键 `lucky_wheel:available:v2`，TTL 30 秒），每次调用从奖池中等概率随机抽取一条。缓存未命中或读取失败时查询数据库；缓存写入失败不影响本次抽奖。命中缓存不会延长 TTL。缓存键升级以避免读取缺少 `seconds` 的旧奖池缓存。
- 抽中后将 `user_id`、`title`、`desc`、`remark`、`vip_secs`、`status` 通过单条 `INSERT` 写入 `lucky_wheel_play_record` 表，其中 `vip_secs` 保存奖品 `seconds` 的快照，`status=0` 表示未领取。奖品时长必须大于 0，写入成功后才返回中奖结果。该流程不使用显式事务，并仅对此次插入关闭 GORM 默认事务；失败时返回错误。
- 成功时 `upp_out_dat_upld` 为对象，包含以下四个字段：

| 实际字段 | 原始字段 | 类型 | 说明 |
| --- | --- | --- | --- |
| `upp_tier_no_upld` | `level` | number | 奖品等级 |
| `upp_head_tx_upld` | `title` | string | 奖品标题，例如 `10分钟免费会员` |
| `upp_intro_tx_upld` | `desc` | string | 奖品描述，例如 `免费会员` |
| `upp_note_tx_upld` | `remark` | string | 奖品备注，例如 `高速会员专属权益` |

两个参与条件均不满足时返回业务状态码 `500`，提示“当前用户暂不可参与幸运转盘”，不会占用当天次数。Redis 中存在当天参与标记时返回业务状态码 `500`，提示“您今天已参与幸运转盘，请明天再试”；删除数据库参与记录不会清除 Redis 标记。无可用记录、Redis 占位失败、数据库查询或记录写入失败时，返回业务状态码 `500`；未登录返回 `401`，HTTP 状态码均为 `200`。本接口保存中奖信息，不修改用户会员时长。

启动时按 `level` 补充缺失的初始记录，将这些等级已有记录的空 `desc` 补为 `免费会员`，并补充为空或为 0 的 `seconds`，保留其他已有内容：

| level | title | seconds | desc | remark |
| --- | --- | --- | --- | --- |
| 1 | 10分钟 | 600 | 免费会员 | 高速会员专属权益 |
| 2 | 20分钟 | 1200 | 免费会员 | 高速会员专属权益 |
| 3 | 30分钟 | 1800 | 免费会员 | 高速会员专属权益 |
| 4 | 1天 | 86400 | 免费会员 | 高速会员专属权益 |
| 5 | 7天 | 604800 | 免费会员 | 高速会员专属权益 |

## 幸运转盘状态

- 请求：`GET /uppiaing/maycallers_v/whl_sts`（原始路由：`/api/v1/lucky_wheel_get_status`）。
- 鉴权：`Authorization: Bearer <token>`，无需请求参数。
- `today_played` 根据北京时间当天的 Redis 参与标记是否存在返回，与抽奖接口的每日限制一致；正在抽奖的占位也视为已参与。只查询，不写入或延长 TTL。Redis 读取失败返回业务状态码 `500`。
- 四个开关字段原样返回下表所列系统配置的字符串值，保留大小写和空白，不限制为 `on/off`；配置缺失、为空或读取失败返回空字符串。沿用系统配置缓存刷新规则，不根据用户注册时长修改配置值。

| 实际字段 | 原始字段 | 类型 | 说明 |
| --- | --- | --- | --- |
| `upp_day_done_upld` | `today_played` | boolean | 当天已有参与标记为 `true`，否则为 `false` |
| `upp_new_on_upld` | `news_enabled` | string | 新用户幸运转盘开关，来自 `lucky_wheel.new_user_enabled` |
| `upp_all_on_upld` | `general_enabled` | string | 通用幸运转盘开关，来自 `lucky_wheel.general_enabled` |
| `upp_new_cls_upld` | `news_close_enabled` | string | 新用户幸运转盘界面关闭开关，来自 `lucky_wheel.new_user_close_enabled` |
| `upp_all_cls_upld` | `general_close_enabled` | string | 通用幸运转盘界面关闭开关，来自 `lucky_wheel.general_close_enabled` |

以上五个字段位于 `upp_out_dat_upld` 对象中，成功时全部返回，包括值为空字符串的配置字段。四个开关的约定值为 `on` 开启、`off` 关闭；两个 `close_enabled` 字段表示界面关闭功能的开关，不代表用户当天是否已参与。初始化配置中两个参与开关为 `off`，两个界面关闭开关为 `on`；这些初始化值不替代查询时缺失配置返回的空字符串。

成功响应示例：

```json
{
  "upp_reccd1_upld": 200,
  "upp_remms1_upld": "success",
  "upp_out_dat_upld": {
    "upp_day_done_upld": false,
    "upp_new_on_upld": "off",
    "upp_all_on_upld": "off",
    "upp_new_cls_upld": "on",
    "upp_all_cls_upld": "on"
  }
}
```

成功业务状态码为 `200`；未登录或登录失效为 `401`；Redis 读取失败为 `500`。HTTP 状态码均为 `200`，失败时 `upp_out_dat_upld` 为空对象，错误信息位于 `upp_remms1_upld`。

## 领取幸运转盘奖励

- 请求：`POST /uppiaing/maycallers_v/whl_clm`（原始路由：`/api/v1/lucky_wheel_get_reward`）。
- 鉴权：`Authorization: Bearer <token>`，无需请求参数或请求体。
- 事务前使用 Redis `SET NX` 原子占位，键为 `lucky_wheel:claimed:YYYYMMDD:<user_id>`，按北京时间计算日期，TTL 为距离次日零点的剩余时间。已有标记时直接返回业务状态码 `500`，提示“幸运转盘奖励正在领取或已领取，请勿重复操作”，不访问数据库，也不刷新 TTL。
- 事务提交成功或数据库确认已领取时保留标记；其他失败通过比较请求标识的 Lua 脚本释放本次占位。进程中断或释放失败可能导致标记保留至零点。Redis 不可用时回退数据库，仍保留领取状态校验、行锁和事务，避免缓存丢失后重复发奖。
- 按北京时间当天 `[00:00, 次日00:00)` 查询当前用户的参与记录，按 ID 取第一条；不按领取状态过滤，避免重复请求领取其他记录。抽奖时已校验开关与用户类型条件，领奖时根据已保存的参与记录发放奖励。
- 使用参与记录的 `vip_secs` 延长会员时间；会员未过期时从原到期时间累加，否则从当前时间累加。
- 仅修改当前用户的 `vip_time`。对于有账号的用户（`type=2` 且 `username` 非空），同步账号表及同账号正常用户的 `vip_time`；不修改支付统计或用户、账号表的其他字段。
- 在同一事务内锁定参与记录与用户记录、更新会员时间，并将 `status` 从 `0` 改为 `1`。任一步骤失败全部回滚，同一参与记录只允许领取一次。
- 成功返回业务状态码 `200`、消息 `success`，`upp_out_dat_upld` 为空对象。当天无记录、已领取、记录已删除或奖励时长无效时返回业务状态码 `500`。
- 旧参与记录不会根据标题推算或补填 `vip_secs`；没有有效奖励时长的记录无法领取。

## 用户中奖记录

- 请求：`GET /uppiaing/maycallers_v/whl_win`（原始路由：`/api/v1/lucky_wheel_winners`）。
- 鉴权：`Authorization: Bearer <token>`，无需请求参数。
- 每次请求读取 `lucky_wheel` 表中全部未删除的数据，再等概率有放回抽取，组成 30 条展示记录，不保存生成的中奖记录。
- `upp_out_dat_upld` 为记录数组，每条记录包含：

| 实际字段 | 原始字段 | 类型 | 说明 |
| --- | --- | --- | --- |
| `upp_usr_no_upld` | `user_id` | string | 100–999 的随机数 + `...` + 0–9 的随机数，例如 `123...4` |
| `upp_head_tx_upld` | `title` | string | 抽中转盘记录的标题 |
| `upp_intro_tx_upld` | `desc` | string | 抽中转盘记录的描述，例如 `免费会员` |
| `upp_note_tx_upld` | `remark` | string | 抽中转盘记录的备注，例如 `高速会员专属权益` |

成功响应中 `upp_reccd1_upld` 为 `200`，`upp_remms1_upld` 为 `success`；数组固定包含 30 条记录，允许用户标识或奖品重复。每条结果的 `title`、`desc`、`remark` 来自同一条数据库记录。

无可用转盘记录或数据库查询失败时返回业务状态码 `500`，HTTP 状态码为 `200`。

## 公共说明

### 基础地址

本地测试：

```text
http://127.0.0.1:8080
```

Uppiaing 接口统一前缀：

```text
/uppiaing/maycallers_v/
```

所有客户端请求和响应字段均使用本文件列出的 Uppiaing 映射字段名。支付运营商回调使用独立路径和运营商字段，客户端无需调用。

### 响应格式

所有接口统一返回：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {

                       }
}
```

字段说明：

| 字段       | 类型         | 说明                                                                     |
| ---------- | ------------ | ------------------------------------------------------------------------ |
| upp_reccd1_upld | number       | `200` 表示成功，`401` 表示登录状态失效，`901` 表示未开通会员或会员已过期，`500` 表示其他业务失败 |
| upp_remms1_upld    | string       | 响应消息                                                                 |
| upp_out_dat_upld    | object/array | 响应数据                                                                 |

<!-- client-status-codes:start -->
### 客户端业务状态码与处理

所有由客户端业务接口正常返回的响应，HTTP 状态码都固定为 `200`。即使 JSON 中的业务状态码是 `401`、`500` 或 `901`，HTTP 状态码仍然是 `200`。
前端必须读取响应 JSON 中的 `upp_reccd1_upld` 判断业务结果，不能只判断 HTTP 状态码。

域名无法访问、Nginx 或网关错误、接口路径不存在、请求超时等未进入正常业务响应的情况，HTTP 状态码可能不是 `200`，应按网络错误处理。

| 业务状态码 | 含义 | 前端处理方式 |
| --- | --- | --- |
| `200` | 请求成功 | 正常处理响应数据。 |
| `401` | 登录状态失效 | 清除旧 Token，重新调用自动登录接口 `/uppiaing/maycallers_v/gst_ent`，并保存返回的 `upp_out_dat_upld.upp_auth_key_upld`。 |
| `500` | 系统或业务错误 | 直接展示 `upp_remms1_upld` 返回的错误信息，不要重新自动登录。 |
| `901` | 用户没有有效 VIP | 只会在获取节点相关接口中返回；提示用户开通或续费 VIP，不要刷新 Token。 |

#### 401 处理

Token 过期、Token 无效、用户已被删除或设备不匹配等需要重新登录的情况统一返回业务码 `401`。

前端收到 `401` 后：

1. 清除本地旧 Token。
2. 重新调用 `POST /uppiaing/maycallers_v/gst_ent`。
3. 保存自动登录响应中的 `upp_out_dat_upld.upp_auth_key_upld`。

#### 500 处理

前端直接展示 `upp_remms1_upld` 中的错误信息，不需要重新自动登录。

#### 901 处理

`901` 仅用于以下获取节点接口，表示当前用户没有有效 VIP：

- `POST /uppiaing/maycallers_v/nod_get`
- `POST /uppiaing/maycallers_v/nod_egr`
- `POST /uppiaing/maycallers_v/nod_cfg`

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
| Uppiaing-PF-Name | 建议必填     | 平台，支持 `iphone/android` | `iphone`     |
| Uppiaing-C-Ver     | 建议必填     | 展示版本号                  | `1.0.0`      |
| Uppiaing-S-Ver        | 版本检测必填 | 数字版本号                  | `100`        |
| Uppiaing-D-C      | 建议必填     | 设备唯一标识                | `device-001` |
| Uppiaing-Time       | 建议必填     | 客户端本地 RFC3339 时间（含偏移量） | `2026-07-11T20:30:15.123+08:00` |
| Uppiaing-TZ-Off      | 建议必填     | 客户端 IANA 时区名            | `Asia/Shanghai` |
| Uppiaing-Reg | 可选 | 苹果 App Store 地区 | `CHN` |
| Uppiaing-Lang-Code | 可选 | 当前用户语言；任意 `zh`、`zh-*` 或 `zh_*`（含繁体标识）均显示简体中文，其他非空语言显示英文 | `zh-Hans` |

多语言响应规则：

- 不携带 `Uppiaing-Lang-Code` 时默认显示简体中文，兼容旧客户端。
- 套餐、弹窗、延迟弹窗、广告、通知、版本内容和线路国家名称在数据库字段为 `{"zh-Hans":"中文","en":"English"}` 时，按本次请求头选择语言。
- 数据库字段仍是普通字符串时原样返回，不会因为英文请求而改变旧内容。
- 双语 JSON 缺少目标语言或目标语言为空时，回退到另一种已有语言。
- 业务错误提示、VIP 提示、邀请奖励文字和本机设备名称等后端固定文字也按本次请求头返回简体中文或英文。

### 用户字段

用户信息结构：

```json
{
  "upp_obj_no_upld": 1,
  "upp_auth_key_upld": "jwt-token",
  "upp_dev_no_upld": "device-001",
  "upp_login_nm_upld": "",
  "upp_kind_tp_upld": 1,
  "upp_vip_ok_upld": 0,
  "upp_vip_end_upld": "",
  "upp_fresh_usr_upld": 1,
  "upp_plat_cd_upld": "iphone",
  "upp_app_ver_upld": "1.0.0",
  "upp_sign_cnt_upld": 1,
  "upp_crt_at_upld": "2026-07-06 10:00:00",
  "upp_last_in_upld": "2026-07-06 10:00:00",
  "upp_sec_key_upld": "pass001"
}
```

字段说明：

| 字段         | 说明                                         |
| ------------ | -------------------------------------------- |
| upp_obj_no_upld     | 用户 ID                                      |
| upp_auth_key_upld  | JWT 登录 token                               |
| upp_dev_no_upld    | 设备唯一标识                                 |
| upp_login_nm_upld  | 账号名，游客为空                             |
| upp_kind_tp_upld         | `1=游客`，`2=账号用户`                       |
| upp_vip_ok_upld    | `1=会员有效`，`0=非会员`                     |
| upp_vip_end_upld  | 会员到期时间，格式 `yyyy-MM-dd HH:mm:ss`，非会员为空 |
| upp_fresh_usr_upld | 是否新用户，`1=首次创建的新用户`，`0=老用户` |
| upp_plat_cd_upld | 客户端平台，例如 `iphone` 或 `android` |
| upp_app_ver_upld | 客户端版本号 |
| upp_sign_cnt_upld   | 登录次数                                     |
| upp_crt_at_upld | 用户创建时间，格式 `yyyy-MM-dd HH:mm:ss` |
| upp_last_in_upld | 本次登录时间，格式 `yyyy-MM-dd HH:mm:ss` |
| upp_sec_key_upld | 绑定账号的明文密码，游客为空字符串 |

## 公开接口

### 健康检查

```http
GET /uppiaing/maycallers_v/srv_chk
```

响应示例：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {

                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/srv_chk:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/srv_chk:end -->

### 获取静态资源

```http
GET /uppiaing/maycallers_v/fil_get/{filepath}
```

无需登录。后端会从服务运行目录的 `upload` 文件夹读取文件，适合放图片等静态资源。

示例：

```text
/uppiaing/maycallers_v/fil_get/banner.png
/uppiaing/maycallers_v/fil_get/ad/home.png
```

说明：

| 项目     | 说明                                              |
| -------- | ------------------------------------------------- |
| filepath | `upload` 目录下的相对路径                         |
| 返回内容 | 直接返回文件内容，Content-Type 由文件类型自动识别 |
| 不存在   | 返回 HTTP 404                                     |

<!-- endpoint-fields:/uppiaing/maycallers_v/fil_get/*filepath:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `filepath` | `string` | 是 | 资源相对路径 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `HTTP Body` | `binary` | 成功时 | 直接返回文件内容，不使用统一 JSON 响应结构。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/fil_get/*filepath:end -->

### 游客登录

```http
POST /uppiaing/maycallers_v/gst_ent
```

请求体：

```json
{
  "upp_dev_no_upld": "device-001",
  "upp_plat_cd_upld": "iphone",
  "upp_app_ver_upld": "1.0.0",
  "upp_dev_nm_upld": "iPhone",
  "upp_os_ver_upld": "iOS 17.5",
  "upp_dev_mdl_upld": "iPhone 15",
  "upp_src_ch_upld": "default"
}
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_obj_no_upld":  1,
                           "upp_auth_key_upld":  "jwt-token",
                           "upp_dev_no_upld":  "device-001",
                           "upp_login_nm_upld":  "",
                           "upp_kind_tp_upld":  1,
                           "upp_vip_ok_upld":  0,
                           "upp_vip_end_upld":  "",
                           "upp_fresh_usr_upld":  1,
                           "upp_plat_cd_upld":  "iphone",
                           "upp_app_ver_upld":  "1.0.0",
                           "upp_sign_cnt_upld":  1,
                           "upp_crt_at_upld":  "2026-07-06 10:00:00",
                           "upp_last_in_upld":  "2026-07-06 10:00:00",
                           "upp_sec_key_upld":  ""
                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/gst_ent:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_dev_no_upld` | `string` | 是 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_plat_cd_upld` | `string` | 否 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_app_ver_upld` | `string` | 否 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_dev_nm_upld` | `string` | 否 | 客户端设备名称，例如 `iPhone`。 |
| `upp_os_ver_upld` | `string` | 否 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_dev_mdl_upld` | `string` | 否 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_src_ch_upld` | `string` | 否 | 客户端安装或获客来源渠道；未区分时可传 `default`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_obj_no_upld` | `integer` | 成功时 | 用户ID |
| `upp_out_dat_upld.upp_auth_key_upld` | `string` | 成功时 | JWT登录凭证 |
| `upp_out_dat_upld.upp_dev_no_upld` | `string` | 成功时 | 设备唯一标识 |
| `upp_out_dat_upld.upp_login_nm_upld` | `string` | 成功时 | 账号名，游客为空 |
| `upp_out_dat_upld.upp_kind_tp_upld` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `upp_out_dat_upld.upp_vip_ok_upld` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `upp_out_dat_upld.upp_vip_end_upld` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `upp_out_dat_upld.upp_fresh_usr_upld` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `upp_out_dat_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台 |
| `upp_out_dat_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端版本号 |
| `upp_out_dat_upld.upp_sign_cnt_upld` | `integer` | 成功时 | 登录次数 |
| `upp_out_dat_upld.upp_crt_at_upld` | `string` | 成功时 | 用户创建时间 |
| `upp_out_dat_upld.upp_last_in_upld` | `string` | 成功时 | 本次登录时间 |
| `upp_out_dat_upld.upp_sec_key_upld` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/uppiaing/maycallers_v/gst_ent:end -->

### 注册账号

```http
POST /uppiaing/maycallers_v/acc_reg
```

需要先调用游客登录拿到 `upp_auth_key_upld`，并在请求头携带：

```http
Authorization: Bearer <token>
```

请求体：

```json
{
  "upp_dev_no_upld": "device-001",
  "upp_login_nm_upld": "user001",
  "upp_sec_key_upld": "pass001"
}
```

账号和密码规则：6 到 20 位字母或数字。

响应 `upp_out_dat_upld` 为用户信息。

<!-- endpoint-fields:/uppiaing/maycallers_v/acc_reg:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_dev_no_upld` | `string` | 是 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_login_nm_upld` | `string` | 是 | 绑定账号名，规则为 6–20 位字母或数字。 |
| `upp_sec_key_upld` | `string` | 是 | 账号密码，规则为 6–20 位字母或数字。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_obj_no_upld` | `integer` | 成功时 | 用户ID |
| `upp_out_dat_upld.upp_auth_key_upld` | `string` | 成功时 | JWT登录凭证 |
| `upp_out_dat_upld.upp_dev_no_upld` | `string` | 成功时 | 设备唯一标识 |
| `upp_out_dat_upld.upp_login_nm_upld` | `string` | 成功时 | 账号名，游客为空 |
| `upp_out_dat_upld.upp_kind_tp_upld` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `upp_out_dat_upld.upp_vip_ok_upld` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `upp_out_dat_upld.upp_vip_end_upld` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `upp_out_dat_upld.upp_fresh_usr_upld` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `upp_out_dat_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台 |
| `upp_out_dat_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端版本号 |
| `upp_out_dat_upld.upp_sign_cnt_upld` | `integer` | 成功时 | 登录次数 |
| `upp_out_dat_upld.upp_crt_at_upld` | `string` | 成功时 | 用户创建时间 |
| `upp_out_dat_upld.upp_last_in_upld` | `string` | 成功时 | 本次登录时间 |
| `upp_out_dat_upld.upp_sec_key_upld` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/uppiaing/maycallers_v/acc_reg:end -->

### 账号登录

```http
POST /uppiaing/maycallers_v/acc_ent
```

需要先调用游客登录拿到当前设备的 `upp_auth_key_upld`，并在请求头携带：

```http
Authorization: Bearer <token>
```

请求体：

```json
{
  "upp_dev_no_upld": "device-001",
  "upp_login_nm_upld": "user001",
  "upp_sec_key_upld": "pass001",
  "upp_plat_cd_upld": "iphone",
  "upp_app_ver_upld": "1.0.0",
  "upp_dev_nm_upld": "iPhone",
  "upp_os_ver_upld": "iOS 17.5",
  "upp_dev_mdl_upld": "iPhone 15",
  "upp_src_ch_upld": "default"
}
```

响应 `upp_out_dat_upld` 为用户信息。

<!-- endpoint-fields:/uppiaing/maycallers_v/acc_ent:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_dev_no_upld` | `string` | 是 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_login_nm_upld` | `string` | 是 | 绑定账号名，规则为 6–20 位字母或数字。 |
| `upp_sec_key_upld` | `string` | 是 | 账号密码，规则为 6–20 位字母或数字。 |
| `upp_plat_cd_upld` | `string` | 否 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_app_ver_upld` | `string` | 否 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_dev_nm_upld` | `string` | 否 | 客户端设备名称，例如 `iPhone`。 |
| `upp_os_ver_upld` | `string` | 否 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_dev_mdl_upld` | `string` | 否 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_src_ch_upld` | `string` | 否 | 客户端安装或获客来源渠道；未区分时可传 `default`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_obj_no_upld` | `integer` | 成功时 | 用户ID |
| `upp_out_dat_upld.upp_auth_key_upld` | `string` | 成功时 | JWT登录凭证 |
| `upp_out_dat_upld.upp_dev_no_upld` | `string` | 成功时 | 设备唯一标识 |
| `upp_out_dat_upld.upp_login_nm_upld` | `string` | 成功时 | 账号名，游客为空 |
| `upp_out_dat_upld.upp_kind_tp_upld` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `upp_out_dat_upld.upp_vip_ok_upld` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `upp_out_dat_upld.upp_vip_end_upld` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `upp_out_dat_upld.upp_fresh_usr_upld` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `upp_out_dat_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台 |
| `upp_out_dat_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端版本号 |
| `upp_out_dat_upld.upp_sign_cnt_upld` | `integer` | 成功时 | 登录次数 |
| `upp_out_dat_upld.upp_crt_at_upld` | `string` | 成功时 | 用户创建时间 |
| `upp_out_dat_upld.upp_last_in_upld` | `string` | 成功时 | 本次登录时间 |
| `upp_out_dat_upld.upp_sec_key_upld` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/uppiaing/maycallers_v/acc_ent:end -->

## 用户接口

### 退出登录

```http
POST /uppiaing/maycallers_v/acc_out
```

当前设备退出账号，设备恢复为游客状态，账号会员保留。

响应 `upp_out_dat_upld` 为用户信息。

<!-- endpoint-fields:/uppiaing/maycallers_v/acc_out:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_obj_no_upld` | `integer` | 成功时 | 用户ID |
| `upp_out_dat_upld.upp_auth_key_upld` | `string` | 成功时 | JWT登录凭证 |
| `upp_out_dat_upld.upp_dev_no_upld` | `string` | 成功时 | 设备唯一标识 |
| `upp_out_dat_upld.upp_login_nm_upld` | `string` | 成功时 | 账号名，游客为空 |
| `upp_out_dat_upld.upp_kind_tp_upld` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `upp_out_dat_upld.upp_vip_ok_upld` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `upp_out_dat_upld.upp_vip_end_upld` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `upp_out_dat_upld.upp_fresh_usr_upld` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `upp_out_dat_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台 |
| `upp_out_dat_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端版本号 |
| `upp_out_dat_upld.upp_sign_cnt_upld` | `integer` | 成功时 | 登录次数 |
| `upp_out_dat_upld.upp_crt_at_upld` | `string` | 成功时 | 用户创建时间 |
| `upp_out_dat_upld.upp_last_in_upld` | `string` | 成功时 | 本次登录时间 |
| `upp_out_dat_upld.upp_sec_key_upld` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/uppiaing/maycallers_v/acc_out:end -->

### 账号注销

```http
POST /uppiaing/maycallers_v/acc_del
```

账号注销接口。后端通过配置表 `account.real_logoff_versions` 控制真实注销版本，配置值为英文逗号分隔的版本号。

如果当前请求头 `Uppiaing-C-Ver` 命中该配置：

- 当前设备未登录账号时，删除当前设备用户
- 当前设备已登录账号时，删除账号表记录，并删除该账号已登录的所有设备用户
- 删除后旧 token 立即失效，当前设备会直接创建一个新的游客用户，不赠送新用户会员
- 接口会直接返回新游客用户的登录 token，客户端保存该 token 后即可继续调用鉴权接口，无需再次自动登录

真实注销成功响应只返回新 token：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_auth_key_upld":  "new-jwt-token"
                       }
}
```

如果版本未命中配置，则返回成功和空对象，不删除数据，当前 token 继续有效。

<!-- endpoint-fields:/uppiaing/maycallers_v/acc_del:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_auth_key_upld` | `string` | 成功时 | JWT 登录凭证，后续鉴权接口通过 Bearer Token 携带。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/acc_del:end -->

### 修改密码

```http
POST /uppiaing/maycallers_v/pwd_upd
```

请求体：

```json
{
  "upp_new_sec_upld": "pass002"
}
```

当前登录账号无需提交旧密码；新密码不能与当前密码相同。

<!-- endpoint-fields:/uppiaing/maycallers_v/pwd_upd:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_new_sec_upld` | `string` | 是 | 需要设置的新密码，规则为 6–20 位字母或数字。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/pwd_upd:end -->

### 获取设备列表

```http
POST /uppiaing/maycallers_v/dev_inf
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_now_dev_upld":  {
                                                 "upp_obj_no_upld":  1,
                                                 "upp_dev_no_upld":  "device-001",
                                                 "upp_dev_nm_upld":  "本机设备",
                                                 "upp_os_ver_upld":  "iOS 17.5",
                                                 "upp_dev_mdl_upld":  "iPhone 15",
                                                 "upp_plat_cd_upld":  "iphone",
                                                 "upp_app_ver_upld":  "1.0.0",
                                                 "upp_last_in_upld":  "2026-07-06 10:00:00"
                                             },
                           "upp_oth_dev_upld":  {

                                              },
                           "upp_dev_ls_upld":  {

                                              }
                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/dev_inf:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_now_dev_upld` | `object` | 成功时 | 当前发起请求的本机设备信息。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_obj_no_upld` | `integer` | 成功时 | 设备记录 ID。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_dev_no_upld` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_dev_nm_upld` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_os_ver_upld` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_dev_mdl_upld` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_last_in_upld` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_oth_dev_upld` | `object` | 成功时 | 账号绑定的另一台设备；不存在时返回空对象。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_obj_no_upld` | `integer` | 成功时 | 设备记录 ID。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_dev_no_upld` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_dev_nm_upld` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_os_ver_upld` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_dev_mdl_upld` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_last_in_upld` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_dev_ls_upld` | `array<object>` | 成功时 | 当前账号绑定的设备列表。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_obj_no_upld` | `integer` | 成功时 | 设备记录 ID。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_dev_no_upld` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_dev_nm_upld` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_os_ver_upld` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_dev_mdl_upld` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_plat_cd_upld` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_app_ver_upld` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_last_in_upld` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/dev_inf:end -->

### 移除设备

```http
POST /uppiaing/maycallers_v/dev_out
```

请求体：

```json
{
  "upp_obj_no_upld": 2
}
```

不能移除当前设备。

<!-- endpoint-fields:/uppiaing/maycallers_v/dev_out:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_obj_no_upld` | `integer` | 是 | 设备记录 ID。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_now_dev_upld` | `object` | 成功时 | 当前发起请求的本机设备信息。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_obj_no_upld` | `integer` | 成功时 | 设备记录 ID。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_dev_no_upld` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_dev_nm_upld` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_os_ver_upld` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_dev_mdl_upld` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_out_dat_upld.upp_now_dev_upld.upp_last_in_upld` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_oth_dev_upld` | `object` | 成功时 | 账号绑定的另一台设备；不存在时返回空对象。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_obj_no_upld` | `integer` | 成功时 | 设备记录 ID。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_dev_no_upld` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_dev_nm_upld` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_os_ver_upld` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_dev_mdl_upld` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_out_dat_upld.upp_oth_dev_upld.upp_last_in_upld` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_dev_ls_upld` | `array<object>` | 成功时 | 当前账号绑定的设备列表。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_obj_no_upld` | `integer` | 成功时 | 设备记录 ID。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_dev_no_upld` | `string` | 成功时 | 客户端设备唯一标识；同一安装应保持稳定。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_dev_nm_upld` | `string` | 成功时 | 客户端设备名称，例如 `iPhone`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_os_ver_upld` | `string` | 成功时 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_dev_mdl_upld` | `string` | 成功时 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_plat_cd_upld` | `string` | 成功时 | 客户端平台，当前支持 `iphone` 或 `android`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_app_ver_upld` | `string` | 成功时 | 客户端展示版本号，例如 `1.0.0`。 |
| `upp_out_dat_upld.upp_dev_ls_upld[].upp_last_in_upld` | `string` | 成功时 | 设备或用户最近登录时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/dev_out:end -->

### 获取邀请码

```http
POST /uppiaing/maycallers_v/ref_gen
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_ref_key_upld":  "A8K29QXZ"
                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/ref_gen:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_ref_key_upld` | `string` | 成功时 | 当前用户自己的长期有效邀请码；不存在时由后端自动生成。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ref_gen:end -->

### 填写邀请码

```http
POST /uppiaing/maycallers_v/ref_use
```

请求体：

```json
{
  "upp_ref_key_upld": "A8K29QXZ"
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/ref_use:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_ref_key_upld` | `string` | 是 | 需要绑定的其他用户邀请码；后端会转为大写后校验。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ref_use:end -->

### 邀请详情

```http
POST /uppiaing/maycallers_v/ref_inf
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_ref_key_upld":  "A8K29QXZ",
                           "upp_ref_ok_upld":  true,
                           "upp_send_cnt_upld":  3,
                           "upp_grant_at_upld":  10800,
                           "upp_ref_sec_upld":  3600,
                           "upp_ref_txt_upld":  "1小时会员",
                           "upp_goal_ls_upld":  {
                                                  "upp_cnt_upld":  24,
                                                  "upp_bonus_sec_upld":  2592000,
                                                  "upp_bonus_tx_upld":  "1个月会员"
                                              }
                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/ref_inf:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_ref_key_upld` | `string` | 成功时 | 当前用户自己的邀请码。 |
| `upp_out_dat_upld.upp_ref_ok_upld` | `boolean` | 成功时 | 当前用户是否仍可填写他人邀请码：未绑定邀请人且仍在允许填写的注册时间范围内时为 `true`。 |
| `upp_out_dat_upld.upp_send_cnt_upld` | `integer` | 成功时 | 用户当前累计成功邀请人数。 |
| `upp_out_dat_upld.upp_grant_at_upld` | `integer` | 成功时 | 用户累计获得的邀请奖励时长，单位为秒。 |
| `upp_out_dat_upld.upp_ref_sec_upld` | `integer` | 成功时 | 每成功邀请一名用户奖励的会员秒数。 |
| `upp_out_dat_upld.upp_ref_txt_upld` | `string` | 成功时 | 每次邀请奖励时长的展示文案。 |
| `upp_out_dat_upld.upp_goal_ls_upld` | `array<object>` | 成功时 | 邀请人数阶梯奖励列表。 |
| `upp_out_dat_upld.upp_goal_ls_upld[].upp_cnt_upld` | `integer` | 成功时 | 获得本阶梯奖励所需的累计成功邀请人数。 |
| `upp_out_dat_upld.upp_goal_ls_upld[].upp_bonus_sec_upld` | `integer` | 成功时 | 达到当前条件后奖励的会员秒数。 |
| `upp_out_dat_upld.upp_goal_ls_upld[].upp_bonus_tx_upld` | `string` | 成功时 | 当前奖励时长的展示文案。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ref_inf:end -->

### 用户信息

```http
GET /uppiaing/maycallers_v/usr_inf
```

响应 `upp_out_dat_upld` 为用户信息。

<!-- endpoint-fields:/uppiaing/maycallers_v/usr_inf:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_obj_no_upld` | `integer` | 成功时 | 用户ID |
| `upp_out_dat_upld.upp_auth_key_upld` | `string` | 成功时 | JWT登录凭证 |
| `upp_out_dat_upld.upp_dev_no_upld` | `string` | 成功时 | 设备唯一标识 |
| `upp_out_dat_upld.upp_login_nm_upld` | `string` | 成功时 | 账号名，游客为空 |
| `upp_out_dat_upld.upp_kind_tp_upld` | `integer` | 成功时 | 用户类型(1=游客,2=账号用户) |
| `upp_out_dat_upld.upp_vip_ok_upld` | `integer` | 成功时 | 会员状态(0=非会员,1=会员有效) |
| `upp_out_dat_upld.upp_vip_end_upld` | `string` | 成功时 | 会员到期时间，格式yyyy-MM-dd HH:mm:ss |
| `upp_out_dat_upld.upp_fresh_usr_upld` | `integer` | 成功时 | 是否新用户(0=否,1=是) |
| `upp_out_dat_upld.upp_plat_cd_upld` | `string` | 成功时 | 客户端平台 |
| `upp_out_dat_upld.upp_app_ver_upld` | `string` | 成功时 | 客户端版本号 |
| `upp_out_dat_upld.upp_sign_cnt_upld` | `integer` | 成功时 | 登录次数 |
| `upp_out_dat_upld.upp_crt_at_upld` | `string` | 成功时 | 用户创建时间 |
| `upp_out_dat_upld.upp_last_in_upld` | `string` | 成功时 | 本次登录时间 |
| `upp_out_dat_upld.upp_sec_key_upld` | `string` | 成功时 | 绑定账号的明文密码，游客为空 |
<!-- endpoint-fields:/uppiaing/maycallers_v/usr_inf:end -->

## 系统接口

### 初始化配置

```http
POST /uppiaing/maycallers_v/app_ini
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_agr_set_upld":  {
                                                "upp_terms_url_upld":  "",
                                                "upp_priv_url_upld":  ""
                                            },
                           "upp_pub_set_upld":  {
                                                     "upp_qr_img_upld":  "",
                                                     "upp_lnk_ls_upld":  [

                                                                        ]
                                                 },
                           "upp_ref_box_upld":  {
                                                   "upp_live_hr_upld":  72,
                                                   "upp_ref_sec_upld":  3600,
                                                   "upp_ref_txt_upld":  "1小时会员",
                                                   "upp_cap_day_upld":  300,
                                                   "upp_goal_ls_upld":  {
                                                                          "upp_cnt_upld":  24,
                                                                          "upp_bonus_sec_upld":  2592000,
                                                                          "upp_bonus_tx_upld":  "1个月会员"
                                                                      }
                                               },
                           "upp_cli_app_upld":  {
                                                 "upp_home_url_upld":  "",
                                                 "upp_trial_sec_upld":  0
                                             },
                           "upp_util_set_upld":  {
                                                    "upp_svc_on_upld":  "off",
                                                    "upp_svc_url_upld":  "",
                                                    "upp_mem_on_upld":  "off",
                                                    "upp_msg_on_upld":  "on"
                                                },
                           "upp_prx_set_upld":  {
                                                    "upp_pass_dom_upld":  [

                                                                             ]
                                                }
                       }
}
```

`upp_util_set_upld` 字段说明：

| 字段           | 说明                                                                                             |
| -------------- | ------------------------------------------------------------------------------------------------ |
| `upp_svc_on_upld` | 在线客服入口开关，`on=展示`，`off=隐藏`                                                          |
| `upp_svc_url_upld`     | 在线客服地址，仅 `upp_svc_on_upld=on` 时使用；后端会把配置模板中的 `#ID` 替换为当前用户 ID 后返回 |
| `upp_mem_on_upld` | 清理内存入口开关，`on=展示`，`off=隐藏`                                                          |
| `upp_msg_on_upld` | 消息通知入口开关，`on=展示`，`off=隐藏`                                                          |

<!-- endpoint-fields:/uppiaing/maycallers_v/app_ini:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 响应状态码 |
| `upp_remms1_upld` | `string` | 是 | 响应消息 |
| `upp_out_dat_upld` | `object` | 是 | 初始化配置 |
| `upp_out_dat_upld.upp_agr_set_upld` | `object` | 成功时 | 协议配置 |
| `upp_out_dat_upld.upp_agr_set_upld.upp_terms_url_upld` | `string` | 成功时 | 用户服务协议地址 |
| `upp_out_dat_upld.upp_agr_set_upld.upp_priv_url_upld` | `string` | 成功时 | 隐私协议地址 |
| `upp_out_dat_upld.upp_pub_set_upld` | `object` | 成功时 | 分享配置 |
| `upp_out_dat_upld.upp_pub_set_upld.upp_qr_img_upld` | `string` | 成功时 | 分享二维码图片地址 |
| `upp_out_dat_upld.upp_pub_set_upld.upp_lnk_ls_upld` | `array<string>` | 成功时 | 分享链接列表 |
| `upp_out_dat_upld.upp_ref_box_upld` | `object` | 成功时 | 邀请配置 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_live_hr_upld` | `integer` | 成功时 | 新用户注册后可填写邀请码的小时数 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_ref_sec_upld` | `integer` | 成功时 | 每邀请一个用户奖励的会员秒数 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_ref_txt_upld` | `string` | 成功时 | 每邀请一个用户奖励的会员时长文案 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_cap_day_upld` | `integer` | 成功时 | 邀请累计最大奖励天数 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_goal_ls_upld` | `array<object>` | 成功时 | 邀请人数阶梯奖励列表 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_goal_ls_upld[].upp_cnt_upld` | `integer` | 成功时 | 达到邀请人数 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_goal_ls_upld[].upp_bonus_sec_upld` | `integer` | 成功时 | 额外奖励会员秒数 |
| `upp_out_dat_upld.upp_ref_box_upld.upp_goal_ls_upld[].upp_bonus_tx_upld` | `string` | 成功时 | 额外奖励会员时长文案 |
| `upp_out_dat_upld.upp_cli_app_upld` | `object` | 成功时 | 应用配置 |
| `upp_out_dat_upld.upp_cli_app_upld.upp_home_url_upld` | `string` | 成功时 | 官网地址 |
| `upp_out_dat_upld.upp_cli_app_upld.upp_trial_sec_upld` | `integer` | 成功时 | 新用户默认赠送会员秒数 |
| `upp_out_dat_upld.upp_util_set_upld` | `object` | 成功时 | 工具入口配置 |
| `upp_out_dat_upld.upp_util_set_upld.upp_svc_on_upld` | `string` | 成功时 | 在线客服入口开关(on=开启,off=关闭) |
| `upp_out_dat_upld.upp_util_set_upld.upp_svc_url_upld` | `string` | 成功时 | 在线客服地址，配置模板中的#ID会替换为当前用户ID |
| `upp_out_dat_upld.upp_util_set_upld.upp_mem_on_upld` | `string` | 成功时 | 清理内存入口开关(on=开启,off=关闭) |
| `upp_out_dat_upld.upp_util_set_upld.upp_msg_on_upld` | `string` | 成功时 | 消息通知入口开关(on=开启,off=关闭) |
| `upp_out_dat_upld.upp_prx_set_upld` | `object` | 成功时 | 代理配置 |
| `upp_out_dat_upld.upp_prx_set_upld.upp_pass_dom_upld` | `array<string>` | 成功时 | 不走代理的域名列表 |
<!-- endpoint-fields:/uppiaing/maycallers_v/app_ini:end -->

### 版本检测

```http
POST /uppiaing/maycallers_v/ver_chk
```

必填请求头：

```http
Uppiaing-PF-Name: iphone
Uppiaing-C-Ver: 1.0.0
Uppiaing-S-Ver: 100
```

无更新：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_new_avl_upld":  false
                       }
}
```

有更新：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_new_avl_upld":  true,
                           "upp_disp_nm_upld":  "iOS 1.0.2",
                           "upp_plat_cd_upld":  "iphone",
                           "upp_app_ver_upld":  "1.0.2",
                           "upp_bld_no_upld":  102,
                           "upp_must_do_upld":  false,
                           "upp_web_url_upld":  "https://example.com/app",
                           "upp_byte_sz_upld":  "23.5MB",
                           "upp_body_tx_upld":  "修复已知问题"
                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/ver_chk:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 响应状态码 |
| `upp_remms1_upld` | `string` | 是 | 响应消息 |
| `upp_out_dat_upld` | `object` | 是 | 版本检测结果 |
| `upp_out_dat_upld.upp_new_avl_upld` | `boolean` | 成功时 | 是否有新版本 |
| `upp_out_dat_upld.upp_disp_nm_upld` | `string` | 成功时 | 版本名称 |
| `upp_out_dat_upld.upp_plat_cd_upld` | `string` | 成功时 | 设备平台(iphone=苹果,android=安卓) |
| `upp_out_dat_upld.upp_app_ver_upld` | `string` | 成功时 | 展示版本号 |
| `upp_out_dat_upld.upp_bld_no_upld` | `integer` | 成功时 | 数字版本号 |
| `upp_out_dat_upld.upp_must_do_upld` | `boolean` | 成功时 | 是否强制更新 |
| `upp_out_dat_upld.upp_web_url_upld` | `string` | 成功时 | 下载地址 |
| `upp_out_dat_upld.upp_byte_sz_upld` | `string` | 成功时 | 安装包大小 |
| `upp_out_dat_upld.upp_body_tx_upld` | `string` | 成功时 | 更新内容 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ver_chk:end -->

### 广告位列表

```http
GET /uppiaing/maycallers_v/ads_lst
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_obj_no_upld":  1,
                           "upp_slot_no_upld":  "banner",
                           "upp_head_tx_upld":  "会员优惠",
                           "upp_body_tx_upld":  "限时开通会员享优惠",
                           "upp_pic_url_upld":  "https://example.com/advert.png",
                           "upp_lnk_url_upld":  "https://example.com",
                           "upp_view_cnt_upld":  1,
                           "upp_cap_show_upld":  3,
                           "upp_ord_idx_upld":  100
                       }
}
```

`upp_slot_no_upld` 固定只有三个值：

| `upp_slot_no_upld`      | 含义         |
| ------------ | ------------ |
| `splash`     | 启动页广告   |
| `home_popup` | 首页弹窗广告 |
| `banner`     | banner 广告  |

<!-- endpoint-fields:/uppiaing/maycallers_v/ads_lst:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld[].upp_obj_no_upld` | `integer` | 成功时 | 广告ID |
| `upp_out_dat_upld[].upp_slot_no_upld` | `string` | 成功时 | 广告位标识(splash=启动页广告,home_popup=首页弹窗广告,banner=banner广告) |
| `upp_out_dat_upld[].upp_head_tx_upld` | `string` | 成功时 | 广告标题 |
| `upp_out_dat_upld[].upp_body_tx_upld` | `string` | 成功时 | 广告内容 |
| `upp_out_dat_upld[].upp_pic_url_upld` | `string` | 成功时 | 广告图片地址 |
| `upp_out_dat_upld[].upp_lnk_url_upld` | `string` | 成功时 | 广告点击后使用浏览器打开的链接地址 |
| `upp_out_dat_upld[].upp_view_cnt_upld` | `integer` | 成功时 | 当前用户已展示次数 |
| `upp_out_dat_upld[].upp_cap_show_upld` | `integer` | 成功时 | 每个用户最大展示次数，0表示不限次数 |
| `upp_out_dat_upld[].upp_ord_idx_upld` | `integer` | 成功时 | 排序值，越大越优先 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ads_lst:end -->

### 全量通知

```http
GET /uppiaing/maycallers_v/ntc_lst
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_pub_ntc_upld":  {
                                                 "upp_itm_ls_upld":  {
                                                                        "upp_obj_no_upld":  1,
                                                                        "upp_head_tx_upld":  "系统维护通知",
                                                                        "upp_body_tx_upld":  "今晚维护",
                                                                        "upp_seen_ok_upld":  0,
                                                                        "upp_crt_at_upld":  "2026-07-06 10:00:00"
                                                                    },
                                                 "upp_all_cnt_upld":  1,
                                                 "upp_new_cnt_upld":  1
                                             },
                           "upp_usr_box_upld":  {
                                                 "upp_itm_ls_upld":  {

                                                                    },
                                                 "upp_all_cnt_upld":  0,
                                                 "upp_new_cnt_upld":  0
                                             }
                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/ntc_lst:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_pub_ntc_upld` | `object` | 成功时 | 系统通知分组。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_obj_no_upld` | `integer` | 成功时 | 通知记录 ID。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_head_tx_upld` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_body_tx_upld` | `string` | 成功时 | 当前记录的正文内容。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_seen_ok_upld` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_crt_at_upld` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_all_cnt_upld` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_new_cnt_upld` | `integer` | 成功时 | 当前未读通知总数。 |
| `upp_out_dat_upld.upp_usr_box_upld` | `object` | 成功时 | 当前用户的个人通知分组。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_obj_no_upld` | `integer` | 成功时 | 通知记录 ID。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_head_tx_upld` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_body_tx_upld` | `string` | 成功时 | 当前记录的正文内容。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_seen_ok_upld` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_crt_at_upld` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_all_cnt_upld` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_new_cnt_upld` | `integer` | 成功时 | 当前未读通知总数。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ntc_lst:end -->

### 未读通知

```http
GET /uppiaing/maycallers_v/ntc_new
```

响应结构同全量通知，只返回未读消息。

<!-- endpoint-fields:/uppiaing/maycallers_v/ntc_new:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_pub_ntc_upld` | `object` | 成功时 | 系统通知分组。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_obj_no_upld` | `integer` | 成功时 | 通知记录 ID。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_head_tx_upld` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_body_tx_upld` | `string` | 成功时 | 当前记录的正文内容。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_seen_ok_upld` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_itm_ls_upld[].upp_crt_at_upld` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_all_cnt_upld` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `upp_out_dat_upld.upp_pub_ntc_upld.upp_new_cnt_upld` | `integer` | 成功时 | 当前未读通知总数。 |
| `upp_out_dat_upld.upp_usr_box_upld` | `object` | 成功时 | 当前用户的个人通知分组。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld` | `array<object>` | 成功时 | 当前接口返回的数据列表。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_obj_no_upld` | `integer` | 成功时 | 通知记录 ID。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_head_tx_upld` | `string` | 成功时 | 当前广告、通知或弹窗的标题。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_body_tx_upld` | `string` | 成功时 | 当前记录的正文内容。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_seen_ok_upld` | `integer` | 成功时 | 通知是否已读：`0` 未读，`1` 已读。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_itm_ls_upld[].upp_crt_at_upld` | `string` | 成功时 | 记录创建时间，格式为 `yyyy-MM-dd HH:mm:ss`。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_all_cnt_upld` | `integer` | 成功时 | 当前查询结果的总数量。 |
| `upp_out_dat_upld.upp_usr_box_upld.upp_new_cnt_upld` | `integer` | 成功时 | 当前未读通知总数。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ntc_new:end -->

### 获取弹窗

```http
GET /uppiaing/maycallers_v/pop_get
```

无弹窗时：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {

                       }
}
```

有弹窗时：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_obj_no_upld":  1,
                           "upp_head_tx_upld":  "会员活动",
                           "upp_body_tx_upld":  "限时开通会员享优惠",
                           "upp_pic_url_upld":  [
                                                   "https://example.com/popup.png"
                                               ],
                           "upp_nav_tp_upld":  "internal",
                           "upp_nav_dst_upld":  "purchase",
                           "upp_cls_ok_upld":  1,
                           "upp_view_cnt_upld":  1,
                           "upp_cap_show_upld":  3
                       }
}
```

后台配置 `upp_pic_url_upld` 可填写单个 URL、英文逗号分隔的多个 URL，或 JSON 字符串数组，接口始终返回 JSON URL 数组。`upp_cls_ok_upld` 取值：`1=可关闭`、`0=不可关闭`。

`upp_nav_tp_upld` 取值：

| 值       | 说明       |
| -------- | ---------- |
| none     | 不跳转     |
| internal | App 内跳转 |
| external | 浏览器打开 |

<!-- endpoint-fields:/uppiaing/maycallers_v/pop_get:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_obj_no_upld` | `integer` | 成功时 | 弹窗ID |
| `upp_out_dat_upld.upp_head_tx_upld` | `string` | 成功时 | 弹窗标题 |
| `upp_out_dat_upld.upp_body_tx_upld` | `string` | 成功时 | 弹窗内容 |
| `upp_out_dat_upld.upp_pic_url_upld` | `array<string>` | 成功时 | 弹窗图片地址数组 |
| `upp_out_dat_upld.upp_nav_tp_upld` | `string` | 成功时 | 跳转方式(none=不跳转,internal=内部跳转,external=外部浏览器) |
| `upp_out_dat_upld.upp_nav_dst_upld` | `string` | 成功时 | 跳转目标，内部跳转填业务code，外部跳转填URL |
| `upp_out_dat_upld.upp_cls_ok_upld` | `integer` | 成功时 | 是否可关闭(0=不可关闭,1=可关闭) |
| `upp_out_dat_upld.upp_view_cnt_upld` | `integer` | 成功时 | 当前用户已展示次数 |
| `upp_out_dat_upld.upp_cap_show_upld` | `integer` | 成功时 | 每个用户最大展示次数，0表示不限次数 |
<!-- endpoint-fields:/uppiaing/maycallers_v/pop_get:end -->

### 延迟迁移弹窗

```http
GET /uppiaing/maycallers_v/pop_dly
```

客户端成功请求后保存到本地，当连续断网达到 `upp_wait_day_upld` 天后展示给用户，用于引导用户转移到新软件。

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_obj_no_upld":  1,
                           "upp_head_tx_upld":  "服务迁移提醒",
                           "upp_body_tx_upld":  "如果当前软件长时间无法连接，请使用转移码前往新软件兑换会员权益",
                           "upp_pic_url_upld":  [
                                                   "/uppiaing/maycallers_v/fil_get/pop.png"
                                               ],
                           "upp_lnk_url_upld":  "https://example.com/download",
                           "upp_cls_ok_upld":  1,
                           "upp_wait_day_upld":  3,
                           "upp_move_key_upld":  "origin8f3k9q"
                       }
}
```

延迟弹窗的 `upp_pic_url_upld` 规则与普通弹窗一致，始终返回 JSON URL 数组。`upp_cls_ok_upld` 取值：`1=可关闭`、`0=不可关闭`。

无可用配置时：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {

                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/pop_dly:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_obj_no_upld` | `integer` | 成功时 | 弹窗ID |
| `upp_out_dat_upld.upp_head_tx_upld` | `string` | 成功时 | 弹窗标题 |
| `upp_out_dat_upld.upp_body_tx_upld` | `string` | 成功时 | 弹窗内容 |
| `upp_out_dat_upld.upp_pic_url_upld` | `array<string>` | 成功时 | 弹窗图片地址数组 |
| `upp_out_dat_upld.upp_lnk_url_upld` | `string` | 成功时 | 弹窗跳转链接 |
| `upp_out_dat_upld.upp_cls_ok_upld` | `integer` | 成功时 | 是否可关闭(0=不可关闭,1=可关闭) |
| `upp_out_dat_upld.upp_wait_day_upld` | `integer` | 成功时 | 断网后延迟展示天数 |
| `upp_out_dat_upld.upp_move_key_upld` | `string` | 成功时 | 当前用户转移码 |
<!-- endpoint-fields:/uppiaing/maycallers_v/pop_dly:end -->

### 标记通知已读

```http
POST /uppiaing/maycallers_v/ntc_ack
```

请求体：

```json
{
  "upp_pub_ids_upld": [1, 2],
  "upp_usr_ids_upld": [3, 4]
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/ntc_ack:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_pub_ids_upld` | `array<integer>` | 否 | 需要标记为已读的系统通知 ID 数组。 |
| `upp_usr_ids_upld` | `array<integer>` | 否 | 需要标记为已读的个人通知 ID 数组。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ntc_ack:end -->

## 上报接口

### 错误上报

```http
POST /uppiaing/maycallers_v/err_log
```

请求体：

```json
{
  "upp_err_tx_upld": "vpn connect timeout",
  "upp_os_ver_upld": "iOS 17.5",
  "upp_dev_mdl_upld": "iPhone 15",
  "upp_err_tp_upld": "vpn"
}
```

`upp_err_tp_upld` 可选：

| 值  | 说明     |
| --- | -------- |
| app | App 错误 |
| vpn | VPN 错误 |

<!-- endpoint-fields:/uppiaing/maycallers_v/err_log:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_err_tx_upld` | `string` | 是 | 客户端需要上报的错误详情，例如 VPN 连接超时或配置解析失败。 |
| `upp_os_ver_upld` | `string` | 是 | 客户端操作系统版本，例如 `iOS 17.5`。 |
| `upp_dev_mdl_upld` | `string` | 是 | 客户端设备型号，例如 `iPhone 15`。 |
| `upp_err_tp_upld` | `string` | 否 | 错误分类：`upp_cli_app_upld` 表示 App 错误，`vpn` 表示 VPN 错误；未传或空值时默认 `upp_cli_app_upld`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/err_log:end -->

## 套餐接口

### 套餐列表

```http
GET /uppiaing/maycallers_v/pkg_lst
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_obj_no_upld":  1,
                           "upp_ios_sku_upld":  "vip_month",
                           "upp_disp_nm_upld":  "月度会员",
                           "upp_alt_nm_upld":  "连续30天高速线路",
                           "upp_pick_ok_upld":  1,
                           "upp_tag_tx_upld":  "推荐",
                           "upp_fee_upld":  1990,
                           "upp_fee_txt_upld":  "¥19.9",
                           "upp_list_amt_upld":  "¥29.9",
                           "upp_raw_v_upld":  "畅享全部会员线路",
                           "upp_note_tx_upld":  "适合短期使用",
                           "upp_dur_day_upld":  30
                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/pkg_lst:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld[].upp_obj_no_upld` | `integer` | 成功时 | 套餐ID |
| `upp_out_dat_upld[].upp_ios_sku_upld` | `string` | 成功时 | 苹果内购产品ID |
| `upp_out_dat_upld[].upp_disp_nm_upld` | `string` | 成功时 | 套餐名称 |
| `upp_out_dat_upld[].upp_alt_nm_upld` | `string` | 成功时 | 套餐副标题 |
| `upp_out_dat_upld[].upp_pick_ok_upld` | `integer` | 成功时 | 是否默认选中(1=选中,0=未选中) |
| `upp_out_dat_upld[].upp_tag_tx_upld` | `string` | 成功时 | 角标文案 |
| `upp_out_dat_upld[].upp_fee_upld` | `integer` | 成功时 | 真实价格，单位分 |
| `upp_out_dat_upld[].upp_fee_txt_upld` | `string` | 成功时 | 展示价格文案 |
| `upp_out_dat_upld[].upp_list_amt_upld` | `string` | 成功时 | 原价文案 |
| `upp_out_dat_upld[].upp_raw_v_upld` | `string` | 成功时 | 底部描述内容 |
| `upp_out_dat_upld[].upp_note_tx_upld` | `string` | 成功时 | 套餐描述 |
| `upp_out_dat_upld[].upp_dur_day_upld` | `integer` | 成功时 | 套餐天数 |
<!-- endpoint-fields:/uppiaing/maycallers_v/pkg_lst:end -->

## 支付接口

### 获取公共 Apple ID

```http
GET /uppiaing/maycallers_v/ios_pub
```

无需登录。用于获取可用的公共 Apple ID 列表，每个 IP 24 小时内返回同一组账号。

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_obj_no_upld":  1,
                           "upp_ios_acc_upld":  "apple@example.com",
                           "upp_sec_key_upld":  "apple-password"
                       }
}
```

字段说明：

| 字段          | 说明          |
| ------------- | ------------- |
| `upp_obj_no_upld`      | 记录 ID       |
| `upp_ios_acc_upld`  | Apple ID 账号 |
| `upp_sec_key_upld` | Apple ID 密码 |

<!-- endpoint-fields:/uppiaing/maycallers_v/ios_pub:start -->
#### 请求字段说明

鉴权：公开接口，无需登录 token。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld[].upp_obj_no_upld` | `integer` | 成功时 | 记录ID |
| `upp_out_dat_upld[].upp_ios_acc_upld` | `string` | 成功时 | Apple ID账号 |
| `upp_out_dat_upld[].upp_sec_key_upld` | `string` | 成功时 | Apple ID密码 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ios_pub:end -->

### 发起支付

```http
POST /uppiaing/maycallers_v/pay_ini
```

请求体：

```json
{
  "upp_plan_no_upld": 1
}
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_kind_tp_upld":  "apple_iap",
                           "upp_aim_no_upld":  "com.lamp.app.premium.7days",
                           "upp_acc_tkn_upld":  "C56A4180-65AA-42EC-A945-5FD21DEC0538"
                       }
}
```

或：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_kind_tp_upld":  "h5",
                           "upp_aim_no_upld":  "https://pay.example.com/order/xxx",
                           "upp_acc_tkn_upld":  ""
                       }
}
```

字段说明：

| 字段            | 说明                                                                                                      |
| --------------- | --------------------------------------------------------------------------------------------------------- |
| `upp_kind_tp_upld`      | 支付方式，`apple_iap=苹果内购`，`h5=三方 H5 支付`                                                         |
| `upp_aim_no_upld`       | `apple_iap` 时返回 Apple 商品 ID，`h5` 时返回浏览器跳转支付地址                                           |
| `upp_acc_tkn_upld` | `apple_iap` 时返回该订单对应的 UUID，前端调起 StoreKit 时必须原样作为 `appAccountToken` 携带；`h5` 时为空 |

苹果内购注意事项：

| 事项      | 说明                                                                           |
| --------- | ------------------------------------------------------------------------------ |
| 商品 ID   | 使用 `upp_aim_no_upld` 调起内购                                                      |
| 订单 UUID | 使用 `upp_acc_tkn_upld` 调起内购，必须传给 StoreKit 的 `appAccountToken`        |
| 验单      | 支付完成后调用 `POST /uppiaing/maycallers_v/ios_vfy`，只提交 `upp_deal_no_upld` |
| 匹配订单  | 后端通过苹果交易中的 `appAccountToken` 找到本次发起支付创建的订单              |

判断规则：

| 规则        | 说明                                                                    |
| ----------- | ----------------------------------------------------------------------- |
| 客户端时区  | 请求头 `Uppiaing-TZ-Off` 不是中国大陆时区或未传时，直接返回 `apple_iap`；该规则优先级最高 |
| 仅内购地区  | 用户当前 IP 地区命中配置时返回 `apple_iap`                              |
| 仅内购版本  | 请求头 `Uppiaing-C-Ver` 命中配置时返回 `apple_iap`                       |
| H5 开放金额 | 今日苹果内购收款额度未达到配置金额时返回 `apple_iap`，达到后可返回 `h5` |
| 默认        | 不命中以上限制时返回 `h5`                                               |

中国大陆时区包括 `Asia/Shanghai`、`Asia/Chongqing`、`Asia/Harbin`、`Asia/Urumqi` 等中国大陆 IANA 时区；`Asia/Hong_Kong`、`Asia/Macau`、`Asia/Taipei` 等不属于中国大陆时区。

当支付配置开启“已成功三方支付用户直接 H5”后，中国大陆时区内已存在成功三方支付订单的用户会直接返回 `h5`，该规则优先于地区、版本和今日苹果内购金额限制。

<!-- endpoint-fields:/uppiaing/maycallers_v/pay_ini:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_plan_no_upld` | `integer` | 是 | 套餐记录 ID，必须使用套餐列表接口返回且当前已上架的有效 ID。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_kind_tp_upld` | `string` | 成功时 | 支付方式(apple_iap=苹果内购,h5=H5支付) |
| `upp_out_dat_upld.upp_aim_no_upld` | `string` | 成功时 | 支付目标，内购返回apple_id，H5返回支付页面地址 |
| `upp_out_dat_upld.upp_acc_tkn_upld` | `string` | 成功时 | 苹果内购订单标识，内购时前端作为appAccountToken传给StoreKit |
<!-- endpoint-fields:/uppiaing/maycallers_v/pay_ini:end -->

### 苹果订单验证

```http
POST /uppiaing/maycallers_v/ios_vfy
```

客户端完成苹果内购后提交苹果交易号。后端会通过苹果交易信息中的商品 ID 和 `appAccountToken` 匹配发起支付时创建的订单，验证成功后发放会员权益。

请求体：

```json
{
  "upp_deal_no_upld": "1000000000000000"
}
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_ord_no_upld":  "20260707120000123456",
                           "upp_pay_st_upld":  3,
                           "upp_pay_tp_upld":  "apple_iap",
                           "upp_deal_no_upld":  "1000000000000000",
                           "upp_vip_end_upld":  "2026-08-06 12:00:00"
                       }
}
```

字段说明：

| 字段             | 说明                             |
| ---------------- | -------------------------------- |
| `upp_ord_no_upld`       | 后端订单号                       |
| `upp_pay_st_upld`     | 支付状态，`1=未支付`，`3=已支付` |
| `upp_pay_tp_upld`       | 支付方式                         |
| `upp_deal_no_upld` | 苹果交易号                       |
| `upp_vip_end_upld`      | 会员到期时间                     |

<!-- endpoint-fields:/uppiaing/maycallers_v/ios_vfy:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_deal_no_upld` | `string` | 是 | Apple 支付成功后返回的交易号，不能为空。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_ord_no_upld` | `string` | 成功时 | 订单号 |
| `upp_out_dat_upld.upp_pay_st_upld` | `integer` | 成功时 | 支付状态(1=未支付,3=已支付) |
| `upp_out_dat_upld.upp_pay_tp_upld` | `string` | 成功时 | 支付方式 |
| `upp_out_dat_upld.upp_deal_no_upld` | `string` | 成功时 | 苹果交易号 |
| `upp_out_dat_upld.upp_vip_end_upld` | `string` | 成功时 | 会员到期时间 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ios_vfy:end -->

## VPN 接口

### 线路区域列表

```http
POST /uppiaing/maycallers_v/loc_lst
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  [{
                           "upp_ctry_upld":  "香港",
                           "upp_reg_cd_upld":  "HK",
                           "upp_min_sec_upld":  100,
                           "upp_max_sec_upld":  300
                       }]
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/loc_lst:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `array<object>` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld[].upp_ctry_upld` | `string` | 成功时 | 国家或地区的展示名称。 |
| `upp_out_dat_upld[].upp_reg_cd_upld` | `string` | 成功时 | 线路代码，例如 `HK` 或 `AUTO`。 |
| `upp_out_dat_upld[].upp_min_sec_upld` | `integer` | 成功时 | 线路建议的最小连接耗时阈值，单位为毫秒。 |
| `upp_out_dat_upld[].upp_max_sec_upld` | `integer` | 成功时 | 线路建议的最大连接耗时阈值，单位为毫秒。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/loc_lst:end -->

### 节点返回模式

母版当前默认实现三种节点返回方式：

| 接口 | 返回内容 | 状态 |
| --- | --- | --- |
| `POST /uppiaing/maycallers_v/nod_get` | 加密节点 URL | 母版和现有 VPN 项目已实现 |
| `POST /uppiaing/maycallers_v/nod_cfg` | 加密的完整 JSON 配置 | 母版和大部分项目已实现，少数老项目尚无 |
| `POST /uppiaing/maycallers_v/nod_egr` | 分别加密的节点 URL 和 `upp_eg_ls_upld` JSON | 已实现 |

三种方式的节点选择、会员校验、审核节点和临时节点记录语义必须一致。前端只能调用当前产品文档和 Swagger 中实际存在的接口。

### 获取节点

```http
POST /uppiaing/maycallers_v/nod_get
```

请求体：

```json
{
  "upp_reg_cd_upld": "HK"
}
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_lnk_url_upld":  "x/k5A0v9kiJjL0r3m6X9dA=="
                       }
}
```

说明：

| 规则     | 说明                                                                          |
| -------- | ----------------------------------------------------------------------------- |
| 节点链接 | `upp_lnk_url_upld` 是加密后的节点连接数据，所有节点类型都按字符串返回               |
| 解密流程 | 客户端先对返回字符串做标准 Base64 解码得到 AES 密文，再以 AES-CBC/PKCS7 解密得到内部 Base64 字符串，最后再次标准 Base64 解码得到原始节点内容 |
| 加密 key | 后端配置提供，前端使用项目约定的节点解密 key                                  |
| 免费线路 | `FREE` 或 `FREE_` 开头的线路不校验会员                                        |
| 会员线路 | 需要会员未过期，未开通会员或会员已过期时返回 `upp_reccd1_upld=901`                 |
| 连接确认 | 成功拿到节点后，需要连接成功再调用在线接口                                    |

会员过期响应：

```json
{
    "upp_reccd1_upld":  901,
    "upp_remms1_upld":  "未开通会员或会员已过期，请开通后使用",
    "upp_out_dat_upld":  {

                       }
}
```

<!-- endpoint-fields:/uppiaing/maycallers_v/nod_get:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_reg_cd_upld` | `string` | 是 | 线路代码，例如 `HK` 或 `AUTO`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_lnk_url_upld` | `string` | 成功时 | 加密节点 URL；先对该字符串做标准 Base64 解码得到 AES 密文，再执行 AES-CBC/PKCS7 解密，最后对解密所得内部 Base64 字符串再次解码得到原始节点 URL。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/nod_get:end -->

### 获取节点 URL 和 outbounds

```http
POST /uppiaing/maycallers_v/nod_egr
```

请求体：

```json
{
  "upp_reg_cd_upld": "HK"
}
```

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_lnk_url_upld":  "x/k5A0v9kiJjL0r3m6X9dA==",
                           "upp_eg_ls_upld":  "x/k5A0v9kiJjL0r3m6X9dA=="
                       }
}
```

| 字段 | 类型 | 必填/必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reg_cd_upld` | string | 必填 | 线路代码，与其他两种节点接口一致 |
| `upp_lnk_url_upld` | string | 必返 | 与 `/uppiaing/maycallers_v/nod_get` 相同的加密节点 URL；先做标准 Base64 解码，再执行 AES-CBC/PKCS7 解密，最后再次标准 Base64 解码 |
| `upp_eg_ls_upld` | string | 必返 | 独立加密的 JSON 数组，必须与 `upp_lnk_url_upld` 分别解密；解密方式相同，解密后与 `/uppiaing/maycallers_v/nod_cfg` 完整配置中的 `upp_eg_ls_upld` 完全一致 |

<!-- endpoint-fields:/uppiaing/maycallers_v/nod_egr:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_reg_cd_upld` | `string` | 是 | 线路代码，例如 `HK` 或 `AUTO`。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_lnk_url_upld` | `string` | 成功时 | 独立加密的节点 URL；完整解密流程为外层标准 Base64 解码、AES-CBC/PKCS7 解密、内部标准 Base64 解码。 |
| `upp_out_dat_upld.upp_eg_ls_upld` | `string` | 成功时 | 独立加密的 outbounds JSON 数组；必须单独完成与 `upp_out_dat_upld.upp_lnk_url_upld` 相同的完整解密流程。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/nod_egr:end -->

### 获取 JSON 节点配置

```http
POST /uppiaing/maycallers_v/nod_cfg
```

请求体：

```json
{
  "upp_reg_cd_upld": "HK",
  "upp_kind_tp_upld": "fast"
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `upp_reg_cd_upld` | 线路代码 |
| `upp_kind_tp_upld` | 连接模式，`fast` 或 `极速` 表示极速模式，`global` 或 `全局` 表示全局模式 |

响应：

```json
{
    "upp_reccd1_upld":  200,
    "upp_remms1_upld":  "success",
    "upp_out_dat_upld":  {
                           "upp_opt_txt_upld":  "x/k5A0v9kiJjL0r3m6X9dA=="
                       }
}
```

`upp_opt_txt_upld` 是完整 JSON 节点配置的加密结果，解密流程与 `upp_lnk_url_upld` 相同：先对返回字符串做标准 Base64 解码得到 AES 密文，再以 AES-CBC/PKCS7 解密得到内部 Base64 字符串，最后再次标准 Base64 解码得到 JSON 配置。该接口与 `/uppiaing/maycallers_v/nod_get` 使用相同的线路选择、会员校验和临时节点记录逻辑。

<!-- endpoint-fields:/uppiaing/maycallers_v/nod_cfg:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_reg_cd_upld` | `string` | 是 | 线路代码 |
| `upp_kind_tp_upld` | `string` | 是 | 连接模式(fast=极速,global=全局) |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_opt_txt_upld` | `string` | 成功时 | 加密的完整 JSON 节点配置；完整解密流程为外层标准 Base64 解码、AES-CBC/PKCS7 解密、内部标准 Base64 解码。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/nod_cfg:end -->

### 确认已连接

```http
POST /uppiaing/maycallers_v/ses_onl
```

客户端成功建立 VPN 后调用。

<!-- endpoint-fields:/uppiaing/maycallers_v/ses_onl:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ses_onl:end -->

### 心跳

```http
POST /uppiaing/maycallers_v/ses_hbt
```

连接中定时调用，用于刷新在线状态。

客户端应每 30 秒调用一次。服务端在线状态记录有效期为 90 秒；记录过期或不存在时，只要会员仍有效，心跳仍返回继续连接，并重新等待后续连接状态上报。只有会员失效时才返回断开指令。

响应 `upp_out_dat_upld.upp_ses_st_upld` 表示连接指令：`1` 继续连接，`0` 断开连接。仅会员过期时返回 `0`；当前没有连接记录但会员有效时仍返回 `1`，避免连接成功上报失败导致客户端被心跳断开。

<!-- endpoint-fields:/uppiaing/maycallers_v/ses_hbt:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
| `upp_out_dat_upld.upp_ses_st_upld` | `integer` | 成功时 | 连接状态(1=继续连接,0=断开连接) |
<!-- endpoint-fields:/uppiaing/maycallers_v/ses_hbt:end -->

### 断开连接

```http
POST /uppiaing/maycallers_v/ses_off
```

主动断开 VPN 时调用。

<!-- endpoint-fields:/uppiaing/maycallers_v/ses_off:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| （无） | - | - | 本接口没有 JSON 请求字段。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/ses_off:end -->

### 流量上报

```http
POST /uppiaing/maycallers_v/net_flo
```

请求体：

```json
{
  "upp_traf_sz_upld": 1024
}
```

`upp_traf_sz_upld` 单位为 K，必须大于 0。

<!-- endpoint-fields:/uppiaing/maycallers_v/net_flo:start -->
#### 请求字段说明

鉴权：需要携带 `Authorization: Bearer <token>`。

| 字段 | 类型 | 是否必填 | 说明 |
| --- | --- | --- | --- |
| `upp_traf_sz_upld` | `integer` | 是 | 本次上报的 VPN 使用流量，单位为 K，必须大于 0。 |

#### 响应字段说明

| 字段 | 类型 | 是否必返 | 说明 |
| --- | --- | --- | --- |
| `upp_reccd1_upld` | `integer` | 是 | 业务状态码：`200` 成功，`401` 登录状态失效，节点接口 `901` 表示无有效会员，`500` 表示业务失败。 |
| `upp_remms1_upld` | `string` | 是 | 业务提示信息；失败时前端可按产品交互展示该文案。 |
| `upp_out_dat_upld` | `object` | 是 | 接口业务数据；具体结构见下方以该字段开头的嵌套字段。 |
<!-- endpoint-fields:/uppiaing/maycallers_v/net_flo:end -->

<!-- detailed-client-flows:start -->
## 推荐调用流程

本节描述前端实际接入顺序、状态保存和异常分支。字段名和路径均为当前产品的真实映射值，可直接用于客户端实现。

### 所有流程共同遵守的请求规则

每次请求都应携带当前平台和版本环境：`Uppiaing-PF-Name`、`Uppiaing-C-Ver`、`Uppiaing-S-Ver`、`Uppiaing-D-C`、`Uppiaing-Time`、`Uppiaing-TZ-Off`。可选扩展请求头为：`Uppiaing-Reg`（App Store 三位地区代码，例如 `CHN`）、`Uppiaing-Lang-Code`（中文可传 `zh-Hans`、`zh-Hant`、`zh-CN`、`zh-TW` 等 `zh` 标识，其他语言按英文处理）。可选头未携带时不能阻止正常请求。

除自动登录等公开接口外，鉴权接口必须携带 `Authorization: Bearer <token>`。前端收到响应后先读取 `upp_reccd1_upld`：`200` 才处理数据；`401` 清除旧 Token 并重新自动登录；`500` 展示 `upp_remms1_upld`；获取节点接口的 `901` 只表示没有有效 VIP，应进入购买或续费页面，不能刷新 Token。网络超时、DNS、Nginx `502` 等没有形成正常业务 JSON 的情况按网络错误处理。

### 应用首次启动与日常启动

| 阶段 | 触发条件与请求 | 成功后前端处理 | 异常与注意事项 |
| --- | --- | --- | --- |
| 建立登录态 | 应用首次安装、本地没有 Token，或鉴权接口返回业务码 `401` 时，调用 `POST /uppiaing/maycallers_v/gst_ent`。请求体至少传 `upp_dev_no_upld`；平台、版本、设备名称、系统版本、设备型号和来源渠道按该接口字段表传递。 | 保存 `upp_out_dat_upld.upp_auth_key_upld`，并缓存 `upp_out_dat_upld.upp_obj_no_upld`、`upp_out_dat_upld.upp_vip_ok_upld`、`upp_out_dat_upld.upp_vip_end_upld`。同一安装应始终使用稳定的设备号，避免重复创建游客。 | 自动登录本身失败时展示业务提示或网络重试；不要拿已经收到 `401` 的旧 Token 循环重试原接口。 |
| 获取运营配置 | 已取得 Token 后调用 `POST /uppiaing/maycallers_v/app_ini`。 | 缓存协议、分享、邀请、官网、试用时长、客服入口、工具开关和代理白名单。后端配置可热更新，因此每次冷启动应重新获取。 | 单个可选配置为空时使用客户端默认值，不应把空配置当成登录失败。 |
| 检查版本 | 调用 `POST /uppiaing/maycallers_v/ver_chk`，构建号从请求头 `Uppiaing-S-Ver` 读取。 | 根据返回的是否更新、是否强制、版本号、下载地址、大小和更新内容决定是否展示更新界面。 | 没有匹配的更新版本时返回空结果属于正常情况；强制更新时应阻止继续进入主界面。 |
| 获取首页内容 | 按页面需要调用 `GET /uppiaing/maycallers_v/pop_get`、`GET /uppiaing/maycallers_v/pop_dly`、`GET /uppiaing/maycallers_v/ads_lst` 和 `GET /uppiaing/maycallers_v/ntc_new`。 | 普通弹窗按后端返回直接展示；延迟弹窗按返回延迟控制；未读数用于消息角标；广告按启用状态和展示位置渲染。 | 返回空对象或空数组通常表示当前没有可展示内容，不应提示系统错误。弹窗展示次数和新用户、地区限制由后端判断。 |

日常启动如果本地已有 Token，可以先直接调用鉴权接口；只有收到业务码 `401` 时才重新自动登录并替换 Token。不要仅因为应用重启就丢弃仍有效的 Token。

### VPN 线路选择、连接、心跳与断开

#### 获取线路并选择节点返回格式

进入线路页面后调用 `POST /uppiaing/maycallers_v/loc_lst`。使用 `upp_out_dat_upld[].upp_ctry_upld` 展示线路名称，提交获取节点请求时必须原样使用同一项的 `upp_out_dat_upld[].upp_reg_cd_upld`，例如 `HK` 或 `AUTO`，不能根据展示文案自行拼接代码。

客户端应根据自身实现固定选择下面一种节点接口；一次正常连接不需要把三种接口全部调用一遍。三种接口共用会员校验、审核版本、试用节点和负载均衡逻辑，区别只在返回格式。

| 客户端能力 | 请求接口与请求体 | 成功响应的使用方式 |
| --- | --- | --- |
| 客户端自行拼接完整配置 | 调用 `POST /uppiaing/maycallers_v/nod_get`，请求体传 `upp_reg_cd_upld`。 | 解密 `upp_out_dat_upld.upp_lnk_url_upld` 得到原始节点 URL，再由客户端生成完整配置。 |
| 客户端只拼接配置其余部分 | 调用 `POST /uppiaing/maycallers_v/nod_egr`，请求体传 `upp_reg_cd_upld`。 | 分别解密 `upp_out_dat_upld.upp_lnk_url_upld` 和 `upp_out_dat_upld.upp_eg_ls_upld`；后者是可直接嵌入配置的 `upp_eg_ls_upld` JSON 数组。 |
| 客户端直接使用完整 JSON | 调用 `POST /uppiaing/maycallers_v/nod_cfg`，请求体传 `upp_reg_cd_upld` 和 `upp_kind_tp_upld`；连接模式使用 `fast`/`极速` 或 `global`/`全局`。 | 解密 `upp_out_dat_upld.upp_opt_txt_upld` 得到完整 JSON 配置并交给连接内核。 |

所有加密节点数据都使用当前产品约定的节点密钥和既有 AES-CBC/PKCS7 + Base64 流程。前端不得把密钥、解密后的节点 URL 或完整配置写入用户可见日志。业务码 `901` 时停止连接并进入会员购买提示；业务码 `401` 时先自动登录再重新获取节点；业务码 `500` 时展示服务端提示。

#### 建立连接后的状态闭环

| 连接状态 | 前端必须调用 | 服务端响应与前端动作 |
| --- | --- | --- |
| VPN 内核尚未确认连通 | 暂时不要调用 `POST /uppiaing/maycallers_v/ses_onl`。如果内核直接失败，可按产品交互调用 `POST /uppiaing/maycallers_v/err_log` 上报错误信息。 | 获取到节点不等于连接成功，不能提前产生在线记录。 |
| VPN 内核已确认连通 | 立即调用 `POST /uppiaing/maycallers_v/ses_onl`。 | 业务码 `200` 后开始心跳；如果该上报因瞬时网络问题失败，只要会员仍有效，后续心跳没有在线记录时也会返回继续连接。 |
| VPN 正在连接 | 每 30 秒调用 `POST /uppiaing/maycallers_v/ses_hbt`。 | `upp_out_dat_upld.upp_ses_st_upld=1` 时保持连接；`upp_out_dat_upld.upp_ses_st_upld=0` 时立即让内核断开。当前逻辑只有会员失效才要求断开。 |
| 用户主动断开或内核结束 | 调用 `POST /uppiaing/maycallers_v/ses_off`，并停止心跳定时器。 | 即使断开上报失败，也必须先清理客户端本地连接状态，后续可在有网络时记录错误。 |
| 产生可统计流量 | 在适当的周期或断开前调用 `POST /uppiaing/maycallers_v/net_flo`，请求体 `upp_traf_sz_upld` 传本次增量流量。 | 单位为 K，值必须大于 `0`；不要重复累计上报同一段流量。 |

### 套餐购买、苹果验单与会员状态刷新

| 阶段 | 请求与判断 | 前端处理 |
| --- | --- | --- |
| 展示套餐 | 调用 `GET /uppiaing/maycallers_v/pkg_lst`。 | 使用返回数组渲染套餐；用户选择后保存该项 `upp_out_dat_upld[].upp_obj_no_upld`。发起支付时只能提交接口实际返回且当前上架的套餐 ID，不能把数组下标当套餐 ID。 |
| 创建订单并决定支付渠道 | 调用 `POST /uppiaing/maycallers_v/pay_ini`，请求体 `upp_plan_no_upld` 传选中的套餐 ID。支付渠道由后端根据时区、IP 地区、版本和支付配置决定。 | 读取 `upp_out_dat_upld.upp_kind_tp_upld`，不要由客户端自行判断走苹果还是 H5。 |
| 返回 `apple_iap` | `upp_out_dat_upld.upp_aim_no_upld` 是 Apple 商品 ID，`upp_out_dat_upld.upp_acc_tkn_upld` 是本次后端订单 UUID。 | 使用商品 ID 调起 StoreKit，并把订单 UUID 原样作为 `appAccountToken`；支付成功取得苹果交易号后调用 `POST /uppiaing/maycallers_v/ios_vfy`，请求体传 `upp_deal_no_upld`。验单业务码 `200` 后，以 `upp_out_dat_upld.upp_vip_end_upld` 更新界面，并再调用用户信息接口校准会员状态。 |
| 返回 `h5` | `upp_out_dat_upld.upp_aim_no_upld` 是三方支付页面 URL，`upp_out_dat_upld.upp_acc_tkn_upld` 为空。 | 用系统浏览器或约定 WebView 打开 URL。支付成功由三方服务端回调后端发放会员，客户端不要调用支付回调接口；用户返回 App 后调用 `GET /uppiaing/maycallers_v/usr_inf` 刷新 `upp_out_dat_upld.upp_vip_ok_upld` 和 `upp_out_dat_upld.upp_vip_end_upld`。 |
| 支付失败或取消 | 苹果取消、验单失败、H5 页面关闭或业务码 `500`。 | 保持原会员状态；`500` 展示 `upp_remms1_upld`。允许用户重新发起支付，但不要复用上一笔的 `upp_out_dat_upld.upp_acc_tkn_upld` 或苹果交易号。 |

苹果和三方支付回调是服务端对服务端接口，不属于客户端接入范围。客户端只调用发起支付与苹果验单两个接口。

### 游客、注册、账号登录、设备管理与注销

| 用户动作 | 调用方式 | Token 与本地状态处理 |
| --- | --- | --- |
| 游客首次进入 | 调用 `POST /uppiaing/maycallers_v/gst_ent`，设备号使用稳定值。 | 保存 `upp_out_dat_upld.upp_auth_key_upld`。后端可能创建新游客，也可能找到已有设备用户；前端不需要区分创建接口。 |
| 游客注册账号 | 在已有游客 Token 下调用 `POST /uppiaing/maycallers_v/acc_reg`，提交 `upp_dev_no_upld`、`upp_login_nm_upld`、`upp_sec_key_upld`。账号和密码均为 6–20 位字母或数字。 | 业务码 `200` 后必须用响应中的 `upp_out_dat_upld.upp_auth_key_upld` 覆盖本地 Token，并刷新用户类型、账号名和会员状态。 |
| 已有账号登录当前设备 | 调用 `POST /uppiaing/maycallers_v/acc_ent`，至少提交 `upp_dev_no_upld`、`upp_login_nm_upld`、`upp_sec_key_upld`，其余设备环境字段按接口字段表传递。 | 业务码 `200` 后用 `upp_out_dat_upld.upp_auth_key_upld` 覆盖本地 Token。账号不存在、密码错误、设备数达到上限等返回 `500`，直接展示提示。 |
| 刷新个人资料 | 调用 `GET /uppiaing/maycallers_v/usr_inf`。 | 使用服务端返回覆盖会员状态和到期时间；支持 App Store 地区或语言记录的产品也会在此接口按非空有效请求头更新用户资料。 |
| 查看或移除其他设备 | 调用 `POST /uppiaing/maycallers_v/dev_inf` 获取本机及其他设备；移除设备时调用 `POST /uppiaing/maycallers_v/dev_out`，用 `upp_obj_no_upld` 提交设备列表返回的记录 ID。 | 不能提交用户 ID 或设备号代替设备记录 ID。移除操作会把目标设备恢复成游客并清除该设备会员；目标设备的原 Token 仍可继续鉴权，但后续读取到的是游客状态。 |
| 当前设备退出账号 | 调用 `POST /uppiaing/maycallers_v/acc_out`。 | 当前设备恢复游客状态并清空本设备会员，账号会员仍保留；保存接口返回的新用户信息和 Token。不要把“退出账号”当成删除账号。 |
| 修改密码 | 已登录账号调用 `POST /uppiaing/maycallers_v/pwd_upd`，只提交 `upp_new_sec_upld`，无需旧密码。 | 新密码必须为 6–20 位字母或数字且不能与当前密码相同；成功后 Token 不需要更换。 |
| 注销账号 | 调用 `POST /uppiaing/maycallers_v/acc_del`。是否真实注销由请求头 `Uppiaing-C-Ver` 是否命中服务端配置决定。 | 真实注销会删除账号或设备用户并创建新游客，必须保存 `upp_out_dat_upld.upp_auth_key_upld`；未命中真实注销版本时返回成功空对象，原 Token 继续有效。前端必须兼容这两种成功响应。 |

### 通知读取、邀请和客户端错误上报

通知页面先调用 `GET /uppiaing/maycallers_v/ntc_lst` 获取系统与个人通知；用户打开通知后调用 `POST /uppiaing/maycallers_v/ntc_ack`，分别用 `upp_pub_ids_upld` 和 `upp_usr_ids_upld` 提交系统通知 ID 数组与个人通知 ID 数组。读取成功后再更新本地已读状态和角标，避免网络失败时界面与服务端不一致。

邀请页面调用 `POST /uppiaing/maycallers_v/ref_gen` 获取自己的邀请码，输入他人邀请码时调用 `POST /uppiaing/maycallers_v/ref_use` 并将输入值放入 `upp_ref_key_upld`；邀请记录和奖励进度从 `POST /uppiaing/maycallers_v/ref_inf` 获取。当前用户是否仍可填写他人邀请码及奖励是否到账以接口返回为准，不由客户端本地计算。

VPN 内核、配置解析或其他可诊断错误可调用 `POST /uppiaing/maycallers_v/err_log` 上报，用 `upp_err_tp_upld` 标识错误分类，用 `upp_err_tx_upld` 提交可诊断信息，并同时提交 `upp_os_ver_upld` 操作系统版本和 `upp_dev_mdl_upld` 设备型号。上报内容不得包含登录密码、支付密钥、完整 Token、节点解密密钥或其他敏感信息；错误上报失败也不能阻塞用户退出连接或继续使用 App。
<!-- detailed-client-flows:end -->
