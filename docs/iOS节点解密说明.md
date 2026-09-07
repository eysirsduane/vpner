# 母版 iOS 节点数据解密说明

## 1. 适用范围

母版当前提供三种节点返回方式：

| 接口 | 加密字段 | 解密结果 |
| --- | --- | --- |
| `POST /api/v1/node` | `result.link_url` | 原始节点 URL，例如 `vmess://...`、`vless://...`、`anytls://...` 或 `chimney://...` |
| `POST /api/v1/node_outbounds` | `result.link_url` | 原始节点 URL |
| `POST /api/v1/node_outbounds` | `result.outbounds` | 可直接嵌入完整配置的 `outbounds` JSON 数组字符串 |
| `POST /api/v1/node_config` | `result.config` | 完整节点 JSON 配置字符串 |

上述加密字段使用完全相同的加密和解密流程。

`node_outbounds` 返回的 `link_url` 和 `outbounds` 是两个各自独立的加密字符串，客户端必须分别解密。不能先拼接两个字段，也不能使用其中一个字段的解密结果替代另一个字段。

具体产品由 mapping 映射接口路径和字段名称。生成产品文档时必须以该产品 mapping 为准，不能把本文件中的母版原始路径和字段名直接交给产品前端。

## 2. 后端加密流程

后端对每一个待加密内容分别执行：

```text
原始内容的 UTF-8 字节
→ 标准 Base64 编码
→ 使用 AES-CBC/PKCS7 加密
→ 将 AES 密文二进制做标准 Base64 编码
→ 作为字符串放入 JSON 响应
```

因此，接口返回的字符串不是可以直接交给 AES 的原始二进制数据。客户端必须先进行一次标准 Base64 解码，得到 AES 密文二进制。

## 3. iOS 解密流程

```text
接口返回的加密字符串
→ 标准 Base64 解码
→ 得到 AES 密文二进制
→ AES-CBC 解密并执行 PKCS7 去填充
→ 得到内部标准 Base64 字符串
→ 再次标准 Base64 解码
→ 得到原始内容的 UTF-8 字符串
```

注意：

- 外层和内层都使用标准 Base64，不是 Base64URL。
- 标准 Base64 字符串可能包含 `+`、`/` 和末尾的 `=`。
- 不要对响应字段执行 URL 解码。
- 不要删除 Base64 字符串末尾的 `=`。
- AES 解密得到的是内部 Base64 文本，不是最终内容，必须再进行一次 Base64 解码。
- 最终结果可能是节点 URL、JSON 数组或 JSON 对象，取决于接口和字段。

## 4. AES 参数

| 参数 | 当前约定 |
| --- | --- |
| 算法 | AES |
| 模式 | CBC |
| 填充 | PKCS7 |
| Key | 当前产品约定的 `node.link_aes_key`，按 UTF-8 转为字节 |
| Key 长度 | 必须为 16、24 或 32 字节，分别对应 AES-128、AES-192、AES-256 |
| IV | Key UTF-8 字节的前 16 字节 |

Key 属于产品私密配置，不应写入公开文档、日志或错误上报。客户端应通过项目现有的安全配置方式保存。不同产品可能使用不同 Key，不能跨产品复用。

## 5. 原生 Swift 实现

以下实现使用 iOS 自带的 `CommonCrypto`，不依赖第三方 AES 库。

