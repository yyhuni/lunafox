import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import {
  SNAPSHOT_PATHS,
  branchName,
  publishDeployment,
  snapshotShaFromBody,
  validateSnapshot,
  waitForPullRequestValidation,
  waitForValidation,
} from "./publish-public-deployment.mjs";
import { bindCompositionToManifest, compositionCorePayload, FINGERPRINT_SCHEMA_VERSION, sha256Digest } from "./resolve-release-component-composition.mjs";
import { canonicalPreheatManifestBytes } from "./preheat-manifest.mjs";

const TAG = "v1.2.3-alpha.4";
const SOURCE_SHA = "a".repeat(40);
const COMMIT_SHA = "b".repeat(40);
const MERGE_SHA = "c".repeat(40);

function fingerprint(componentId) {
  const inputs = {
    schemaVersion: FINGERPRINT_SCHEMA_VERSION, componentId, kind: componentId.split(".")[0], contextPath: ".",
    dockerfile: `${componentId.replaceAll(".", "/")}/Dockerfile`, dockerignore: "",
    files: [], namedContexts: {}, buildArgs: {}, platforms: ["linux/amd64", "linux/arm64"],
    baseImages: [], baseImagesResolved: true, builderPolicy: {}, generatedInputs: [],
  };
  return { version: FINGERPRINT_SCHEMA_VERSION, algorithm: "sha256-canonical-json-v1", digest: sha256Digest(inputs), baseImagesResolved: true, inputs };
}

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-deployment-publication-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const env = "RELEASE_REGISTRY=docker.io\n";
  const compositionValue = {
    schemaVersion: 1,
    kind: "lunafox.runtime-composition",
    releaseTag: TAG,
    components: [{
      id: "runtime.frontend",
      kind: "runtime",
      name: "frontend",
      inputFingerprint: fingerprint("runtime.frontend"),
      artifact: { ref: `ghcr.io/yyhuni/lunafox-frontend@sha256:${"a".repeat(64)}`, digest: `sha256:${"a".repeat(64)}` },
      disposition: "built",
      sourceRelease: { tag: TAG },
      evidence: { image: "image.json", provenance: "provenance.json", sbom: "sbom.json", signature: "signature.json" },
    }],
    capabilities: { dynamicFrontendUpstream: true },
  };
  compositionValue.compositionDigest = sha256Digest(compositionCorePayload(compositionValue));
  const thirdPartyEntries = [
    { service: "postgres", profiles: ["embedded"], registry: "docker.io", repository: "library/postgres", digest: `sha256:${"a".repeat(64)}`, contentAudit: "reviewed", publisherSignatureVerification: "not-approved", evidence: "release-frozen Docker Official Image digest reviewed by LunaFox" },
    { service: "redis", profiles: ["embedded", "external"], registry: "docker.io", repository: "library/redis", digest: `sha256:${"b".repeat(64)}`, contentAudit: "reviewed", publisherSignatureVerification: "not-approved", evidence: "release-frozen Docker Official Image digest reviewed by LunaFox" },
    { service: "loki", profiles: ["embedded", "external"], registry: "docker.io", repository: "grafana/loki", digest: `sha256:${"c".repeat(64)}`, contentAudit: "reviewed", publisherSignatureVerification: "not-approved", evidence: "release-frozen Grafana image digest reviewed by LunaFox" },
    { service: "alloy", profiles: ["embedded", "external"], registry: "docker.io", repository: "grafana/alloy", digest: `sha256:${"d".repeat(64)}`, contentAudit: "reviewed", publisherSignatureVerification: "not-approved", evidence: "release-frozen Grafana image digest reviewed by LunaFox" },
  ];
  const thirdPartyCompose = thirdPartyEntries.map((entry) => `  ${entry.service}:\n    image: ${entry.registry}/${entry.repository}@${entry.digest}\n`).join("");
  const files = {
    ".env": env,
    ".env.example": env,
    "compose.yaml": "services:\n  server:\n    image: ${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-server@sha256:" + "d".repeat(64) + "\n" + thirdPartyCompose,
    "engine-inventory.yaml": "enginePackages: []\n",
    "third-party-image-policy.json": JSON.stringify({
      schemaVersion: 1,
      provenanceClaim: "lunafox-reviewed-content",
      entries: thirdPartyEntries,
    }, null, 2) + "\n",
    "release.manifest.yaml": `releaseVersion: "1.2.3-alpha.4"\nreleaseNotes:\n  digest: "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"\n  body: |\n    ## English\n\n    - Test release notes.\n\n    ## 简体中文\n\n    - 测试发布说明。\nruntimeComposition:\n  schemaVersion: 1\n  asset: "runtime-composition.json"\n  sha256: "${compositionValue.compositionDigest}"\n`,
  };
  for (const [name, content] of Object.entries(files)) fs.writeFileSync(path.join(root, name), content);
  // The reverse manifest binding is added after the manifest bytes are fixed;
  // it is excluded from compositionDigest to avoid a circular hash.
  compositionValue.manifestBinding = {
    manifestDigest: sha256Digest(fs.readFileSync(path.join(root, "release.manifest.yaml"))),
  };
  fs.writeFileSync(path.join(root, "runtime-composition.json"), `${JSON.stringify(compositionValue, null, 2)}\n`);
  const engineDigest = `sha256:${"b".repeat(64)}`;
  const engineIdentity = `yyhuni/lunafox-engine-runtime-port-scan@${engineDigest}`;
  const entry = {
    candidates: [`docker.io/${engineIdentity}`, `ghcr.io/${engineIdentity}`],
    digest: engineDigest,
    identityReference: `ghcr.io/${engineIdentity}`,
    platforms: ["linux/amd64", "linux/arm64"],
    profiles: ["embedded", "external"],
    repository: "yyhuni/lunafox-engine-runtime-port-scan",
    sources: [{ kind: "engine-runtime", name: "engine.lunafox.port_scan" }],
    trust: "first-party",
  };
  const closures = ["embedded", "external"].map((profile) => {
    const entries = [engineIdentity];
    return { profile, entries, digest: sha256Digest({ profile, entries }) };
  });
  const preheat = {
    schemaVersion: 1,
    kind: "lunafox.preheat-manifest",
    release: {
      tag: TAG,
      manifestDigest: sha256Digest(fs.readFileSync(path.join(root, "release.manifest.yaml"))),
      compositionDigest: compositionValue.compositionDigest,
      composeDigest: sha256Digest(Buffer.from(files["compose.yaml"])),
      thirdPartyPolicyDigest: sha256Digest(Buffer.from(files["third-party-image-policy.json"])),
    },
    entries: [entry],
    profileClosures: closures,
  };
  const preheatBytes = canonicalPreheatManifestBytes({ ...preheat, manifestDigest: sha256Digest(preheat) });
  fs.writeFileSync(path.join(root, "preheat-manifest.json"), preheatBytes);
  fs.appendFileSync(path.join(root, ".env"), `LUNAFOX_PREHEAT_MANIFEST_DIGEST=${JSON.parse(preheatBytes).manifestDigest}\n`);
  fs.appendFileSync(path.join(root, ".env.example"), `LUNAFOX_PREHEAT_MANIFEST_DIGEST=${JSON.parse(preheatBytes).manifestDigest}\n`);
  return root;
}

