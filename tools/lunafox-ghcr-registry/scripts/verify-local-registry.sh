#!/usr/bin/env bash

set -euo pipefail

required_vars=(REGISTRY_HOST IMAGE_REPOSITORY IMAGE_DIGEST BLOB_DIGEST)
for variable in "${required_vars[@]}"; do
  if [[ -z "${!variable:-}" ]]; then
    echo "${variable} is required" >&2
    exit 2
  fi
done

if [[ ! "${IMAGE_REPOSITORY}" =~ ^yyhuni/lunafox-[a-z0-9][a-z0-9._-]*$ ]]; then
  echo "IMAGE_REPOSITORY must be a public yyhuni/lunafox-* repository" >&2
  exit 2
fi
if [[ ! "${IMAGE_DIGEST}" =~ ^sha256:[a-f0-9]{64}$ ]] || [[ ! "${BLOB_DIGEST}" =~ ^sha256:[a-f0-9]{64}$ ]]; then
  echo "IMAGE_DIGEST and BLOB_DIGEST must be sha256 digests" >&2
  exit 2
fi

base_url="https://${REGISTRY_HOST}"
reference="${REGISTRY_HOST}/${IMAGE_REPOSITORY}@${IMAGE_DIGEST}"
output_dir="$(mktemp -d)"
trap 'rm -rf "${output_dir}"' EXIT

expect_status() {
  local expected="$1"
  shift
  local actual
  actual="$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' "$@")"
  if [[ "${actual}" != "${expected}" ]]; then
    echo "expected HTTP ${expected}, got ${actual}: $*" >&2
    exit 1
  fi
}

echo "Local-network evidence only: ${REGISTRY_HOST}"
expect_status 200 --head "${base_url}/v2/"

docker pull "${reference}"
oras pull "${reference}" --output "${output_dir}/oras"

curl --fail --silent --show-error --range 0-1023 --output /dev/null \
  "${base_url}/v2/${IMAGE_REPOSITORY}/blobs/${BLOB_DIGEST}"

expect_status 403 "${base_url}/v2/${IMAGE_REPOSITORY}/manifests/latest"
expect_status 404 "${base_url}/v2/${IMAGE_REPOSITORY}/referrers/${IMAGE_DIGEST}"
expect_status 403 "${base_url}/v2/${IMAGE_REPOSITORY}/referrers/${IMAGE_DIGEST}?artifactType=application%2Fvnd.oci.image.manifest.v1%2Bjson"
expect_status 404 "${base_url}/v2/${IMAGE_REPOSITORY}/manifests/sha256-${IMAGE_DIGEST#sha256:}"
expect_status 403 "${base_url}/v2/_catalog"
expect_status 405 -X POST "${base_url}/v2/${IMAGE_REPOSITORY}/blobs/uploads/"

echo "Registry verification succeeded on this local network only."
