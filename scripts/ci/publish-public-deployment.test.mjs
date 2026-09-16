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
} from "./publish-public-deployment.mjs";

const TAG = "v1.2.3-alpha.4";
const SOURCE_SHA = "a".repeat(40);
const COMMIT_SHA = "b".repeat(40);
const MERGE_SHA = "c".repeat(40);

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-deployment-publication-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const env = "RELEASE_REGISTRY=docker.io\n";
  const files = {
    ".env": env,
    ".env.example": env,
    "compose.yaml": "services:\n  server:\n    image: ${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-server@sha256:" + "d".repeat(64) + "\n",
    "engine-inventory.yaml": "enginePackages: []\n",
    "release.manifest.yaml": 'releaseVersion: "1.2.3-alpha.4"\n',
  };
  for (const [name, content] of Object.entries(files)) fs.writeFileSync(path.join(root, name), content);
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

test("publication creates one protected PR, dispatches validation, and returns final main SHA", async (t) => {
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
      const dispatchCount = calls.filter((call) => call.endpoint.includes("/dispatches")).length;
      return response({ object: { sha: dispatchCount >= 1 ? MERGE_SHA : SOURCE_SHA } });
    }
    if (parsed.pathname === `/repos/yyhuni/lunafox/git/commits/${SOURCE_SHA}` && method === "GET") return response({ tree: { sha: "tree_base" } });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/blobs" && method === "POST") return response({ sha: `blob_${blob++}` });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/trees" && method === "POST") return response({ sha: "tree_snapshot" });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/commits" && method === "POST") return response({ sha: COMMIT_SHA });
    if (parsed.pathname === "/repos/yyhuni/lunafox/git/refs" && method === "POST") return response({ ref: body.ref });
    if (parsed.pathname === "/repos/yyhuni/lunafox/pulls" && method === "POST") return response({ number: 71, state: "open", body: body.body });
    if (parsed.pathname.endsWith(`/actions/workflows/public-validate.yml/dispatches`) && method === "POST") return response({}, 204);
    if (parsed.pathname === "/repos/yyhuni/lunafox/pulls/71" && method === "GET") {
      return response({ number: 71, state: "closed", merged_at: "2026-09-16T00:00:00Z", merge_commit_sha: MERGE_SHA });
    }
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
    assert.equal(calls.filter((call) => call.endpoint.includes("/dispatches")).length, 2);
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
    if (parsed.pathname === `/repos/yyhuni/lunafox/commits/${MERGE_SHA}/check-runs` && method === "GET") return response({ check_runs: [{ name: "Public Projection Validation", conclusion: "success" }] });
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
