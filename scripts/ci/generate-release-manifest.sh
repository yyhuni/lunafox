#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUTPUT_FILE="$ROOT_DIR/release.manifest.yaml"
VERIFY_CONTRACT=0
ENGINE_REF_DIR=""
FINAL_RUNTIME_CONTRACT=0
RELEASE_NOTES_FILE=""
RUNTIME_COMPOSITION_SHA256_ARG=""
RUNTIME_COMPOSITION_SHA256_ARG_SET=0
RELEASE_PROFILE_ARG=""
RELEASE_PROFILE_ARG_SET=0

usage() {
	cat <<'USAGE'
用法:
  RELEASE_VERSION=1.2.3 \
	RUNTIME_SERVER_REFS='docker.io/ns/lunafox-server@sha256:...,ghcr.io/owner/lunafox-server@sha256:...' \
	RUNTIME_FRONTEND_REFS='...' RUNTIME_NGINX_REFS='...' RUNTIME_AGENT_REFS='...' \
	RUNTIME_BOOTSTRAP_REFS='...' \
	RUNTIME_COMPOSITION_SHA256='sha256:<canonical-composition-digest>' \
	./scripts/ci/generate-release-manifest.sh --engine-ref-dir dist/engine-digests [--runtime-composition-sha256 <sha256:...>] [--release-profile <modern|alpha164-bridge>] [--output <path>] [--release-notes-file <path>] [--final-runtime-contract] [--verify-contract]

每组必须恰好包含 Docker Hub 和 GHCR 的同一个 sha256 manifest digest。引擎组从
--engine-ref-dir 中每个构建产物生成的 `ENGINE_ID` 和 `ENGINE_REFS` 读取。
	`runtimeImages` 始终只生成 server/frontend/nginx/agent/bootstrap；`--final-runtime-contract`
与 `--verify-contract` 配合时选择最终公开发布校验，不会开启另一套运行镜像契约。
USAGE
}

fail() {
	echo "✗ $*" >&2
	exit 1
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	-o | --output)
		[ "$#" -ge 2 ] || fail "$1 缺少参数"
		OUTPUT_FILE="$2"
		shift 2
		;;
	--engine-ref-dir)
		[ "$#" -ge 2 ] || fail "$1 缺少参数"
		ENGINE_REF_DIR="$2"
		shift 2
		;;
	--release-notes-file)
		[ "$#" -ge 2 ] || fail "$1 缺少参数"
		RELEASE_NOTES_FILE="$2"
		shift 2
		;;
	--final-runtime-contract)
		FINAL_RUNTIME_CONTRACT=1
		shift
		;;
	--verify-contract)
		VERIFY_CONTRACT=1
		shift
		;;
	--runtime-composition-sha256)
		[ "$#" -ge 2 ] || fail "$1 缺少参数"
		[ "$RUNTIME_COMPOSITION_SHA256_ARG_SET" -eq 0 ] || fail "--runtime-composition-sha256 不得重复"
		RUNTIME_COMPOSITION_SHA256_ARG="$2"
		RUNTIME_COMPOSITION_SHA256_ARG_SET=1
		shift 2
		;;
	--release-profile)
		[ "$#" -ge 2 ] || fail "$1 缺少参数"
		[ "$RELEASE_PROFILE_ARG_SET" -eq 0 ] || fail "--release-profile 不得重复"
		RELEASE_PROFILE_ARG="$2"
		RELEASE_PROFILE_ARG_SET=1
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*) fail "不支持的参数: $1" ;;
	esac
done

