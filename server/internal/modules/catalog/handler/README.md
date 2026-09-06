# catalog/handler

该目录作为 catalog 模块的 HTTP handler 层。Handler 只依赖 facade 或以用例命名的
application 接口，不直接保存 concrete application service；对象图仍由
`bootstrap/wiring` 组装。

Wordlist handler contract：响应的 `name` 必须是 `wordlists/{id}`，上传文件 basename
只读出现在 `fileName`。PATCH 只允许 `description,tags`；`name`、`displayName` 和
`fileName` update mask 必须在持久化前拒绝。下载和内容编辑路径按 canonical ID 解析，
不能把 URL 中的文件名当作资源身份。
