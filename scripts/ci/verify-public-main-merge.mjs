#!/usr/bin/env node

/**
 * Verify that a generated export PR was merged into the canonical public main
 * commit and that the secretless public validation check succeeded.  This is a
 * read-only gate; it never merges, pushes, or edits the destination repository.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DESTINATION_REPOSITORY = "yyhuni/lunafox";
const APP_TOKEN_ENV = "LUNAFOX_PUBLIC_REPO_APP_TOKEN";
const FALLBACK_TOKEN_NAMES = ["GITHUB_TOKEN", "GH_TOKEN", "GITHUB_PAT", "GH_PAT", "PAT"];

function fail(message) { throw new Error(message); }

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function parseArgs(argv) {
  const args = {
    repo: DESTINATION_REPOSITORY,
    tag: "",
    // The initial tag run cannot know the merge SHA until a maintainer merges
    // the generated PR. When omitted, the verifier discovers it from the
    // merged tag-scoped PR and emits the exact SHA for downstream jobs.
    mergeSha: "",
    prNumber: "",
    apiBase: "https://api.github.com",
    fixture: "",
    dryRun: false,
    json: false,
  };
  const valueFlags = new Set(["--repo", "--tag", "--merge-sha", "--pr-number", "--api-base", "--fixture"]);
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--dry-run") { args.dryRun = true; continue; }
    if (arg === "--json") { args.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-public-main-merge.mjs --tag <tag> [--merge-sha <sha>] [--pr-number <n>] [--fixture <json>] [--dry-run] [--json]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++i];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--repo") args.repo = value;
    else if (arg === "--tag") args.tag = value;
    else if (arg === "--merge-sha") args.mergeSha = value;
    else if (arg === "--pr-number") args.prNumber = value;
    else if (arg === "--api-base") args.apiBase = value;
    else if (arg === "--fixture") args.fixture = path.resolve(value);
  }
  if (args.repo !== DESTINATION_REPOSITORY) fail(`destination repository must be ${DESTINATION_REPOSITORY}`);
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(args.tag)) fail(`invalid release tag: ${args.tag}`);
  if (args.mergeSha && !/^[0-9a-f]{40}$/.test(args.mergeSha)) fail("--merge-sha must be a 40-character commit SHA");
  if (args.prNumber && !/^\d+$/.test(args.prNumber)) fail("--pr-number must be numeric");
  return args;
}

function rejectFallbackCredentials() {
  for (const name of FALLBACK_TOKEN_NAMES) {
    if (process.env[name]) fail(`refusing ${name}; public merge verification requires the destination App token`);
  }
}

async function request(apiBase, endpoint, token) {
  const response = await fetch(`${apiBase.replace(/\/$/, "")}${endpoint}`, {
    headers: {
      Accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2022-11-28",
      Authorization: `Bearer ${token}`,
      "User-Agent": "lunafox-public-merge-verifier/1",
    },
  });
  const text = await response.text();
  let body;
  try { body = text ? JSON.parse(text) : {}; } catch { body = { message: text.slice(0, 500) }; }
  if (!response.ok) fail(`GitHub API GET ${endpoint} failed (${response.status}): ${body.message ?? "unknown error"}`);
  return body;
}

function assertMergedPullRequest(pr, tag, mergeSha) {
  const expectedBranch = new RegExp(`^${escapeRegExp(`export/${tag}`)}(?:-retry-[0-9]+)?$`);
  if (pr.base?.ref !== "main") fail("public export PR must target main");
  if (!expectedBranch.test(pr.head?.ref ?? "")) fail(`public export PR branch is not tag-scoped: ${pr.head?.ref ?? ""}`);
  if (!pr.merged_at) fail("public export PR has not been merged into public main");
  if (pr.state && pr.state !== "closed") fail("public export PR must be closed after merge");
  if (!/^[0-9a-f]{40}$/.test(pr.merge_commit_sha ?? "")) fail("public export PR did not provide a canonical merge commit SHA");
  if (pr.merge_commit_sha !== mergeSha) fail("public export PR merge commit does not match the requested SHA");
  return { number: pr.number, branch: pr.head.ref, mergeSha: pr.merge_commit_sha, mergedAt: pr.merged_at };
}

function assertPublicValidation(checkRuns, statuses = [], expectedSha = "") {
  const runs = Array.isArray(checkRuns?.check_runs) ? checkRuns.check_runs : [];
  const matchingRuns = runs.filter((run) =>
    run.name === "Public Projection Validation" ||
    String(run.name ?? "").startsWith("Public Projection Validation / ")
  );
  const validation = matchingRuns.find((run) =>
    run.conclusion === "success" && (!expectedSha || run.head_sha === expectedSha)
  );
  if (!validation) {
    fail("Public Projection Validation check is not successful on the exact merge commit");
  }
  if (expectedSha && validation.head_sha !== expectedSha) {
    fail("Public Projection Validation check is not attached to the exact merge commit");
  }
  if (statuses.some((status) => status.state !== "success")) {
    fail("a required public status check is not successful on the exact merge commit");
  }
  return { check: validation.name, conclusion: validation.conclusion };
}

function assertProtection(protection) {
  const reviews = protection?.required_pull_request_reviews;
  if (!reviews) fail("public main branch protection does not require pull-request review");
  if (!Number.isInteger(reviews.required_approving_review_count) || reviews.required_approving_review_count < 1) {
    fail("public main branch protection must require at least one approving review");
  }
  if (protection?.enforce_admins?.enabled !== true) fail("public main branch protection must enforce rules for administrators");
  if (protection?.allow_force_pushes?.enabled === true) fail("public main must reject force pushes");
  if (protection?.allow_deletions?.enabled === true) fail("public main must reject branch deletion");
  const requiredChecks = protection?.required_status_checks;
  if (!requiredChecks) fail("public main branch protection must require status checks");
  const checkNames = [
    ...(Array.isArray(requiredChecks.checks) ? requiredChecks.checks.map((check) => check?.context) : []),
    ...(Array.isArray(requiredChecks.contexts) ? requiredChecks.contexts : []),
  ].filter((name) => typeof name === "string");
  if (!checkNames.some((name) => name === "Public Projection Validation" || name.startsWith("Public Projection Validation / "))) {
    fail("public main branch protection must require Public Projection Validation");
  }
  return {
    requiredReview: true,
    requiredChecks: checkNames,
    forcePushDisabled: protection?.allow_force_pushes?.enabled !== true,
  };
}

function loadFixture(file) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read merge verification fixture: ${error.message}`); }
}

async function verify(options) {
  if (options.dryRun) {
    return {
      schemaVersion: 1,
      dryRun: true,
      repository: options.repo,
      tag: options.tag,
      mergeSha: options.mergeSha,
      branchPattern: `export/${options.tag}`,
      requiredCheck: "Public Projection Validation",
      requiresProtectedReview: true,
    };
  }
  let fixture;
  if (options.fixture) fixture = loadFixture(options.fixture);
  const token = (process.env[APP_TOKEN_ENV] ?? "").trim();
  if (!fixture) {
    rejectFallbackCredentials();
    if (!token) fail(`${APP_TOKEN_ENV} is required; source GITHUB_TOKEN/PAT fallback is forbidden`);
  }
  const [owner, repo] = options.repo.split("/");
  // Query every PR targeting main.  GitHub's `head` filter cannot express the
  // retry branch suffix while preserving the exact tag boundary, so branch
  // matching is deliberately performed locally below.
  const pulls = fixture?.pulls ?? await request(options.apiBase, `/repos/${owner}/${repo}/pulls?state=all&base=main&per_page=100`, token);
  const candidates = Array.isArray(pulls) ? pulls : [];
  const branchPattern = new RegExp(`^${escapeRegExp(`export/${options.tag}`)}(?:-retry-[0-9]+)?$`);
  const branchCandidates = candidates.filter((item) =>
    item?.base?.ref === "main" && branchPattern.test(item?.head?.ref ?? "")
  );
  const pr = options.prNumber
    ? branchCandidates.find((item) => String(item.number) === options.prNumber)
    : options.mergeSha
      ? branchCandidates.find((item) => item.merged_at && item.merge_commit_sha === options.mergeSha)
      // Sort by merge time so a retry can supersede an older rejected/stale PR.
      : [...branchCandidates]
        .filter((item) => item.merged_at && /^[0-9a-f]{40}$/.test(item.merge_commit_sha ?? ""))
        .sort((left, right) => String(right.merged_at).localeCompare(String(left.merged_at)))[0];
  if (!pr) fail(`no merged export PR found for ${options.tag}${options.mergeSha ? ` and ${options.mergeSha}` : ""}`);
  const mergeSha = pr.merge_commit_sha;
  if (!/^[0-9a-f]{40}$/.test(mergeSha ?? "")) fail("merged export PR did not provide a commit SHA");
  if (options.mergeSha && mergeSha !== options.mergeSha) fail("discovered merge commit does not match --merge-sha");
  const merged = assertMergedPullRequest(pr, options.tag, mergeSha);
  const checks = fixture
    ? (fixture.checks ?? { check_runs: [] })
    : await request(options.apiBase, `/repos/${owner}/${repo}/commits/${mergeSha}/check-runs`, token);
  const statuses = fixture
    ? (fixture.statuses ?? [])
    : await request(options.apiBase, `/repos/${owner}/${repo}/commits/${mergeSha}/status`, token);
  const validation = assertPublicValidation(checks, statuses, mergeSha);
  const protectionPayload = fixture
    ? (fixture.protection ?? {})
    : await request(options.apiBase, `/repos/${owner}/${repo}/branches/main/protection`, token);
  const protection = assertProtection(protectionPayload);
  return { schemaVersion: 1, passed: true, repository: options.repo, tag: options.tag, merged, mergeSha, validation, protection };
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = await verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `public main merge verified: ${result.tag} (${result.mergeSha})`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => { process.stderr.write(`public main merge verification failed: ${error.message}\n`); process.exitCode = 1; });
}

export { assertMergedPullRequest, assertProtection, assertPublicValidation, parseArgs, verify };
