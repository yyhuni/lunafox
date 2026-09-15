#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EVIDENCE_FILE=""
REQUIRE_FIRST_RELEASE=0

usage() {
	cat <<'USAGE'
Usage: scripts/ci/verify-public-deployment.sh [--root-dir <dir>] [--evidence <file>] [--require-first-release]

Secretless static validation for the direct Docker Compose deployment closure.
It never starts services, builds images, accesses private source, or reads user data.
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
	resources/loki/loki-config.yaml resources/alloy/config.alloy \
	resources/fingerprints/web_fingerprint_v4.json resources/wordlists/manifest.json \
	docker/bootstrap/Dockerfile docker/bootstrap/bootstrap.sh docker/bootstrap/cert-init.sh \
	docker/nginx/Dockerfile docker/nginx/nginx.conf \
	scripts/ci/audit-public-security-scope.mjs scripts/ci/check-public-channel.mjs \
	scripts/ci/check-public-release-policy.mjs scripts/ci/generate-compose-deployment.mjs \
	scripts/ci/generate-compose-deployment.test.mjs scripts/ci/verify-public-release.mjs \
	scripts/ci/verify-compose-cert-init-selftest.sh \
	scripts/ci/verify-public-runtime-source.sh scripts/ci/verify-public-runtime-contexts.mjs; do
	require_file "$file"
done

for required in server server/scripts contracts engine-go proto extensions docker/nginx docker/bootstrap tools/engine-release tools/engine-oci-publish; do
	[ -d "$ROOT_DIR/$required" ] || fail "public Runtime source closure is missing: $required"
done
for forbidden in install.sh start.sh restart.sh stop.sh uninstall.sh scripts/deploy \
	tools/installer worker docker/base-tools docker/ci-tools docker/dev-engine-publisher \
	docker/nginx/ssl scripts/cli scripts/installer scripts/shared checksums.txt; do
	[ ! -e "$ROOT_DIR/$forbidden" ] || fail "retired, private, or development deployment path is present: $forbidden"
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

bash "$ROOT_DIR/scripts/ci/verify-public-runtime-source.sh" --root-dir "$ROOT_DIR"
node "$ROOT_DIR/scripts/ci/verify-public-runtime-contexts.mjs" --root-dir "$ROOT_DIR"
node --test "$ROOT_DIR/scripts/ci/generate-compose-deployment.test.mjs"
bash "$ROOT_DIR/scripts/ci/verify-compose-cert-init-selftest.sh"

compose="$ROOT_DIR/compose.yaml"
compose_json="$(
	env \
		DB_NAME=lunafox \
		DB_USER=postgres \
		DB_PASSWORD=placeholder \
		JWT_SECRET=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
		SERVER_IMAGE_REF=docker.io/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
		FRONTEND_IMAGE_REF=docker.io/yyhuni/lunafox-frontend@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
		NGINX_IMAGE_REF=docker.io/yyhuni/lunafox-nginx@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc \
		AGENT_IMAGE_REF=docker.io/yyhuni/lunafox-agent@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd \
		BOOTSTRAP_IMAGE_REF=docker.io/yyhuni/lunafox-bootstrap@sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee \
		RELEASE_VERSION=1.2.3 AGENT_VERSION=1.2.3 \
		PUBLIC_HOST=example.invalid PUBLIC_URL=https://example.invalid PUBLIC_PORT=443 \
		ENGINE_INVENTORY_HOST_PATH=./engine-inventory.yaml ENGINE_INSTALL_REGISTRY=docker.io \
		LUNAFOX_SHARED_DATA_VOLUME_BIND=lunafox_data:/opt/lunafox:rw \
		docker compose -f "$compose" config --format json
)" || fail "root Compose configuration is invalid"

