#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf -- "$TEST_ROOT"' EXIT

fail() {
	echo "prepare deployment self-test failed: $*" >&2
	exit 1
}

sha256_file() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

write_metadata() {
	local directory="$1" dockerhub_digest="$2" ghcr_digest="$3"
	cat >"$directory/deployment-packages.json" <<EOF
[
  {
    "name": "lunafox-v1.2.3-dockerhub.zip",
    "sha256": "$dockerhub_digest"
  },
  {
    "name": "lunafox-v1.2.3-ghcr.zip",
    "sha256": "$ghcr_digest"
  }
]
EOF
}

make_package() {
	local directory="$1" registry="$2"
	local package_root="$directory/package-$registry"
	mkdir -p "$package_root/resources"
	cat >"$package_root/compose.yaml" <<EOF
services:
  server:
    image: ${registry}.example/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
EOF
	cat >"$package_root/.env" <<'EOF'
PUBLIC_HOST=localhost
PUBLIC_PORT=443
EOF
	cp "$package_root/.env" "$package_root/.env.example"
	printf 'enginePackages: []\n' >"$package_root/engine-inventory.yaml"
	printf 'releaseVersion: "1.2.3"\n' >"$package_root/release.manifest.yaml"
	printf '# Deployment package\n' >"$package_root/README.md"
	python3 - "$package_root" "$directory/lunafox-v1.2.3-$registry.zip" <<'PY'
import pathlib
import sys
import zipfile

root = pathlib.Path(sys.argv[1])
with zipfile.ZipFile(sys.argv[2], "w") as archive:
    for path in sorted(root.rglob("*")):
        if path.is_file():
            archive.write(path, path.relative_to(root))
PY
}

make_checkout() {
	local directory="$1" tagged="$2"
	mkdir -p "$directory"
	cp "$ROOT_DIR/prepare-deployment.sh" "$directory/prepare-deployment.sh"
	chmod +x "$directory/prepare-deployment.sh"
	git -C "$directory" init -q
	git -C "$directory" config user.email test@example.invalid
	git -C "$directory" config user.name Test
	git -C "$directory" add prepare-deployment.sh
	git -C "$directory" commit -qm fixture
	if [ "$tagged" = "tagged" ]; then
		git -C "$directory" tag v1.2.3
	fi
}

run_prepare() {
	local checkout="$1"
	shift
	(
		cd "$checkout"
		PATH="$FAKE_BIN:$PATH" \
			FIXTURE_ASSET_DIR="$ASSET_DIR" \
			FIXTURE_CURL_LOG="$CURL_LOG" \
			./prepare-deployment.sh "$@"
	)
}

assert_failure() {
	local expected="$1"
	shift
	local output
	if output="$("$@" 2>&1)"; then
		fail "command unexpectedly succeeded: $*"
	fi
	grep -Fq "$expected" <<<"$output" || fail "missing failure text '$expected': $output"
}

ASSET_DIR="$TEST_ROOT/assets"
FAKE_BIN="$TEST_ROOT/bin"
CURL_LOG="$TEST_ROOT/curl.log"
mkdir -p "$ASSET_DIR" "$FAKE_BIN"
make_package "$ASSET_DIR" dockerhub
make_package "$ASSET_DIR" ghcr
DOCKERHUB_SHA="$(sha256_file "$ASSET_DIR/lunafox-v1.2.3-dockerhub.zip")"
GHCR_SHA="$(sha256_file "$ASSET_DIR/lunafox-v1.2.3-ghcr.zip")"
write_metadata "$ASSET_DIR" "$DOCKERHUB_SHA" "$GHCR_SHA"

cat >"$FAKE_BIN/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
destination=""
url=""
while [ "$#" -gt 0 ]; do
	case "$1" in
	--output)
		destination="$2"
		shift 2
		;;
	--proto | --max-redirs)
		shift 2
		;;
	--fail | --silent | --show-error | --location | --tlsv1.2)
		shift
		;;
	*)
		url="$1"
		shift
		;;
	esac
done
[ -n "$destination" ] && [ -n "$url" ]
printf '%s\n' "$url" >>"$FIXTURE_CURL_LOG"
name="${url##*/}"
if [ "${FIXTURE_FAIL_ASSET:-}" = "$name" ]; then
	exit 22
fi
cp "$FIXTURE_ASSET_DIR/$name" "$destination"
EOF
chmod +x "$FAKE_BIN/curl"

