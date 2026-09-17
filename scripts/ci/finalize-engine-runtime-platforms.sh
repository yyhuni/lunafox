#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENGINE_ROOT="${ENGINE_ROOT:-$ROOT_DIR/extensions/engines}"
ENGINE_ID="${ENGINE_RELEASE_ENGINE_ID:-}"
INPUT_ROOT="${ENGINE_PLATFORM_BUILDS_ROOT:-}"
TAG="${ENGINE_IMAGE_TAG:-}"
OUTPUT_ROOT="${ENGINE_RUNTIME_IMAGE_SHARD_ROOT:-$ROOT_DIR/dist/public-engine-runtime-shard}"

fail() {
	echo "finalize Engine Runtime platforms: $*" >&2
	exit 1
}
[ -n "$ENGINE_ID" ] || fail "ENGINE_RELEASE_ENGINE_ID is required"
[ -d "$INPUT_ROOT" ] || fail "ENGINE_PLATFORM_BUILDS_ROOT must be a directory"
[ -z "$(find "$INPUT_ROOT" -type l -print -quit)" ] || fail "platform receipts must not contain symlinks"
[[ "$TAG" =~ ^public-[a-f0-9]{40}$ ]] || fail "ENGINE_IMAGE_TAG must bind the protected commit SHA"
[ ! -e "$OUTPUT_ROOT" ] || fail "ENGINE_RUNTIME_IMAGE_SHARD_ROOT must not exist"
for command in docker jq node oras go; do command -v "$command" >/dev/null 2>&1 || fail "$command is required"; done

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
receipts=()
while IFS= read -r -d '' receipt; do receipts+=("$receipt"); done < <(find "$INPUT_ROOT" -type f -name platform-build.json -print0)
[ "${#receipts[@]}" -eq 2 ] || fail "expected exactly two platform receipts, got ${#receipts[@]}"
records="$tmp_dir/records.json"
jq -s '.' "${receipts[@]}" >"$records"
jq -e --arg engineId "$ENGINE_ID" --arg tag "$TAG" '
  length == 2 and all(.[];
    (keys | sort) == (["schemaVersion","engineId","directory","dockerfile","buildContext","repository","platform","digest","sourceRef","tagIdentity"] | sort) and
    .schemaVersion == "lunafox.engine-runtime-platform-build.v1" and .engineId == $engineId and
    .tagIdentity == $tag and (.digest | test("^sha256:[a-f0-9]{64}$")) and
    .sourceRef == ("docker.io/yyhuni/" + .repository + "@" + .digest)) and
  ([.[].platform] | sort) == ["linux/amd64","linux/arm64"] and
  ([.[].repository] | unique | length) == 1 and ([.[].dockerfile] | unique | length) == 1 and
  ([.[].buildContext] | unique | length) == 1 and ([.[].directory] | unique | length) == 1
' "$records" >/dev/null || fail "platform receipts do not form one exact dual-architecture Engine build"

repository="$(jq -er '.[0].repository' "$records")"
directory="$(jq -er '.[0].directory' "$records")"
dockerfile="$(jq -er '.[0].dockerfile' "$records")"
build_context="$(jq -er '.[0].buildContext' "$records")"
docker_tag="docker.io/yyhuni/$repository:$TAG"
ghcr_tag="ghcr.io/yyhuni/$repository:$TAG"
source_refs=()
while IFS= read -r source_ref; do source_refs+=("$source_ref"); done < <(jq -er 'sort_by(.platform) | .[].sourceRef' "$records")
# Re-derive identity from this checkout before any final tag is written.
(cd "$ROOT_DIR/tools/engine-release" && go run . -command discover -engine-root "$ENGINE_ROOT" -engine-id "$ENGINE_ID" -output "$tmp_dir/discovery.json")
jq -e --slurpfile records "$records" '
  (.engines | length) == 1 and all($records[0][];
    .engineId == $ARGS.named.engineId)
' --arg engineId "$ENGINE_ID" "$tmp_dir/discovery.json" >/dev/null
jq -e --slurpfile discovery "$tmp_dir/discovery.json" '
  all(.[]; .engineId == $discovery[0].engines[0].engineId and
    .repository == $discovery[0].engines[0].repository and
    .directory == $discovery[0].engines[0].directory and
    .dockerfile == $discovery[0].engines[0].dockerfile and
    .buildContext == $discovery[0].engines[0].buildContext)
