# LunaFox Bootstrap Wiring 子包结构说明

## 1. 目标

在完成 DDD-Strict 改造后，`server/internal/bootstrap` 不再承载大量平铺的 `wiring_*.go` 文件，而是统一收敛到 `wiring/` 子包目录，达到以下目标：

- 降低 `bootstrap` 根目录复杂度
- 明确各业务模块的组装边界
- 保持 `wiring.go` 仅做“依赖编排入口”
- 通过 `exports.go` 统一对外暴露最小构造接口

## 2. 目录现状

```text
server/internal/bootstrap/
├── wiring.go
└── wiring/
    ├── asset/
    ├── catalog/
    ├── identity/
    ├── scan/
    ├── scanlog/
    ├── security/
    ├── snapshot/
    └── worker/
```

说明：

- `wiring.go`：总装配入口（composition root）
- `wiring/<module>/`：该模块相关 adapter/mapper/assembler 的内部实现

## 3. 子包职责划分

### 3.1 `wiring/asset`

负责 asset 应用层端口适配：

- `target lookup` 适配
- `website/subdomain/endpoint/directory/host_port/screenshot` store 适配
- `domain <-> persistence` mapper

### 3.2 `wiring/catalog`

负责 catalog 应用层端口适配：

- `engine/target/wordlist` store 适配
- catalog 与 organization 协同适配
- `domain <-> persistence` mapper

### 3.3 `wiring/identity`

负责 identity 应用层端口适配：

- user/organization store 适配
- 组织目标关联错误语义转换
- `domain <-> persistence` mapper

### 3.4 `wiring/scan`

负责 scan 应用层端口与运行态适配：

- scan store/query 适配
- task store/canceller/runtime 适配
- command store（domain repository）桥接

### 3.5 `wiring/scanlog`

负责 scan log 用例所需适配：

- log store 适配
- scan lookup 适配
- scan log application service 组装入口

### 3.6 `wiring/security`

负责 security 用例所需 target lookup 适配：

- vulnerability service 的 target 读取桥接

### 3.7 `wiring/snapshot`

负责 snapshot 相关完整组装适配：

- scan lookup 适配
- 各 snapshot store 适配
- asset/security sync 适配
- raw output codec
- snapshot mapper

### 3.8 `wiring/worker`

负责 worker 用例组装适配：

- scan lookup 适配
- subfinder settings store 适配
- worker application service 组装入口

## 4. 对外导出约定（重要）

每个子包统一通过 `exports.go` 导出有限入口：

- `NewXxx...` 构造函数（对 `wiring.go` 可见）
- 内部实现函数保持小写（包内私有）

约束：

- `wiring.go` 只能调用各子包 `exports.go` 导出的入口
- 禁止在 `wiring.go` 直接依赖子包内部私有类型/函数

## 5. `wiring.go` 角色边界

`wiring.go` 仅承担以下职责：

- 创建 repository 实例
- 调用各 wiring 子包构造 adapter/service
- 组装 handler 与最终依赖对象

`wiring.go` 不应承担：

- 领域规则判断
- 复杂映射逻辑
- 跨模块数据转换细节

## 6. 新增模块的接入流程

当新增模块 `foo` 时，按以下步骤落地：

1. 新建 `server/internal/bootstrap/wiring/foo/`
2. 在该目录实现 adapter/mapper 等内部文件
3. 新增 `exports.go` 暴露最小构造接口
4. 在 `wiring.go` 引入 `foowiring` 并调用导出入口
5. 执行：
   - `cd server && go test ./...`
   - `cd server && make check-architecture`

## 7. 命名规范（建议）

- 子包名：`<module>wiring`（如 `assetwiring`、`scanwiring`）
- 导出函数：`New<Resource><Role>Adapter` / `New...Service`
- 文件名保持可读：
  - `wiring_<module>_<resource>_adapter.go`
  - `wiring_<module>_mappers.go`
  - `exports.go`

## 8. 验收标准

满足以下条件视为 wiring 结构合格：

- `bootstrap` 根目录只保留编排入口文件（如 `wiring.go`）
- 模块 wiring 代码均位于 `wiring/<module>/`
- `go test ./...` 通过
- `make check-architecture` 通过

