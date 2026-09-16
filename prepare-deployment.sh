#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_BASE_URL="https://github.com/yyhuni/lunafox/releases/download"
VERSION=""
REGISTRY="dockerhub"
OUTPUT_DIR="$ROOT_DIR/.lunafox-deployment"
TEMP_DIR=""
LOCK_DIR=""

usage() {
	cat <<'USAGE'
Usage: ./prepare-deployment.sh [options]

Prepare a verified LunaFox release package from a public source checkout.

Options:
  --version <vX.Y.Z[-alpha.N|-beta.N|-rc.N]>
                         Release version. Defaults to the unique release tag at HEAD.
  --registry <dockerhub|ghcr>
                         Complete image registry closure (default: dockerhub).
  --output <directory>   Deployment directory (default: ./.lunafox-deployment).
  -h, --help             Show this help.
USAGE
}

fail() {
	echo "LunaFox deployment preparation failed: $*" >&2
	exit 1
}

cleanup() {
	if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
		rm -rf -- "$TEMP_DIR"
	fi
	if [ -n "$LOCK_DIR" ] && [ -d "$LOCK_DIR" ]; then
		rmdir "$LOCK_DIR" 2>/dev/null || true
	fi
}
trap cleanup EXIT

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "required command is unavailable: $1"
}

resolve_version() {
	if [ -n "$VERSION" ]; then
		return
	fi
	require_command git
	git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1 ||
		fail "cannot infer a release version outside a Git checkout; pass --version"

	local tag release_tags=() candidate
	while IFS= read -r tag; do
		if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)\.[0-9]+)?$ ]]; then
			release_tags+=("$tag")
		fi
	done < <(git -C "$ROOT_DIR" tag --points-at HEAD)

	if [ "${#release_tags[@]}" -ne 1 ]; then
		candidate="${release_tags[*]:-none}"
		fail "HEAD must have exactly one LunaFox release tag (found: $candidate); pass --version"
	fi
	VERSION="${release_tags[0]}"
}

sha256_file() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{print $1}'
	else
		fail "required SHA-256 tool is unavailable: install sha256sum or shasum"
	fi
}

download() {
	local url="$1" destination="$2"
	curl --fail --silent --show-error --location \
		--proto '=https' --tlsv1.2 --max-redirs 3 \
		--output "$destination" "$url" || fail "download failed: $url"
}

metadata_sha_for_asset() {
	local metadata="$1" asset="$2" count digest
	count="$(grep -F -c "\"name\": \"$asset\"" "$metadata" || true)"
	[ "$count" = "1" ] || fail "package metadata must contain exactly one asset named $asset"
	digest="$(awk -v needle="\"name\": \"$asset\"" '
		index($0, needle) {
			if (getline <= 0) exit 1
			line = $0
			sub(/^.*"sha256"[[:space:]]*:[[:space:]]*"/, "", line)
			sub(/".*$/, "", line)
			print line
			exit
		}
	' "$metadata")"
	[[ "$digest" =~ ^[a-f0-9]{64}$ ]] || fail "package metadata contains an invalid SHA-256 for $asset"
	printf '%s\n' "$digest"
}

validate_existing_output() {
	local marker="$OUTPUT_DIR/.lunafox-deployment.env"
	[ -d "$OUTPUT_DIR" ] || fail "output path exists and is not a directory: $OUTPUT_DIR"
	for required in "$marker" "$OUTPUT_DIR/compose.yaml" "$OUTPUT_DIR/.env" \
		"$OUTPUT_DIR/.env.example" "$OUTPUT_DIR/engine-inventory.yaml" \
		"$OUTPUT_DIR/release.manifest.yaml" "$OUTPUT_DIR/README.md"; do
		[ -f "$required" ] && [ ! -L "$required" ] ||
			fail "existing output is incomplete; refusing to overwrite: $required"
	done
	[ "$(wc -l <"$marker" | tr -d ' ')" = "5" ] ||
		fail "existing output identity is malformed; refusing to overwrite: $marker"
	grep -Fqx 'SCHEMA_VERSION=1' "$marker" || fail "existing output identity schema is unsupported"
	grep -Fqx "RELEASE_TAG=$VERSION" "$marker" || fail "existing output uses a different release version"
	grep -Fqx "REGISTRY=$REGISTRY" "$marker" || fail "existing output uses a different registry"
	grep -Fqx "PACKAGE_NAME=$PACKAGE_NAME" "$marker" || fail "existing output uses a different package"
	grep -Fqx "PACKAGE_SHA256=$EXPECTED_SHA256" "$marker" || fail "existing output uses a different package digest"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--version)
		[ "$#" -ge 2 ] || fail "--version requires a value"
		VERSION="$2"
		shift 2
		;;
	--registry)
		[ "$#" -ge 2 ] || fail "--registry requires a value"
		REGISTRY="$2"
		shift 2
		;;
	--output)
		[ "$#" -ge 2 ] || fail "--output requires a value"
		OUTPUT_DIR="$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*) fail "unknown argument: $1" ;;
	esac
