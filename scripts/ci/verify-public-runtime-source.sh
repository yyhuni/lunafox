#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
JSON_OUTPUT=0

usage() {
	cat <<'EOF'
Usage: bash scripts/ci/verify-public-runtime-source.sh [--root-dir <dir>] [--json]
EOF
}

fail() {
	echo "public Runtime source verification failed: $*" >&2
	exit 1
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--root-dir)
		[ "$#" -ge 2 ] || fail "--root-dir requires a value"
		ROOT_DIR="$(cd "$2" && pwd)"
		shift 2
		;;
	--json)
		JSON_OUTPUT=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		fail "unknown argument: $1"
		;;
	esac
done

[ -d "$ROOT_DIR" ] || fail "root directory is missing: $ROOT_DIR"

# Keep this list explicit. A public Runtime image is reproducible only when
# its source and local module replacements are present in the projection.
required_dirs=(frontend server contracts engine-go proto extensions/engines extensions/workflows docker/nginx docker/bootstrap tools/engine-release tools/engine-oci-publish)
for directory in "${required_dirs[@]}"; do
	[ -d "$ROOT_DIR/$directory" ] || fail "required public source directory is missing: $directory"
done

required_files=(
	frontend/package.json
	frontend/pnpm-lock.yaml
	frontend/Dockerfile
	server/go.mod
	server/go.sum
	server/Dockerfile
	contracts/go.mod
	contracts/go.sum
	engine-go/go.mod
	engine-go/go.sum
	proto/buf.yaml
	proto/scripts/check-generated.sh
	extensions/engines/go.mod
	extensions/engines/go.sum
	extensions/workflows/default.scan-workflow.json
	docker/nginx/Dockerfile
	docker/nginx/nginx.conf
	docker/bootstrap/Dockerfile
	docker/bootstrap/bootstrap.sh
	scripts/ci/publish-engine-runtime-images.sh
	scripts/ci/check-engine-image-tool-inventory.mjs
	scripts/ci/verify-distribution-registry-v2.mjs
	scripts/ci/verify-runtime-image-index.mjs
	tools/engine-release/go.mod
	tools/engine-release/go.sum
	tools/engine-oci-publish/go.mod
	tools/engine-oci-publish/go.sum
)
for file in "${required_files[@]}"; do
	[ -f "$ROOT_DIR/$file" ] || fail "required public Runtime input is missing: $file"
done

for forbidden in agent tools/installer worker docker/base-tools docker/ci-tools docker/dev-engine-publisher docker/nginx/ssl; do
	[ ! -e "$ROOT_DIR/$forbidden" ] || fail "private or development path is present in public Runtime tree: $forbidden"
done

if [ -e "$ROOT_DIR/tools" ]; then
	for tool_path in "$ROOT_DIR/tools"/*; do
		[ -e "$tool_path" ] || continue
		case "$(basename "$tool_path")" in
		engine-release | engine-oci-publish) ;;
		*) fail "private or development tool is present in public Runtime tree: ${tool_path#"$ROOT_DIR"/}" ;;
		esac
	done
fi

if find "$ROOT_DIR" -type f \( \
	-name '*.pem' -o -name '*.key' -o -name '*.crt' -o \
	-name '.env' -o -name '.env.local' -o -name '*.tsbuildinfo' \
	\) -not -path "$ROOT_DIR/.git/*" -print -quit | grep -q .; then
	fail "private certificate, environment, or generated material is present"
fi

if find "$ROOT_DIR" -type d \( \
	-name node_modules -o -name .next -o -name dist -o -name build -o \
	-name coverage -o -name reports -o -name test-results -o -name test-plan \
	\) -not -path "$ROOT_DIR/.git/*" -print -quit | grep -q .; then
	fail "runtime output or local test evidence is present"
fi

grep -Fq 'SPDX-License-Identifier: GPL-3.0-only' "$ROOT_DIR/LICENSE" || fail "LICENSE must declare GPL-3.0-only"
for document in README.md CONTRIBUTING.md docs/public-deployment.md NOTICE-CLOSED-ARTIFACTS.md; do
	grep -Eiq 'Agent.{0,80}(private|closed|not|不|私有)' "$ROOT_DIR/$document" ||
		fail "$document must describe the private Agent boundary"
done

if [ "$JSON_OUTPUT" -eq 1 ]; then
	printf '%s\n' '{"schemaVersion":1,"passed":true,"sourceClosure":"frontend,server,contracts,engine-go,proto,extensions,nginx,bootstrap,engine-release-tools","secretless":true}'
else
	echo "public Runtime source closure verified (secretless)"
fi
