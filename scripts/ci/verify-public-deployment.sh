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

require_regular_file() {
	if [ ! -f "$ROOT_DIR/$1" ] || [ -L "$ROOT_DIR/$1" ]; then
		fail "required public file must be a regular file: $1"
	fi
}

for file in README.md README.zh-CN.md CONTRIBUTING.md LICENSE NOTICE-CLOSED-ARTIFACTS.md \
	.gitignore docs/public-deployment.md docs/public-deployment.zh-CN.md .env .env.example compose.yaml engine-inventory.yaml release.manifest.yaml \
	deploy/compose.template.yaml deploy/.env.example \
	resources/loki/loki-config.yaml resources/alloy/config.alloy \
	resources/fingerprints/web_fingerprint_v4.json resources/wordlists/manifest.json \
	docker/bootstrap/Dockerfile docker/bootstrap/bootstrap.sh docker/bootstrap/cert-init.sh docker/bootstrap/config-init.sh \
	server/cmd/lunafox-upgrader/main.go server/internal/modules/upgrade/upgrader/compose.go \
	server/internal/modules/upgrade/upgrader/daemon.go server/internal/modules/upgrade/upgrader/journal.go \
	docker/nginx/Dockerfile docker/nginx/nginx.conf \
	scripts/ci/audit-public-security-scope.mjs scripts/ci/check-public-channel.mjs \
	scripts/ci/check-public-release-policy.mjs scripts/ci/generate-compose-deployment.mjs \
	scripts/ci/generate-compose-deployment.test.mjs scripts/ci/verify-public-release.mjs \
	scripts/ci/resolve-release-component-composition.mjs scripts/ci/verify-release-component-composition.mjs \
	scripts/ci/verify-compose-cert-init-selftest.sh scripts/ci/verify-compose-config-init-selftest.sh \
	scripts/ci/verify-public-runtime-source.sh scripts/ci/verify-public-runtime-contexts.mjs; do
	require_file "$file"
done
require_regular_file release.manifest.yaml

LEGACY_V1_BOOTSTRAP=0
if [ -e "$ROOT_DIR/runtime-composition.json" ] || [ -L "$ROOT_DIR/runtime-composition.json" ]; then
	require_regular_file runtime-composition.json
else
	# alpha.114 predates composition evidence. Its raw manifest identity is fixed
	# by policy, so this branch remains v1/full-only rather than granting scope.
	if node --input-type=module - "$ROOT_DIR" <<'NODE'; then
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