release_version="$(echo "${RELEASE_VERSION:-}" | xargs)"
[[ "$release_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([+-][-0-9A-Za-z.+_]+)?$ ]] || fail "RELEASE_VERSION 必须是裸语义化版本"

release_profile_env="${RELEASE_COMPATIBILITY_PROFILE-}"
if [ "$RELEASE_PROFILE_ARG_SET" -eq 1 ]; then
	if [ -n "$release_profile_env" ] && [ "$release_profile_env" != "$RELEASE_PROFILE_ARG" ]; then
		fail "--release-profile 与 RELEASE_COMPATIBILITY_PROFILE 不一致"
	fi
	release_profile_request="$RELEASE_PROFILE_ARG"
else
	release_profile_request="$release_profile_env"
fi
release_profile_args=(--release-version "$release_version")
if [ -n "$release_profile_request" ]; then
	release_profile_args+=(--release-profile "$release_profile_request")
fi
release_profile="$(node "$ROOT_DIR/scripts/ci/release-compatibility-profile.mjs" "${release_profile_args[@]}")" || fail "无法解析 release compatibility profile"

runtime_composition_sha256_env="${RUNTIME_COMPOSITION_SHA256-}"
if [ "$RUNTIME_COMPOSITION_SHA256_ARG_SET" -eq 1 ]; then
	if [ -n "$runtime_composition_sha256_env" ] && [ "$runtime_composition_sha256_env" != "$RUNTIME_COMPOSITION_SHA256_ARG" ]; then
		fail "--runtime-composition-sha256 与 RUNTIME_COMPOSITION_SHA256 不一致"
	fi
	runtime_composition_sha256="$RUNTIME_COMPOSITION_SHA256_ARG"
else
	runtime_composition_sha256="$runtime_composition_sha256_env"
fi
[[ "$runtime_composition_sha256" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "必须显式提供 runtime composition canonical sha256 digest（--runtime-composition-sha256 或 RUNTIME_COMPOSITION_SHA256）"
release_major="${release_version%%.*}"
release_major_number=$((10#$release_major))
next_release_major=$((release_major_number + 1))

MIGRATION_POLICY_FILE="$ROOT_DIR/server/cmd/server/migrations/policy.json"
MIGRATION_POLICY_VERSION="$(node -e 'const fs=require("node:fs"); const file=process.argv[1]; const policy=JSON.parse(fs.readFileSync(file,"utf8")); process.stdout.write(String(policy.schemaVersion));' "$MIGRATION_POLICY_FILE")" || fail "无法读取 migration policy version"
RELEASE_MIGRATION_ID="$(echo "${RELEASE_MIGRATION_ID:-}" | xargs)"
RELEASE_MIGRATION_TYPE="$(echo "${RELEASE_MIGRATION_TYPE:-none}" | xargs)"
RELEASE_MIGRATION_CHECKSUM="$(echo "${RELEASE_MIGRATION_CHECKSUM:-}" | xargs)"
if [ -n "$RELEASE_MIGRATION_ID" ]; then
	[ -n "$RELEASE_MIGRATION_CHECKSUM" ] || fail "RELEASE_MIGRATION_CHECKSUM is required when RELEASE_MIGRATION_ID is set"
	case "$RELEASE_MIGRATION_TYPE" in
	compatible | preserve-data | destructive) ;;
	*) fail "RELEASE_MIGRATION_TYPE must be compatible, preserve-data, or destructive" ;;
	esac
else
	[ "$RELEASE_MIGRATION_TYPE" = none ] || fail "RELEASE_MIGRATION_TYPE must be none when no migration is declared"
	[ -z "$RELEASE_MIGRATION_CHECKSUM" ] || fail "RELEASE_MIGRATION_CHECKSUM requires RELEASE_MIGRATION_ID"
fi
if [ "${RUNTIME_WORKER_REFS+x}" = x ]; then
	fail "RUNTIME_WORKER_REFS 已删除，Engine Runtime Image 由 package v2 绑定"
fi

if [[ "$OUTPUT_FILE" != /* ]]; then OUTPUT_FILE="$ROOT_DIR/$OUTPUT_FILE"; fi
if [ -z "$RELEASE_NOTES_FILE" ]; then
	RELEASE_NOTES_FILE="$ROOT_DIR/release-notes/v${release_version}.md"
elif [[ "$RELEASE_NOTES_FILE" != /* ]]; then
	RELEASE_NOTES_FILE="$ROOT_DIR/$RELEASE_NOTES_FILE"
fi
if [ "$release_version" != "0.0.0-dev" ] && [ ! -f "$RELEASE_NOTES_FILE" ]; then
	fail "缺少与 releaseVersion 匹配的 Release Notes: $RELEASE_NOTES_FILE"
fi
if [ "$release_version" != "0.0.0-dev" ] && [ "$(basename "$RELEASE_NOTES_FILE")" != "v${release_version}.md" ]; then
	fail "Release Notes 文件名必须匹配 releaseVersion: v${release_version}.md"
fi
mkdir -p "$(dirname "$OUTPUT_FILE")"
tmp_file="$(mktemp "$(dirname "$OUTPUT_FILE")/.release.manifest.XXXXXX.tmp")"

split_group() {
	local raw="$1" label="$2"
	IFS=',' read -r -a refs <<<"$raw"
	[ "${#refs[@]}" -eq 2 ] || fail "$label 必须包含恰好两个 refs"
	local first="${refs[0]//[[:space:]]/}" second="${refs[1]//[[:space:]]/}"
	[[ "$first" =~ ^docker\.io/.+@sha256:([a-f0-9]{64})$ ]] || fail "$label 首项必须是 docker.io sha256 ref"
	local digest="${BASH_REMATCH[1]}"
	[[ "$second" =~ ^ghcr\.io/.+@sha256:${digest}$ ]] || fail "$label 次项必须是同 digest 的 ghcr.io ref"
	printf '%s\n%s\n' "$first" "$second"
}

append_group() {
	local kind="$1" name="$2" raw="$3" label="$4"
	[ -n "$raw" ] || fail "缺少候选引用: $label"
	local expected_repository=""
	if [ "$kind" = engine ]; then
		# This is the first-party release boundary. Generic package consumers
		# intentionally allow arbitrary repositories, but a release manifest
		# must bind every Engine ID to the same Runtime Image/Package identity.
		[[ "$name" =~ ^engine\.lunafox\.([a-z][a-z0-9_]*)$ ]] || fail "Engine ID 不是规范的 first-party identity: $name"
		local local_name="${BASH_REMATCH[1]}"
		if [[ "$local_name" =~ _v[0-9]+(_[0-9]+)*$ ||
			"$local_name" =~ _sha256_[a-f0-9]{64}$ ||
			"$local_name" =~ _[a-f0-9]{7,64}$ ||
			"$local_name" =~ _release_[a-z0-9]+$ ||
			"$local_name" =~ _[0-9]+$ ]]; then
			fail "Engine ID 不得带 release-scoped suffix: $name"
		fi
		expected_repository="lunafox-engine-runtime-${local_name//_/-}"
	fi
	local refs=()
	local ref
	while IFS= read -r ref; do
		refs+=("$ref")
	done < <(split_group "$raw" "$label")
	if [ "$kind" = engine ]; then
		for ref in "${refs[@]}"; do
			local repository="${ref%@*}"
			repository="${repository##*/}"
			[ "$repository" = "$expected_repository" ] || fail "$label 必须使用由 Engine ID 推导的 repository ${expected_repository}，实际为 ${repository}"
		done
	fi
	if [ "$kind" = runtime ]; then
		printf '  - name: %s\n    refs:\n      - "%s"\n      - "%s"\n' "$name" "${refs[0]}" "${refs[1]}" >>"$tmp_file"
	else
		printf '  - refs:\n      - "%s"\n      - "%s"\n' "${refs[0]}" "${refs[1]}" >>"$tmp_file"
	fi
}

cat >"$tmp_file" <<EOF
# Auto-generated by scripts/ci/generate-release-manifest.sh. Do not edit.
releaseVersion: "$release_version"
EOF
if [ "$release_profile" = modern ] && [ -f "$RELEASE_NOTES_FILE" ]; then
	release_notes_digest="sha256:$(sha256sum "$RELEASE_NOTES_FILE" | awk '{print $1}')"
	cat >>"$tmp_file" <<EOF
releaseNotes:
  digest: "$release_notes_digest"
  body: |
EOF
	# Preserve the validated Markdown as inert YAML text; the shared manifest
	# parser verifies UTF-8, bilingual sections, size, and this digest.
	sed 's/^/    /' "$RELEASE_NOTES_FILE" >>"$tmp_file"
fi
cat >>"$tmp_file" <<'EOF'
runtimeImages:
EOF
runtime_components=(server frontend nginx agent bootstrap)
for component in "${runtime_components[@]}"; do
	upper="$(echo "$component" | tr '[:lower:]' '[:upper:]')"
	env_key="RUNTIME_${upper}_REFS"
	append_group runtime "$component" "${!env_key:-}" "$env_key"
done
printf 'enginePackages:\n' >>"$tmp_file"
[ -n "$ENGINE_REF_DIR" ] || fail "--engine-ref-dir 是必填参数"
if [[ "$ENGINE_REF_DIR" != /* ]]; then ENGINE_REF_DIR="$ROOT_DIR/$ENGINE_REF_DIR"; fi
[ -d "$ENGINE_REF_DIR" ] || fail "引擎候选目录不存在: $ENGINE_REF_DIR"
engine_ref_files=("$ENGINE_REF_DIR"/*.env)
[ -f "${engine_ref_files[0]}" ] || fail "引擎候选目录为空: $ENGINE_REF_DIR"
for engine_ref_file in "${engine_ref_files[@]}"; do
	engine_id="$(awk -F= '$1 == "ENGINE_ID" { print substr($0, index($0, "=") + 1); exit }' "$engine_ref_file" | xargs)"
	engine_refs="$(awk -F= '$1 == "ENGINE_REFS" { print substr($0, index($0, "=") + 1); exit }' "$engine_ref_file" | xargs)"
	[ -n "$engine_id" ] || fail "缺少 ENGINE_ID: $engine_ref_file"
	append_group engine "$engine_id" "$engine_refs" "$engine_ref_file"
done

if [ "$release_profile" = modern ]; then
	cat >>"$tmp_file" <<EOF
runtimeComposition:
  schemaVersion: 1
  asset: "runtime-composition.json"
  sha256: "$runtime_composition_sha256"
EOF
fi

cat >>"$tmp_file" <<EOF
upgrade:
  manifestId: "lunafox-$release_version"
  deploymentMode: "single-node-compose"
  compatibilityRange: ">=${release_major_number}.0.0 <${next_release_major}.0.0"
  maintenanceWindowMinutes: 15
  requiresAdminConfirmation: true
  databaseMigration:
    hasDatabaseMigration: $(if [ -n "$RELEASE_MIGRATION_ID" ]; then echo true; else echo false; fi)
    migrationType: "$RELEASE_MIGRATION_TYPE"
    migrationId: "$RELEASE_MIGRATION_ID"
    checksum: "$RELEASE_MIGRATION_CHECKSUM"
    policyVersion: $MIGRATION_POLICY_VERSION
EOF

# The shared Go contract owns strict manifest decoding. The public projection
# separately owns publication policy checks; keeping this check outside the
# installer avoids making the retired host-binary path a release prerequisite.
# Development manifests are intentionally not public release candidates, so
# they may omit release notes and must not enter the production preflight.
if [ "$release_version" != "0.0.0-dev" ]; then
	node "$ROOT_DIR/scripts/ci/verify-public-release.mjs" \
		--manifest "$tmp_file" \
		--tag "v$release_version" \
		--release-profile "$release_profile" >/dev/null
fi
node "$ROOT_DIR/scripts/ci/check-migration-baseline-policy.mjs" \
	--release-manifest "$tmp_file" \
	--release-channel release-candidate \
	--deployment-mode disposable >/dev/null
mv "$tmp_file" "$OUTPUT_FILE"

if [ "$VERIFY_CONTRACT" -eq 1 ]; then
	if [ "$FINAL_RUNTIME_CONTRACT" -eq 1 ]; then
		# The final manifest is composed in the public repository.  Its contract
		# is the public release verifier; the private release-contract script
		# intentionally remains a source-tree gate for the private workflow.
		node "$ROOT_DIR/scripts/ci/verify-public-release.mjs" \
			--manifest "$OUTPUT_FILE" --tag "v$release_version" \
			--release-profile "$release_profile" >/dev/null
	else
		bash "$ROOT_DIR/scripts/ci/verify-release-contract.sh" "$OUTPUT_FILE" "$ROOT_DIR/.tmp-non-existent.env" current
	fi
elif [ "$FINAL_RUNTIME_CONTRACT" -eq 1 ]; then
	fail "--final-runtime-contract requires --verify-contract"
fi
echo "✓ 已生成 release manifest: $OUTPUT_FILE"