function response(value = {}, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => status === 204 ? "" : JSON.stringify(value),
    json: async () => value,
  };
}

test("snapshot contract is deterministic and Registry-selectable", (t) => {
  const root = fixture(t);
  const first = validateSnapshot(root, TAG);
  const second = validateSnapshot(root, TAG);
  assert.deepEqual([...first.files.keys()], SNAPSHOT_PATHS);
  assert.equal(first.sha256, second.sha256);
  assert.equal(branchName(TAG), `deployment/${TAG}`);
  assert.equal(branchName(TAG, 2), `deployment/${TAG}-retry-2`);
  assert.equal(snapshotShaFromBody(`Deployment snapshot SHA-256: ${first.sha256}\n`), first.sha256);
  fs.writeFileSync(path.join(root, ".env"), "RELEASE_REGISTRY=ghcr.io\n");
  assert.throws(() => validateSnapshot(root, TAG), /must match/);
});

test("the alpha.164 bridge snapshot binds independent composition evidence", (t) => {
  const root = fixture(t);
  const bridgeTag = "v0.0.1-alpha.183";
  const manifest = path.join(root, "release.manifest.yaml");
  fs.writeFileSync(manifest, fs.readFileSync(manifest, "utf8")
    .replace('releaseVersion: "1.2.3-alpha.4"', 'releaseVersion: "0.0.1-alpha.183"')
    .replace(/^releaseNotes:\n[\s\S]*?(?=^runtimeComposition:)/m, "")
    .replace(/^runtimeComposition:\n[\s\S]*$/m, ""));
  // The bridge is intentionally the last package that predates install-time
  // preheating. It must not smuggle a modern manifest fingerprint or asset
  // into the historical snapshot while still retaining its composition file.
  fs.rmSync(path.join(root, "preheat-manifest.json"));
  for (const envName of [".env", ".env.example"]) {
    const envPath = path.join(root, envName);
    fs.writeFileSync(envPath, fs.readFileSync(envPath, "utf8").replace(/^LUNAFOX_PREHEAT_MANIFEST_DIGEST=.*\n/m, ""));
  }
  const composition = JSON.parse(fs.readFileSync(path.join(root, "runtime-composition.json"), "utf8"));
  composition.releaseTag = bridgeTag;
  composition.components[0].sourceRelease.tag = bridgeTag;
  delete composition.manifestBinding;
  composition.compositionDigest = sha256Digest(compositionCorePayload(composition));
  fs.writeFileSync(
    path.join(root, "runtime-composition.json"),
    `${JSON.stringify(bindCompositionToManifest(composition, sha256Digest(fs.readFileSync(manifest))), null, 2)}\n`,
  );
  assert.equal(validateSnapshot(root, bridgeTag, "alpha164-bridge").releaseProfile, "alpha164-bridge");
  assert.equal(fs.existsSync(path.join(root, "preheat-manifest.json")), false);
  assert.equal(fs.readFileSync(path.join(root, ".env"), "utf8").includes("LUNAFOX_PREHEAT_MANIFEST_DIGEST="), false);
  assert.throws(() => validateSnapshot(root, bridgeTag, "modern"), /does not match registered profile/);
});

