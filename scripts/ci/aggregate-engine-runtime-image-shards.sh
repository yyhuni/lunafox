#!/usr/bin/env bash
set -euo pipefail

# The protected public workflow publishes one immutable Runtime Image receipt per
# Engine. This helper is the only place those candidate shards become the
# complete receipt that signing and Package generation may consume.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENGINE_ROOT="${ENGINE_ROOT:-$ROOT_DIR/extensions/engines}"
MODE="${ENGINE_RELEASE_MODE:-production}"
DISCOVERY_INPUT="${ENGINE_RUNTIME_IMAGE_DISCOVERY:-}"
RELEASE_CONTEXT_INPUT="${ENGINE_RUNTIME_IMAGE_RELEASE_CONTEXT:-}"
SHARDS_ROOT="${ENGINE_RUNTIME_IMAGE_SHARDS_ROOT:-}"
OUTPUT_ROOT="${ENGINE_RUNTIME_IMAGE_AGGREGATE_OUTPUT_ROOT:-$ROOT_DIR/dist/public-engine-runtime}"

usage() {
	cat <<'USAGE'
Usage: scripts/ci/aggregate-engine-runtime-image-shards.sh

Environment:
  ENGINE_ROOT                              Builtin Engine source root
  ENGINE_RELEASE_MODE                       Must be production
  ENGINE_RUNTIME_IMAGE_DISCOVERY            Discovery artifact JSON from the
                                             protected source-bound job
  ENGINE_RUNTIME_IMAGE_RELEASE_CONTEXT      Release context JSON from the
                                             protected source-bound job
  ENGINE_RUNTIME_IMAGE_SHARDS_ROOT          Directory containing all selected
                                             Engine Runtime Image artifacts
  ENGINE_RUNTIME_IMAGE_AGGREGATE_OUTPUT_ROOT
                                             Empty output directory for the
                                             complete receipt and evidence
USAGE
}

fail() {
	echo "aggregate Engine Runtime Image shards: $*" >&2
	exit 1
}

require_regular_file() {
	local path="$1" label="$2"
	[ -n "$path" ] || fail "$label is required"
	[ -f "$path" ] && [ ! -L "$path" ] || fail "$label must be a regular non-symlink file: $path"
}

require_directory() {
	local path="$1" label="$2"
	[ -n "$path" ] || fail "$label is required"
	[ -d "$path" ] && [ ! -L "$path" ] || fail "$label must be a directory, not a symlink: $path"
}

safe_engine_id() {
	local engine_id="$1" safe_id
	safe_id="${engine_id//[^A-Za-z0-9_.-]/_}"
	[ "$safe_id" = "$engine_id" ] || fail "Engine ID is unsafe for an artifact path: $engine_id"
	printf '%s\n' "$safe_id"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
		;;
	*) fail "unknown argument: $1" ;;
	esac
done

[ "$MODE" = production ] || fail "ENGINE_RELEASE_MODE must be production"
command -v go >/dev/null 2>&1 || fail "go is required"
command -v jq >/dev/null 2>&1 || fail "jq is required"
command -v node >/dev/null 2>&1 || fail "node is required"
command -v cmp >/dev/null 2>&1 || fail "cmp is required"

ENGINE_ROOT="$(cd "$ENGINE_ROOT" && pwd)" || fail "ENGINE_ROOT does not exist: $ENGINE_ROOT"
require_directory "$ENGINE_ROOT" "ENGINE_ROOT"
require_regular_file "$DISCOVERY_INPUT" "ENGINE_RUNTIME_IMAGE_DISCOVERY"
require_regular_file "$RELEASE_CONTEXT_INPUT" "ENGINE_RUNTIME_IMAGE_RELEASE_CONTEXT"
require_directory "$SHARDS_ROOT" "ENGINE_RUNTIME_IMAGE_SHARDS_ROOT"
if [ -e "$OUTPUT_ROOT" ] || [ -L "$OUTPUT_ROOT" ]; then
	fail "ENGINE_RUNTIME_IMAGE_AGGREGATE_OUTPUT_ROOT must not already exist: $OUTPUT_ROOT"
fi
if [ -n "$(find "$SHARDS_ROOT" -type l -print -quit)" ]; then
	fail "Runtime Image shard artifacts must not contain symlinks"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
tool="$tmp_dir/engine-release"
current_discovery="$tmp_dir/current-discovery.json"
records_dir="$tmp_dir/records"
aggregate_root="$tmp_dir/public-engine-runtime"
aggregate_evidence="$aggregate_root/registry-evidence"
mkdir -p "$records_dir" "$aggregate_evidence"

