#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EVIDENCE_FILE=""
REQUIRE_FIRST_RELEASE=0

usage() {
	cat <<'USAGE'
Usage: scripts/ci/verify-public-deployment.sh [--root-dir <dir>] [--evidence <file>] [--require-first-release]

Secretless static validation for the generated Compose-first deployment closure.
It never builds images, accesses private source, or reads release credentials.
USAGE
}

fail() {
	echo "public deployment verification failed: $*" >&2
	exit 1
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--root-dir)
		[ "$#" -ge 2 ] || fail "--root-dir requires a value"
		ROOT_DIR="$(cd "$2" && pwd)"
		shift 2
		;;
	--evidence)
		[ "$#" -ge 2 ] || fail "--evidence requires a value"
		EVIDENCE_FILE="$2"
		shift 2
		;;
	--require-first-release)
		REQUIRE_FIRST_RELEASE=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*) fail "unknown argument: $1" ;;
	esac
done

require_file() {
	[ -f "$ROOT_DIR/$1" ] || fail "missing required public file: $1"
}

for file in README.md CONTRIBUTING.md LICENSE NOTICE-CLOSED-ARTIFACTS.md \
	docs/public-deployment.md compose.yaml .env.example \
	resources/loki/loki-config.yaml install.sh start.sh restart.sh stop.sh uninstall.sh \
	scripts/deploy/lifecycle.sh scripts/deploy/receipt/invalidate.sh \
	scripts/deploy/receipt/finalize.sh scripts/deploy/receipt/verify.sh \
	scripts/ci/audit-public-security-scope.mjs scripts/ci/check-public-channel.mjs \
	scripts/ci/check-public-release-policy.mjs scripts/ci/check-public-export.mjs \
	scripts/ci/generate-public-channel.mjs scripts/ci/publish-public-export.mjs \
	scripts/ci/verify-public-release.mjs scripts/ci/verify-public-main-merge.mjs \
	scripts/ci/verify-public-runtime-contexts.mjs scripts/ci/verify-public-runtime-image-evidence.mjs \
	scripts/ci/verify-public-runtime-source.sh scripts/ci/publish-engine-runtime-images.sh \
	scripts/ci/check-engine-image-tool-inventory.mjs scripts/ci/verify-distribution-registry-v2.mjs \
	scripts/ci/verify-runtime-image-index.mjs; do
	require_file "$file"
done

for required in server contracts engine-go proto extensions docker/nginx docker/bootstrap tools/engine-release tools/engine-oci-publish; do
	[ -d "$ROOT_DIR/$required" ] || fail "public Runtime source closure is missing: $required"
done
for forbidden in tools/installer worker docker/base-tools docker/ci-tools docker/dev-engine-publisher \
	docker/nginx/ssl scripts/cli scripts/installer scripts/shared checksums.txt; do
	[ ! -e "$ROOT_DIR/$forbidden" ] || fail "private/development or retired delivery path is present in public projection: $forbidden"
