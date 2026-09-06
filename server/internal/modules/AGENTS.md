# modules 目录说明

适用范围：`server/internal/modules/**`

修改本目录下代码时：

- 先读最近的 `README.md`、契约、测试和脚本说明。
- 模块内若已有更近的 `README.md`，以近代码文档为准。
- 通用命名规则见 `server/internal/modules/README.md`。

## domain 命名与拆分触发条件

- 新增 `domain/` 文件时，优先按具体领域概念命名，如 `scan.go`、`task.go`、`vulnerability.go`。
- 不要把新的独立领域概念继续塞进现有 `entity.go` 或 `entities.go`；优先新建 `<concept>.go`。
- 如果当前改动只涉及 `entity.go` 或 `entities.go` 中的一个独立概念，且拆分不会扩大无关改动，优先顺手拆分。
- 若文件名已不能准确概括其中大多数顶层类型或函数，应优先拆分或重命名到更具体的业务名。
- 禁止新增弱语义文件名：`types.go`、`common.go`、`models.go`、`utils.go`。

## 变更范围

- 不为统一而统一；禁止仅为命名一致性做无关重构或批量重命名。
- 仅在本次改动已经触及相关模块，且拆分/重命名能明显提升可读性时，再做最小收口。