(cd "$ROOT_DIR/tools/engine-release" && go build -trimpath -buildvcs=false -o "$tool" .)
"$tool" -command discover -engine-root "$ENGINE_ROOT" -output "$current_discovery"

# `discover` writes a canonical, stable byte representation. Comparing it
# directly rejects a stale or substituted matrix source instead of treating an
# artifact as an alternative source of truth.
cmp -s "$DISCOVERY_INPUT" "$current_discovery" ||
	fail "discovery artifact does not exactly match current Engine source discovery"
jq -e '
  type == "object" and
  .schemaVersion == "lunafox.engine-release-discovery.v1" and
  (.engines | type == "array" and length > 0) and
  all(.engines[]; (.engineId | type == "string") and (.directory | type == "string") and (.dockerfile | type == "string") and (.buildContext | type == "string") and (.repository | type == "string"))
' "$current_discovery" >/dev/null || fail "current Engine discovery has an invalid shape"
jq -e '
  type == "object" and
  .schemaVersion == "lunafox.engine-release-context.v1" and
  (.releaseTag | type == "string" and test("^v[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z.-]+)?$")) and
  (.publicMergeCommit | type == "string" and test("^[a-f0-9]{40}$")) and
  (.sourceRevisionDigest | type == "string" and test("^sha256:[a-f0-9]{64}$")) and
  (.publicExportManifestSha256 | type == "string" and test("^sha256:[a-f0-9]{64}$")) and
  (.publicProvenanceSha256 | type == "string" and test("^sha256:[a-f0-9]{64}$")) and
  .signerIdentity == "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main"
' "$RELEASE_CONTEXT_INPUT" >/dev/null || fail "release context has an invalid shape"

expected_ids_file="$tmp_dir/expected-engine-ids.txt"
receipt_ids_file="$tmp_dir/receipt-engine-ids.txt"
verification_ids_file="$tmp_dir/verification-engine-ids.txt"
receipt_paths_file="$tmp_dir/receipt-paths.txt"
verification_paths_file="$tmp_dir/verification-paths.txt"
jq -er '.engines[] | .engineId' "$current_discovery" >"$expected_ids_file"
expected_count="$(wc -l <"$expected_ids_file" | tr -d '[:space:]')"
[ "$expected_count" -gt 0 ] || fail "current discovery found no Engines"
while IFS= read -r engine_id; do
	[[ "$engine_id" =~ ^engine\.lunafox\.[a-z][a-z0-9_]*$ ]] || fail "discovery returned a non-canonical Engine ID: $engine_id"
done <"$expected_ids_file"

find "$SHARDS_ROOT" -type f -name build-results.json -print | LC_ALL=C sort >"$receipt_paths_file"
receipt_count="$(wc -l <"$receipt_paths_file" | tr -d '[:space:]')"
[ "$receipt_count" -eq "$expected_count" ] ||
	fail "Runtime Image shard receipt count $receipt_count does not match discovery count $expected_count"
: >"$receipt_ids_file"
while IFS= read -r receipt_path; do
	engine_id="$(jq -er '
    if type != "object" or (.engines | type != "array") or (.engines | length != 1) then
      error("receipt must contain exactly one Engine")
    else
      .engines[0].engineId
	    end
	  ' "$receipt_path")" || fail "shard receipt must contain exactly one Engine: $receipt_path"
	[[ "$engine_id" =~ ^engine\.lunafox\.[a-z][a-z0-9_]*$ ]] || fail "shard receipt names a non-canonical Engine: $engine_id"
	grep -Fqx -- "$engine_id" "$expected_ids_file" || fail "shard receipt names an undiscovered Engine: $engine_id"
	if grep -Fqx -- "$engine_id" "$receipt_ids_file"; then
		fail "duplicate Runtime Image shard receipt for Engine: $engine_id"
	fi
	"$tool" -command validate-build-results -engine-root "$ENGINE_ROOT" -engine-id "$engine_id" -build-results "$receipt_path" -mode "$MODE" >/dev/null ||
		fail "invalid selected-Engine Runtime Image receipt for $engine_id"
	safe_id="$(safe_engine_id "$engine_id")"
	jq -c '.engines[0]' "$receipt_path" >"$records_dir/$safe_id.json"
	printf '%s\n' "$engine_id" >>"$receipt_ids_file"
done <"$receipt_paths_file"
while IFS= read -r engine_id; do
	grep -Fqx -- "$engine_id" "$receipt_ids_file" || fail "missing Runtime Image shard receipt for Engine: $engine_id"
done <"$expected_ids_file"