done

resolve_version
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)\.[0-9]+)?$ ]] ||
	fail "invalid release version: $VERSION"
case "$REGISTRY" in
dockerhub | ghcr) ;;
*) fail "unsupported registry: $REGISTRY (expected dockerhub or ghcr)" ;;
esac

require_command curl
require_command unzip
require_command awk

case "$OUTPUT_DIR" in
/*) ;;
*) OUTPUT_DIR="$PWD/$OUTPUT_DIR" ;;
esac
OUTPUT_PARENT="$(dirname "$OUTPUT_DIR")"
OUTPUT_NAME="$(basename "$OUTPUT_DIR")"
[ "$OUTPUT_NAME" != "." ] && [ "$OUTPUT_NAME" != ".." ] || fail "invalid output directory: $OUTPUT_DIR"
mkdir -p "$OUTPUT_PARENT"
OUTPUT_PARENT="$(cd "$OUTPUT_PARENT" && pwd -P)"
OUTPUT_DIR="$OUTPUT_PARENT/$OUTPUT_NAME"
LOCK_DIR="$OUTPUT_DIR.prepare-lock"
mkdir "$LOCK_DIR" 2>/dev/null || fail "another preparation is running or a stale lock exists: $LOCK_DIR"
TEMP_DIR="$(mktemp -d "$OUTPUT_PARENT/.lunafox-prepare.XXXXXX")"

PACKAGE_NAME="lunafox-${VERSION}-${REGISTRY}.zip"
RELEASE_URL="$RELEASE_BASE_URL/$VERSION"
METADATA_FILE="$TEMP_DIR/deployment-packages.json"
ARCHIVE_FILE="$TEMP_DIR/$PACKAGE_NAME"

echo "Downloading release package metadata for $VERSION..."
download "$RELEASE_URL/deployment-packages.json" "$METADATA_FILE"
EXPECTED_SHA256="$(metadata_sha_for_asset "$METADATA_FILE" "$PACKAGE_NAME")"

if [ -e "$OUTPUT_DIR" ]; then
	validate_existing_output
	echo "Deployment directory already matches $VERSION ($REGISTRY): $OUTPUT_DIR"
	printf 'Run:\n  cd %q\n  docker compose up -d\n' "$OUTPUT_DIR"
	exit 0
fi

echo "Downloading $PACKAGE_NAME..."
download "$RELEASE_URL/$PACKAGE_NAME" "$ARCHIVE_FILE"
ACTUAL_SHA256="$(sha256_file "$ARCHIVE_FILE")"
[ "$ACTUAL_SHA256" = "$EXPECTED_SHA256" ] ||
	fail "package SHA-256 mismatch for $PACKAGE_NAME"

PAYLOAD_DIR="$TEMP_DIR/payload"
mkdir "$PAYLOAD_DIR"
ARCHIVE_ENTRIES="$(unzip -Z1 "$ARCHIVE_FILE")" || fail "cannot inspect deployment package: $PACKAGE_NAME"
while IFS= read -r entry; do
	case "$entry" in
	"" | /* | . | .. | ../* | */.. | */../* | *//* | *\\*) fail "package contains an unsafe path: $entry" ;;
	esac
done <<<"$ARCHIVE_ENTRIES"
unzip -q "$ARCHIVE_FILE" -d "$PAYLOAD_DIR"
if find "$PAYLOAD_DIR" -type l -print -quit | grep -q .; then
	fail "deployment package contains a symbolic link"
fi

for required in compose.yaml .env .env.example engine-inventory.yaml release.manifest.yaml README.md; do
	[ -f "$PAYLOAD_DIR/$required" ] && [ ! -L "$PAYLOAD_DIR/$required" ] ||
		fail "deployment package is missing required file: $required"
done

cat >"$PAYLOAD_DIR/.lunafox-deployment.env" <<EOF
SCHEMA_VERSION=1
RELEASE_TAG=$VERSION
REGISTRY=$REGISTRY
PACKAGE_NAME=$PACKAGE_NAME
PACKAGE_SHA256=$EXPECTED_SHA256
EOF

[ ! -e "$OUTPUT_DIR" ] || fail "output appeared during preparation; refusing to overwrite: $OUTPUT_DIR"
mv "$PAYLOAD_DIR" "$OUTPUT_DIR"

echo "Prepared LunaFox $VERSION ($REGISTRY) in $OUTPUT_DIR"
printf 'Next:\n  cd %q\n  review .env\n  docker compose up -d\n  docker compose ps\n' "$OUTPUT_DIR"
