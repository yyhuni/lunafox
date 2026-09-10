#!/usr/bin/env bash
set -euo pipefail

# Image-first publisher for Engine Runtime Images. This script intentionally
# has no engine list or repository map: the Go discovery tool owns the source
# set and the contracts repository-name function owns the derived repository.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENGINE_ROOT="${ENGINE_ROOT:-$ROOT_DIR/extensions/engines}"
MODE="${ENGINE_RELEASE_MODE:-development}"
VERSION="${ENGINE_VERSION:-}"
OUTPUT="${ENGINE_IMAGE_BUILD_RESULTS:-$ROOT_DIR/dist/engine-runtime-images/build-results.json}"
EVIDENCE_ROOT="${ENGINE_RUNTIME_IMAGE_EVIDENCE_ROOT:-$(dirname "$OUTPUT")/registry-evidence}"
SELECTED_ENGINE_ID="${ENGINE_RELEASE_ENGINE_ID:-}"
REGISTRY_PUBLIC_HOST="${ENGINE_REGISTRY_PUBLIC_HOST:-localhost:${ENGINE_REGISTRY_PUBLIC_PORT:-5000}}"
REGISTRY_PUBLISHER_HOST="${ENGINE_REGISTRY_PUBLISHER_HOST:-$REGISTRY_PUBLIC_HOST}"
REGISTRY_TRANSPORT_HOST="${ENGINE_REGISTRY_TRANSPORT_HOST:-$REGISTRY_PUBLIC_HOST}"
REGISTRY_TRANSPORT_INSECURE="${ENGINE_REGISTRY_TRANSPORT_INSECURE:-false}"
REGISTRY_NAMESPACE="${ENGINE_REGISTRY_NAMESPACE:-lunafox}"
DOCKERHUB_NAMESPACE="${DOCKERHUB_NAMESPACE:-}"
GHCR_OWNER="${GITHUB_REPOSITORY_OWNER:-}"
IMAGE_SOURCE="${ENGINE_IMAGE_SOURCE:-https://github.com/${GITHUB_REPOSITORY:-yyhuni/lunafox}}"
CANONICAL_NAMESPACE="${LUNAFOX_CANONICAL_NAMESPACE:-yyhuni}"
GO_PROXY="${ENGINE_GO_PROXY:-${GOPROXY:-}}"
BUILD_PLATFORMS="${ENGINE_RUNTIME_IMAGE_PLATFORMS:-}"
TAG="${ENGINE_IMAGE_TAG:-}"
REUSE_EXISTING="${ENGINE_REUSE_EXISTING:-false}"
BUILDER_NAME="${ENGINE_BUILDER_NAME:-lunafox-engine-release}"
BUILDER_NETWORK="${ENGINE_BUILDER_NETWORK:-}"
REGISTRY_INSPECT_ATTEMPTS="${ENGINE_REGISTRY_INSPECT_ATTEMPTS:-6}"
REGISTRY_INSPECT_INITIAL_DELAY_SECONDS="${ENGINE_REGISTRY_INSPECT_INITIAL_DELAY_SECONDS:-2}"
REGISTRY_INSPECT_MAX_DELAY_SECONDS="${ENGINE_REGISTRY_INSPECT_MAX_DELAY_SECONDS:-8}"
PROXY_VARIABLES=(HTTP_PROXY HTTPS_PROXY ALL_PROXY NO_PROXY http_proxy https_proxy all_proxy no_proxy)

