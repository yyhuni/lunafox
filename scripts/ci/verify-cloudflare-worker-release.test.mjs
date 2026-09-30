import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import {
  parseArgs,
  staleDigest,
  verify,
} from "./verify-cloudflare-worker-release.mjs";

const root = path.resolve(new URL("../..", import.meta.url).pathname);
const policyPath = path.join(root, "scripts/ci/third-party-image-policy.json");
const workerSource = path.join(root, "tools/lunafox-ghcr-registry/src/registry.ts");
const policy = JSON.parse(fs.readFileSync(policyPath, "utf8"));
const releaseTag = "v0.0.1-alpha.999";
const privateSourceRevision = "sha256:" + "a".repeat(64);
const publicSourceCommit = "b".repeat(40);

function options(snapshotPolicy, output) {
  return {
    policy: policyPath,
    snapshotPolicy,
    workerSource,
    registryHost: "docker.lunafox.cc.cd",
    releaseTag,
    privateSourceRevision,
    publicSourceCommit,
    workerVersionId: "worker-version-1",
    deploymentTag: "lunafox-v0.0.1-alpha.999",
    output,
    timeoutMs: 100,
  };
}

test("verifies the current digest set and rejects an unlisted digest", async () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-worker-release-"));
  const snapshot = path.join(directory, "third-party-image-policy.json");
  fs.copyFileSync(policyPath, snapshot);
  const output = path.join(directory, "evidence.json");
  const stale = staleDigest(policy.entries);
  const fetcher = async (input) => {
    const url = new URL(String(input));
    if (url.pathname === "/v2/") return new Response(null, { status: 200, headers: { "Docker-Distribution-API-Version": "registry/2.0" } });
    const entry = policy.entries.find((candidate) => url.pathname.endsWith(`/manifests/${candidate.digest}`));
    if (entry) return new Response("manifest", { status: 200, headers: { "Docker-Content-Digest": entry.digest } });
    if (url.pathname.endsWith(`/manifests/${stale}`)) return new Response("forbidden", { status: 403 });
    return new Response("unexpected", { status: 500 });
  };

  const evidence = await verify(options(snapshot, output), fetcher);
  assert.equal(evidence.passed, true);
  assert.equal(evidence.smoke.currentDigests.length, policy.entries.length);
  assert.equal(evidence.staleDigest.result.status, 403);
  assert.deepEqual(JSON.parse(fs.readFileSync(output, "utf8")), evidence);
});

test("fails before network access when the snapshot policy drifts", async () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-worker-policy-"));
  const snapshot = path.join(directory, "third-party-image-policy.json");
  const changed = structuredClone(policy);
  changed.entries[0].digest = "sha256:" + "c".repeat(64);
  fs.writeFileSync(snapshot, `${JSON.stringify(changed, null, 2)}\n`);
  let calls = 0;
  await assert.rejects(
    verify(options(snapshot, ""), async () => { calls += 1; return new Response(null, { status: 200 }); }),
    /canonical third-party policy JSON bytes|does not match/,
  );
  assert.equal(calls, 0);
});

test("enforces a bounded request timeout", async () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-worker-timeout-"));
  const snapshot = path.join(directory, "third-party-image-policy.json");
  fs.copyFileSync(policyPath, snapshot);
  await assert.rejects(
    verify({ ...options(snapshot, ""), timeoutMs: 10 }, async () => new Promise(() => {})),
    /timed out after 10ms/,
  );
});

test("parses only the required CLI identity fields", () => {
  const args = parseArgs([
    "--release-tag", releaseTag,
    "--private-source-revision", privateSourceRevision,
    "--public-source-commit", publicSourceCommit,
    "--worker-version-id", "version-1",
    "--deployment-tag", "lunafox-v0.0.1-alpha.999",
    "--registry-host", "https://docker.lunafox.cc.cd",
  ]);
  assert.equal(args.registryHost, "docker.lunafox.cc.cd");
  assert.equal(args.timeoutMs, 15_000);
});

test("keeps the test fixture's hash primitive available for evidence checks", () => {
  assert.match(crypto.createHash("sha256").update("fixture").digest("hex"), /^[a-f0-9]{64}$/);
});