find "$SHARDS_ROOT" -type f -name verification.json -print | LC_ALL=C sort >"$verification_paths_file"
verification_count="$(wc -l <"$verification_paths_file" | tr -d '[:space:]')"
[ "$verification_count" -eq "$expected_count" ] ||
	fail "Runtime Image shard evidence count $verification_count does not match discovery count $expected_count"
: >"$verification_ids_file"
while IFS= read -r verification_path; do
	engine_id="$(jq -er '.engineId' "$verification_path")" || fail "shard evidence has no Engine ID: $verification_path"
	[[ "$engine_id" =~ ^engine\.lunafox\.[a-z][a-z0-9_]*$ ]] || fail "shard evidence names a non-canonical Engine: $engine_id"
	grep -Fqx -- "$engine_id" "$expected_ids_file" || fail "shard evidence names an undiscovered Engine: $engine_id"
	if grep -Fqx -- "$engine_id" "$verification_ids_file"; then
		fail "duplicate Runtime Image shard evidence for Engine: $engine_id"
	fi
	safe_id="$(safe_engine_id "$engine_id")"
	record_path="$records_dir/$safe_id.json"
	require_regular_file "$record_path" "receipt record for $engine_id"
	index_digest="$(jq -er '.indexDigest' "$record_path")"
	docker_ref="$(jq -er '.refs[0]' "$record_path")"
	ghcr_ref="$(jq -er '.refs[1]' "$record_path")"
	jq -e \
		--arg engineID "$engine_id" \
		--arg indexDigest "$index_digest" \
		--arg dockerRef "$docker_ref" \
		--arg ghcrRef "$ghcr_ref" '
			type == "object" and
			.schemaVersion == "lunafox.engine-runtime-image-registry-evidence.v1" and
			.engineId == $engineID and
			.indexDigest == $indexDigest and
			.platforms == ["linux/amd64", "linux/arm64"] and
			.registries.dockerHub.ref == $dockerRef and
			.registries.dockerHub.rawDescriptor == "dockerhub-index.json" and
			.registries.ghcr.ref == $ghcrRef and
			.registries.ghcr.rawDescriptor == "ghcr-index.json" and
			.copiedWithoutRebuild == true and
			.digestEqualityVerified == true and
			.platformSetVerified == true
		' "$verification_path" >/dev/null || fail "shard evidence does not match receipt for Engine: $engine_id"
	evidence_dir="$(dirname "$verification_path")"
	docker_descriptor="$evidence_dir/dockerhub-index.json"
	ghcr_descriptor="$evidence_dir/ghcr-index.json"
	require_regular_file "$docker_descriptor" "Docker Hub descriptor for $engine_id"
	require_regular_file "$ghcr_descriptor" "GHCR descriptor for $engine_id"
	node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" \
		--raw-file "$docker_descriptor" \
		--expected-digest "$index_digest" \
		--expected-platforms "linux/amd64,linux/arm64" >/dev/null
	node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" \
		--raw-file "$ghcr_descriptor" \
		--expected-digest "$index_digest" \
		--expected-platforms "linux/amd64,linux/arm64" >/dev/null
	mkdir -p "$aggregate_evidence/$safe_id"
	cp "$verification_path" "$aggregate_evidence/$safe_id/verification.json"
	cp "$docker_descriptor" "$aggregate_evidence/$safe_id/dockerhub-index.json"
	cp "$ghcr_descriptor" "$aggregate_evidence/$safe_id/ghcr-index.json"
	printf '%s\n' "$engine_id" >>"$verification_ids_file"
done <"$verification_paths_file"
while IFS= read -r engine_id; do
	grep -Fqx -- "$engine_id" "$verification_ids_file" || fail "missing Runtime Image shard evidence for Engine: $engine_id"
done <"$expected_ids_file"

jq -s --arg schema "lunafox.engine-runtime-image-build-results.v1" --arg mode "$MODE" \
	'{schemaVersion:$schema,mode:$mode,engines:(sort_by(.engineId))}' \
	"$records_dir"/*.json >"$aggregate_root/build-results.json"
"$tool" -command validate-build-results -engine-root "$ENGINE_ROOT" -build-results "$aggregate_root/build-results.json" -mode "$MODE" >/dev/null ||
	fail "combined Runtime Image receipt does not match complete source discovery"
cp "$RELEASE_CONTEXT_INPUT" "$aggregate_root/release-context.json"

mkdir -p "$(dirname "$OUTPUT_ROOT")"
mv "$aggregate_root" "$OUTPUT_ROOT"
echo "wrote verified complete Engine Runtime Image receipt: $OUTPUT_ROOT/build-results.json"