usage() {
	cat <<'USAGE'
Usage: scripts/ci/publish-engine-runtime-images.sh

Environment:
  ENGINE_ROOT                 Builtin engine source root (default extensions/engines)
  ENGINE_RELEASE_MODE         development or production (default development)
  ENGINE_VERSION              Required image/package version
  ENGINE_IMAGE_BUILD_RESULTS  Output build-result JSON path
  ENGINE_RUNTIME_IMAGE_EVIDENCE_ROOT
                              Production raw descriptor evidence directory
  ENGINE_RELEASE_ENGINE_ID    Optional canonical Engine ID; emits one verified
                              Runtime Image receipt and evidence shard
  ENGINE_REGISTRY_PUBLIC_HOST Host-reachable dev Registry (default localhost:5000)
  ENGINE_REGISTRY_PUBLISHER_HOST
                              Publisher-process Registry endpoint; defaults to public host
  ENGINE_REGISTRY_TRANSPORT_HOST
                              BuildKit/ORAS Registry endpoint; defaults to the public host
  ENGINE_REGISTRY_TRANSPORT_INSECURE
                              Set true only for an explicitly HTTP BuildKit Registry
  ENGINE_BUILDER_CONFIG       buildkitd.toml applied when creating a fresh builder
  ENGINE_BUILDER_NAME         Buildx builder name (default lunafox-engine-release)
  ENGINE_BUILDER_NETWORK      Optional existing Docker network for a fresh BuildKit builder
  ENGINE_REGISTRY_NAMESPACE   Dev Registry namespace (default lunafox)
  DOCKERHUB_NAMESPACE         Production namespace override (must be yyhuni)
  GITHUB_REPOSITORY_OWNER     Production GHCR owner hint (must be yyhuni)
  LUNAFOX_CANONICAL_NAMESPACE Canonical production namespace (fixed yyhuni)
  ENGINE_IMAGE_SOURCE         OCI source label (publisher configuration)
  ENGINE_GO_PROXY              Optional Go module proxy build override (defaults to GOPROXY)
  ENGINE_RUNTIME_IMAGE_PLATFORMS
                              Comma-separated Linux platforms; development defaults to the Docker daemon platform
  ENGINE_IMAGE_TAG            Optional immutable staging tag
  ENGINE_REUSE_EXISTING       Reuse an existing protected production tag on retry
  ENGINE_REGISTRY_INSPECT_ATTEMPTS
                              Maximum attempts for post-push Registry reads (default 6)
  ENGINE_REGISTRY_INSPECT_INITIAL_DELAY_SECONDS
                              Initial delay between post-push Registry reads (default 2)
  ENGINE_REGISTRY_INSPECT_MAX_DELAY_SECONDS
                              Maximum delay between post-push Registry reads (default 8)
USAGE
}

fail() {
	echo "publish engine runtime images: $*" >&2
	exit 1
}

normalize_arch() {
	case "$1" in
	x86_64) echo amd64 ;;
	aarch64) echo arm64 ;;
	*) echo "$1" ;;
	esac
}

validate_platforms() {
	local raw="$1" platform
	[ -n "$raw" ] || fail "Engine Runtime Image platforms are required"
	IFS=',' read -r -a platform_list <<<"$raw"
	[ "${#platform_list[@]}" -gt 0 ] || fail "Engine Runtime Image platforms are required"
	local seen=","
	for platform in "${platform_list[@]}"; do
		case "$platform" in
		linux/amd64 | linux/arm64) ;;
		*) fail "ENGINE_RUNTIME_IMAGE_PLATFORMS must contain only linux/amd64 and linux/arm64" ;;
		esac
		case "$seen" in
		*",$platform,"*) fail "ENGINE_RUNTIME_IMAGE_PLATFORMS must not repeat $platform" ;;
		esac
		seen="${seen}${platform},"
	done
}

resolve_build_platforms() {
	if [ -n "$BUILD_PLATFORMS" ]; then
		validate_platforms "$BUILD_PLATFORMS"
		return
	fi
	if [ "$MODE" = production ]; then
		BUILD_PLATFORMS="linux/amd64,linux/arm64"
		return
	fi
	local daemon_os daemon_arch
	daemon_os="$(docker info --format '{{.OSType}}')"
	daemon_arch="$(normalize_arch "$(docker info --format '{{.Architecture}}')")"
	[ "$daemon_os" = linux ] || fail "development Docker daemon must run Linux, got $daemon_os"
	BUILD_PLATFORMS="${daemon_os}/${daemon_arch}"
	validate_platforms "$BUILD_PLATFORMS"
}

append_proxy_driver_options() {
	local proxy_name proxy_value driver_option
	for proxy_name in "${PROXY_VARIABLES[@]}"; do
		proxy_value="${!proxy_name:-}"
		[ -n "$proxy_value" ] || continue
		driver_option="env.${proxy_name}=${proxy_value//\"/\"\"}"
		# Buildx parses each --driver-opt value as CSV. Quoting the whole field
		# keeps comma-separated NO_PROXY entries inside one env option.
		create_args+=(--driver-opt "\"${driver_option}\"")
	done
}

