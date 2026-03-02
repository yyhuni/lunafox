# Workflow Contract 生成与路径映射

## 目标
- 统一以 `worker/cmd/workflow-contract-gen` 作为 contract/schema/docs/typed 的生成入口。
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
- `server/internal/engineschema`
- `docs/config-reference`

## 可配置输出变量
`make workflow-contracts-gen-all` 支持以下可选变量：
- `SERVER_SCHEMA_DIR`：server schema 输出目录（默认 `../server/internal/engineschema`）
- `DOCS_DIR`：文档输出目录（默认 `../docs/config-reference`）
- `WORKER_SCHEMA_BASE_DIR`：worker schema 根目录（默认每个 workflow 下 `generated/`）
- `MIRROR_SCHEMA_DIR`：额外镜像输出目录（可选）

示例：
```bash
cd worker
make workflow-contracts-gen-all \
  SERVER_SCHEMA_DIR=../server/internal/engineschema \
  DOCS_DIR=../docs/config-reference \
  WORKER_SCHEMA_BASE_DIR=./out/worker-schemas \
  MIRROR_SCHEMA_DIR=./out/mirror
```

## 生成契约约定
- schema 文件名：`<workflow>-<apiVersion>-<schemaVersion>.schema.json`
- schema `$id`：`lunafox://schemas/engines/<workflow>/<apiVersion>/<schemaVersion>`
- typed config：`config_typed_generated.go`