```swift
import Foundation
import CommonCrypto

enum NodeDecryptError: LocalizedError {
    case invalidKeyLength
    case invalidOuterBase64
    case invalidCipherLength
    case aesDecryptFailed(CCCryptorStatus)
    case invalidInnerBase64Text
    case invalidInnerBase64
    case invalidUTF8
    case invalidJSONStructure

    var errorDescription: String? {
        switch self {
        case .invalidKeyLength:
            return "节点解密 Key 必须为 16、24 或 32 字节"
        case .invalidOuterBase64:
            return "节点字段不是有效的外层 Base64 字符串"
        case .invalidCipherLength:
            return "AES 密文长度不正确"
        case let .aesDecryptFailed(status):
            return "AES 解密失败，状态码：\(status)"
        case .invalidInnerBase64Text:
            return "AES 解密结果不是有效的 UTF-8 Base64 文本"
        case .invalidInnerBase64:
            return "AES 解密结果不是有效的内部 Base64 字符串"
        case .invalidUTF8:
            return "原始节点内容不是有效的 UTF-8 字符串"
        case .invalidJSONStructure:
            return "节点 JSON 的顶层结构不正确"
        }
    }
}

enum NodeDecryptor {
    static func decrypt(
        encryptedValue: String,
        key: String
    ) throws -> String {
        let keyData = Data(key.utf8)

        guard [kCCKeySizeAES128, kCCKeySizeAES192, kCCKeySizeAES256]
            .contains(keyData.count) else {
            throw NodeDecryptError.invalidKeyLength
        }

        // 第一次标准 Base64 解码：字符串 → AES 密文二进制
        guard let cipherData = Data(base64Encoded: encryptedValue) else {
            throw NodeDecryptError.invalidOuterBase64
        }

        guard !cipherData.isEmpty,
              cipherData.count.isMultiple(of: kCCBlockSizeAES128) else {
            throw NodeDecryptError.invalidCipherLength
        }

        let ivData = Data(keyData.prefix(kCCBlockSizeAES128))
        let decryptedData = try decryptAESData(
            cipherData,
            keyData: keyData,
            ivData: ivData
        )

        // AES 解密结果是内部标准 Base64 文本
        guard let innerBase64 = String(
            data: decryptedData,
            encoding: .utf8
        ) else {
            throw NodeDecryptError.invalidInnerBase64Text
        }

        // 第二次标准 Base64 解码：内部 Base64 → 原始内容字节
        guard let originalData = Data(base64Encoded: innerBase64) else {
            throw NodeDecryptError.invalidInnerBase64
        }

        guard let originalContent = String(
            data: originalData,
            encoding: .utf8
        ) else {
            throw NodeDecryptError.invalidUTF8
        }

        return originalContent
    }

    private static func decryptAESData(
        _ cipherData: Data,
        keyData: Data,
        ivData: Data
    ) throws -> Data {
        var output = Data(count: cipherData.count + kCCBlockSizeAES128)
        var outputLength: size_t = 0

        let status: CCCryptorStatus = output.withUnsafeMutableBytes { outputBytes in
            cipherData.withUnsafeBytes { cipherBytes in
                keyData.withUnsafeBytes { keyBytes in
                    ivData.withUnsafeBytes { ivBytes in
                        CCCrypt(
                            CCOperation(kCCDecrypt),
                            CCAlgorithm(kCCAlgorithmAES),
                            CCOptions(kCCOptionPKCS7Padding),
                            keyBytes.baseAddress,
                            keyData.count,
                            ivBytes.baseAddress,
                            cipherBytes.baseAddress,
                            cipherData.count,
                            outputBytes.baseAddress,
                            output.count,
                            &outputLength
                        )
                    }
                }
            }
        }

        guard status == kCCSuccess else {
            throw NodeDecryptError.aesDecryptFailed(status)
        }

        output.removeSubrange(Int(outputLength)..<output.count)
        return output
    }
}
```

如果工程提示 `No such module 'CommonCrypto'`，可以在 Objective-C Bridging Header 中加入：

```objc
#import <CommonCrypto/CommonCrypto.h>
```

通过 Bridging Header 引入后，Swift 文件不需要再写 `import CommonCrypto`。

## 6. 普通节点 URL 使用方式

### 6.1 请求

```http
POST /api/v1/node
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "code": "HK"
}
```

### 6.2 解密

从 `result.link_url` 取出字符串后调用：

```swift
let nodeURL = try NodeDecryptor.decrypt(
    encryptedValue: encryptedLinkURL,
    key: nodeDecryptKey
)
```

结果是完整原始节点 URL，例如：

```text
vmess://...
vless://...
anytls://...
chimney://...
```

不要再对整个 URL 统一执行 Base64 解码。某个具体协议内部是否还有编码，应交给对应协议解析器处理。

## 7. URL 与 outbounds 使用方式

### 7.1 请求

```http
POST /api/v1/node_outbounds
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "code": "HK"
}
```

### 7.2 分别解密两个字段

```swift
let nodeURL = try NodeDecryptor.decrypt(
    encryptedValue: encryptedLinkURL,
    key: nodeDecryptKey
)

let outboundsJSONString = try NodeDecryptor.decrypt(
    encryptedValue: encryptedOutbounds,
    key: nodeDecryptKey
)
```

`nodeURL` 是原始节点 URL。`outboundsJSONString` 是 JSON 数组字符串，可进一步解析：

```swift
let outboundsData = Data(outboundsJSONString.utf8)
let outbounds = try JSONSerialization.jsonObject(
    with: outboundsData,
    options: []
)

guard outbounds is [Any] else {
    throw NodeDecryptError.invalidJSONStructure
}
```

