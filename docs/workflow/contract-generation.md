# Workflow Contract 生成与路径映射

## 目标
- 统一以 `worker/cmd/workflow-contract-gen` 作为 contract 驱动的 server schema/manifest/profile、docs、typed config 生成入口。
- 支持在不同目录布局下输出生成产物，避免脚本硬编码路径耦合。

## 默认命令
```bash
cd worker
make workflow-contracts-gen-all
```

## CI 门禁命令
```bash
cd worker
make workflow-contracts-ci-check
```

该命令会重新生成 contract 产物，并校验以下目录无未提交差异：
- `worker/internal/workflow`
- `server/internal/workflow/schema`
- `server/internal/workflow/profile/profiles`
- `docs/config-reference`

## 可配置输出变量
`make workflow-contracts-gen-all` 支持以下可选变量：
- `SERVER_SCHEMA_DIR`：server workflow schema 输出目录（默认 `../server/internal/workflow/schema`）
- `SERVER_PROFILE_DIR`：server workflow profile 输出目录（默认 `../server/internal/workflow/profile/profiles`）
- `DOCS_DIR`：文档输出目录（默认 `../docs/config-reference`）
- `MIRROR_SCHEMA_DIR`：额外镜像输出目录（可选）

示例：
```bash
cd worker
make workflow-contracts-gen-all \
  SERVER_SCHEMA_DIR=../server/internal/workflow/schema \
  SERVER_PROFILE_DIR=../server/internal/workflow/profile/profiles \
  DOCS_DIR=../docs/config-reference \
    MIRROR_SCHEMA_DIR=./out/mirror
```

## 生成契约约定
- schema 文件名：`<workflow>.schema.json`
- schema `$id`：`lunafox://schemas/workflows/<workflow>`
- profile 文件名：`<workflow>.yaml`
- profile 来源：contract 中的 `DefaultProfile` + 参数 `Default`
- profile `configuration`：直接生成 YAML mapping，不再写入 block-scalar 形式的二次 YAML 字符串
- typed config：`config_typed_generated.go`
