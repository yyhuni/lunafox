import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";

const script = path.resolve("scripts/ci/finalize-engine-runtime-platforms.sh");
const engineId = "engine.lunafox.port_scan";
function setup(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "native-platform-test-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const input = path.join(root, "input"), bin = path.join(root, "bin");
  fs.mkdirSync(input); fs.mkdirSync(bin);
  // Any publication command is a test failure: malformed pairs must fail
  // before a Registry mutation can be attempted.
  for (const name of ["docker", "oras", "go"]) {
    fs.writeFileSync(path.join(bin, name), '#!/bin/sh\necho unexpected-publication-command >&2\nexit 77\n', { mode: 0o755 });
  }
  const record = platform => ({
    schemaVersion: "lunafox.engine-runtime-platform-build.v1",
    engineId, directory: "port_scan", dockerfile: "port_scan/Dockerfile",
    buildContext: ".", repository: "lunafox-engine-runtime-port-scan",
    platform, digest: "sha256:" + "a".repeat(64),
    sourceRef: "docker.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:" + "a".repeat(64),
    tagIdentity: "public-" + "b".repeat(40),
  });
  const write = (name, value) => {
    fs.mkdirSync(path.join(input, name));
    fs.writeFileSync(path.join(input, name, "platform-build.json"), JSON.stringify(value));
  };
  const run = () => {
    const result = spawnSync("bash", [script], {
      encoding: "utf8", env: { ...process.env, PATH: bin + ":" + process.env.PATH,
        ENGINE_RELEASE_ENGINE_ID: engineId, ENGINE_IMAGE_TAG: "public-" + "b".repeat(40),
        ENGINE_PLATFORM_BUILDS_ROOT: input, ENGINE_RUNTIME_IMAGE_SHARD_ROOT: path.join(root, "output") },
    });
    assert.notEqual(result.status, 0);
    assert.doesNotMatch(result.stderr, /unexpected-publication-command/);
    assert.equal(fs.existsSync(path.join(root, "output")), false);
    return result.stderr;
  };
  return { record, write, run };
}
test("missing architecture never publishes an index", t => {
  const f = setup(t); f.write("amd64", f.record("linux/amd64"));
  assert.match(f.run(), /exactly two platform receipts/);
});
test("duplicate architectures never publish an index", t => {
  const f = setup(t); f.write("first", f.record("linux/amd64")); f.write("second", f.record("linux/amd64"));
  assert.match(f.run(), /exact dual-architecture/);
});
for (const [name, change] of Object.entries({
  engine: { engineId: "engine.lunafox.other" },
  release: { tagIdentity: "public-other" },
  digest: { digest: "not-a-digest" },
  registry: { sourceRef: "ghcr.io/attacker/image@sha256:" + "a".repeat(64) },
  source: { dockerfile: "other/Dockerfile" },
})) {
  test("rejects mismatched platform " + name, t => {
    const f = setup(t); f.write("amd64", f.record("linux/amd64"));
    f.write("arm64", { ...f.record("linux/arm64"), ...change });
    assert.match(f.run(), /exact dual-architecture/);
  });
}

// Exercise the cold-pull gate with a stateful daemon double: a cached sibling
// reference must be evicted by image ID before either registry is accepted.
for (const scenario of ["valid", "wrong-digest", "wrong-platform", "cache-remains"]) {
  test("finalized native cold pull: " + scenario, t => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "native-cold-pull-"));
    t.after(() => fs.rmSync(root, { recursive: true, force: true }));
    const source = fs.readFileSync(script, "utf8");
    const functions = source.slice(source.indexOf("normalize_arch() {"), source.indexOf("# The finalizer's native daemon"));
    const harness = `set -euo pipefail
fail() { echo "$*" >&2; exit 1; }
${functions}
cached=1
pulled=0
docker() {
  if [ "$1 $2" = 'image rm' ]; then
    if [ "\${@: -1}" = 'image-id' ] && [ "$SCENARIO" != cache-remains ]; then cached=0; fi
    return 0
  fi
  if [ "$1" = pull ]; then
    [ "$cached" = 0 ] || return 90
    pulled=1; return 0
  fi
  if [ "$1" = info ]; then
    case "$*" in *OSType*) echo linux ;; *) echo x86_64 ;; esac
    return 0
  fi
  if [ "$1 $2" = 'image inspect' ]; then
    case "$*" in
      *'{{.Id}}'*) echo image-id ;;
      *RepoDigests*) if [ "$SCENARIO" = wrong-digest ]; then echo bad; else echo yyhuni/image@sha256:abc; fi ;;
      *'{{.Os}}'*) echo linux ;;
      *'{{.Architecture}}'*) if [ "$SCENARIO" = wrong-platform ]; then echo arm64; else echo amd64; fi ;;
      *) [ "$cached" = 1 ] || [ "$pulled" = 1 ] ;;
    esac
    return $?
  fi
  return 91
}
verify_host_pull docker.io/yyhuni/image@sha256:abc engine test ghcr.io/yyhuni/image@sha256:abc
`;
    const result = spawnSync("bash", ["-c", harness], { encoding: "utf8", env: { ...process.env, SCENARIO: scenario } });
    if (scenario === "valid") assert.equal(result.status, 0, result.stderr);
    else {
      assert.notEqual(result.status, 0);
      assert.match(result.stderr, scenario === "wrong-digest" ? /requested digest/ : scenario === "wrong-platform" ? /pull selected/ : /remained before independent cold pull/);
    }
  });
}

test("retry reuses an existing immutable index instead of comparing provenance graphs", () => {
  const source = fs.readFileSync(script, "utf8");
  assert.match(source, /immutable source of truth/);
  assert.match(source, /retry_transient_registry "oras cp \$docker_ref" oras cp "\$docker_ref" "\$ghcr_tag"/);
  assert.doesNotMatch(source, /already has a different graph/);
  assert.doesNotMatch(source, /proposed-canonical/);
});

test("transient registry retry retries not-found then succeeds", () => {
  const source = fs.readFileSync(script, "utf8");
  const helper = source.slice(source.indexOf("retry_transient_registry() {"), source.indexOf("existing_digest="));
  const harness = `set -euo pipefail
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
${helper}
calls_file="$tmp_dir/calls"
echo 0 >"$calls_file"
fake_inspect() {
  echo $(( $(cat "$calls_file") + 1 )) >"$calls_file"
  if [ "$(cat "$calls_file")" -lt 2 ]; then
    echo 'Error response from registry: sha256:abc: not found' >&2
    return 1
  fi
  echo reused-index
}
out="$(retry_transient_registry inspect fake_inspect)"
[ "$out" = reused-index ]
[ "$(cat "$calls_file")" -eq 2 ]
`;
  const result = spawnSync("bash", ["-c", harness], { encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr + result.stdout);
});