try {
  const root = process.argv[2];
  const policy = JSON.parse(fs.readFileSync(path.join(root, "scripts/ci/public-release-policy.json"), "utf8"));
  const bootstrap = policy.legacyDeploymentBootstrap;
  const expectedKeys = ["manifestSha256", "packageName", "paths", "releaseTag", "sha256"];
  if (!bootstrap || typeof bootstrap !== "object" || Array.isArray(bootstrap) ||
      JSON.stringify(Object.keys(bootstrap).sort()) !== JSON.stringify(expectedKeys) ||
      !/^v\d+\.\d+\.\d+-alpha\.\d+$/.test(bootstrap.releaseTag) ||
      bootstrap.packageName !== `lunafox-${bootstrap.releaseTag}-dockerhub.zip` ||
      !/^[a-f0-9]{64}$/.test(bootstrap.sha256) ||
      !/^[a-f0-9]{64}$/.test(bootstrap.manifestSha256) ||
      !Array.isArray(bootstrap.paths) || bootstrap.paths.includes("runtime-composition.json")) {
    process.exit(1);
  }

  const manifestPath = path.join(root, "release.manifest.yaml");
  const manifestInfo = fs.lstatSync(manifestPath);
  if (!manifestInfo.isFile() || manifestInfo.isSymbolicLink()) process.exit(1);
  const manifest = fs.readFileSync(manifestPath);
  const digest = crypto.createHash("sha256").update(manifest).digest("hex");
  // alpha.114 is pinned below. Later published snapshots that predate the
  // composition asset are pinned by exact manifest bytes, not by release tag.
  const publishedWithoutComposition = policy.publishedManifestsWithoutComposition;
  if (Array.isArray(publishedWithoutComposition) && publishedWithoutComposition.includes(digest)) process.exit(0);
  const text = manifest.toString("utf8");
  const releaseVersion = text.match(/^releaseVersion:[ \t]*["']?([^"'#\s]+)["']?[ \t]*(?:#.*)?$/m)?.[1] ?? "";
  const releaseTag = releaseVersion.startsWith("v") ? releaseVersion : `v${releaseVersion}`;
  if (releaseTag !== bootstrap.releaseTag ||
      digest !== bootstrap.manifestSha256 ||
      /(?:^|\r?\n)[ \t]*runtimeComposition[ \t]*:/.test(text)) {
    process.exit(1);
  }
} catch {
  process.exit(1);
}
NODE
		LEGACY_V1_BOOTSTRAP=1
	else
		fail "missing required public file: runtime-composition.json (only a policy-pinned manifest that predates composition evidence may omit it)"
	fi
fi

cmp -s "$ROOT_DIR/.env" "$ROOT_DIR/.env.example" || fail ".env and .env.example must start from the same generated configuration"

for required in server server/scripts contracts engine-go proto extensions docker/nginx docker/bootstrap tools/engine-release tools/engine-oci-publish; do
	[ -d "$ROOT_DIR/$required" ] || fail "public Runtime source closure is missing: $required"
done
# The public host execution surface is an explicit allowlist: the seven root
# lifecycle scripts plus the helper they all share. Every other root entry, and
# every retired private path, stays rejected.
for script in install.sh start.sh restart.sh stop.sh status.sh logs.sh uninstall.sh lunafox-lifecycle.sh; do
	require_file "$script"
	[ -x "$ROOT_DIR/$script" ] || fail "public lifecycle script must be executable: $script"
done
for script in install.sh start.sh restart.sh stop.sh status.sh logs.sh uninstall.sh; do
	grep -Fq 'lunafox-lifecycle.sh' "$ROOT_DIR/$script" || fail "public lifecycle script must delegate to the shared helper: $script"
done
for forbidden in prepare-deployment.sh scripts/deploy \
	scripts/ci/prepare-deployment-selftest.sh \
	tools/installer worker docker/base-tools docker/ci-tools docker/dev-engine-publisher \
	docker/nginx/ssl scripts/cli scripts/installer scripts/shared checksums.txt; do
	[ ! -e "$ROOT_DIR/$forbidden" ] || fail "retired, private, or development deployment path is present: $forbidden"
done
for entry in "$ROOT_DIR"/*.sh; do
	[ -e "$entry" ] || continue
	case "$(basename "$entry")" in
	install.sh | start.sh | restart.sh | stop.sh | status.sh | logs.sh | uninstall.sh | lunafox-lifecycle.sh) ;;
	*) fail "undeclared root host entry in the public projection: $(basename "$entry")" ;;
	esac
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

compose="$ROOT_DIR/compose.yaml"
render_compose() {
	local registry="$1" database_mode="$2" db_host="$3" db_port="$4" db_sslmode="$5"
	local db_user="$6" db_name="$7" db_password="$8" jwt_secret="$9" public_host="${10}" public_port="${11}"
	(
		cd "$ROOT_DIR"
		env \
			RELEASE_REGISTRY="$registry" \
			DATABASE_MODE="$database_mode" \
			COMPOSE_PROFILES="$database_mode" \
			DB_HOST="$db_host" \
			DB_PORT="$db_port" \
			DB_SSLMODE="$db_sslmode" \
			DB_NAME="$db_name" \
			DB_USER="$db_user" \
			DB_PASSWORD="$db_password" \
			JWT_SECRET="$jwt_secret" \
			PUBLIC_HOST="$public_host" PUBLIC_PORT="$public_port" \
			docker compose --env-file .env -f compose.yaml config --format json
	)
}
compose_json="$(render_compose docker.io embedded '' '' '' '' '' '' '' localhost 443)" || fail "root Compose configuration with default embedded inputs is invalid"
ghcr_compose_json="$(render_compose ghcr.io embedded '' '' '' '' '' '' '' localhost 443)" || fail "root Compose configuration with GHCR inputs is invalid"
custom_compose_json="$(render_compose docker.io embedded '' 5432 disable lunafox_app lunafox_custom operator-db-password operator-jwt-secret example.invalid 8443)" || fail "custom embedded Compose configuration is invalid"
external_default_json="$(render_compose docker.io external database.example '' require '' '' operator-db-password '' localhost 443)" || fail "external Compose configuration with database identity defaults is invalid"
external_custom_json="$(render_compose docker.io external 2001:db8::1 6543 verify-full lunafox_remote lunafox_external operator-db-password operator-jwt-secret example.invalid 8443)" || fail "custom external Compose configuration is invalid"

if [ "$LEGACY_V1_BOOTSTRAP" -eq 0 ]; then
	if ! node "$ROOT_DIR/scripts/ci/verify-release-component-composition.mjs" \
		--composition "$ROOT_DIR/runtime-composition.json" \
		--manifest "$ROOT_DIR/release.manifest.yaml" \
		--policy "$ROOT_DIR/scripts/ci/public-release-policy.json"; then
		fail "runtime composition asset does not match the release manifest"
	fi
fi

jq -e '
  (.services | keys | sort) == (["agent","agent-preflight","alloy","bootstrap","cert-init","config-init","frontend","loki","migrate","nginx","postgres","redis","server","upgrader"] | sort) and
  (all(.services[]; .build == null)) and
  (all(.services[]; .logging.driver == "json-file" and .logging.options["max-size"] == "10m" and .logging.options["max-file"] == "3")) and
  (.services.bootstrap.restart == "no") and
  (.services.migrate.restart == "no") and
  (.services["agent-preflight"].restart == "no") and
  (.services["cert-init"].restart == "no") and
  (.services["config-init"].restart == "no") and
  (.services["config-init"].network_mode == "none") and
  (.services["config-init"].environment.DATABASE_MODE == "embedded") and
  (.services["config-init"].environment.DB_HOST == "") and
  (.services["config-init"].environment.DB_PORT == "5432") and
  (.services["config-init"].environment.DB_SSLMODE == "") and
  (.services.postgres.depends_on["config-init"].condition == "service_completed_successfully") and
  (.services.migrate.depends_on["config-init"].condition == "service_completed_successfully") and
  (.services.bootstrap.depends_on["config-init"].condition == "service_completed_successfully") and
  (.services.server.depends_on["config-init"].condition == "service_completed_successfully") and
  (.services.server.depends_on.bootstrap.condition == "service_completed_successfully") and
  (.services.bootstrap.depends_on["agent-preflight"].condition == "service_completed_successfully") and
  (.services.bootstrap.depends_on.migrate.condition == "service_completed_successfully") and
  (.services.migrate.depends_on.postgres.condition == "service_healthy") and
  (.services.migrate.depends_on.postgres.required == false) and
  (.services.server.depends_on.postgres.condition == "service_healthy") and
  (.services.server.depends_on.postgres.required == false) and
  (.services.agent.depends_on.bootstrap.condition == "service_completed_successfully") and
  (.services.agent.depends_on.server.condition == "service_healthy") and
  (.services.nginx.depends_on["cert-init"].condition == "service_completed_successfully") and
  (.services.agent.container_name == "lunafox-agent") and
  (.services.agent.environment.LUNAFOX_AGENT_DISABLE_SELF_UPDATE == "true") and
  (.services.upgrader.network_mode == "none") and
  (.services.upgrader.ports == null) and
  (.services.upgrader.command == ["--root-dir","/deployment","--layout","public","--registry","docker.io"]) and
  (.services.server.environment.RELEASE_CHANNEL == "canary") and
  (.services.server.environment.RELEASE_METADATA_BASE_URL == "https://raw.githubusercontent.com/yyhuni/lunafox/release-channel") and
  (.services.server.environment.RELEASE_REGISTRY == "docker.io") and
  (.services.postgres.volumes | any(.source == "postgres_data" and .target == "/var/lib/postgresql/data")) and
  (.services.postgres.environment.POSTGRES_DB == "lunafox") and
  (.services.postgres.environment.POSTGRES_USER == "postgres") and
  (.services.server.environment.DB_NAME == "lunafox") and
  (.services.server.environment.DB_USER == "postgres") and
  (.services.server.environment.PUBLIC_HOST == "localhost") and
  (.services.server.environment.PUBLIC_PORT == "443") and
  (.services.nginx.ports | any(.target == 443 and .published == "443")) and
  (.services.server.environment.PUBLIC_URL == null) and
  (.services.server.labels["lunafox.logs.component"] == "server") and
  (.services.agent.labels["lunafox.logs.component"] == "agent")
' <<<"$compose_json" >/dev/null || {
	# Print only the graph facts the assertions above read, so a failing clause is
	# identifiable from the log without exposing a secret value.
	printf 'LunaFox guard diagnostics: rendered service graph\n' >&2
	jq -c '{
    services: (.services | to_entries | map({
      service: .key,
      restart: (.value.restart // null),
      network_mode: (.value.network_mode // null),
      ports: (.value.ports // null),
      build: (.value.build // null),
      logging: (.value.logging // null),
      healthcheck: (if .value.healthcheck then "present" else null end),
      stop_grace_period: (.value.stop_grace_period // null),
      container_name: (.value.container_name // null),
      depends_on: (.value.depends_on // {}),
      volumes: (.value.volumes // [] | map({source: (.source // null), target: (.target // null), read_only: (.read_only // false)})),
      labels: (.value.labels // null)
    })),
    config_init_inputs: (.services["config-init"].environment | {DATABASE_MODE, COMPOSE_PROFILES, DB_HOST, DB_PORT, DB_SSLMODE})
  }' <<<"$compose_json" >&2
	fail "Compose service graph or bounded logging contract is invalid"
}

# An export PR still carries the previous release's rendered compose snapshot, so
# facts introduced by the current template are asserted on the template itself
# and on the freshly generated package contract, never on a lagging snapshot.
template_config_init="$(awk '/^  config-init:$/ {inside=1; next} inside && /^  [a-z]/ {inside=0} inside {print}' "$ROOT_DIR/deploy/compose.template.yaml")"
# The single-quoted pattern is the literal template text, not a shell expansion.
# shellcheck disable=SC2016
printf '%s' "$template_config_init" | grep -Fq 'COMPOSE_PROFILES: ${COMPOSE_PROFILES:-}' ||
	fail "the deployment template must pass the derived Compose profile to config-init"
agent_grace_seconds="$(awk '/^  agent:$/ {inside=1; next} inside && /^  [a-z]/ {inside=0} inside && /^    stop_grace_period:/ {value=$2; gsub(/[^0-9]/, "", value); print value; exit}' "$ROOT_DIR/deploy/compose.template.yaml")"
case "$agent_grace_seconds" in
'' | *[!0-9]*) fail "the resident Agent must declare a stop grace period of at least 30 seconds" ;;
esac
[ "$agent_grace_seconds" -ge 30 ] || fail "the resident Agent must allow at least 30 seconds for Engine Container cleanup"

assert_registry_closure() {
	local registry="$1" other_registry="$2" rendered="$3"
	jq -e --arg registry "$registry" --arg other "$other_registry" '
	  . as $root |
	  (.services.server.environment.RELEASE_REGISTRY == $registry) and
	  (.services.server.environment.ENGINE_INSTALL_REGISTRY == $registry) and
	  (.services.bootstrap.environment.ENGINE_INSTALL_REGISTRY == $registry) and
	  (.services.upgrader.command == ["--root-dir","/deployment","--layout","public","--registry",$registry]) and
	  (["server","frontend","nginx","agent","bootstrap","config-init","migrate","cert-init","upgrader","agent-preflight"] |
	    all(.[]; $root.services[.].image | startswith($registry + "/yyhuni/lunafox-"))) and
	  ([.services[].image] | all(contains($other + "/yyhuni/lunafox-") | not))
	' <<<"$rendered" >/dev/null || fail "Compose mixes first-party registries or does not propagate ${registry}"
}

assert_registry_closure docker.io ghcr.io "$compose_json"
assert_registry_closure ghcr.io docker.io "$ghcr_compose_json"

jq -e '
  (.services.postgres.environment.POSTGRES_DB == "lunafox_custom") and
  (.services.postgres.environment.POSTGRES_USER == "lunafox_app") and
  (.services.server.environment.DB_NAME == "lunafox_custom") and
  (.services.server.environment.DB_USER == "lunafox_app") and
  (.services.bootstrap.environment.DB_NAME == "lunafox_custom") and
  (.services.bootstrap.environment.DB_USER == "lunafox_app") and
  (.services.migrate.environment.DB_NAME == "lunafox_custom") and
  (.services.migrate.environment.DB_USER == "lunafox_app") and
  (.services.postgres.healthcheck.test | tostring | contains("pg_isready -U lunafox_app -d lunafox_custom")) and
  (.services.server.environment.PUBLIC_HOST == "example.invalid") and
  (.services.server.environment.PUBLIC_PORT == "8443") and
  (.services.nginx.ports | any(.target == 443 and .published == "8443"))
' <<<"$custom_compose_json" >/dev/null || fail "database identity or public address overrides are not propagated consistently"

jq -e '
  (.services | has("postgres") | not) and
  (.services["config-init"].environment.DATABASE_MODE == "external") and
  (.services["config-init"].environment.DB_HOST == "database.example") and
  (.services["config-init"].environment.DB_PORT == "5432") and
  (.services["config-init"].environment.DB_SSLMODE == "require") and
  (all(.services.server, .services.bootstrap, .services.migrate;
    .environment.DB_HOST == "database.example" and
    .environment.DB_PORT == "5432" and
    .environment.DB_USER == "postgres" and
    .environment.DB_NAME == "lunafox" and
    .environment.DB_SSLMODE == "require")) and
  (.services.server.depends_on.postgres.condition == "service_healthy") and
  (.services.server.depends_on.postgres.required == false) and
  (.services.migrate.depends_on.postgres.condition == "service_healthy") and
  (.services.migrate.depends_on.postgres.required == false)
' <<<"$external_default_json" >/dev/null || fail "external Compose defaults or optional database dependency are invalid"

jq -e '
  (.services | has("postgres") | not) and
  (all(.services.server, .services.bootstrap, .services.migrate;
    .environment.DB_HOST == "2001:db8::1" and
    .environment.DB_PORT == "6543" and
    .environment.DB_USER == "lunafox_remote" and
    .environment.DB_NAME == "lunafox_external" and
    .environment.DB_SSLMODE == "verify-full"))
' <<<"$external_custom_json" >/dev/null || fail "custom external database inputs are not propagated consistently"

jq -e --arg root "$ROOT_DIR" '
  (.secrets.db_password_input.environment == "DB_PASSWORD") and
  (.secrets.jwt_secret_input.environment == "JWT_SECRET") and
  (.services["config-init"].secrets | any(.source == "db_password_input" and .target == "db-password-input")) and
  (.services["config-init"].secrets | any(.source == "jwt_secret_input" and .target == "jwt-secret-input")) and
  (.services["config-init"].volumes | any(.source == "lunafox_config" and .target == "/var/lib/lunafox-config" and .read_only != true)) and
  (.services.postgres.environment.POSTGRES_PASSWORD == null) and
  (.services.postgres.environment.POSTGRES_PASSWORD_FILE == "/run/lunafox-config/db-password") and
  (.services.server.environment.DB_PASSWORD == null) and
  (.services.server.environment.JWT_SECRET == null) and
  (.services.server.environment.DB_PASSWORD_FILE == "/run/lunafox-config/db-password") and
  (.services.server.environment.JWT_SECRET_FILE == "/run/lunafox-config/jwt-secret") and
  (.services.server.volumes | any(.source == "lunafox_config" and .target == "/run/lunafox-config" and .read_only == true)) and
  (.services.server.volumes | any(.source == "lunafox_upgrade_state" and .target == "/opt/lunafox/.lunafox/upgrade" and .read_only != true)) and
  (.services.upgrader.volumes | any(.type == "bind" and .source == $root and .target == "/deployment" and .read_only != true)) and
  (.services.upgrader.volumes | any(.source == "lunafox_upgrade_state" and .target == "/deployment/.lunafox/upgrade" and .read_only != true)) and
  (.services.bootstrap.volumes | any(.source == "lunafox_config" and .target == "/run/lunafox-config" and .read_only == true)) and
  (.services.migrate.volumes | any(.source == "lunafox_config" and .target == "/run/lunafox-config" and .read_only == true)) and
  (.volumes.lunafox_config.name == "lunafox_config") and
  (.volumes.lunafox_upgrade_state.name == "lunafox_upgrade_state")
' <<<"$compose_json" >/dev/null || fail "Compose persistent secret boundary is invalid"

socket_owners="$(jq -r '.services | to_entries[] | select(any(.value.volumes[]?; .source == "/var/run/docker.sock")) | .key' <<<"$compose_json" | sort)"
[ "$socket_owners" = $'agent\nagent-preflight\nalloy\nupgrader' ] || fail "only the upgrader and existing Agent execution or log collection services may mount the Docker socket"
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
  (.services.upgrader.volumes | any(.source == "/var/run/docker.sock" and .target == "/var/run/docker.sock" and .read_only != true)) and
  (.services.alloy.volumes | any(.source == "alloy_data" and .target == "/var/lib/alloy/data")) and
  (.services.alloy.command | index("--storage.path=/var/lib/alloy/data") != null)
' <<<"$compose_json" >/dev/null || fail "Agent capability preflight or Alloy position storage mount is invalid"

jq -e '
  [.services.postgres.image, .services.redis.image, .services.loki.image, .services.alloy.image]
  | all(test("^docker\\.io/.+@sha256:[a-f0-9]{64}$"))
' <<<"$compose_json" >/dev/null || fail "third-party resident images must be digest-qualified"

template="$ROOT_DIR/deploy/compose.template.yaml"
for key in SERVER_IMAGE_REF FRONTEND_IMAGE_REF NGINX_IMAGE_REF AGENT_IMAGE_REF BOOTSTRAP_IMAGE_REF RELEASE_CHANNEL RELEASE_METADATA_BASE_URL RELEASE_REGISTRY ENGINE_INSTALL_CF_ACCELERATION PUBLIC_HOST PUBLIC_PORT; do
	grep -Fq "\${${key}:?${key} is required}" "$template" || fail "deployment template must require ${key}"
done
for key in PUBLIC_HOST PUBLIC_PORT; do
	grep -Fq "\${${key}:?${key} is required}" "$compose" || fail "public root Compose must require ${key}"
done
if rg -q '\$\{(?:SERVER|FRONTEND|NGINX|AGENT|BOOTSTRAP)_IMAGE_REF' "$compose"; then
	fail "public root Compose must contain resolved Runtime image identities"
fi
grep -Fq 'ENGINE_INSTALL_CF_ACCELERATION: false' "$compose" ||
	fail "public root Compose must keep the direct Compose CF acceleration default disabled"
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

# The shipped .env.example in an export PR is the previous release's snapshot, so
# these facts must hold for every published snapshot. Facts introduced by the
# current template are asserted on deploy/.env.example below, which is always the
# current source of this revision.
shipped_env="$ROOT_DIR/.env.example"
grep -Eq '^PUBLIC_HOST=localhost$' "$shipped_env" || fail ".env.example must provide the local public host"
grep -Eq '^RELEASE_REGISTRY=docker\.io$' "$shipped_env" || fail ".env.example must default to Docker Hub"
grep -Eq '^PUBLIC_PORT=443$' "$shipped_env" || fail ".env.example must provide the HTTPS public port"
grep -Eq '^DATABASE_MODE=embedded$' "$shipped_env" || fail ".env.example must default to embedded database mode"
expected_compose_profile="COMPOSE_PROFILES=\${DATABASE_MODE:-embedded}"
grep -Fqx "$expected_compose_profile" "$shipped_env" || fail ".env.example must derive the Compose profile from database mode"
grep -Eq '^DB_PASSWORD=$' "$shipped_env" || fail ".env.example must leave DB_PASSWORD empty for automatic generation"
grep -Eq '^JWT_SECRET=$' "$shipped_env" || fail ".env.example must leave JWT_SECRET empty for automatic generation"
if grep -Eq '^PUBLIC_URL=' "$shipped_env"; then
	fail ".env.example must not expose derived PUBLIC_URL"
fi
grep -Eiq 'commit local secret' "$shipped_env" || fail ".env.example must warn against committing local secret edits"

source_env="$ROOT_DIR/deploy/.env.example"
active_keys="$(rg -o '^[A-Za-z_][A-Za-z0-9_]*=' "$source_env" | tr -d '=' | sort -u | tr '\n' ' ')"
[ "$active_keys" = "COMPOSE_PROFILES DATABASE_MODE DB_PASSWORD JWT_SECRET PUBLIC_HOST PUBLIC_PORT RELEASE_REGISTRY " ] ||
	fail "deploy/.env.example must only assign the reviewed settings and the two empty secret inputs: $active_keys"
for key in DB_HOST DB_PORT DB_USER DB_NAME DB_SSLMODE; do
	grep -Eq "^#${key}=" "$source_env" || fail "deploy/.env.example must keep ${key} in the commented external database example"
	if grep -Eq "^${key}=" "$source_env"; then
		fail "deploy/.env.example must not assign ${key} outside the commented external database example"
	fi
done
grep -Eq '^DB_PASSWORD=$' "$source_env" || fail "deploy/.env.example must leave DB_PASSWORD empty for automatic generation"
grep -Eq '^JWT_SECRET=$' "$source_env" || fail "deploy/.env.example must leave JWT_SECRET empty for automatic generation"
if grep -Eq '^PUBLIC_URL=' "$source_env"; then
	fail "deploy/.env.example must not expose derived PUBLIC_URL"
fi
grep -Eiq 'commit local secret' "$source_env" || fail "deploy/.env.example must warn against committing local secret edits"

bash "$ROOT_DIR/scripts/ci/verify-public-runtime-source.sh" --root-dir "$ROOT_DIR"
node "$ROOT_DIR/scripts/ci/verify-public-runtime-contexts.mjs" --root-dir "$ROOT_DIR"
node --test "$ROOT_DIR/scripts/ci/generate-compose-deployment.test.mjs"
bash "$ROOT_DIR/scripts/ci/verify-compose-cert-init-selftest.sh"
bash "$ROOT_DIR/scripts/ci/verify-compose-config-init-selftest.sh"

grep -Fq 'docker compose up -d' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must use direct Compose startup"
grep -Fq 'PowerShell' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must cover native Windows PowerShell"
grep -Fq 'docker compose down' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must document direct Compose shutdown"
grep -Fq 'docker compose exec server resetadmin' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must retain administrator reset"
grep -Fq 'Docker Compose 2.24.0' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must state the minimum Compose version"
grep -Fq 'DATABASE_MODE=external' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must describe external PostgreSQL mode"
grep -Fq '## Configuration changes after first start' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must document post-start configuration changes"
grep -Fq 'online database-mode migration is unsupported' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must state that online database-mode migration is unsupported"
grep -Fq 'online credential rotation is unsupported' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must state that online credential rotation is unsupported"
grep -Fq 'lunafox_config' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must identify the persisted configuration volume"
grep -Fq 'lunafox_postgres' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must identify the embedded database volume"
grep -Fq '## 首次启动后的配置变更' "$ROOT_DIR/docs/public-deployment.zh-CN.md" || fail "Chinese deployment docs must document post-start configuration changes"
grep -Fq '不支持在线数据库模式迁移' "$ROOT_DIR/docs/public-deployment.zh-CN.md" || fail "Chinese deployment docs must state that online database-mode migration is unsupported"
grep -Fq '不支持在线凭据轮换' "$ROOT_DIR/docs/public-deployment.zh-CN.md" || fail "Chinese deployment docs must state that online credential rotation is unsupported"
grep -Fq 'docs/public-deployment.md#configuration-changes-after-first-start' "$ROOT_DIR/README.md" "$ROOT_DIR/README.zh-CN.md" || fail "README files must link to post-start configuration guidance"
# A failed start may already have created containers, so the documentation has to
# describe the preserved scene instead of claiming nothing changed.
grep -Fq 'nothing is rolled back' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must state that a failure leaves the deployment in place and rolls nothing back"
if grep -Eq 'configuration untouched|were not changed' "$ROOT_DIR/docs/public-deployment.md"; then
	fail "deployment docs must not claim a failed start left the deployment unchanged"
fi
grep -Fq './uninstall.sh --purge --confirm' "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must document the only command that deletes named volumes"
grep -Fq 'git clone https://github.com/yyhuni/lunafox.git' "$ROOT_DIR/README.md" || fail "public README must document direct main checkout installation"
grep -Fq 'lunafox-<version>.zip' "$ROOT_DIR/README.md" || fail "public README must document the unified release ZIP"
grep -Fq 'RELEASE_REGISTRY=docker.io' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must state the Docker Hub default"
grep -Fq 'RELEASE_REGISTRY=ghcr.io' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/public-deployment.md" || fail "deployment docs must explain explicit GHCR selection"
if rg -n 'prepare-deployment\.sh|-dockerhub\.zip|-ghcr\.zip|--registry (dockerhub|ghcr)' "$ROOT_DIR/README.md" "$ROOT_DIR/docs/public-deployment.md"; then
	fail "deployment docs contain a retired installation path"
fi

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
