# Release Notes

每个新发布 Tag 都需要一份已通过确定性校验的公开说明：

```text
release-notes/<tag>.md
```

例如 Tag 为 `v1.2.3` 时，文件必须是 `release-notes/v1.2.3.md`。文件名只表达
Tag；正文面向最终用户，不是内部部署记录或导出提交摘要。

## 文件格式

每份 Release Notes 同时承载英文和简体中文正文，两个二级标题必须各出现一次且非空：

```markdown
## English

- User-visible change in English.

## 简体中文

- 面向用户的变更说明。
```

公开 GitHub Release 的正文始终来自这个已校验的文件。私有
`release-note-evidence/<tag>.json` 仅保存来源覆盖、身份和摘要绑定，且不会进入公共投影。
它不保存审计输入正文、推理、对话、prompt、凭据或任何外部服务来源信息。

## Manifest 与更新检查

生产 `release.manifest.yaml` 会把同一 Tag 的正文和正文 SHA-256 绑定在
`releaseNotes.body` 与 `releaseNotes.digest` 中；生成器可用
`--release-notes-file` 指定输入，否则读取 `release-notes/v<releaseVersion>.md`。
生产版本缺少 notes、正文不符合上述双语格式或摘要不匹配时必须失败。Server 的更新检查
会投影为 `candidate.releaseNotes.body` 和 `candidate.releaseNotes.sha256`，因此离线实例也
能展示与公开 Release 相同的内容，前端不会额外请求 GitHub。仅 `0.0.0-dev` 开发 manifest
允许省略该对象，About 会明确显示暂无更新说明。

## 发布前流程

语义审计由发布执行者在本地完成。执行者可以是 Codex Agent，也可以是拥有维护者权限的
人工发布者；CI 不做语义分类、翻译或外部服务调用。

1. 在包含待发布 revision 的本地 clone 中，使用已有 GitHub CLI 会话收集事实：

   ```bash
   node scripts/ci/generate-release-notes.mjs collect \
     --tag "$TAG" \
     --target-revision "$TARGET_REVISION" \
     --facts-out "dist/release-notes/$TAG-facts.json" \
     --json
   ```

   收集器以最近一次成功公开 Release 为 baseline，完整盘点到目标 revision 的 commit，
   并关联已合并 PR。缺少本地 `gh auth login` 会话、GitHub 元数据或完整来源时会直接失败。

2. 审阅 facts 文件，并创建临时 audit input。它必须使用 `schemaVersion: 1`，携带完全一致的
   `factsDigest`，并用允许分类、英文和简体中文文本覆盖每个待审来源。`unmapped` 可用来
   显式记录尚未能公开说明的来源，但这种输入不会产生可发布候选，必须处理完后重新审计。

   ```json
   {
     "schemaVersion": 1,
     "factsDigest": "<facts SHA-256>",
     "entries": [{
       "sourceIds": ["pr:123"],
       "category": "Fixed",
       "english": "Fixed a user-visible failure.",
       "chinese": "修复面向用户的失败问题。"
     }],
     "unmapped": []
   }
   ```

   audit input 只是本地临时输入，不能提交到仓库或上传到 CI artifact。

3. 确定性 materializer 校验摘要、来源 ID、覆盖、公开安全和双语 claim，再生成 notes 与
   schema-2 private evidence：

   ```bash
   node scripts/ci/generate-release-notes.mjs materialize \
     --facts-file "dist/release-notes/$TAG-facts.json" \
     --audit-input "dist/release-notes/$TAG-audit.json" \
     --notes-out "release-notes/$TAG.md" \
     --evidence-out "release-note-evidence/$TAG.json" \
     --json
   ```

4. 通过本地 Git 和 `gh auth login` 的维护者会话发布候选。发布器只会创建
   `release-notes/<tag>` 或 `-retry-N` 分支、打开到 `main` 的 PR，并请求 GitHub native
   auto-merge；它不会 force-push、直写 `main` 或调用直接 merge endpoint：

   ```bash
   node scripts/ci/publish-release-notes-pr.mjs \
     --tag "$TAG" \
     --target-revision "$TARGET_REVISION" \
     --notes-file "release-notes/$TAG.md" \
     --evidence-file "release-note-evidence/$TAG.json" \
     --json
   ```

   相同身份和相同内容会复用已有候选。baseline、target 或候选摘要变化时，发布器会撤销旧
   PR 的 auto-merge、关闭旧 PR，并创建带 retry 后缀的新候选。

5. `Release Notes Candidate Validation` 是 `main` 上必须启用的受保护 PR check。它只校验
   提交的唯一 notes/evidence pair、分支形式、摘要、来源覆盖、公开安全以及 target 是 PR
   head 的祖先。候选合并后才能创建 Tag。

6. Tag 触发的私有 `Release` workflow 会再次在 Tag 树中验证 notes/evidence、摘要和 target
   祖先关系，然后才允许导出、签名或发布。

本地可向发布器传入 `--dry-run` 验证候选输入，而不调用 GitHub 写接口。任何来源不完整、
双语不一致、证据过期、缺少本地 GitHub 会话或 PR/Tag 祖先关系不成立都会 fail closed；
不会回退到旧 notes 或 GitHub 自动摘要。

## 内容约定

按读者能感知的变化组织内容。允许分类为：

- **新增**：新能力、支持的资源或工作流。
- **改进**：性能、稳定性、易用性或兼容性改进。
- **修复**：影响用户的错误修复。
- **安全**：安全修复或边界加强；只写公开后不会扩大风险的信息。
- **文档**：影响用户使用、配置或理解产品的文档变更。

只保留对用户有帮助的描述，并在需要时加入公开仓库的完整变更链接：

```markdown
Full Changelog: https://github.com/yyhuni/lunafox/compare/v1.2.2...v1.2.3
```

不要写私有仓库链接、凭据、内部 workflow 名称、内部构建/部署细节，或
`chore(export)`、`chore(deploy)` 这类工程提交。验证器会在发布边界再次检查这些高风险
内容，但公开安全仍由发布执行者和 reviewer 共同负责。

正文必须是有效 UTF-8 Markdown，去除首尾空白后不得为空，且不超过 64 KiB。PR 标签和
结构化文本是分类的优先信号，但不是每个 PR 的额外必填字段；无法证明公开安全或用户影响的
记录应标记为 unmapped，并阻断候选直到人工处理。

双语要求只适用于生效后的新 Tag。历史 Release 不会被自动回填或改写；如需补录，必须作为
单独的人工审核操作执行。
