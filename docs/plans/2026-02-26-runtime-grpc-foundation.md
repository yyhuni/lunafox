# Runtime gRPC Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the protocol and generation foundation for runtime communication migration from WebSocket/HTTP to gRPC.

**Architecture:** Define shared protobuf contracts in a new top-level `proto/` tree, generate Go stubs once into shared module `contracts/gen`, and let `server/agent/worker` import that shared package. Centralize auth metadata + gRPC error mapping as shared runtime conventions. Do not replace existing runtime endpoints in this step.

**Tech Stack:** Go 1.26, Protocol Buffers, gRPC-Go, protoc plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)

---

### Task 1: Define Runtime Proto Contracts

**Files:**
- Create: `proto/lunafox/runtime/v1/runtime.proto`

**Step 1: Write the failing test**

Add a contract-shape test that checks generated symbols are available:
- `server/internal/grpc/runtime/v1/contracts_smoke_test.go`
- `agent/internal/grpc/runtime/v1/contracts_smoke_test.go`
- `worker/internal/grpc/runtime/v1/contracts_smoke_test.go`

The test should compile-reference:
- `AgentRuntimeServiceClient`
- `AgentDataProxyServiceClient`
- `WorkerRuntimeServiceClient`
- key envelope messages (`AgentRuntimeRequest`, `AgentRuntimeEvent`)

**Step 2: Run test to verify it fails**

Run:

```bash
cd server && go test ./internal/grpc/runtime/v1 -run TestGeneratedContractsPresent -count=1
cd ../agent && go test ./internal/grpc/runtime/v1 -run TestGeneratedContractsPresent -count=1
cd ../worker && go test ./internal/grpc/runtime/v1 -run TestGeneratedContractsPresent -count=1
```

Expected: compile failure because packages/files do not exist yet.

**Step 3: Write minimal implementation**

Create `runtime.proto` with:
- package `lunafox.runtime.v1`
- `service AgentRuntimeService { rpc Connect(stream AgentRuntimeRequest) returns (stream AgentRuntimeEvent); }`
- `service AgentDataProxyService` with provider-config, wordlist meta/download, batch upsert
- `service WorkerRuntimeService` with provider-config, ensure-wordlist, post-batch
- oneof envelopes for agent up/down stream messages
- `go_package = "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1;runtimev1"`

**Step 4: Run test to verify it still fails for missing generated code**

Run the same commands in Step 2.

Expected: compile failure now points to missing generated Go files.

---

### Task 2: Add Protobuf Generation Pipeline

**Files:**
- Create: `proto/scripts/gen-go.sh`
- Create: `proto/scripts/check-generated.sh`
- Create: `contracts/go.mod`
- Modify: `Makefile` (workspace root)
- Modify: `server/go.mod`
- Modify: `agent/go.mod`
- Modify: `worker/go.mod`
- Create: `contracts/gen/lunafox/runtime/v1/runtime.pb.go` (generated)
- Create: `contracts/gen/lunafox/runtime/v1/runtime_grpc.pb.go` (generated)

**Step 1: Write the failing test**

Add a script-level guard test:
- `scripts/ci/check-runtime-proto-generated.sh` (invoked by CI/lint target)
- it exits non-zero when regenerating protobuf changes tracked files.

**Step 2: Run test to verify it fails**

Run:

```bash
bash scripts/ci/check-runtime-proto-generated.sh
```

Expected: failure because generator and paths are not implemented.

**Step 3: Write minimal implementation**

- Implement `proto/scripts/gen-go.sh`:
  - installs/uses `protoc-gen-go` and `protoc-gen-go-grpc` from local Go toolchain
  - generates to `contracts/gen/`
- add root `make proto` and `make proto-check`
- add required grpc/protobuf deps to all three go.mod files and add local module mapping to `../contracts`
- update runtime imports in each component to use shared package `github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1`.

**Step 4: Run test to verify it passes**

Run:

```bash
make proto
bash scripts/ci/check-runtime-proto-generated.sh
cd server && go test ./internal/grpc/runtime/v1 -run TestGeneratedContractsPresent -count=1
cd ../agent && go test ./internal/grpc/runtime/v1 -run TestGeneratedContractsPresent -count=1
cd ../worker && go test ./internal/grpc/runtime/v1 -run TestGeneratedContractsPresent -count=1
```

Expected: all green.

---

### Task 3: Define Runtime Auth Metadata and gRPC Error Mapping

**Files:**
- Create: `server/internal/grpc/runtime/auth/metadata.go`
- Create: `server/internal/grpc/runtime/auth/errors.go`
- Create: `agent/internal/grpc/runtime/auth/metadata.go`
- Create: `worker/internal/grpc/runtime/auth/metadata.go`
- Test: `server/internal/grpc/runtime/auth/metadata_test.go`
- Test: `server/internal/grpc/runtime/auth/errors_test.go`

**Step 1: Write the failing test**

Tests assert:
- metadata key constants are exactly:
  - `x-agent-key`
  - `x-worker-token`
  - `x-task-token`
- error mapping is stable:
  - invalid credentials -> `codes.Unauthenticated`
  - forbidden task scope -> `codes.PermissionDenied`
  - unsupported legacy path behavior (future hook) -> `codes.Unimplemented`

**Step 2: Run test to verify it fails**

Run:

```bash
cd server && go test ./internal/grpc/runtime/auth -count=1
```

Expected: package missing / compile failure.

**Step 3: Write minimal implementation**

Add constants + helper functions:
- metadata helpers for extracting/attaching tokens
- typed sentinel errors
- deterministic mapping function from domain/auth errors to gRPC `status` codes

**Step 4: Run test to verify it passes**

Run:

```bash
cd server && go test ./internal/grpc/runtime/auth -count=1
```

Expected: pass.

---

### Task 4: Validate and Update OpenSpec Task State

**Files:**
- Modify: `openspec/changes/refactor-runtime-communication-to-grpc/tasks.md`

**Step 1: Verification run**

Run:

```bash
cd server && go test ./...
cd ../agent && go test ./...
cd ../worker && go test ./...
cd .. && openspec validate refactor-runtime-communication-to-grpc --strict --no-interactive
```

**Step 2: Update checklist**

Mark completed items:
- `1.1`
- `1.2`
- `1.3`
- `1.4` (if metadata + auth mapping are fully wired)

Leave unchecked anything partially implemented.