jq -e '
  (.services | keys | sort) == (["agent","agent-preflight","alloy","bootstrap","cert-init","frontend","loki","migrate","nginx","postgres","redis","server"] | sort) and
  (all(.services[]; .build == null)) and
  (all(.services[]; .logging.driver == "json-file" and .logging.options["max-size"] == "10m" and .logging.options["max-file"] == "3")) and
  (.services.bootstrap.restart == "no") and
  (.services.migrate.restart == "no") and
  (.services["agent-preflight"].restart == "no") and
  (.services["cert-init"].restart == "no") and
  (.services.server.depends_on.bootstrap.condition == "service_completed_successfully") and
  (.services.bootstrap.depends_on["agent-preflight"].condition == "service_completed_successfully") and
  (.services.bootstrap.depends_on.migrate.condition == "service_completed_successfully") and
  (.services.migrate.depends_on.postgres.condition == "service_healthy") and
  (.services.agent.depends_on.bootstrap.condition == "service_completed_successfully") and
  (.services.agent.depends_on.server.condition == "service_healthy") and
  (.services.nginx.depends_on["cert-init"].condition == "service_completed_successfully") and
  (.services.agent.container_name == "lunafox-agent") and
  (.services.agent.environment.LUNAFOX_AGENT_DISABLE_SELF_UPDATE == "true") and
  (.services.server.labels["lunafox.logs.component"] == "server") and
  (.services.agent.labels["lunafox.logs.component"] == "agent")
' <<<"$compose_json" >/dev/null || fail "Compose service graph or bounded logging contract is invalid"

socket_owners="$(jq -r '.services | to_entries[] | select(any(.value.volumes[]?; .source == "/var/run/docker.sock")) | .key' <<<"$compose_json" | sort)"
[ "$socket_owners" = $'agent\nagent-preflight\nalloy' ] || fail "only Agent, its one-shot preflight, and Alloy may mount the Docker socket"
jq -e '
  (.services.agent.volumes | any(.source == "/var/run/docker.sock" and .target == "/var/run/docker.sock")) and
  (.services["agent-preflight"].image == .services.agent.image) and
  (.services["agent-preflight"].network_mode == "none") and
  (.services["agent-preflight"].volumes | any(.source == "/var/run/docker.sock" and .target == "/var/run/docker.sock" and .read_only == true)) and
  (.services["agent-preflight"].volumes | any(.source == "lunafox_engine_execution" and .target == "/var/lib/lunafox/engine-execution")) and
  (.services["agent-preflight"].volumes | any(.source == "lunafox_data" and .target == "/opt/lunafox")) and
  (.services["agent-preflight"].command | tostring | contains("--profile capability")) and
  (.services["agent-preflight"].command | tostring | contains("$$AGENT_IMAGE_REF")) and
  (.services["agent-preflight"].command | tostring | contains("compose-$$HOSTNAME")) and
  (.services.alloy.volumes | any(.source == "/var/run/docker.sock" and .target == "/var/run/docker.sock" and .read_only == true)) and
  (.services.alloy.volumes | any(.source == "alloy_data" and .target == "/var/lib/alloy/data")) and
  (.services.alloy.command | index("--storage.path=/var/lib/alloy/data") != null)
' <<<"$compose_json" >/dev/null || fail "Agent capability preflight or Alloy position storage mount is invalid"

jq -e '
  [.services.postgres.image, .services.redis.image, .services.loki.image, .services.alloy.image]
  | all(test("^docker\\.io/.+@sha256:[a-f0-9]{64}$"))
' <<<"$compose_json" >/dev/null || fail "third-party resident images must be digest-qualified"

for key in SERVER_IMAGE_REF FRONTEND_IMAGE_REF NGINX_IMAGE_REF AGENT_IMAGE_REF BOOTSTRAP_IMAGE_REF; do
	grep -Fq "\${${key}:?${key} is required}" "$compose" || fail "source Compose must require ${key}"
done
if rg -n 'lunafox-loki|LOKI_PLUGIN_REF|logging:[[:space:]]*loki' "$compose" "$ROOT_DIR/.env.example" "$ROOT_DIR/resources/alloy/config.alloy"; then
	fail "direct Compose deployment must not require the Loki Docker plugin"
