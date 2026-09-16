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
