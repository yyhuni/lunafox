#!/usr/bin/env node

/**
 * Publish one generated deployment snapshot through a protected public PR.
 * Wait for the exact PR run, approve its controlled deployment head if needed,
 * then validate the final main commit before publishing release metadata.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { parseRuntimeComposition } from "./verify-public-release.mjs";
import { validateComposition } from "./resolve-release-component-composition.mjs";
import {
  ReleaseCompatibilityProfileAlpha164Bridge,
  assertReleaseCompatibilityProfile,
} from "./release-compatibility-profile.mjs";

const REPOSITORY = "yyhuni/lunafox";
const WORKFLOW = "public-validate.yml";
const TOKEN_ENV = "GITHUB_TOKEN";
const SNAPSHOT_PATHS = Object.freeze([
  ".env",
  ".env.example",
  "compose.yaml",
  "engine-inventory.yaml",
  "release.manifest.yaml",
  "runtime-composition.json",
]);
const AUTHOR = Object.freeze({
  name: "LunaFox Deployment Publisher",
  email: "deployment-publisher@users.noreply.github.com",
});

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const options = {
    snapshotDir: "",
    tag: "",
    sourceSha: "",
    repo: REPOSITORY,
    baseBranch: "main",
    workflow: WORKFLOW,
    apiBase: "https://api.github.com",
    graphqlBase: "https://api.github.com/graphql",
    timeoutSeconds: 10800,
    pollSeconds: 20,
    releaseProfile: "",
    json: false,
  };
  const values = new Set([
    "--snapshot-dir", "--tag", "--source-sha", "--repo", "--base-branch",
    "--workflow", "--api-base", "--graphql-base", "--timeout-seconds", "--poll-seconds", "--release-profile",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/publish-public-deployment.mjs --snapshot-dir <dir> --tag <tag> --source-sha <sha> [--release-profile <modern|alpha164-bridge>] [--timeout-seconds <n>] [--json]\n");
      process.exit(0);
    }
    if (!values.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--snapshot-dir") options.snapshotDir = path.resolve(value);
    else if (arg === "--tag") options.tag = value;
    else if (arg === "--source-sha") options.sourceSha = value;
    else if (arg === "--repo") options.repo = value;
    else if (arg === "--base-branch") options.baseBranch = value;
    else if (arg === "--workflow") options.workflow = value;
    else if (arg === "--api-base") options.apiBase = value.replace(/\/$/, "");
    else if (arg === "--graphql-base") options.graphqlBase = value.replace(/\/$/, "");
    else if (arg === "--timeout-seconds") options.timeoutSeconds = Number(value);
    else if (arg === "--poll-seconds") options.pollSeconds = Number(value);
    else options.releaseProfile = value;
  }
  if (options.repo !== REPOSITORY) fail(`repository must be ${REPOSITORY}`);
  if (options.baseBranch !== "main") fail("base branch must be main");
  if (!/^v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?$/.test(options.tag)) fail(`invalid release tag: ${options.tag}`);
  if (!/^[a-f0-9]{40}$/.test(options.sourceSha)) fail("source SHA must be a 40-character commit SHA");
  if (!options.snapshotDir || !fs.existsSync(options.snapshotDir) || !fs.statSync(options.snapshotDir).isDirectory()) {
    fail(`snapshot directory is missing: ${options.snapshotDir}`);
  }
  if (!Number.isInteger(options.timeoutSeconds) || options.timeoutSeconds < 60) fail("timeout must be at least 60 seconds");
  if (!Number.isInteger(options.pollSeconds) || options.pollSeconds < 1) fail("poll interval must be at least one second");
  return options;
}

function branchName(tag, attempt = 0) {
  return attempt === 0 ? `deployment/${tag}` : `deployment/${tag}-retry-${attempt}`;
}

function containedFile(root, relative) {
  const target = path.join(root, ...relative.split("/"));
  if (!fs.existsSync(target) || !fs.statSync(target).isFile() || fs.lstatSync(target).isSymbolicLink()) {
    fail(`snapshot is missing regular file: ${relative}`);
  }
  return fs.readFileSync(target);
}

function validateSnapshot(snapshotDir, tag, requestedReleaseProfile = "") {
  const files = new Map(SNAPSHOT_PATHS.map((relative) => [relative, containedFile(snapshotDir, relative)]));
  if (!files.get(".env").equals(files.get(".env.example"))) fail("snapshot .env and .env.example must match");
  if (!/^RELEASE_REGISTRY=docker\.io$/m.test(files.get(".env").toString("utf8"))) {
    fail("snapshot must default RELEASE_REGISTRY to docker.io");
  }
  const manifestText = files.get("release.manifest.yaml").toString("utf8");
  const releaseVersion = manifestText.match(/^releaseVersion:\s*["']?([^"'\s]+)["']?/m)?.[1];
  if (`v${releaseVersion}` !== tag) fail("snapshot release manifest does not match the requested tag");
  const releaseProfile = assertReleaseCompatibilityProfile(releaseVersion, requestedReleaseProfile);
  const bridge = releaseProfile === ReleaseCompatibilityProfileAlpha164Bridge;
  const hasReleaseNotes = /^releaseNotes:[ \t]*$/m.test(manifestText);
  const hasRuntimeComposition = /^runtimeComposition:[ \t]*$/m.test(manifestText);
  if (bridge && (hasReleaseNotes || hasRuntimeComposition)) {
    fail("snapshot alpha164-bridge manifest must omit both releaseNotes and runtimeComposition");
  }
  if (!bridge && (!hasReleaseNotes || !hasRuntimeComposition)) {
    fail("snapshot modern manifest must contain releaseNotes and runtimeComposition");
  }
  // The bridge keeps only the reverse asset-to-Manifest binding because its
  // YAML must remain readable by alpha.164's strict decoder.
  const compositionBinding = bridge ? null : parseRuntimeComposition(manifestText);
  let composition;
  try { composition = JSON.parse(files.get("runtime-composition.json").toString("utf8")); }
  catch (error) { fail(`snapshot runtime composition is not valid JSON: ${error.message}`); }
  let normalizedComposition;
  try { normalizedComposition = validateComposition(composition, { requireManifestBinding: true }); }
  catch (error) { fail(`snapshot runtime composition is invalid: ${error.message}`); }
  if (normalizedComposition.releaseTag.replace(/^v/, "") !== tag.replace(/^v/, "")) fail("snapshot runtime composition release tag does not match the requested tag");
  if (compositionBinding && normalizedComposition.compositionDigest !== compositionBinding.sha256) fail("snapshot runtime composition digest does not match the release manifest");
  if (normalizedComposition.manifestBinding.manifestDigest !== `sha256:${crypto.createHash("sha256").update(files.get("release.manifest.yaml")).digest("hex")}`) {
    fail("snapshot runtime composition manifest binding does not match the release manifest bytes");
  }
  const compose = files.get("compose.yaml").toString("utf8");
  if (!compose.includes("${RELEASE_REGISTRY:-docker.io}/yyhuni/")) {
    fail("snapshot Compose does not contain the unified Registry selector");
  }
  const hash = crypto.createHash("sha256");
  for (const relative of SNAPSHOT_PATHS) {
    hash.update(relative);
    hash.update("\0");
    hash.update(files.get(relative));
    hash.update("\0");
  }
  return { files, releaseProfile, sha256: hash.digest("hex") };
}

function snapshotShaFromBody(body) {
  return String(body ?? "").match(/(?:^|\n)Deployment snapshot SHA-256:\s*([a-f0-9]{64})(?:\n|$)/i)?.[1]?.toLowerCase() ?? "";
}

async function request(options, endpoint, { method = "GET", body, accept = "application/vnd.github+json" } = {}) {
  const response = await fetch(`${options.apiBase}${endpoint}`, {
    method,
    headers: {
      Accept: accept,
      Authorization: `Bearer ${options.token}`,
      "Content-Type": "application/json",
      "User-Agent": "lunafox-deployment-publisher/1",
      "X-GitHub-Api-Version": "2022-11-28",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  let parsed = {};
  try { parsed = text ? JSON.parse(text) : {}; } catch { parsed = { message: text.slice(0, 500) }; }
  if (!response.ok) fail(`GitHub API ${method} ${endpoint} failed (${response.status}): ${parsed.message ?? "unknown error"}`);
  return parsed;
}

async function optionalRequest(options, endpoint) {
  try { return { found: true, value: await request(options, endpoint) }; }
  catch (error) {
    if (/failed \(404\):/.test(error.message)) return { found: false, value: null };
    throw error;
  }
}

async function graphql(options, query, variables) {
  const response = await fetch(options.graphqlBase, {
    method: "POST",
    headers: {
      Accept: "application/vnd.github+json",
      Authorization: `Bearer ${options.token}`,
      "Content-Type": "application/json",
      "User-Agent": "lunafox-deployment-publisher/1",
    },
    body: JSON.stringify({ query, variables }),
  });
  const body = await response.json();
  if (!response.ok || body.errors?.length) {
    fail(`GitHub GraphQL request failed: ${body.errors?.map((error) => error.message).join("; ") ?? response.status}`);
  }
  return body.data ?? {};
}

async function findPublication(options, snapshotSha) {
  const [owner] = options.repo.split("/");
  for (let attempt = 0; attempt <= 100; attempt += 1) {
    const branch = branchName(options.tag, attempt);
    const ref = await optionalRequest(options, `/repos/${options.repo}/git/ref/heads/${encodeURIComponent(branch)}`);
    const pulls = await request(options, `/repos/${options.repo}/pulls?state=all&head=${encodeURIComponent(`${owner}:${branch}`)}&base=main&per_page=100`);
    const matches = (Array.isArray(pulls) ? pulls : []).filter((pr) => pr.head?.ref === branch && pr.base?.ref === "main");
    const merged = matches.find((pr) => pr.merged_at);
    if (merged) {
      if (snapshotShaFromBody(merged.body) !== snapshotSha) fail(`deployment ${options.tag} already merged with a different snapshot`);
      return { branch, refSha: ref.value?.object?.sha ?? merged.head?.sha ?? "", pr: merged, merged: true };
    }
    const open = matches.find((pr) => pr.state === "open" && snapshotShaFromBody(pr.body) === snapshotSha && ref.found);
    if (open) return { branch, refSha: ref.value.object.sha, pr: open, merged: false };
    if (!ref.found && matches.length === 0) return { branch, refSha: "", pr: null, merged: false };
  }
  fail(`no deployment branch remains for ${options.tag}`);
}

async function createCommit(options, snapshot, branch) {
  const ref = await request(options, `/repos/${options.repo}/git/ref/heads/${options.baseBranch}`);
  const baseSha = ref.object?.sha;
  if (baseSha !== options.sourceSha) {
    fail(`public main moved before deployment publication: expected ${options.sourceSha}, got ${baseSha ?? "missing"}`);
  }
  const baseCommit = await request(options, `/repos/${options.repo}/git/commits/${baseSha}`);
  const tree = [];
  for (const relative of SNAPSHOT_PATHS) {
    const blob = await request(options, `/repos/${options.repo}/git/blobs`, {
      method: "POST",
      body: { content: snapshot.files.get(relative).toString("base64"), encoding: "base64" },
    });
    tree.push({ path: relative, mode: "100644", type: "blob", sha: blob.sha });
  }
  const generatedTree = await request(options, `/repos/${options.repo}/git/trees`, {
    method: "POST",
    body: { base_tree: baseCommit.tree?.sha, tree },
  });
  const commit = await request(options, `/repos/${options.repo}/git/commits`, {
    method: "POST",
    body: {
      message: `chore(deploy): finalize deployment snapshot ${options.tag}`,
      tree: generatedTree.sha,
      parents: [baseSha],
      author: AUTHOR,
      committer: AUTHOR,
    },
  });
  await request(options, `/repos/${options.repo}/git/refs`, {
    method: "POST",
    body: { ref: `refs/heads/${branch}`, sha: commit.sha },
  });
  return commit.sha;
}

async function createPullRequest(options, branch, snapshotSha) {
  return request(options, `/repos/${options.repo}/pulls`, {
    method: "POST",
    body: {
      title: `chore(deploy): finalize deployment snapshot ${options.tag}`,
      head: branch,
      base: options.baseBranch,
      maintainer_can_modify: false,
      body: `Generated after the complete dual-Registry release closure passed.\n\nDeployment snapshot SHA-256: ${snapshotSha}\nSource projection SHA: ${options.sourceSha}\n\nThe six root deployment files are an atomic, generated snapshot.`,
    },
  });
}

async function dispatchValidation(options, ref) {
  await request(options, `/repos/${options.repo}/actions/workflows/${encodeURIComponent(options.workflow)}/dispatches`, {
    method: "POST",
    body: { ref },
  });
}

async function enableAutoMerge(options, number) {
  const [owner, name] = options.repo.split("/");
  const query = `query($owner:String!,$name:String!,$number:Int!){repository(owner:$owner,name:$name){pullRequest(number:$number){id state isDraft autoMergeRequest{mergeMethod}}}}`;
  const data = await graphql(options, query, { owner, name, number });
  const pr = data.repository?.pullRequest;
  if (!pr?.id || pr.state !== "OPEN" || pr.isDraft) fail("deployment PR is not eligible for auto-merge");
  if (pr.autoMergeRequest) return;
  const mutation = `mutation($id:ID!){enablePullRequestAutoMerge(input:{pullRequestId:$id,mergeMethod:SQUASH}){pullRequest{number autoMergeRequest{mergeMethod}}}}`;
  const enabled = await graphql(options, mutation, { id: pr.id });
  if (!enabled.enablePullRequestAutoMerge?.pullRequest?.autoMergeRequest) fail("GitHub did not enable deployment PR auto-merge");
}

function delay(milliseconds) { return new Promise((resolve) => setTimeout(resolve, milliseconds)); }

async function poll(options, operation, label) {
  const deadline = Date.now() + options.timeoutSeconds * 1000;
  while (Date.now() < deadline) {
    const result = await operation();
    if (result?.done) return result.value;
    await options.sleep(options.pollSeconds * 1000);
  }
  fail(`${label} did not complete within ${options.timeoutSeconds} seconds`);
}

export async function waitForPullRequestValidation(options, number, branch, headSha) {
  const workflow = encodeURIComponent(options.workflow);
  const deadline = Date.now() + options.timeoutSeconds * 1000;
  const approved = new Set();
  while (Date.now() < deadline) {
    const pr = await request(options, `/repos/${options.repo}/pulls/${number}`);
    if (pr.head?.sha !== headSha) fail("deployment PR head changed during validation");
    if (pr.state === "closed" && !pr.merged_at) fail("deployment PR closed without merging");
    const result = await request(options, `/repos/${options.repo}/actions/workflows/${workflow}/runs?event=pull_request&head_sha=${headSha}&per_page=100`);
    const run = (result.workflow_runs ?? []).find(item =>
      item.head_sha === headSha && item.head_branch === branch && item.event === "pull_request" &&
      item.path === `.github/workflows/${options.workflow}` && item.head_repository?.full_name === options.repo);
    if (run) {
      if (run.conclusion === "action_required") {
        // Approve only the controlled same-repository deployment head. A
        // workflow_dispatch check cannot replace this PR-associated gate.
        if (!approved.has(run.id)) {
          await request(options, `/repos/${options.repo}/actions/runs/${run.id}/approve`, { method: "POST" });
          approved.add(run.id);
        }
      } else if (run.status === "completed") {
        if (run.conclusion !== "success") fail(`deployment PR validation failed: ${run.conclusion}`);
        return;
      }
    }
    await options.sleep(options.pollSeconds * 1000);
  }
  fail("deployment PR validation did not complete; inspect the PR-associated workflow run");
}

async function waitForMerge(options, number) {
  return poll(options, async () => {
    const pr = await request(options, `/repos/${options.repo}/pulls/${number}`);
    if (pr.merged_at) {
      if (!/^[a-f0-9]{40}$/.test(pr.merge_commit_sha ?? "")) fail("merged deployment PR has no canonical merge SHA");
      return { done: true, value: pr.merge_commit_sha };
    }
    if (pr.state !== "open") fail("deployment PR closed without merging");
    return { done: false };
  }, "deployment PR auto-merge");
}

export async function waitForValidation(options, sha) {
  let dispatched = false;
  return poll(options, async () => {
    const result = await request(options, `/repos/${options.repo}/actions/workflows/${encodeURIComponent(options.workflow)}/runs?head_sha=${sha}&per_page=100`);
    // Check names alone are not authority: another app or workflow can emit
    // the same name. Bind reuse to this workflow, repository, branch and SHA.
    const runs = (result.workflow_runs ?? []).filter(run =>
      run.head_sha === sha && run.head_branch === options.baseBranch &&
      ["push", "workflow_dispatch"].includes(run.event) &&
      run.path === `.github/workflows/${options.workflow}` &&
      run.head_repository?.full_name === options.repo);
    const run = runs.sort((a, b) => b.id - a.id)[0];
    if (run?.status === "completed") {
      if (run.conclusion !== "success") fail(`deployment validation failed on ${sha}: ${run.conclusion}`);
      return { done: true, value: true };
    }
    if (!run && !dispatched) {
      const main = await request(options, `/repos/${options.repo}/git/ref/heads/${options.baseBranch}`);
      if (main.object?.sha !== sha) fail("public main moved before deployment validation dispatch");
      await dispatchValidation(options, options.baseBranch);
      dispatched = true;
    }
    return { done: false };
  }, `Public Projection Validation on ${sha}`);
}

async function publishDeployment(input) {
  const options = { ...input, token: input.token ?? (process.env[TOKEN_ENV] ?? "").trim(), sleep: input.sleep ?? delay };
  if (!options.token) fail(`${TOKEN_ENV} is required`);
  const snapshot = validateSnapshot(options.snapshotDir, options.tag, options.releaseProfile);
  const publication = await findPublication(options, snapshot.sha256);
  let pr = publication.pr;
  let commitSha = publication.refSha;
  if (!pr) {
    commitSha = await createCommit(options, snapshot, publication.branch);
    pr = await createPullRequest(options, publication.branch, snapshot.sha256);
  }
  let mergeSha = pr.merge_commit_sha ?? "";
  if (!pr.merged_at) {
    await enableAutoMerge(options, Number(pr.number));
    await waitForPullRequestValidation(options, Number(pr.number), publication.branch, commitSha);
    mergeSha = await waitForMerge(options, Number(pr.number));
  }
  await poll(options, async () => {
    const main = await request(options, `/repos/${options.repo}/git/ref/heads/${options.baseBranch}`);
    return main.object?.sha === mergeSha ? { done: true, value: true } : { done: false };
  }, "deployment merge visibility on public main");
  await waitForValidation(options, mergeSha);
  return {
    schemaVersion: 1,
    tag: options.tag,
    sourceSha: options.sourceSha,
    snapshotSha256: snapshot.sha256,
    branch: publication.branch,
    commitSha,
    pullRequestNumber: Number(pr.number),
    mergeSha,
    reused: Boolean(publication.pr),
  };
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = await publishDeployment(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : result.mergeSha}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  main().catch((error) => {
    process.stderr.write(`public deployment publication failed: ${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}

export {
  SNAPSHOT_PATHS,
  branchName,
  parseArgs,
  publishDeployment,
  snapshotShaFromBody,
  validateSnapshot,
};