' "$records" >/dev/null || fail "platform receipt does not match checked-out discovery"
for platform in linux/amd64 linux/arm64; do
	source_ref="$(jq -er --arg platform "$platform" '.[] | select(.platform == $platform) | .sourceRef' "$records")"
	docker buildx imagetools inspect --raw "$source_ref" >"$tmp_dir/platform-${platform#linux/}.json"
	node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" --raw-file "$tmp_dir/platform-${platform#linux/}.json" --expected-digest "${source_ref##*@}" --expected-platforms "$platform" >/dev/null
done
# Platform rebuilds attach fresh provenance/SBOM bytes, so a retry of the same
# protected merge can propose a different graph. The first successful index
# write for this tag is the immutable source of truth; later retries reuse it.
resolve_optional_tag() {
	local tag="$1" output error
	output="$(mktemp "$tmp_dir/resolve.XXXXXX")"
	error="$(mktemp "$tmp_dir/resolve-error.XXXXXX")"
	if oras resolve "$tag" >"$output" 2>"$error"; then
		cat "$output"
	elif grep -Eqi 'MANIFEST_UNKNOWN|manifest unknown|not found|404' "$error"; then
		return 0
	else
		cat "$error" >&2
		return 1
	fi
}
retry_transient_registry() {
	local description="$1"
	shift
	local attempt=1 delay=2 output_file error_file
	output_file="$(mktemp "$tmp_dir/retry-output.XXXXXX")"
	error_file="$(mktemp "$tmp_dir/retry-error.XXXXXX")"
	while :; do
		: >"$output_file"
		: >"$error_file"
		if "$@" >"$output_file" 2>"$error_file"; then
			cat "$output_file"
			return 0
		fi
		if [ "$attempt" -ge 6 ] || ! grep -Eqi 'MANIFEST_UNKNOWN|manifest unknown|not found|404' "$error_file"; then
			cat "$error_file" >&2
			return 1
		fi
		printf 'Registry %s failed (attempt %s/6); retrying in %ss\n' "$description" "$attempt" "$delay" >&2
		sleep "$delay"
		delay=$((delay * 2))
		[ "$delay" -le 8 ] || delay=8
		attempt=$((attempt + 1))
	done
}
existing_digest="$(resolve_optional_tag "$docker_tag")"
if [ -n "$existing_digest" ]; then
	retry_transient_registry "inspect $docker_tag" \
		docker buildx imagetools inspect --raw "$docker_tag" >"$tmp_dir/existing-index.json"
	node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" \
		--raw-file "$tmp_dir/existing-index.json" --expected-digest "$existing_digest" --expected-platforms linux/amd64,linux/arm64 >/dev/null
	printf 'reusing immutable Engine index %s\n' "$existing_digest" >&2
else
	docker buildx imagetools create --tag "$docker_tag" "${source_refs[@]}" >/dev/null
fi
index_digest="$(retry_transient_registry "inspect digest $docker_tag" \
	docker buildx imagetools inspect "$docker_tag" | awk '/^Digest:/ && digest == "" {digest=$2} END {print digest}')"
[[ "$index_digest" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "final index digest is invalid"
[ -z "$existing_digest" ] || [ "$existing_digest" = "$index_digest" ] || fail "resolved immutable digest drifted during inspect"
docker_raw="$tmp_dir/dockerhub-index.json"
retry_transient_registry "inspect raw $docker_tag" \
	docker buildx imagetools inspect --raw "$docker_tag" >"$docker_raw"
node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" --raw-file "$docker_raw" --expected-digest "$index_digest" --expected-platforms linux/amd64,linux/arm64 >/dev/null

docker_ref="docker.io/yyhuni/$repository@$index_digest"
ghcr_existing="$(resolve_optional_tag "$ghcr_tag")"
[ -z "$ghcr_existing" ] || [ "$ghcr_existing" = "$index_digest" ] || fail "immutable GHCR index tag already has a different digest"
retry_transient_registry "oras cp $docker_ref" oras cp "$docker_ref" "$ghcr_tag"
ghcr_digest="$(retry_transient_registry "inspect digest $ghcr_tag" \
	docker buildx imagetools inspect "$ghcr_tag" | awk '/^Digest:/ && digest == "" {digest=$2} END {print digest}')"
[ "$ghcr_digest" = "$index_digest" ] || fail "cross-Registry index digest drift"
ghcr_raw="$tmp_dir/ghcr-index.json"
retry_transient_registry "inspect raw $ghcr_tag" \
	docker buildx imagetools inspect --raw "$ghcr_tag" >"$ghcr_raw"
node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" --raw-file "$ghcr_raw" --expected-digest "$index_digest" --expected-platforms linux/amd64,linux/arm64 >/dev/null
ghcr_ref="ghcr.io/yyhuni/$repository@$index_digest"

normalize_arch() {
	case "$1" in
	x86_64) echo amd64 ;;
	aarch64) echo arm64 ;;
	*) echo "$1" ;;
	esac
}

evict_host_pull_candidates() {
	local candidate image_id
	for candidate in "$@"; do
		[ -n "$candidate" ] || continue
		image_id="$(docker image inspect --format '{{.Id}}' "$candidate" 2>/dev/null || true)"
		docker image rm --force "$candidate" >/dev/null 2>&1 || true
		if [ -n "$image_id" ]; then
			# Removing the image ID clears sibling repository references and
			# prevents the second Registry pull from reusing the first pull's
			# manifest/layer ownership as its evidence.
			docker image rm --force "$image_id" >/dev/null 2>&1 || true
		fi
	done
	for candidate in "$@"; do
		[ -n "$candidate" ] || continue
		if docker image inspect "$candidate" >/dev/null 2>&1; then
			fail "Runtime Image candidate remained before independent cold pull: $candidate"
		fi
	done
}