解密后的 `outbounds` 数组必须与完整配置接口返回 JSON 中的 `outbounds` 数组一致。

## 8. 完整 JSON 配置使用方式

### 8.1 请求

```http
POST /api/v1/node_config
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "code": "HK",
  "type": "fast"
}
```

`type` 必填：`fast` 或 `极速` 表示极速模式，`global` 或 `全局` 表示全局模式。

### 8.2 解密并解析

```swift
let jsonString = try NodeDecryptor.decrypt(
    encryptedValue: encryptedConfig,
    key: nodeDecryptKey
)

let jsonData = Data(jsonString.utf8)
let config = try JSONSerialization.jsonObject(
    with: jsonData,
    options: []
)

guard config is [String: Any] else {
    throw NodeDecryptError.invalidJSONStructure
}
```

## 9. 业务状态处理

只在业务状态码 `code` 为 `200` 且对应加密字段非空时执行解密：

| 业务状态码 | 客户端处理方式 |
| --- | --- |
| `200` | 读取并解密节点字段 |
| `401` | 当前登录状态失效，重新调用自动登录获取 Token |
| `901` | 仅节点接口使用，表示无有效会员，提示用户开通会员 |
| `500` | 展示后端返回的友好错误信息，不执行解密 |

HTTP 请求正常到达后端时，业务错误通常仍通过 HTTP 200 响应承载，客户端应读取 JSON 中的业务状态码。

## 10. 接入检查清单

1. 只在业务状态码为 `200` 且目标字段非空时执行解密。
2. 从 JSON 响应中取得字符串后，直接做标准 Base64 解码，不做 URL 解码。
3. Key 必须与当前产品后端配置一致，并按 UTF-8 字节处理。
4. IV 必须使用 Key 字节的前 16 字节，不能使用全零 IV，也不能从密文中截取 IV。
5. AES 使用 CBC 模式和 PKCS7 填充，不能改为 ECB、GCM 或 NoPadding。
6. AES 解密后还要执行第二次标准 Base64 解码。
7. `node_outbounds` 的 `link_url` 和 `outbounds` 必须分别解密。
8. 普通节点结果按 URL 字符串使用，`outbounds` 按 JSON 数组解析，完整配置按 JSON 对象解析。
9. 解密失败时不要打印 Key、完整密文或完整节点内容到线上日志。
10. 产品项目必须使用其映射后的接口路径和字段名，不能照搬母版原始名称。

## 11. 本地联调测试向量

以下内容仅用于验证 iOS 解密实现，不是线上 Key：

```text
测试 Key：0123456789abcdef
接口加密字符串：ggFLxfolSFWbXBGnBs1IqvrrKRq3MQXAPZ+yBDT+cHa7qNR0sPEEDaQUIMFltYT1
预期解密结果：vmess://example.com:443
```

调用示例：

```swift
let result = try NodeDecryptor.decrypt(
    encryptedValue: "ggFLxfolSFWbXBGnBs1IqvrrKRq3MQXAPZ+yBDT+cHa7qNR0sPEEDaQUIMFltYT1",
    key: "0123456789abcdef"
)

assert(result == "vmess://example.com:443")
```

测试向量通过后，再替换为当前产品的真实 Key 和接口返回值进行联调。

## 12. 常见错误

### AES 接口要求二进制，但响应是字符串

响应字符串是 AES 密文的标准 Base64 表示。先执行：

```swift
Data(base64Encoded: encryptedValue)
```

得到密文二进制后再调用 AES。

### AES 解密失败或提示 padding 错误

优先检查：

- 是否遗漏了外层 Base64 解码；
- Key 是否属于当前产品；
- Key 的 UTF-8 字节数是否为 16、24 或 32；
- IV 是否为 Key 的前 16 字节；
- 是否使用 CBC 和 PKCS7。

### AES 解密成功但不是节点 URL 或 JSON

AES 解密得到的是内部 Base64 字符串，还需要再进行一次标准 Base64 解码。

### Base64 解码偶尔失败

不要把标准 Base64 当作 Base64URL 处理，也不要删除字符串中的 `+`、`/` 和末尾的 `=`。如果字段经过 URL 查询参数传递，`+` 可能被错误转换为空格；节点字段应直接从 JSON 响应读取。

### `outbounds` 解析成了字符串而不是数组

`NodeDecryptor.decrypt` 的返回值是 JSON 文本。对 `result.outbounds` 解密后，还要使用 `JSONSerialization` 或 `JSONDecoder` 将文本解析为数组。
