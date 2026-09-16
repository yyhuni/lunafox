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
# Refuse an existing immutable tag with a different graph, including retries
# whose platform build returned different provenance bytes.
docker buildx imagetools create --dry-run "${source_refs[@]}" >"$tmp_dir/proposed-index.json"
existing_digest="$(oras resolve "$docker_tag" 2>/dev/null || true)"
if [ -n "$existing_digest" ]; then
	docker buildx imagetools inspect --raw "$docker_tag" >"$tmp_dir/existing-index.json"
	jq -S . "$tmp_dir/proposed-index.json" >"$tmp_dir/proposed-canonical.json"
	jq -S . "$tmp_dir/existing-index.json" >"$tmp_dir/existing-canonical.json"
	cmp -s "$tmp_dir/proposed-canonical.json" "$tmp_dir/existing-canonical.json" || fail "immutable index tag already has a different graph"
else
	docker buildx imagetools create --tag "$docker_tag" "${source_refs[@]}" >/dev/null
fi
index_digest="$(docker buildx imagetools inspect "$docker_tag" | awk '/^Digest:/ && digest == "" {digest=$2} END {print digest}')"
[[ "$index_digest" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "final index digest is invalid"
docker_raw="$tmp_dir/dockerhub-index.json"
docker buildx imagetools inspect --raw "$docker_tag" >"$docker_raw"
node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" --raw-file "$docker_raw" --expected-digest "$index_digest" --expected-platforms linux/amd64,linux/arm64 >/dev/null

docker_ref="docker.io/yyhuni/$repository@$index_digest"
ghcr_existing="$(oras resolve "$ghcr_tag" 2>/dev/null || true)"
[ -z "$ghcr_existing" ] || [ "$ghcr_existing" = "$index_digest" ] || fail "immutable GHCR index tag already has a different digest"
oras cp "$docker_ref" "$ghcr_tag"
ghcr_digest="$(docker buildx imagetools inspect "$ghcr_tag" | awk '/^Digest:/ && digest == "" {digest=$2} END {print digest}')"
[ "$ghcr_digest" = "$index_digest" ] || fail "cross-Registry index digest drift"
ghcr_raw="$tmp_dir/ghcr-index.json"
docker buildx imagetools inspect --raw "$ghcr_tag" >"$ghcr_raw"
node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" --raw-file "$ghcr_raw" --expected-digest "$index_digest" --expected-platforms linux/amd64,linux/arm64 >/dev/null
ghcr_ref="ghcr.io/yyhuni/$repository@$index_digest"

# The finalizer's native daemon proves the selected platform and image-local
# tool closure independently through both Registries.
for ref in "$docker_ref" "$ghcr_ref"; do
	docker image rm --force "$docker_ref" "$ghcr_ref" >/dev/null 2>&1 || true
	docker pull "$ref" >/dev/null
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