verify_host_pull() {
	local ref="$1" engine_id="$2" label="$3"
	shift 3
	local dockerhub_normalized_ref="${ref#docker.io/}"
	# Docker Engine drops the explicit docker.io/ prefix from RepoDigests for
	# Docker Hub images. The digest remains immutable; accept that transport
	# canonicalization while retaining an exact digest comparison.
	# Pull through the same host Docker daemon Agent uses. A registry index can
	# be structurally valid while its selected platform is unavailable to that
	# daemon. Evict both same-digest candidate identities before each pull so
	# Docker Hub and GHCR independently prove their manifest and blob closure.
	evict_host_pull_candidates "$ref" "$@"
	docker pull "$ref" >/dev/null
	local selected_os selected_arch daemon_os daemon_arch
	local repo_digests
	repo_digests="$(docker image inspect --format '{{range .RepoDigests}}{{println .}}{{end}}' "$ref")"
	if ! printf '%s\n' "$repo_digests" | grep -Fqx "$ref" &&
		! printf '%s\n' "$repo_digests" | grep -Fqx "$dockerhub_normalized_ref"; then
		fail "$label pull did not retain the requested digest reference for $engine_id: $ref"
	fi
	selected_os="$(docker image inspect --format '{{.Os}}' "$ref")"
	selected_arch="$(normalize_arch "$(docker image inspect --format '{{.Architecture}}' "$ref")")"
	daemon_os="$(docker info --format '{{.OSType}}')"
	daemon_arch="$(normalize_arch "$(docker info --format '{{.Architecture}}')")"
	[ "$selected_os/$selected_arch" = "$daemon_os/$daemon_arch" ] ||
		fail "$label pull selected $selected_os/$selected_arch for $engine_id, want $daemon_os/$daemon_arch"
}

# The finalizer's native daemon proves the selected platform and image-local
# tool closure independently through both Registries.
for ref in "$docker_ref" "$ghcr_ref"; do
	verify_host_pull "$ref" "$ENGINE_ID" "finalized Runtime Image" "$docker_ref" "$ghcr_ref"
	node "$ROOT_DIR/scripts/ci/check-engine-image-tool-inventory.mjs" \
		--repo-root "$ROOT_DIR" --skip-product-image-source-check --engine-image "$directory=$ref" >/dev/null
done

evidence="$OUTPUT_ROOT/registry-evidence/${ENGINE_ID//[^A-Za-z0-9_.-]/_}"
mkdir -p "$evidence"
cp "$docker_raw" "$evidence/dockerhub-index.json"
cp "$ghcr_raw" "$evidence/ghcr-index.json"
jq -n --arg engineId "$ENGINE_ID" --arg indexDigest "$index_digest" --arg dockerHubRef "$docker_ref" --arg ghcrRef "$ghcr_ref" \
	'{schemaVersion:"lunafox.engine-runtime-image-registry-evidence.v1",engineId:$engineId,indexDigest:$indexDigest,platforms:["linux/amd64","linux/arm64"],registries:{dockerHub:{ref:$dockerHubRef,rawDescriptor:"dockerhub-index.json"},ghcr:{ref:$ghcrRef,rawDescriptor:"ghcr-index.json"}},copiedWithoutRebuild:true,digestEqualityVerified:true,platformSetVerified:true}' \
	>"$evidence/verification.json"
jq -n --arg engineId "$ENGINE_ID" --arg dockerfile "$dockerfile" --arg buildContext "$build_context" --arg repository "$repository" --arg indexDigest "$index_digest" --arg dockerRef "$docker_ref" --arg ghcrRef "$ghcr_ref" \
	'{schemaVersion:"lunafox.engine-runtime-image-build-results.v1",mode:"production",engines:[{engineId:$engineId,dockerfile:$dockerfile,buildContext:$buildContext,repository:$repository,buildCount:2,indexDigest:$indexDigest,indexMediaType:"application/vnd.oci.image.index.v1+json",platforms:["linux/amd64","linux/arm64"],refs:[$dockerRef,$ghcrRef],sourceRef:$dockerRef,copiedRefs:[$ghcrRef]}]}' \
	>"$OUTPUT_ROOT/build-results.json"
(cd "$ROOT_DIR/tools/engine-release" && go run . -command validate-build-results -engine-root "$ENGINE_ROOT" -engine-id "$ENGINE_ID" -build-results "$OUTPUT_ROOT/build-results.json" -mode production) >/dev/null
echo "wrote verified native multi-architecture Engine receipt: $OUTPUT_ROOT/build-results.json"