append_proxy_build_args() {
	local proxy_name proxy_value
	for proxy_name in "${PROXY_VARIABLES[@]}"; do
		proxy_value="${!proxy_name:-}"
		[ -n "$proxy_value" ] || continue
		build_args+=(--build-arg "${proxy_name}=${proxy_value}")
	done
}

append_proxy_build_hosts() {
	local proxy_name proxy_value proxy_host proxy_address
	local added_hosts=","
	for proxy_name in "${PROXY_VARIABLES[@]}"; do
		proxy_value="${!proxy_name:-}"
		[ -n "$proxy_value" ] || continue
		proxy_host="$(printf '%s' "$proxy_value" | sed -E 's#^[[:alpha:]][[:alnum:]+.-]*://([^/@]*@)?(\[[^]]+\]|[^:/]+).*#\2#')"
		case "$proxy_host" in
		host.docker.internal | host.containers.internal)
			case "$added_hosts" in
			*",$proxy_host,"*) continue ;;
			esac
			proxy_address="$(resolve_builder_host_address "$proxy_host")"
			# BuildKit's sandbox does not inherit the builder container's host aliases.
			# Map only a proxy host explicitly requested by the caller, using the
			# concrete address visible to this docker-container builder.
			build_args+=(--add-host "${proxy_host}:${proxy_address}")
			added_hosts="${added_hosts}${proxy_host},"
			;;
		esac
	done
}