test("publication creates one protected PR, reuses validation, and returns final main SHA", async (t) => {
  const snapshotDir = fixture(t);
  const snapshot = validateSnapshot(snapshotDir, TAG);
  const originalFetch = globalThis.fetch;
  const calls = [];
  let blob = 0;
  globalThis.fetch = async (url, init = {}) => {
    const parsed = new URL(url);
    const endpoint = `${parsed.pathname}${parsed.search}`;
    const method = init.method ?? "GET";
    const body = init.body ? JSON.parse(init.body) : null;
    calls.push({ endpoint, method, body });
    if (parsed.pathname === "/graphql") {
      if (String(body.query).startsWith("query")) {
        return response({ data: { repository: { pullRequest: { id: "PR_node", state: "OPEN", isDraft: false, autoMergeRequest: null } } } });
      }
      return response({ data: { enablePullRequestAutoMerge: { pullRequest: { number: 71, autoMergeRequest: { mergeMethod: "SQUASH" } } } } });
    }
    if (parsed.pathname.includes("/git/ref/heads/deployment%2F") && method === "GET") return response({ message: "Not Found" }, 404);
    if (parsed.pathname === "/repos/yyhuni/lunafox/pulls" && method === "GET") return response([]);
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/ref/heads/main" && method === "GET") {
      const dispatchCount = calls.filter((call) => call.endpoint.includes("/actions/workflows/public-validate.yml/runs")).length;
      return response({ object: { sha: dispatchCount >= 1 ? MERGE_SHA : SOURCE_SHA } });
    }
    if (parsed.pathname === `/repos/yyhuni/lunafox/git/commits/${SOURCE_SHA}` && method === "GET") return response({ tree: { sha: "tree_base" } });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/blobs" && method === "POST") return response({ sha: `blob_${blob++}` });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/trees" && method === "POST") return response({ sha: "tree_snapshot" });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/commits" && method === "POST") return response({ sha: COMMIT_SHA });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/refs" && method === "POST") return response({ ref: body.ref });
    if (parsed.pathname === "/repos/yyhuni/lunafox/pulls" && method === "POST") return response({ number: 71, state: "open", body: body.body });
    if (parsed.pathname.endsWith(`/actions/workflows/public-validate.yml/dispatches`) && method === "POST") return response({}, 204);
    if (parsed.pathname.endsWith("/actions/workflows/public-validate.yml/runs") && parsed.searchParams.get("event") === "pull_request") return response({ workflow_runs: [{ id: 123, head_sha: COMMIT_SHA, head_branch: `deployment/${TAG}`, event: "pull_request", path: ".github/workflows/public-validate.yml", head_repository: { full_name: "yyhuni/lunafox" }, status: "completed", conclusion: "success" }] });
    if (parsed.pathname === "/repos/yyhuni/lunafox/pulls/71" && method === "GET") {
      return response({ head: { sha: COMMIT_SHA }, number: 71, state: "closed", merged_at: "2026-09-16T00:00:00Z", merge_commit_sha: MERGE_SHA });
    }
    if (parsed.pathname.endsWith("/actions/workflows/public-validate.yml/runs")) return response({ workflow_runs: [mainRun()] });
    if (parsed.pathname === `/repos/yyhuni/lunafox/commits/${MERGE_SHA}/check-runs` && method === "GET") {
      return response({ check_runs: [{ name: "Public Projection Validation", status: "completed", conclusion: "success", head_sha: MERGE_SHA }] });
    }
    throw new Error(`unexpected request: ${method} ${endpoint}`);
  };
  try {
    const result = await publishDeployment({
      snapshotDir,
      tag: TAG,
      sourceSha: SOURCE_SHA,
      repo: "yyhuni/lunafox",
      baseBranch: "main",
      workflow: "public-validate.yml",
      apiBase: "https://api.test",
      graphqlBase: "https://api.test/graphql",
      timeoutSeconds: 60,
      pollSeconds: 1,
      token: "workflow-token",
      sleep: async () => {},
    });
    assert.equal(result.mergeSha, MERGE_SHA);
    assert.equal(result.snapshotSha256, snapshot.sha256);
    assert.equal(result.pullRequestNumber, 71);
    assert.equal(calls.filter((call) => call.endpoint.includes("/dispatches")).length, 0);
    const tree = calls.find((call) => call.endpoint === "/repos/yyhuni/lunafox/git/trees");
    assert.deepEqual(tree.body.tree.map((entry) => entry.path), SNAPSHOT_PATHS);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("merged publication is reused only for the same immutable snapshot", async (t) => {
  const snapshotDir = fixture(t);
  const snapshot = validateSnapshot(snapshotDir, TAG);
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async (url, init = {}) => {
    const parsed = new URL(url);
    const method = init.method ?? "GET";
    if (parsed.pathname.includes("/git/ref/heads/deployment%2F") && method === "GET") return response({ object: { sha: COMMIT_SHA } });
    if (parsed.pathname === "/repos/yyhuni/lunafox/pulls" && method === "GET") {
      return response([{ number: 71, state: "closed", merged_at: "2026-09-16T00:00:00Z", merge_commit_sha: MERGE_SHA, head: { ref: `deployment/${TAG}`, sha: COMMIT_SHA }, base: { ref: "main" }, body: `Deployment snapshot SHA-256: ${snapshot.sha256}\n` }]);
    }
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/ref/heads/main" && method === "GET") return response({ object: { sha: MERGE_SHA } });
    if (parsed.pathname.endsWith("/dispatches") && method === "POST") return response({}, 204);
    if (parsed.pathname.endsWith("/actions/workflows/public-validate.yml/runs")) return response({ workflow_runs: [mainRun()] });
    if (parsed.pathname === `/repos/yyhuni/lunafox/commits/${MERGE_SHA}/check-runs` && method === "GET") return response({ check_runs: [{ name: "Public Projection Validation", head_sha: MERGE_SHA, status: "completed", conclusion: "success" }] });
    throw new Error(`unexpected request: ${method} ${parsed.pathname}`);
  };
  try {
    const result = await publishDeployment({ snapshotDir, tag: TAG, sourceSha: SOURCE_SHA, repo: "yyhuni/lunafox", baseBranch: "main", workflow: "public-validate.yml", apiBase: "https://api.test", graphqlBase: "https://api.test/graphql", timeoutSeconds: 60, pollSeconds: 1, token: "workflow-token", sleep: async () => {} });
    assert.equal(result.reused, true);
    assert.equal(result.mergeSha, MERGE_SHA);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

function mainRun(overrides = {}) {
  return { id: 124, head_sha: MERGE_SHA, head_branch: "main", event: "push",
    path: ".github/workflows/public-validate.yml", head_repository: { full_name: "yyhuni/lunafox" },
    status: "completed", conclusion: "success", ...overrides };
}
const OPTIONS = { repo: "yyhuni/lunafox", baseBranch: "main", workflow: "public-validate.yml",
  apiBase: "https://api.test", token: "test", tag: TAG, timeoutSeconds: 60, pollSeconds: 1, sleep: async () => {} };

test("PR approval binds exact identity and approves a controlled run only once", async t => {
  const original = globalThis.fetch; t.after(() => { globalThis.fetch = original; });
  let approvals = 0, reads = 0;
  globalThis.fetch = async (url, init = {}) => {
    if (url.endsWith("/pulls/71")) return response({ head: { sha: COMMIT_SHA }, state: "open" });
    if (url.endsWith("/approve")) { approvals++; return response({}, 204); }
    reads++;
    return response({ workflow_runs: [mainRun({ id: 123, head_sha: COMMIT_SHA,
      head_branch: `deployment/${TAG}`, event: "pull_request",
      conclusion: reads < 3 ? "action_required" : "success" })] });
  };
  await waitForPullRequestValidation(OPTIONS, 71, `deployment/${TAG}`, COMMIT_SHA);
  assert.equal(approvals, 1);
});

test("PR changed head and failed validation stop publication", async t => {
  const original = globalThis.fetch; t.after(() => { globalThis.fetch = original; });
  globalThis.fetch = async () => response({ head: { sha: SOURCE_SHA }, state: "open" });
  await assert.rejects(waitForPullRequestValidation(OPTIONS, 71, `deployment/${TAG}`, COMMIT_SHA), /head changed/);
  globalThis.fetch = async url => url.endsWith("/pulls/71")
    ? response({ head: { sha: COMMIT_SHA }, state: "open" })
    : response({ workflow_runs: [mainRun({ head_sha: COMMIT_SHA, head_branch: `deployment/${TAG}`, event: "pull_request", conclusion: "failure" })] });
  await assert.rejects(waitForPullRequestValidation(OPTIONS, 71, `deployment/${TAG}`, COMMIT_SHA), /validation failed/);
});

test("main validation waits for running push without duplicate dispatch", async t => {
  const original = globalThis.fetch; t.after(() => { globalThis.fetch = original; });
  let reads = 0;
  globalThis.fetch = async (url, init = {}) => {
    assert.equal(init.method ?? "GET", "GET");
    return response({ workflow_runs: [mainRun(++reads === 1 ? { status: "in_progress", conclusion: null } : {})] });
  };
  await waitForValidation(OPTIONS, MERGE_SHA);
  assert.equal(reads, 2);
});

test("main rejects foreign success and dispatches once for missing canonical run", async t => {
  const original = globalThis.fetch; t.after(() => { globalThis.fetch = original; });
  let dispatched = 0;
  let dispatchBody;
  globalThis.fetch = async (url, init = {}) => {
    if (url.endsWith("/dispatches")) {
      dispatched++;
      dispatchBody = JSON.parse(init.body);
      return response({}, 204);
    }
    if (url.endsWith("/git/ref/heads/main")) return response({ object: { sha: MERGE_SHA } });
    return response({ workflow_runs: [mainRun(dispatched ? {} : { head_repository: { full_name: "attacker/fork" } })] });
  };
  await waitForValidation(OPTIONS, MERGE_SHA);
  assert.equal(dispatched, 1);
  assert.deepEqual(dispatchBody, { ref: "main", inputs: { publish: "false", release_tag: TAG } });
});

test("failed main workflow cannot be hidden by an older success", async t => {
  const original = globalThis.fetch; t.after(() => { globalThis.fetch = original; });
  globalThis.fetch = async () => response({ workflow_runs: [mainRun(), mainRun({ id: 125, conclusion: "failure" })] });
  await assert.rejects(waitForValidation(OPTIONS, MERGE_SHA), /validation failed/);
});
