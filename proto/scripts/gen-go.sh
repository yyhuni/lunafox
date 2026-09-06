#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
PROTO_DIR="${REPO_ROOT}/proto"
GEN_ROOT="${REPO_ROOT}/contracts/gen"
ENGINE_GEN_ROOT="${REPO_ROOT}/engine-go"

for bin in protoc protoc-gen-go protoc-gen-go-grpc; do
	if ! command -v "${bin}" >/dev/null 2>&1; then
		echo "missing required tool: ${bin}" >&2
		exit 1
	fi
done

# Ubuntu's protobuf package keeps well-known types outside this repository.
# Resolve that include root explicitly so generated contracts are portable across CI and local setups.
WELL_KNOWN_PROTO_DIR=""
for candidate in \
	"${PROTOBUF_INCLUDE_DIR:-}" \
	"$(dirname "$(command -v protoc)")/../include" \
	/usr/local/include \
	/usr/include \
	/opt/homebrew/include; do
	if [[ -n "${candidate}" && -f "${candidate}/google/protobuf/duration.proto" ]]; then
		WELL_KNOWN_PROTO_DIR="${candidate}"
		break
	fi
done

if [[ -z "${WELL_KNOWN_PROTO_DIR}" ]]; then
	echo "unable to locate protobuf well-known type includes (expected google/protobuf/duration.proto)" >&2
	exit 1
fi

mkdir -p \
	"${GEN_ROOT}/lunafox/agent/control/v1" \
	"${GEN_ROOT}/lunafox/agent/execution/v1" \
	"${GEN_ROOT}/lunafox/agent/data/v1" \
	"${GEN_ROOT}/lunafox/taskprogress/v1" \
	"${ENGINE_GEN_ROOT}/protocol"

rm -f \
	"${GEN_ROOT}/lunafox/agent/control/v1"/*.pb.go \
	"${GEN_ROOT}/lunafox/agent/execution/v1"/*.pb.go \
	"${GEN_ROOT}/lunafox/agent/data/v1"/*.pb.go \
	"${GEN_ROOT}/lunafox/taskprogress/v1"/*.pb.go \
	"${ENGINE_GEN_ROOT}/protocol"/*.pb.go

protoc \
	--proto_path="${PROTO_DIR}" \
	--proto_path="${WELL_KNOWN_PROTO_DIR}" \
	--go_out="${GEN_ROOT}" \
	--go_opt=paths=source_relative \
	--go-grpc_out="${GEN_ROOT}" \
	--go-grpc_opt=paths=source_relative \
	"lunafox/agent/control/v1/agent_control.proto" \
	"lunafox/agent/execution/v1/resolved_engine_execution_plan.proto" \
	"lunafox/agent/data/v1/agent_data.proto" \
	"lunafox/agent/data/v1/execution_artifact.proto" \
	"lunafox/taskprogress/v1/task_progress.proto"

protoc \
	--proto_path="${PROTO_DIR}" \
	--proto_path="${WELL_KNOWN_PROTO_DIR}" \
	--go_out="${ENGINE_GEN_ROOT}" \
	--go_opt=module=github.com/yyhuni/lunafox/engine-go \
	--go_opt=paths=import \
	--go-grpc_out="${ENGINE_GEN_ROOT}" \
	--go-grpc_opt=module=github.com/yyhuni/lunafox/engine-go \
	--go-grpc_opt=paths=import \
	"lunafox/engine/execution/v2/engine_execution_context.proto" \
	"lunafox/engine/execution/v2/engine_execution_diagnostics.proto" \
	"lunafox/engine/execution/v2/engine_execution_input.proto" \
	"lunafox/engine/execution/v2/engine_execution_reporting.proto"

echo "generated gRPC contracts"