resolve_builder_host_address() {
	local proxy_host="$1" builder_container proxy_address
	builder_container="$(docker ps --format '{{.Names}}' | awk -v builder="$BUILDER_NAME" '$0 ~ "^buildx_buildkit_" builder "[0-9]+$" { print; exit }')"
	[ -n "$builder_container" ] || fail "cannot locate running BuildKit container for $BUILDER_NAME"
	proxy_address="$(docker exec "$builder_container" getent ahostsv4 "$proxy_host" | awk 'NR == 1 { print $1 }')"
	[ -n "$proxy_address" ] || fail "BuildKit cannot resolve proxy host $proxy_host; use an address reachable from the BuildKit container"
	echo "$proxy_address"
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

verify_engine_payload() {
	local directory="$1" ref="$2"
	node "$ROOT_DIR/scripts/ci/check-engine-image-tool-inventory.mjs" \
		--repo-root "$ROOT_DIR" \
		--skip-product-image-source-check \
		--engine-image "$directory=$ref" >/dev/null
}

retry_registry_inspect() {
	local ref="$1"
	shift
	local attempt=1 delay="$REGISTRY_INSPECT_INITIAL_DELAY_SECONDS"
	local output_file error_file
	output_file="$(mktemp "$tmp_dir/registry-inspect-output.XXXXXX")"
	error_file="$(mktemp "$tmp_dir/registry-inspect-error.XXXXXX")"

	# A Registry may acknowledge a manifest write before its read replicas expose
	# the tag. Retry only the read and preserve the final error so a real auth or
	# descriptor failure still stops the release rather than being downgraded.
	while :; do
		: >"$output_file"
		: >"$error_file"
		if "$@" "$ref" >"$output_file" 2>"$error_file"; then
			cat "$output_file"
			rm -f "$output_file" "$error_file"
			return 0
		fi
		if [ "$attempt" -ge "$REGISTRY_INSPECT_ATTEMPTS" ]; then
			cat "$error_file" >&2
			rm -f "$output_file" "$error_file"
			return 1
		fi
		printf 'Registry inspect failed for %s (attempt %s/%s); retrying in %ss\n' \
			"$ref" "$attempt" "$REGISTRY_INSPECT_ATTEMPTS" "$delay" >&2
		sleep "$delay"
		if [ "$delay" -lt "$REGISTRY_INSPECT_MAX_DELAY_SECONDS" ]; then
			delay=$((delay * 2))
			[ "$delay" -le "$REGISTRY_INSPECT_MAX_DELAY_SECONDS" ] || delay="$REGISTRY_INSPECT_MAX_DELAY_SECONDS"
		fi
		attempt=$((attempt + 1))
	done
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

[ -n "$VERSION" ] || fail "ENGINE_VERSION is required"
[ "$MODE" = development ] || [ "$MODE" = production ] || fail "ENGINE_RELEASE_MODE must be development or production"
[ "$REUSE_EXISTING" = true ] || [ "$REUSE_EXISTING" = false ] || fail "ENGINE_REUSE_EXISTING must be true or false"
[ "$CANONICAL_NAMESPACE" = yyhuni ] || fail "LUNAFOX_CANONICAL_NAMESPACE is fixed to yyhuni"
[ -n "$REGISTRY_NAMESPACE" ] || fail "ENGINE_REGISTRY_NAMESPACE is required"
[ -n "$REGISTRY_PUBLIC_HOST" ] || fail "ENGINE_REGISTRY_PUBLIC_HOST is required"
[ -n "$REGISTRY_PUBLISHER_HOST" ] || fail "ENGINE_REGISTRY_PUBLISHER_HOST is required"
[ -n "$REGISTRY_TRANSPORT_HOST" ] || fail "ENGINE_REGISTRY_TRANSPORT_HOST is required"
[ "$REGISTRY_TRANSPORT_INSECURE" = true ] || [ "$REGISTRY_TRANSPORT_INSECURE" = false ] ||
	fail "ENGINE_REGISTRY_TRANSPORT_INSECURE must be true or false"
[[ "$REGISTRY_INSPECT_ATTEMPTS" =~ ^[1-9][0-9]*$ ]] ||
	fail "ENGINE_REGISTRY_INSPECT_ATTEMPTS must be a positive integer"
[[ "$REGISTRY_INSPECT_INITIAL_DELAY_SECONDS" =~ ^[0-9]+$ ]] ||
	fail "ENGINE_REGISTRY_INSPECT_INITIAL_DELAY_SECONDS must be a non-negative integer"
[[ "$REGISTRY_INSPECT_MAX_DELAY_SECONDS" =~ ^[0-9]+$ ]] ||
	fail "ENGINE_REGISTRY_INSPECT_MAX_DELAY_SECONDS must be a non-negative integer"
[ "$REGISTRY_INSPECT_ATTEMPTS" -le 12 ] ||
	fail "ENGINE_REGISTRY_INSPECT_ATTEMPTS must not exceed 12"
[ "$REGISTRY_INSPECT_MAX_DELAY_SECONDS" -le 60 ] ||
	fail "ENGINE_REGISTRY_INSPECT_MAX_DELAY_SECONDS must not exceed 60 seconds"
[ "$REGISTRY_INSPECT_INITIAL_DELAY_SECONDS" -le "$REGISTRY_INSPECT_MAX_DELAY_SECONDS" ] ||
	fail "ENGINE_REGISTRY_INSPECT_INITIAL_DELAY_SECONDS must not exceed ENGINE_REGISTRY_INSPECT_MAX_DELAY_SECONDS"
case "$REGISTRY_PUBLIC_HOST:$REGISTRY_PUBLISHER_HOST:$REGISTRY_TRANSPORT_HOST" in
*[[:space:]/]*) fail "Registry hosts must be host[:port] values without whitespace, scheme, or path" ;;
esac
case "$BUILDER_NETWORK" in
*[[:space:]/]*) fail "ENGINE_BUILDER_NETWORK must be a Docker network name without whitespace or slash" ;;
esac
case "$GO_PROXY" in
*[[:space:]]*) fail "ENGINE_GO_PROXY must not contain whitespace" ;;
esac
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v jq >/dev/null 2>&1 || fail "jq is required"
command -v node >/dev/null 2>&1 || fail "node is required"
[ -n "$BUILDER_NAME" ] || fail "ENGINE_BUILDER_NAME is required"
resolve_build_platforms
BUILD_PLATFORMS_JSON="$(jq -cn --arg platforms "$BUILD_PLATFORMS" '$platforms | split(",")')"

