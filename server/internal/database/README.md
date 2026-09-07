# database

`NewDatabase` 负责创建 Server 的 GORM 连接、统一 UTC `NowFunc` 与连接池设置。

- GORM 诊断必须经过共享 Zap JSON pipeline。新 Server 进程产生的每条数据库日志使用既有
  `level`、`timestamp`、`caller`、`msg` envelope，并使用 `db.query.text`、
  `db.query.duration_ms`、`db.rows_affected` 和安全分类后的 `error` 字段；不得重新引入
  标准库纯文本 logger 或平行日志输出。
- 默认保持 GORM `Warn` log mode、200ms 慢查询阈值、`IgnoreRecordNotFoundError=true` 和
  参数化查询。预期的 `record not found` 分支不记录数据库错误，由 HTTP 或业务层记录最终
  状态；其他错误优先于慢查询，普通查询只有在 GORM Info mode 才记录。
- 参数 filter 必须阻止绑定值进入 SQL，错误字段只记录稳定分类，不能透传数据库错误正文。
  `caller` 必须是查询调用位置，adapter 不得同时产生第二个顶层 caller。
- 切换只影响新进程产生的行。Loki、system-log API 和前端继续原样传递历史纯文本或新 JSON
  行；本模块不增加 API level 字段、前端筛选，也不改写保留历史。
