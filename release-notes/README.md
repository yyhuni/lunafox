# Release Notes

每个新的发布 Tag 都需要一份人工审核的公开说明：

```text
release-notes/<tag>.md
```

例如 Tag 为 `v1.2.3` 时，文件必须是 `release-notes/v1.2.3.md`。文件名只表达
Tag，正文面向最终用户，不是内部部署记录或导出提交摘要。

## 文件格式

每份新的 Release Notes 使用同一个文件同时承载英文和简体中文正文，两个二级标题都必须存在且非空：

```markdown
## English

- User-visible change in English.

## 简体中文

- 面向用户的变更说明。
```

英文段落和中文段落由提交者人工维护；发布校验器只检查段落存在、非空、UTF-8、大小限制和公开安全边界，不会自动翻译或生成摘要。公共 GitHub Release 的正文始终来自同一个已校验的 `release-notes/<tag>.md` 文件。

## 内容约定

按读者能感知的变化组织内容。常用分类包括：

- **新增**：新能力、支持的资源或工作流。
- **改进**：性能、稳定性、易用性或兼容性改进。
- **修复**：影响用户的错误修复。
- **安全**：安全修复或边界加强；只写公开后不会扩大风险的信息。

只保留对用户有帮助的描述，并在需要时加入公开仓库的完整变更链接：

```markdown
Full Changelog: https://github.com/yyhuni/lunafox/compare/v1.2.2...v1.2.3
```

不要写私有仓库链接、凭据、内部 workflow 名称、内部构建/部署细节，或
`chore(export)`、`chore(deploy)` 这类工程提交。验证器会在发布边界再次检查这些
高风险内容，但公开安全仍需要提交者和 reviewer 负责。

正文限制为有效 UTF-8 Markdown，去除首尾空白后不得为空，且不超过 64 KiB。第一阶段
只要求逐 Tag 文件；PR 标签、changeset 或自动聚合可以作为未来的初稿来源，但不是本
契约的额外必填字段。

双语要求只适用于生效后的新 Tag。历史 Release 不会由 workflow 自动回填或改写；如需
补录，必须作为单独的人工审核操作执行。