if [ "$MODE" = production ]; then
	[ "$REGISTRY_TRANSPORT_INSECURE" = false ] || fail "ENGINE_REGISTRY_TRANSPORT_INSECURE is forbidden in production"
	if [ -n "$DOCKERHUB_NAMESPACE" ] && [ "$DOCKERHUB_NAMESPACE" != "$CANONICAL_NAMESPACE" ]; then
		fail "DOCKERHUB_NAMESPACE must be $CANONICAL_NAMESPACE in production"
	fi
	if [ -n "$GHCR_OWNER" ] && [ "$GHCR_OWNER" != "$CANONICAL_NAMESPACE" ]; then
		fail "GITHUB_REPOSITORY_OWNER must be $CANONICAL_NAMESPACE in production"
	fi
	DOCKERHUB_NAMESPACE="$CANONICAL_NAMESPACE"
	GHCR_OWNER="$CANONICAL_NAMESPACE"
	# Engine is public even when a private product release consumes the image.
	IMAGE_SOURCE="${ENGINE_IMAGE_SOURCE:-https://github.com/yyhuni/lunafox}"
	command -v oras >/dev/null 2>&1 || fail "oras is required in production to copy one image index"
	REGISTRY_NAMESPACE="$CANONICAL_NAMESPACE"
else
	case "$REGISTRY_PUBLIC_HOST" in
	localhost | localhost:*) ;;
	*) fail "development Runtime Image ref must use host-reachable localhost Registry, got $REGISTRY_PUBLIC_HOST" ;;
	esac
	if [ "$REGISTRY_TRANSPORT_HOST" = "$REGISTRY_PUBLIC_HOST" ] && [ "$REGISTRY_TRANSPORT_INSECURE" != true ]; then
		fail "development public Registry uses plain HTTP; set ENGINE_REGISTRY_TRANSPORT_INSECURE=true when BuildKit uses the same endpoint"
	fi
fi

case "$TAG" in
"") TAG="engine-${VERSION}-$(date -u +%Y%m%d%H%M%S)" ;;
*[[:space:]/]*) fail "ENGINE_IMAGE_TAG must not contain whitespace or slash" ;;
esac

ENGINE_ROOT="$(cd "$ENGINE_ROOT" && pwd)" || fail "ENGINE_ROOT does not exist: $ENGINE_ROOT"
mkdir -p "$(dirname "$OUTPUT")"
if [ "$MODE" = production ]; then
	if [ -e "$EVIDENCE_ROOT" ] && [ -n "$(find "$EVIDENCE_ROOT" -mindepth 1 -print -quit 2>/dev/null)" ]; then
		fail "production Runtime Image evidence root must be absent or empty: $EVIDENCE_ROOT"
	fi
	mkdir -p "$EVIDENCE_ROOT"
fi
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

if [ "$MODE" = development ]; then
	# This endpoint becomes package identity and must be real from the publisher
	# host before a costly image build begins. The post-build Docker pull below
	# remains the authoritative host-daemon reachability and platform check.
	if ! node "$ROOT_DIR/scripts/ci/verify-distribution-registry-v2.mjs" \
		--endpoint "http://$REGISTRY_PUBLISHER_HOST/v2/"; then
		fail "development publisher Registry endpoint is not a reachable Distribution Registry v2 endpoint"
	fi
fi

builder_config="${ENGINE_BUILDER_CONFIG:-}"
generated_development_builder_config=false
if [ "$MODE" = development ] && [ "$REGISTRY_TRANSPORT_INSECURE" = true ] && [ -z "$builder_config" ]; then
	builder_config="$tmp_dir/buildkitd.toml"
	generated_development_builder_config=true
	printf '[registry."%s"]\n  http = true\n  insecure = true\n' "$REGISTRY_TRANSPORT_HOST" >"$builder_config"
fi
if [ -n "$builder_config" ] && [ ! -f "$builder_config" ]; then
	fail "ENGINE_BUILDER_CONFIG does not point to a readable buildkitd.toml: $builder_config"
fi

if ! docker buildx inspect "$BUILDER_NAME" >/dev/null 2>&1; then
	create_args=(--name "$BUILDER_NAME" --driver docker-container)
	if [ -n "$BUILDER_NETWORK" ]; then
		create_args+=(--driver-opt "network=$BUILDER_NETWORK")
	fi
	append_proxy_driver_options
	if [ -n "$builder_config" ]; then
		create_args+=(--buildkitd-config "$builder_config")
	fi
	docker buildx create "${create_args[@]}" >/dev/null
elif [ -n "$builder_config" ] && [ "$generated_development_builder_config" != true ]; then
	fail "Builder $BUILDER_NAME already exists and cannot apply ENGINE_BUILDER_CONFIG; set ENGINE_BUILDER_NAME to a fresh builder"
fi
docker buildx inspect "$BUILDER_NAME" --bootstrap >/dev/null