done
if [ -e "$ROOT_DIR/tools" ]; then
	for tool_path in "$ROOT_DIR/tools"/*; do
		[ -e "$tool_path" ] || continue
		case "$(basename "$tool_path")" in
		engine-release | engine-oci-publish) ;;
		*) fail "private/development tool is present in public projection: ${tool_path#"$ROOT_DIR"/}" ;;
		esac
	done
fi

for file in server/Dockerfile server/.dockerignore server/go.mod server/go.sum \
	contracts/go.mod contracts/go.sum engine-go/go.mod engine-go/go.sum \
	proto/buf.yaml proto/scripts/check-generated.sh extensions/engines/go.mod \
	extensions/engines/go.sum extensions/workflows/default.scan-workflow.json \
	docker/nginx/Dockerfile docker/nginx/nginx.conf docker/bootstrap/Dockerfile \
	docker/bootstrap/bootstrap.sh; do
	require_file "$file"
done

bash "$ROOT_DIR/scripts/ci/verify-public-runtime-source.sh" --root-dir "$ROOT_DIR"
node "$ROOT_DIR/scripts/ci/verify-public-runtime-contexts.mjs" --root-dir "$ROOT_DIR"

for file in server/Dockerfile server/.dockerignore server/go.mod server/go.sum \
	contracts/go.mod contracts/go.sum engine-go/go.mod engine-go/go.sum \
	proto/buf.yaml proto/scripts/check-generated.sh extensions/engines/go.mod \
	extensions/engines/go.sum extensions/workflows/default.scan-workflow.json \
	docker/nginx/Dockerfile docker/nginx/nginx.conf docker/bootstrap/Dockerfile \
	docker/bootstrap/bootstrap.sh; do
	require_file "$file"
done

bash "$ROOT_DIR/scripts/ci/verify-public-runtime-source.sh" --root-dir "$ROOT_DIR"
node "$ROOT_DIR/scripts/ci/verify-public-runtime-contexts.mjs" --root-dir "$ROOT_DIR"

# The public projection intentionally includes a reviewed frontend source
# closure for community debugging. Its generated output and local test evidence
# remain outside that closure; the reviewed Docker context is intentionally public.
for forbidden in frontend/node_modules frontend/.pnpm-store frontend/.next frontend/out \
	frontend/dist frontend/build frontend/output frontend/coverage frontend/reports \
	frontend/test-results frontend/test-plan frontend/.playwright-cli \
	frontend/storybook-static frontend/.turbo frontend/.cache; do
	[ ! -e "$ROOT_DIR/$forbidden" ] || fail "private frontend build or local-test path is present in public projection: $forbidden"
done
for file in frontend/package.json frontend/pnpm-lock.yaml frontend/Dockerfile frontend/.dockerignore frontend/app/page.tsx \
	frontend/scripts/route-inventory.mjs frontend/__tests__/vitest.config.contract.test.ts; do
	require_file "$file"
done

compose="$ROOT_DIR/compose.yaml"
for service in postgres redis loki server frontend bootstrap nginx \
	receipt-invalidator receipt-finalizer receipt-verifier; do
	grep -Eq "^[[:space:]]{2}${service}:" "$compose" || fail "production Compose is missing ${service}"
done
for key in SERVER_IMAGE_REF FRONTEND_IMAGE_REF NGINX_IMAGE_REF AGENT_IMAGE_REF BOOTSTRAP_IMAGE_REF; do
	grep -Fq "\${${key}:?${key} is required}" "$compose" || fail "Compose must require ${key}"
done
grep -Fq 'restart: "no"' "$compose" || fail "one-shot helpers and bootstrap must opt out of automatic restart"
for service in postgres redis loki server frontend nginx; do
	awk -v wanted="$service" '
		$0 ~ "^  " wanted ":" { active=1; next }
		active && /^  [A-Za-z0-9_-]+:/ { exit }
		active && /restart: unless-stopped/ { found=1 }
		END { exit found ? 0 : 1 }
	' "$compose" || fail "resident service ${service} must use restart: unless-stopped"
done
awk '
	/^  nginx:/ { active=1; next }
	active && /^  [A-Za-z0-9_-]+:/ { exit }
	active && /healthcheck:/ { healthcheck=1 }
	active && /healthChecks\/current/ { endpoint=1 }
	END { exit healthcheck && endpoint ? 0 : 1 }
' "$compose" || fail "nginx must use an HTTPS application health check"
if grep -Eq '^[[:space:]]+build:' "$compose"; then
	fail "public production Compose must consume released images and never build source"
fi
# The lifecycle implementation contains literal retired-name checks so it can
# fail closed when handed a stale manifest. Scan operational inputs here;
# validator/policy source may mention those names as detection rules.
if rg -n 'docker/docker-compose|scripts/(?:cli|installer)|tools/installer|lunafox-installer|checksums\.txt|go run' \
	"$compose" "$ROOT_DIR/.env.example" "$ROOT_DIR"/{install,start,restart,stop,uninstall}.sh; then
	fail "public deployment closure contains a retired installer or development fallback"
fi

lifecycle="$ROOT_DIR/scripts/deploy/lifecycle.sh"
grep -Fq 'ensure_fresh_runtime_volumes' "$lifecycle" || fail "fresh install must provision all runtime volumes before bootstrap"
grep -Fq 'require_runtime_volumes' "$lifecycle" || fail "ordinary lifecycle commands must reject missing runtime volumes"
grep -Fq 'stop_project_agent' "$lifecycle" || fail "stop lifecycle must manage the bootstrap-created Agent"
grep -Fq 'start_project_agent' "$lifecycle" || fail "start lifecycle must restore the bootstrap-created Agent"
grep -Fq 'restart_project_agent' "$lifecycle" || fail "restart lifecycle must restart the bootstrap-created Agent"
awk '
	/^run_mount_capability_probe\(\)/ { active=1; next }
	active && /^[A-Za-z0-9_]+\(\)/ { exit }
	active && /docker volume create/ { bad=1 }
	END { exit bad ? 1 : 0 }
' "$lifecycle" || fail "mount preflight must not silently create project volumes"

# Receipt storage is a deliberately narrow trust boundary. Runtime and
# bootstrap services must never receive it, while the verifier is read-only.
for service in postgres redis loki server frontend bootstrap nginx; do
	awk -v wanted="$service" '
		$0 ~ "^  " wanted ":" { active=1; next }
		active && /^  [A-Za-z0-9_-]+:/ { exit }
		active && /lunafox_receipt/ { bad=1 }
		END { exit bad ? 1 : 0 }
	' "$compose" || fail "${service} must not mount the completion receipt"
done
awk '
	/^  receipt-invalidator:/ { active=1; next }
	active && /^  [A-Za-z0-9_-]+:/ { exit }
	active && /lunafox_receipt:\/receipt:ro/ { exit 1 }
	active && /lunafox_receipt:\/receipt/ { found=1 }
	END { exit found ? 0 : 1 }
' "$compose" || fail "receipt invalidator must mount receipt storage read-write"
awk '
	/^  receipt-finalizer:/ { active=1; next }
	active && /^  [A-Za-z0-9_-]+:/ { exit }
	active && /lunafox_receipt:\/receipt:ro/ { exit 1 }
	active && /lunafox_receipt:\/receipt/ { found=1 }
	END { exit found ? 0 : 1 }
' "$compose" || fail "receipt finalizer must mount receipt storage read-write"
awk '
	/^  receipt-verifier:/ { active=1; next }
	active && /^  [A-Za-z0-9_-]+:/ { exit }
	active && /read_only: true/ { readonly=1 }
	active && /lunafox_receipt:\/receipt:ro/ { mount=1 }
	END { exit readonly && mount ? 0 : 1 }
' "$compose" || fail "receipt verifier must be a read-only helper with a read-only receipt mount"

compose_env="$(mktemp)"
trap 'rm -f "$compose_env"' EXIT
cat >"$compose_env" <<'EOF'
DB_NAME=lunafox
DB_USER=postgres
DB_PASSWORD=placeholder
JWT_SECRET=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
SERVER_IMAGE_REF=docker.io/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
FRONTEND_IMAGE_REF=docker.io/yyhuni/lunafox-frontend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
NGINX_IMAGE_REF=docker.io/yyhuni/lunafox-nginx@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
AGENT_IMAGE_REF=docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
BOOTSTRAP_IMAGE_REF=docker.io/yyhuni/lunafox-bootstrap@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
RELEASE_VERSION=0.0.1-alpha.57
AGENT_VERSION=0.0.1-alpha.57
PUBLIC_HOST=example.invalid
PUBLIC_URL=https://example.invalid
PUBLIC_PORT=443
LOKI_PUSH_URL=https://example.invalid/loki/api/v1/push
ENGINE_INVENTORY_HOST_PATH=.lunafox/engine-inventory.yaml
ENGINE_INSTALL_REGISTRY=docker.io
LUNAFOX_SHARED_DATA_VOLUME_BIND=lunafox_data:/opt/lunafox:rw
EOF
docker compose --env-file "$compose_env" -f "$compose" config -q >/dev/null || fail "root Compose configuration is invalid"

grep -Fq 'SPDX-License-Identifier: GPL-3.0-only' "$ROOT_DIR/LICENSE" || fail "GPL-3.0-only SPDX notice is missing"
if rg -n 'GPL-3\.0-or-later' "$ROOT_DIR/LICENSE" "$ROOT_DIR/README.md" "$ROOT_DIR/docs" "$ROOT_DIR/NOTICE-CLOSED-ARTIFACTS.md" >/dev/null; then
	fail "public notices must not imply GPL-3.0-or-later"
fi

private_release_repo='lunafox-private'
if rg -n "github\\.com/yyhuni/${private_release_repo}/releases|gitee\\.com/yyhuni/${private_release_repo}/releases" "$ROOT_DIR" \
	--glob '!PUBLIC_EXPORT_MANIFEST.json' --glob '!*.manifest.json' --glob '!.git/**' >/dev/null; then
	fail "private Release fallback appears in public deployment files"
fi

channel_args=(--root-dir "$ROOT_DIR" --allow-empty)
if [ "$REQUIRE_FIRST_RELEASE" -eq 1 ]; then
	channel_args+=(--require-first-release)
fi
node "$ROOT_DIR/scripts/ci/check-public-channel.mjs" "${channel_args[@]}"

grep -Eiq 'GENERATED|READ[- ]ONLY' "$ROOT_DIR/README.md" || fail "README lacks generated/read-only marker"
grep -Eiq 'long-lived.{0,40}bearer|eight-character|8 位|八位' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/public-deployment.md" || fail "Agent bearer limitation is not documented"
grep -Eiq 'TLS|certificate-chain|mTLS' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/public-deployment.md" || fail "deferred Agent TLS scope is not documented"

node "$ROOT_DIR/scripts/ci/audit-public-security-scope.mjs" --root-dir "$ROOT_DIR"
node "$ROOT_DIR/scripts/ci/check-public-release-policy.mjs" --root-dir "$ROOT_DIR" --public-only

if [ -n "$EVIDENCE_FILE" ]; then
	mkdir -p "$(dirname "$EVIDENCE_FILE")"
	cat >"$EVIDENCE_FILE" <<EOF
{
  "schemaVersion": 2,
  "passed": true,
  "secretless": true,
  "delivery": "compose-first",
  "requireFirstRelease": $REQUIRE_FIRST_RELEASE,
  "canonicalRepository": "yyhuni/lunafox"
}
EOF
fi

echo "public Compose deployment closure verified (secretless)"
