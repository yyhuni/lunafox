# 离线 Agent 授权签名

`LF1.<base64url canonical JSON>.<base64url Ed25519 signature>` 是本产品的版本化授权格式，
不是通用 JWT。签名消息为 `lunafox.agent-license.v1\n` 加原始 JSON 字节。
算法固定为 Ed25519，不接受载荷指定算法。官方实现依据：[Go crypto/ed25519](https://pkg.go.dev/crypto/ed25519)。

- 产品标识固定为 `lunafox.agent-expansion`；授权编号为非空 canonical UUID。
- 额度为大于 3 的 JavaScript 安全整数；期限为 1 至 120 个日历月。
- 日期为 Unix UTC 秒；`issuedAt` 同时是生效时间。到期日按 UTC 月份增加并夹取月底，
  例如 1 月 31 日签发一个月，到期为 2 月最后一天的同一 UTC 时刻。
- 所有字段必填。验签后要求重新序列化字节完全一致，拒绝重复字段、未知字段、别名和尾随内容。
- `Verify` 不判断当前时间，使存量过期授权仍能展示。调用者必须在同一权威时间点判断
  `issuedAt <= now < expiresAt`；固定日期不会因导入、换机、重部署或重放而变化。
- 只有没有持久记录才是未激活；数据库/签名失败必须返回错误，不可静默降为基础额度。

`trusted-public-key.txt` 是本发行版作者的**公钥**，由 `NewDefaultVerifier` 嵌入使用。
产品中没有私钥，也不开放 HTTP 修改信任根。测试通过 `NewVerifier` 显式注入临时公钥。
公钥变更会使旧 Key 失效，不能当普通配置自动轮换。

作者签发与私钥备份说明见 [`tools/license-issuer/README.md`](../../../../tools/license-issuer/README.md)。

验证：`cd server && go test ./internal/agentlicense ./cmd/lunafox-license-issuer`。