discovery_bin="$tmp_dir/engine-release"
(cd "$ROOT_DIR/tools/engine-release" && go build -trimpath -buildvcs=false -o "$discovery_bin" .)
discovery="$tmp_dir/discovery.json"
discovery_args=(-command discover -engine-root "$ENGINE_ROOT" -output "$discovery")
if [ -n "$SELECTED_ENGINE_ID" ]; then
	# The release tool validates this against the full canonical discovery before
	# any image build starts; the shell never owns a handwritten Engine set.
	discovery_args+=(-engine-id "$SELECTED_ENGINE_ID")
fi
"$discovery_bin" "${discovery_args[@]}"

records_dir="$tmp_dir/records"
mkdir -p "$records_dir"
engine_count="$(jq '.engines | length' "$discovery")"
[ "$engine_count" -gt 0 ] || fail "discovery found no engines"
if [ -n "$SELECTED_ENGINE_ID" ]; then
	[ "$engine_count" -eq 1 ] || fail "selected Engine discovery must contain exactly one Engine"
fi

while IFS=$'\t' read -r engine_id directory dockerfile build_context repository; do
	[ -n "$engine_id" ] || fail "discovery returned an empty engineId"
	safe_engine_id="${engine_id//[^A-Za-z0-9_.-]/_}"
	[ -n "$directory" ] || fail "discovery returned an empty engine directory for $engine_id"
	[ -n "$build_context" ] || fail "discovery returned an empty buildContext for $engine_id"
	case "$build_context" in
	/* | *..* | *\\*) fail "discovery returned an unsafe buildContext for $engine_id: $build_context" ;;
	esac
	build_context_path="$ENGINE_ROOT/$build_context"
	[ -d "$build_context_path" ] || fail "discovered buildContext does not exist for $engine_id: $build_context"
	case "$repository" in
	*lunafox-engine-runtime-*) ;;
	*) fail "discovery returned an invalid runtime repository for $engine_id: $repository" ;;
	esac

	if [ "$MODE" = development ]; then
		# BuildKit may run in a separate VM/network from the host Docker daemon.
		# Keep the transport location private to the publisher; only the public
		# host is written into the receipt/package consumed by Agent.
		transport_image_location="$REGISTRY_TRANSPORT_HOST/$REGISTRY_NAMESPACE/$repository"
		publisher_image_location="$REGISTRY_PUBLISHER_HOST/$REGISTRY_NAMESPACE/$repository"
		public_image_location="$REGISTRY_PUBLIC_HOST/$REGISTRY_NAMESPACE/$repository"
	else
		transport_image_location="docker.io/$DOCKERHUB_NAMESPACE/$repository"
		publisher_image_location="$transport_image_location"
		public_image_location="$transport_image_location"
	fi
	# BuildKit must use the transport endpoint visible from its builder
	# container, while the local Docker CLI must use the publisher endpoint.
	# These endpoints can name the same Registry through different hostnames
	# (for example host.docker.internal versus localhost on Docker Desktop).
	# Keep the distinction through single-platform index creation so the CLI
	# never tries to resolve a builder-only hostname from the host.
	transport_staging_ref="$transport_image_location:$TAG"
	publisher_staging_ref="$publisher_image_location:$TAG"
	build_staging_ref="$transport_staging_ref"
	publisher_build_staging_ref="$publisher_staging_ref"
	if [[ "$BUILD_PLATFORMS" != *,* ]]; then
		build_staging_ref="${transport_staging_ref}-platform"
		publisher_build_staging_ref="${publisher_staging_ref}-platform"
	fi
	# Runtime Image publisher inspection must use the explicit publisher
	# Registry endpoint; the staging ref is equivalent but hides that boundary.
	inspect_ref="$publisher_image_location:$TAG"

	# One BuildKit invocation produces the configured platform manifests and one
	# OCI index. No package bytes or package digest is passed to the image build.
	build_args=(
		--platform "$BUILD_PLATFORMS"
		--output "type=image,push=true,oci-mediatypes=true"
		--build-context "contracts=$ROOT_DIR/contracts"
		--build-context "engine-go=$ROOT_DIR/engine-go"
		--build-arg "ENGINE_IMAGE_VERSION=$VERSION"
		--build-arg "ENGINE_IMAGE_SOURCE=$IMAGE_SOURCE"
		--tag "$build_staging_ref"
		--file "$ENGINE_ROOT/$dockerfile"
	)
	if [ "$MODE" = production ]; then
		# Public Engine releases carry their own provenance and SBOM attestations;
		# development's local Registry remains intentionally metadata-free.
		build_args+=(--provenance=mode=max --sbom=true)
	else
		build_args+=(--provenance=false --sbom=false)
	fi
	if [ "${ENGINE_RUNTIME_IMAGE_PULL:-true}" = false ]; then
		build_args+=(--pull=false)
	fi
	if [ -n "$GO_PROXY" ]; then
		build_args+=(--build-arg "GOPROXY=$GO_PROXY")
	fi
	append_proxy_build_args
	append_proxy_build_hosts
	existing_index_digest=""
	if [ "$MODE" = production ] && [ "$REUSE_EXISTING" = true ]; then
		existing_index_digest="$(docker buildx imagetools inspect --builder "$BUILDER_NAME" "$inspect_ref" 2>/dev/null | awk '/^Digest:/ && digest == "" { digest = $2 } END { print digest }' || true)"
	fi
	if [[ "$existing_index_digest" =~ ^sha256:[a-f0-9]{64}$ ]]; then
		# A retry of the same protected merge must reuse the existing immutable
		# digest. Registry replication and evidence are still re-verified below.
		index_digest="$existing_index_digest"
	else
		docker buildx build --builder "$BUILDER_NAME" "${build_args[@]}" "$build_context_path"
		if [ "$build_staging_ref" != "$transport_staging_ref" ]; then
			# Buildx can export a lone platform as a manifest. The package/bootstrap
			# contract requires an OCI index even for local single-platform images.
			# Resolve both source and destination through the publisher endpoint;
			# the Registry is shared with the BuildKit transport endpoint above.
			docker buildx imagetools create --builder "$BUILDER_NAME" --prefer-index --tag "$publisher_staging_ref" "$publisher_build_staging_ref" >/dev/null
		fi
		# imagetools resolves through the publisher host, not through BuildKit's
		# transport network. Inspect the same object through its public endpoint.
		# Consume the complete inspect stream: exiting awk early can make buildx
		# report SIGPIPE as exit 255 under pipefail after a successful push.
		index_digest="$(retry_registry_inspect "$inspect_ref" docker buildx imagetools inspect --builder "$BUILDER_NAME" | awk '/^Digest:/ && digest == "" { digest = $2 } END { print digest }')"
	fi
	[[ "$index_digest" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "invalid image index digest for $engine_id: $index_digest"
	raw_index="$tmp_dir/${safe_engine_id}.index.json"
	retry_registry_inspect "$inspect_ref" docker buildx imagetools inspect --builder "$BUILDER_NAME" --raw >"$raw_index"
	node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" \
		--raw-file "$raw_index" \
		--expected-digest "$index_digest" \
		--expected-platforms "$BUILD_PLATFORMS" >/dev/null

	if [ "$MODE" = development ]; then
		# The executable ref must be resolvable by the host daemon used by Agent;
		# Compose-only service DNS is intentionally never advertised.
		published_ref="$public_image_location@$index_digest"
		ENGINE_REGISTRY_PUBLIC_HOST="$REGISTRY_PUBLIC_HOST" \
			"$ROOT_DIR/scripts/dev/verify-engine-runtime-image-pull.sh" \
			--ref "$published_ref" \
			--engine-id "$engine_id"
		verify_engine_payload "$directory" "$published_ref"
		refs_json="$(jq -n --arg ref "$published_ref" '[ $ref ]')"
		source_ref="$published_ref"
		copied_refs_json='[]'
	else
		engine_evidence_root="$EVIDENCE_ROOT/$safe_engine_id"
		mkdir -p "$engine_evidence_root"
		cp "$raw_index" "$engine_evidence_root/dockerhub-index.json"
		docker_ref="$public_image_location@$index_digest"
		ghcr_location="ghcr.io/$GHCR_OWNER/$repository"
		ghcr_tag_ref="$ghcr_location:$TAG"
		# oras copies the already-built index and blobs. It never rebuilds the
		# engine image for GHCR, so both candidates retain one digest identity.
		oras cp "$docker_ref" "$ghcr_tag_ref"
		ghcr_digest="$(retry_registry_inspect "$ghcr_tag_ref" docker buildx imagetools inspect --builder "$BUILDER_NAME" | awk '/^Digest:/ && digest == "" { digest = $2 } END { print digest }')"
		[ "$ghcr_digest" = "$index_digest" ] || fail "cross-Registry Runtime Image digest drift for $engine_id: Docker Hub=$index_digest GHCR=$ghcr_digest"
		ghcr_raw="$tmp_dir/${safe_engine_id}.ghcr.index.json"
		retry_registry_inspect "$ghcr_tag_ref" docker buildx imagetools inspect --builder "$BUILDER_NAME" --raw >"$ghcr_raw"
		node "$ROOT_DIR/scripts/ci/verify-runtime-image-index.mjs" \
			--raw-file "$ghcr_raw" \
			--expected-digest "$index_digest" \
			--expected-platforms "$BUILD_PLATFORMS" >/dev/null
		ghcr_ref="$ghcr_location@$ghcr_digest"
		cp "$ghcr_raw" "$engine_evidence_root/ghcr-index.json"
		jq -n \
			--arg engineId "$engine_id" \
			--arg indexDigest "$index_digest" \
			--arg dockerHubRef "$docker_ref" \
			--arg ghcrRef "$ghcr_ref" \
			'{
				schemaVersion:"lunafox.engine-runtime-image-registry-evidence.v1",
				engineId:$engineId,
				indexDigest:$indexDigest,
				platforms:["linux/amd64","linux/arm64"],
				registries:{
					dockerHub:{ref:$dockerHubRef,rawDescriptor:"dockerhub-index.json"},
					ghcr:{ref:$ghcrRef,rawDescriptor:"ghcr-index.json"}
				},
				copiedWithoutRebuild:true,
				digestEqualityVerified:true,
				platformSetVerified:true
			}' >"$engine_evidence_root/verification.json"
		verify_host_pull "$docker_ref" "$engine_id" "Docker Hub Runtime Image" "$ghcr_ref"
		verify_engine_payload "$directory" "$docker_ref"
		verify_host_pull "$ghcr_ref" "$engine_id" "GHCR Runtime Image" "$docker_ref"
		verify_engine_payload "$directory" "$ghcr_ref"
		refs_json="$(jq -n --arg docker "$docker_ref" --arg ghcr "$ghcr_ref" '[ $docker, $ghcr ]')"
		source_ref="$docker_ref"
		copied_refs_json="$(jq -n --arg ghcr "$ghcr_ref" '[ $ghcr ]')"
	fi

	jq -n \
		--arg engineId "$engine_id" \
		--arg dockerfile "$dockerfile" \
		--arg buildContext "$build_context" \
		--arg repository "$repository" \
		--arg indexDigest "$index_digest" \
		--arg indexMediaType 'application/vnd.oci.image.index.v1+json' \
		--arg sourceRef "$source_ref" \
		--argjson platforms "$BUILD_PLATFORMS_JSON" \
		--argjson refs "$refs_json" \
		--argjson copiedRefs "$copied_refs_json" \
		'{engineId:$engineId,dockerfile:$dockerfile,buildContext:$buildContext,repository:$repository,buildCount:1,indexDigest:$indexDigest,indexMediaType:$indexMediaType,platforms:$platforms,refs:$refs,sourceRef:$sourceRef,copiedRefs:$copiedRefs}' \
		>"$records_dir/${safe_engine_id}.json"
done < <(jq -r '.engines[] | [.engineId,.directory,.dockerfile,.buildContext,.repository] | @tsv' "$discovery")

jq -s --arg schema 'lunafox.engine-runtime-image-build-results.v1' --arg mode "$MODE" \
	'{schemaVersion:$schema,mode:$mode,engines:(sort_by(.engineId))}' \
	"$records_dir"/*.json >"$OUTPUT"

validation_args=(-command validate-build-results -engine-root "$ENGINE_ROOT" -build-results "$OUTPUT" -mode "$MODE")
if [ -n "$SELECTED_ENGINE_ID" ]; then
	validation_args+=(-engine-id "$SELECTED_ENGINE_ID")
fi
"$discovery_bin" "${validation_args[@]}" >/dev/null
echo "wrote verified Engine Runtime Image build results: $OUTPUT"
