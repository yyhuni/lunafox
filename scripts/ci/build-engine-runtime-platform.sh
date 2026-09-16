#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENGINE_ROOT="${ENGINE_ROOT:-$ROOT_DIR/extensions/engines}"
ENGINE_ID="${ENGINE_RELEASE_ENGINE_ID:-}"
PLATFORM="${ENGINE_RUNTIME_IMAGE_PLATFORM:-}"
VERSION="${ENGINE_VERSION:-}"
OUTPUT="${ENGINE_PLATFORM_BUILD_OUTPUT:-$ROOT_DIR/dist/engine-runtime-platform/platform-build.json}"
IMAGE_SOURCE="${ENGINE_IMAGE_SOURCE:-https://github.com/${GITHUB_REPOSITORY:-yyhuni/lunafox}}"
TAG_IDENTITY="${ENGINE_IMAGE_TAG:-}"

fail() {
	echo "build Engine Runtime platform: $*" >&2
	exit 1
}
normalize_arch() { case "$1" in x86_64) echo amd64 ;; aarch64 | arm64) echo arm64 ;; *) echo "$1" ;; esac }

[ -n "$ENGINE_ID" ] || fail "ENGINE_RELEASE_ENGINE_ID is required"
[ -n "$VERSION" ] || fail "ENGINE_VERSION is required"
[[ "$TAG_IDENTITY" =~ ^public-[a-f0-9]{40}$ ]] || fail "ENGINE_IMAGE_TAG must bind the protected commit SHA"
case "$PLATFORM" in linux/amd64 | linux/arm64) ;; *) fail "ENGINE_RUNTIME_IMAGE_PLATFORM must be linux/amd64 or linux/arm64" ;; esac
[ "linux/$(normalize_arch "$(uname -m)")" = "$PLATFORM" ] || fail "runner architecture does not match $PLATFORM"
for command in docker go jq; do command -v "$command" >/dev/null 2>&1 || fail "$command is required"; done

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
tool="$tmp_dir/engine-release"
discovery="$tmp_dir/discovery.json"
metadata="$tmp_dir/build-metadata.json"
(cd "$ROOT_DIR/tools/engine-release" && go build -trimpath -buildvcs=false -o "$tool" .)
"$tool" -command discover -engine-root "$ENGINE_ROOT" -engine-id "$ENGINE_ID" -output "$discovery"
[ "$(jq -er '.engines | length' "$discovery")" = 1 ] || fail "selected discovery must contain exactly one Engine"
directory="$(jq -er '.engines[0].directory' "$discovery")"
dockerfile="$(jq -er '.engines[0].dockerfile' "$discovery")"
build_context="$(jq -er '.engines[0].buildContext' "$discovery")"
repository="$(jq -er '.engines[0].repository' "$discovery")"
image="docker.io/yyhuni/$repository"
arch="${PLATFORM#linux/}"

args=(
	--platform "$PLATFORM"
	--output "type=image,name=$image,push-by-digest=true,name-canonical=true,push=true,oci-mediatypes=true"
	--metadata-file "$metadata"
	--build-context "contracts=$ROOT_DIR/contracts"
	--build-context "engine-go=$ROOT_DIR/engine-go"
	--build-arg "ENGINE_IMAGE_VERSION=$VERSION"
	--build-arg "ENGINE_IMAGE_SOURCE=$IMAGE_SOURCE"
	--file "$ENGINE_ROOT/$dockerfile"
	--provenance=mode=max
	--sbom=true
	--cache-from "type=registry,ref=ghcr.io/yyhuni/$repository:buildcache-$arch"
	--cache-from "type=registry,ref=ghcr.io/yyhuni/$repository:buildcache"
	--cache-to "type=registry,ref=ghcr.io/yyhuni/$repository:buildcache-$arch,mode=max,image-manifest=true,oci-mediatypes=true,ignore-error=true"
)
attempt=1
until docker buildx build "${args[@]}" "$ENGINE_ROOT/$build_context"; do
	[ "$attempt" -lt 3 ] || fail "build failed after $attempt attempts"
	sleep "$((attempt * 2))"
	attempt=$((attempt + 1))
done
digest="$(jq -er '."containerimage.digest"' "$metadata")"
[[ "$digest" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "BuildKit returned an invalid digest"
# Execute conformance on the native architecture before recording a shard.
docker pull "$image@$digest" >/dev/null
node "$ROOT_DIR/scripts/ci/check-engine-image-tool-inventory.mjs" \
	--repo-root "$ROOT_DIR" --skip-product-image-source-check --engine-image "$directory=$image@$digest" >/dev/null
mkdir -p "$(dirname "$OUTPUT")"
jq -n \
	--arg engineId "$ENGINE_ID" --arg directory "$directory" --arg dockerfile "$dockerfile" \
	--arg buildContext "$build_context" --arg repository "$repository" --arg platform "$PLATFORM" \
	--arg digest "$digest" --arg sourceRef "$image@$digest" --arg tagIdentity "$TAG_IDENTITY" \
	'{schemaVersion:"lunafox.engine-runtime-platform-build.v1",engineId:$engineId,directory:$directory,dockerfile:$dockerfile,buildContext:$buildContext,repository:$repository,platform:$platform,digest:$digest,sourceRef:$sourceRef,tagIdentity:$tagIdentity}' \
	>"$OUTPUT"
echo "wrote native platform build receipt: $OUTPUT"