fi

for marker in \
	'loki\.source\.docker[[:space:]]+"lunafox"' \
	'__meta_docker_container_label_lunafox_logs_component' \
	'target_label[[:space:]]*=[[:space:]]*"container_name"' \
	'agent_id[[:space:]]*=[[:space:]]*""' \
	'url[[:space:]]*=[[:space:]]*"http://loki:3100/loki/api/v1/push"'; do
	rg -q "$marker" "$ROOT_DIR/resources/alloy/config.alloy" || fail "Alloy configuration is missing: $marker"
done

grep -Eq '^DB_PASSWORD=$' "$ROOT_DIR/.env.example" || fail ".env.example must leave DB_PASSWORD for the operator"
grep -Eq '^JWT_SECRET=$' "$ROOT_DIR/.env.example" || fail ".env.example must leave JWT_SECRET for the operator"
if ! grep -Eiq 'GENERATED' "$ROOT_DIR/.env.example" || ! grep -Eiq 'READ[- ]ONLY' "$ROOT_DIR/.env.example"; then
	fail ".env.example lacks complete generated/read-only marker"
fi
grep -Fq 'docker compose up -d' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must use direct Compose startup"
grep -Fq 'PowerShell' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must cover native Windows PowerShell"
grep -Fq 'docker compose down' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must document direct Compose shutdown"
grep -Fq 'docker compose exec server resetadmin' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must retain administrator reset"

workflow="$ROOT_DIR/.github/workflows/public-validate.yml"
for marker in 'Build immutable Docker Compose deployment packages' 'generate-compose-deployment.mjs' 'dist/final/deployment/*.zip' 'deployment-packages.json'; do
	grep -Fq "$marker" "$workflow" || fail "public release workflow is missing deployment package publication: $marker"
done

grep -Fq 'SPDX-License-Identifier: GPL-3.0-only' "$ROOT_DIR/LICENSE" || fail "GPL-3.0-only SPDX notice is missing"
if rg -n 'GPL-3\.0-or-later' "$ROOT_DIR/LICENSE" "$ROOT_DIR/README.md" "$ROOT_DIR/docs" "$ROOT_DIR/NOTICE-CLOSED-ARTIFACTS.md" >/dev/null; then
	fail "public notices must not imply GPL-3.0-or-later"
fi

channel_args=(--root-dir "$ROOT_DIR" --allow-empty)
if [ "$REQUIRE_FIRST_RELEASE" -eq 1 ]; then
	channel_args+=(--require-first-release)
fi
node "$ROOT_DIR/scripts/ci/check-public-channel.mjs" "${channel_args[@]}"

grep -Eiq 'GENERATED|READ[- ]ONLY' "$ROOT_DIR/README.md" || fail "README lacks generated/read-only marker"
grep -Eiq 'long-lived.{0,40}bearer|eight-character' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/public-deployment.md" || fail "Agent bearer limitation is not documented"
grep -Eiq 'TLS|certificate-chain|mTLS' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/public-deployment.md" || fail "deferred Agent TLS scope is not documented"

node "$ROOT_DIR/scripts/ci/audit-public-security-scope.mjs" --root-dir "$ROOT_DIR"
node "$ROOT_DIR/scripts/ci/check-public-release-policy.mjs" --root-dir "$ROOT_DIR" --public-only

if [ -n "$EVIDENCE_FILE" ]; then
	mkdir -p "$(dirname "$EVIDENCE_FILE")"
	cat >"$EVIDENCE_FILE" <<EOF
{
  "schemaVersion": 3,
  "passed": true,
  "secretless": true,
  "delivery": "direct-compose-package",
  "fullInstallationRun": false,
  "requireFirstRelease": $REQUIRE_FIRST_RELEASE,
  "canonicalRepository": "yyhuni/lunafox"
}
EOF
fi

echo "public direct Compose deployment closure verified (secretless)"