TAGGED_CHECKOUT="$TEST_ROOT/tagged"
make_checkout "$TAGGED_CHECKOUT" tagged
: >"$CURL_LOG"
run_prepare "$TAGGED_CHECKOUT" >/dev/null
grep -Fqx 'RELEASE_TAG=v1.2.3' "$TAGGED_CHECKOUT/.lunafox-deployment/.lunafox-deployment.env" || fail "exact tag was not recorded"
grep -Fqx 'REGISTRY=dockerhub' "$TAGGED_CHECKOUT/.lunafox-deployment/.lunafox-deployment.env" || fail "default registry was not recorded"
grep -Fq 'dockerhub.zip' "$CURL_LOG" || fail "Docker Hub package was not requested"

printf 'PUBLIC_HOST=changed.example\nPUBLIC_PORT=8443\n' >"$TAGGED_CHECKOUT/.lunafox-deployment/.env"
BEFORE_ENV="$(sha256_file "$TAGGED_CHECKOUT/.lunafox-deployment/.env")"
run_prepare "$TAGGED_CHECKOUT" >/dev/null
[ "$(sha256_file "$TAGGED_CHECKOUT/.lunafox-deployment/.env")" = "$BEFORE_ENV" ] || fail "matching reuse overwrote .env"

EXPLICIT_CHECKOUT="$TEST_ROOT/explicit"
make_checkout "$EXPLICIT_CHECKOUT" untagged
: >"$CURL_LOG"
run_prepare "$EXPLICIT_CHECKOUT" --version v1.2.3 --registry ghcr >/dev/null
grep -Fqx 'REGISTRY=ghcr' "$EXPLICIT_CHECKOUT/.lunafox-deployment/.lunafox-deployment.env" || fail "explicit GHCR selection was not recorded"
grep -Fq 'ghcr.zip' "$CURL_LOG" || fail "GHCR package was not requested"
if grep -Fq 'dockerhub.zip' "$CURL_LOG"; then
	fail "explicit GHCR selection requested Docker Hub"
fi

UNTAGGED_CHECKOUT="$TEST_ROOT/untagged"
make_checkout "$UNTAGGED_CHECKOUT" untagged
: >"$CURL_LOG"
assert_failure 'pass --version' run_prepare "$UNTAGGED_CHECKOUT"
[ ! -s "$CURL_LOG" ] || fail "untagged checkout contacted the release service"

BAD_METADATA="$TEST_ROOT/bad-metadata"
make_checkout "$BAD_METADATA" untagged
cp "$ASSET_DIR/deployment-packages.json" "$TEST_ROOT/metadata.backup"
printf '[]\n' >"$ASSET_DIR/deployment-packages.json"
assert_failure 'exactly one asset' run_prepare "$BAD_METADATA" --version v1.2.3
[ ! -e "$BAD_METADATA/.lunafox-deployment" ] || fail "missing metadata asset created output"

write_metadata "$ASSET_DIR" "$DOCKERHUB_SHA" "$GHCR_SHA"
perl -0pi -e 's/\n  \}\n\]/\n  },\n  {\n    "name": "lunafox-v1.2.3-dockerhub.zip",\n    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"\n  }\n]/' "$ASSET_DIR/deployment-packages.json"
assert_failure 'exactly one asset' run_prepare "$BAD_METADATA" --version v1.2.3

write_metadata "$ASSET_DIR" "$(printf '0%.0s' {1..64})" "$GHCR_SHA"
assert_failure 'SHA-256 mismatch' run_prepare "$BAD_METADATA" --version v1.2.3
[ ! -e "$BAD_METADATA/.lunafox-deployment" ] || fail "digest mismatch created output"

write_metadata "$ASSET_DIR" "$DOCKERHUB_SHA" "$GHCR_SHA"
: >"$CURL_LOG"
export FIXTURE_FAIL_ASSET=lunafox-v1.2.3-dockerhub.zip
assert_failure 'deployment preparation failed' run_prepare "$BAD_METADATA" --version v1.2.3
unset FIXTURE_FAIL_ASSET
if grep -Fq 'ghcr.zip' "$CURL_LOG"; then
	fail "Docker Hub failure automatically fell back to GHCR"
fi

CONFLICT_CHECKOUT="$TEST_ROOT/conflict"
make_checkout "$CONFLICT_CHECKOUT" untagged
mkdir "$CONFLICT_CHECKOUT/.lunafox-deployment"
printf 'keep\n' >"$CONFLICT_CHECKOUT/.lunafox-deployment/user-file"
assert_failure 'refusing to overwrite' run_prepare "$CONFLICT_CHECKOUT" --version v1.2.3
grep -Fqx 'keep' "$CONFLICT_CHECKOUT/.lunafox-deployment/user-file" || fail "conflicting output was modified"

echo "prepare deployment self-test passed"
