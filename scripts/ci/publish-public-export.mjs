#!/usr/bin/env node

/**
 * Publish one already-validated projection to the public repository.
 *
 * This utility intentionally has no fallback credential.  A destination-only
 * GitHub App installation token is required for mutation; the private source
 * GITHUB_TOKEN and personal PATs are rejected before the first API write.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { execFileSync } from "node:child_process";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DESTINATION_REPOSITORY = "yyhuni/lunafox";
const APP_TOKEN_ENV = "LUNAFOX_PUBLIC_REPO_APP_TOKEN";
const APP_ID_ENV = "LUNAFOX_PUBLIC_REPO_APP_ID";
const APP_INSTALLATION_ENV = "LUNAFOX_PUBLIC_REPO_APP_INSTALLATION_ID";
const APP_PRIVATE_KEY_ENV = "LUNAFOX_PUBLIC_REPO_APP_PRIVATE_KEY";
const REQUIRED_APP_PERMISSIONS = Object.freeze({ contents: "write", pull_requests: "write", metadata: "read" });
const AGENT_BUNDLE_FILES = Object.freeze([
  "lunafox-agent-linux-amd64",
  "lunafox-agent-linux-arm64",
  "lunafox-engine-mount-preflight-linux-amd64",
  "lunafox-engine-mount-preflight-linux-arm64",
  "agent-bundle.json",
  "agent-bundle.sha256",
  "agent-bundle.sigstore.json",
]);

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const args = {
    exportDir: "",
    tag: "",
    repo: DESTINATION_REPOSITORY,
    baseBranch: "main",
    apiBase: "https://api.github.com",
    attempt: "",
    destinationOwnedRoot: "",
    dryRun: false,
    noValidate: false,
    json: false,
  };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--export-dir") args.exportDir = path.resolve(argv[++i] ?? "");
    else if (arg === "--tag") args.tag = argv[++i] ?? "";
    else if (arg === "--repo") args.repo = argv[++i] ?? "";
    else if (arg === "--base-branch") args.baseBranch = argv[++i] ?? "";
    else if (arg === "--api-base") args.apiBase = argv[++i] ?? "";
    else if (arg === "--attempt") args.attempt = argv[++i] ?? "";
    else if (arg === "--destination-owned-root") args.destinationOwnedRoot = path.resolve(argv[++i] ?? "");
    else if (arg === "--dry-run") args.dryRun = true;
    else if (arg === "--no-validate") args.noValidate = true;
    else if (arg === "--json") args.json = true;
    else if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/publish-public-export.mjs --export-dir <dir> --tag <tag> [--attempt <n>] [--destination-owned-root <dir>] [--dry-run] [--json]\n");
      process.exit(0);
    } else fail(`unknown argument: ${arg}`);
  }
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(args.tag)) fail(`invalid release tag: ${args.tag}`);
  if (args.repo !== DESTINATION_REPOSITORY) fail(`destination repository must be ${DESTINATION_REPOSITORY}`);
  if (!/^main$/.test(args.baseBranch)) fail("base branch must be main");
  if (!args.exportDir || !fs.existsSync(args.exportDir)) fail(`export directory is missing: ${args.exportDir}`);
  if (args.destinationOwnedRoot && (!fs.existsSync(args.destinationOwnedRoot) || !fs.statSync(args.destinationOwnedRoot).isDirectory())) {
    fail(`destination-owned source root is missing: ${args.destinationOwnedRoot}`);
  }
  if (args.attempt && !/^\d+$/.test(args.attempt)) fail("--attempt must be a non-negative integer");
  return args;
}

function branchName(tag, attempt) {
  const normalized = attempt === undefined || attempt === null ? "" : String(attempt);
  return normalized === "" || normalized === "0"
    ? `export/${tag}`
    : `export/${tag}-retry-${normalized}`;
}

function walkFiles(root) {
  const files = [];
  const visit = (directory, relative = "") => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      if (entry.name === ".git") continue;
      const full = path.join(directory, entry.name);
      const rel = path.posix.join(relative, entry.name);
      if (entry.isDirectory()) visit(full, rel);
      else if (entry.isFile()) files.push({ path: rel, full });
      else fail(`export contains unsupported entry: ${rel}`);
    }
  };
  visit(root);
  return files;
}

function validateExportTree(exportDir) {
  const checker = path.join(SCRIPT_DIR, "check-public-export.mjs");
  try {
    execFileSync(process.execPath, [checker, "--repo-root", exportDir, "--require-agent-bundle"], { stdio: "pipe" });
  } catch (error) {
    const detail = error.stderr?.toString().trim() || error.stdout?.toString().trim() || error.message;
    fail(`export preflight failed before public mutation: ${detail}`);
  }
  const manifestPath = path.join(exportDir, "PUBLIC_EXPORT_MANIFEST.json");
  if (!fs.existsSync(manifestPath)) fail("export preflight did not produce PUBLIC_EXPORT_MANIFEST.json");
  return JSON.parse(fs.readFileSync(manifestPath, "utf8"));
}

function agentBundleSha256(exportDir, tag) {
  const bundleDir = path.join(exportDir, "agent", "bin", tag);
  const hash = crypto.createHash("sha256");
  for (const name of AGENT_BUNDLE_FILES) {
    const file = path.join(bundleDir, name);
    if (!fs.existsSync(file) || !fs.statSync(file).isFile()) {
      fail(`export is missing immutable Agent bundle member: ${name}`);
    }
    hash.update(name);
    hash.update("\0");
    hash.update(fs.readFileSync(file));
    hash.update("\0");
  }
  return hash.digest("hex");
}

function rejectFallbackCredentials() {
  for (const name of ["GITHUB_TOKEN", "GH_TOKEN", "GITHUB_PAT", "PAT", "GH_PAT"]) {
    if (process.env[name]) fail(`refusing ${name} for cross-repository publication; use ${APP_TOKEN_ENV}`);
  }
  for (const name of [APP_TOKEN_ENV]) {
    if (process.env[name] && /^(?:ghp_|github_pat_)/.test(process.env[name])) fail(`${name} looks like a personal token; destination App token is required`);
  }
}

function base64Url(value) {
  return Buffer.from(value).toString("base64").replace(/=/g, "").replace(/\+/g, "-").replace(/\//g, "_");
}

function appJwt() {
  const appId = (process.env[APP_ID_ENV] ?? "").trim();
  const privateKey = process.env[APP_PRIVATE_KEY_ENV] ?? "";
  if (!/^\d+$/.test(appId)) fail(`${APP_ID_ENV} is required to mint an installation token`);
  if (!privateKey.includes("BEGIN RSA PRIVATE KEY") && !privateKey.includes("BEGIN PRIVATE KEY")) fail(`${APP_PRIVATE_KEY_ENV} is required to mint an installation token`);
  const now = Math.floor(Date.now() / 1000);
  const header = base64Url(JSON.stringify({ alg: "RS256", typ: "JWT" }));
  const payload = base64Url(JSON.stringify({ iat: now - 30, exp: now + 540, iss: appId }));
  const signingInput = `${header}.${payload}`;
  const signature = crypto.createSign("RSA-SHA256").update(signingInput).sign(privateKey);
  return `${signingInput}.${base64Url(signature)}`;
}

async function request(apiBase, endpoint, { method = "GET", token = "", body } = {}) {
  const headers = {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
    "User-Agent": "lunafox-public-export-publisher/1",
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(`${apiBase.replace(/\/$/, "")}${endpoint}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  let parsed;
  try { parsed = text ? JSON.parse(text) : {}; } catch { parsed = { message: text.slice(0, 500) }; }
  if (!response.ok) {
    const message = typeof parsed?.message === "string" ? parsed.message : `HTTP ${response.status}`;
    fail(`GitHub API ${method} ${endpoint} failed (${response.status}): ${message}`);
  }
  return parsed;
}

async function requestOptional(apiBase, endpoint, token) {
  try {
    return { found: true, value: await request(apiBase, endpoint, { token }) };
  } catch (error) {
    if (/\(404\):/.test(error.message)) return { found: false, value: null };
    throw error;
  }
}

function normalizeManifestSha(value) {
  const normalized = String(value ?? "").trim().toLowerCase();
  return /^[a-f0-9]{64}$/.test(normalized) ? normalized : "";
}

/**
 * Read the manifest identity written into a generated PR body. The body is
 * deliberately a strict, machine-readable line so a rerun can distinguish an
 * idempotent publication from a stale or hand-edited branch.
 */
function manifestShaFromPublication(publication) {
  for (const value of [publication?.manifestSha256, publication?.manifest_sha256]) {
    const normalized = normalizeManifestSha(value);
    if (normalized) return normalized;
  }
  const body = String(publication?.body ?? "");
  const match = body.match(/(?:^|\r?\n)\s*Export manifest SHA-256:\s*([a-f0-9]{64})\s*(?:\r?\n|$)/i);
  return normalizeManifestSha(match?.[1]);
}

function bundleShaFromPublication(publication) {
  for (const value of [publication?.agentBundleSha256, publication?.agent_bundle_sha256]) {
    const normalized = normalizeManifestSha(value);
    if (normalized) return normalized;
  }
  const body = String(publication?.body ?? "");
  const match = body.match(/(?:^|\r?\n)\s*Agent bundle SHA-256:\s*([a-f0-9]{64})\s*(?:\r?\n|$)/i);
  return normalizeManifestSha(match?.[1]);
}

function publicationState(publication) {
  if (publication?.merged_at || publication?.merged === true) return "merged";
  return String(publication?.state ?? "").toLowerCase() === "open" ? "open" : "closed";
}

/**
 * Decide whether an occupied tag-scoped branch can be reused. A merged PR
 * with a different manifest is terminal: creating another PR for the same tag
 * would make the immutable release identity ambiguous. Closed/unmerged or
 * open-but-stale PRs are retained and bypassed with a retry branch.
 */
function classifyExistingPublication(publication, expectedManifestSha, expectedBundleSha = "", { branchExists = true } = {}) {
  const state = publicationState(publication);
  const expected = normalizeManifestSha(expectedManifestSha);
  const expectedBundle = normalizeManifestSha(expectedBundleSha);
  const actualManifestSha = manifestShaFromPublication(publication);
  const actualBundleSha = bundleShaFromPublication(publication);
  const manifestMatches = Boolean(expected && actualManifestSha && actualManifestSha === expected);
  const bundleMatches = !expectedBundle || Boolean(actualBundleSha && actualBundleSha === expectedBundle);
  const identityMatches = manifestMatches && bundleMatches;
  if (state === "merged") {
    return identityMatches
      ? { action: "reuse", state, manifestSha256: actualManifestSha, agentBundleSha256: actualBundleSha }
      : { action: "conflict", state, manifestSha256: actualManifestSha, agentBundleSha256: actualBundleSha };
  }
  if (state === "open" && identityMatches && branchExists) {
    return { action: "reuse", state, manifestSha256: actualManifestSha, agentBundleSha256: actualBundleSha };
  }
  return {
    action: "retry",
    state,
    manifestSha256: actualManifestSha,
    agentBundleSha256: actualBundleSha,
    reason: state === "open" && !branchExists ? "open PR head branch is missing" : "stale or rejected export PR",
  };
}

async function findExistingPublication({ apiBase, owner, repo, baseBranch, tag, attempt, token, manifestSha256, agentBundleSha256 = "" }) {
  const startAttempt = attempt === "" || attempt === undefined || attempt === null ? 0 : Number(attempt);
  for (let candidateAttempt = startAttempt; candidateAttempt <= 100; candidateAttempt += 1) {
    const branch = branchName(tag, candidateAttempt === 0 ? "" : String(candidateAttempt));
    const refResult = await requestOptional(apiBase, `/repos/${owner}/${repo}/git/ref/heads/${branch}`, token);
    const refSha = refResult.value?.object?.sha;
    const pulls = await request(
      apiBase,
      `/repos/${owner}/${repo}/pulls?state=all&head=${encodeURIComponent(`${owner}:${branch}`)}&base=${encodeURIComponent(baseBranch)}&per_page=100`,
      { token },
    );
    const matches = (Array.isArray(pulls) ? pulls : []).filter((item) =>
      item?.base?.ref === baseBranch && item?.head?.ref === branch,
    );

    if (!refResult.found && matches.length === 0) {
      return { branch, refSha: null, pr: null, state: "available" };
    }

    // A merged PR is checked first so a conflicting immutable publication
    // cannot be hidden by an unrelated open retry PR on the same branch.
    for (const publication of matches.filter((item) => publicationState(item) === "merged")) {
      const decision = classifyExistingPublication(publication, manifestSha256, agentBundleSha256, { branchExists: refResult.found });
      if (decision.action === "conflict") {
        fail(`destination branch ${branch} already has a merged PR with a different export manifest or Agent bundle`);
      }
      if (decision.action === "reuse") {
        if (publication.head?.sha && refSha && publication.head.sha !== refSha) {
          fail(`destination branch ${branch} no longer points at the merged PR head`);
        }
        return { branch, refSha, pr: publication, state: decision.state };
      }
    }
    for (const publication of matches.filter((item) => publicationState(item) !== "merged")) {
      const decision = classifyExistingPublication(publication, manifestSha256, agentBundleSha256, { branchExists: refResult.found });
      if (decision.action === "reuse") {
        // Do not reuse a branch whose PR head no longer points at the branch
        // ref; that is evidence of a stale/re-written review head.
        if (publication.head?.sha && refSha && publication.head.sha !== refSha) continue;
        return { branch, refSha, pr: publication, state: decision.state };
      }
    }

    // Existing branches/PRs are immutable review history. Never delete or
    // force-update them; move to the next retry suffix instead.
  }
  fail(`no available tag-scoped export branch remains for ${tag} (attempts exhausted)`);
}

async function resolveToken(apiBase) {
  rejectFallbackCredentials();
  const direct = (process.env[APP_TOKEN_ENV] ?? "").trim();
  if (direct) return direct;
  const installation = (process.env[APP_INSTALLATION_ENV] ?? "").trim();
  if (!/^\d+$/.test(installation)) fail(`set ${APP_TOKEN_ENV}, or provide ${APP_ID_ENV}, ${APP_INSTALLATION_ENV}, and ${APP_PRIVATE_KEY_ENV}`);
  const jwt = appJwt();
  const response = await request(apiBase, `/app/installations/${installation}/access_tokens`, { method: "POST", token: jwt, body: {} });
  if (!response.token) fail("GitHub App installation token response did not contain token");
  return response.token;
}

async function verifyDestinationInstallation(apiBase, token, destinationRepository) {
  // GitHub does not expose the App installation permission map through an
  // installation token at /installation. The workflow scopes this short-lived
  // token explicitly with permission-* inputs; the supported token endpoint
  // below verifies the resulting repository boundary before the first write.
  const installed = await request(apiBase, "/installation/repositories?per_page=100", { token });
  const repositories = Array.isArray(installed.repositories) ? installed.repositories : [];
  const names = repositories.map((repository) => repository?.full_name).filter((name) => typeof name === "string");
  if (installed.repository_selection && installed.repository_selection !== "selected") {
    fail(`destination App installation token must use selected repository scope, got ${String(installed.repository_selection)}`);
  }
  if (names.length !== 1 || names[0] !== destinationRepository) {
    fail(`destination App installation must be scoped only to ${destinationRepository}`);
  }
  return { repositorySelection: installed.repository_selection ?? "selected", repositories: names };
}

function runGit(repository, args, { env = {}, maxBuffer = 10 * 1024 * 1024 } = {}) {
  try {
    return execFileSync("git", ["-C", repository, ...args], {
      encoding: "utf8",
      env: { ...process.env, GIT_TERMINAL_PROMPT: "0", ...env },
      maxBuffer,
      stdio: ["ignore", "pipe", "pipe"],
    }).trim();
  } catch (error) {
    const detail = error.stderr?.toString().trim() || error.stdout?.toString().trim() || error.message;
    const token = process.env[APP_TOKEN_ENV];
    const sanitized = token ? detail.replaceAll(token, "[redacted]") : detail;
    fail(`Git publication command failed: ${sanitized}`);
  }
}

function gitAuthEnv(token) {
  return {
    GIT_CONFIG_COUNT: "1",
    GIT_CONFIG_KEY_0: "http.extraHeader",
    // GitHub's smart HTTP transport uses the x-access-token username for
    // App installation tokens, even though the REST API uses Bearer syntax.
    GIT_CONFIG_VALUE_0: `Authorization: Basic ${Buffer.from(`x-access-token:${token}`).toString("base64")}`,
  };
}

function gitObjectExists(repository, object) {
  try {
    execFileSync("git", ["-C", repository, "cat-file", "-e", object], {
      stdio: "ignore",
      env: { ...process.env, GIT_TERMINAL_PROMPT: "0" },
    });
    return true;
  } catch {
    return false;
  }
}

function destinationOwnedPaths(exportDir) {
  const policyPath = path.join(exportDir, "scripts", "ci", "public-export-policy.json");
  if (!fs.existsSync(policyPath)) fail(`export is missing the public export policy: ${path.relative(exportDir, policyPath)}`);
  let policy;
  try {
    policy = JSON.parse(fs.readFileSync(policyPath, "utf8"));
  } catch (error) {
    fail(`cannot read public export policy for destination-owned paths: ${error.message}`);
  }
  const paths = [...new Set(policy.destinationOwnedExact ?? [])];
  for (const destinationPath of paths) {
    if (!destinationPath || destinationPath.startsWith("/") || destinationPath.includes("\\") || destinationPath.split("/").some((part) => !part || part === "." || part === "..")) {
      fail(`invalid destination-owned path in public export policy: ${destinationPath}`);
    }
  }
  return paths.sort();
}

function pushProjection({ exportDir, remoteUrl, branch, baseSha, tag, token, destinationPaths = [], destinationOwnedRoot = "" }) {
  const repository = fs.mkdtempSync(path.join(process.env.RUNNER_TEMP || os.tmpdir(), "lunafox-public-export-git-"));
  const message = `chore(export): generated deployment projection ${tag}`;
  try {
    runGit(repository, ["init", "--quiet"]);
    runGit(repository, ["config", "user.name", "LunaFox Public Projection"]);
    runGit(repository, ["config", "user.email", "public-projection@users.noreply.github.com"]);
    runGit(repository, ["remote", "add", "origin", remoteUrl]);
    // Avoid carrying any inherited index state from the temporary repository;
    // the export directory itself is the complete source of the new tree.
    runGit(repository, ["read-tree", "--empty"]);

    // The parent is fetched without downloading its blobs. This keeps the
    // commit connected while avoiding a second copy of the public tree.
    runGit(repository, ["fetch", "--no-tags", "--depth=1", "--filter=blob:none", "origin", baseSha], {
      env: gitAuthEnv(token),
    });
    const fetchedSha = runGit(repository, ["rev-parse", `${baseSha}^{commit}`]);
    if (fetchedSha !== baseSha) fail(`public base commit fetch resolved to ${fetchedSha}, expected ${baseSha}`);
    for (const destinationPath of destinationPaths) {
      if (gitObjectExists(repository, `${baseSha}:${destinationPath}`)) continue;
      if (!destinationOwnedRoot) fail(`public base commit is missing destination-owned path: ${destinationPath}`);
      const sourcePath = path.join(destinationOwnedRoot, ...destinationPath.split("/"));
      if (!fs.existsSync(sourcePath) || !fs.statSync(sourcePath).isFile() || fs.lstatSync(sourcePath).isSymbolicLink()) {
        fail(`destination-owned bootstrap source is missing a regular file: ${destinationPath}`);
      }
      const stagedPath = path.join(exportDir, ...destinationPath.split("/"));
      fs.mkdirSync(path.dirname(stagedPath), { recursive: true });
      fs.copyFileSync(sourcePath, stagedPath);
      fs.chmodSync(stagedPath, fs.statSync(sourcePath).mode & 0o7777);
    }

    // A fresh index makes the commit an exact projection: files absent from
    // the private export cannot survive from the public bootstrap commit.
    runGit(repository, ["--work-tree", exportDir, "add", "--all", "--", "."]);
    // Destination-owned files are intentionally excluded from the private
    // manifest, but must survive the new commit from protected public main.
    // Checkout happens after the projection add so it cannot be removed by
    // the fresh index operation above.
    for (const destinationPath of destinationPaths) {
      if (gitObjectExists(repository, `${baseSha}:${destinationPath}`)) {
        runGit(repository, ["checkout", baseSha, "--", destinationPath]);
      }
    }
    const treeSha = runGit(repository, ["write-tree"]);
    const commitSha = runGit(repository, ["commit-tree", treeSha, "-p", baseSha, "-m", message], {
      env: {
        GIT_AUTHOR_NAME: "LunaFox Public Projection",
        GIT_AUTHOR_EMAIL: "public-projection@users.noreply.github.com",
        GIT_COMMITTER_NAME: "LunaFox Public Projection",
        GIT_COMMITTER_EMAIL: "public-projection@users.noreply.github.com",
      },
    });
    if (!/^[0-9a-f]{40}$/.test(commitSha)) fail(`Git did not produce a valid projection commit: ${commitSha}`);

    // The branch was proven absent immediately before this call. Never use
    // force-push: a race with another publisher must fail instead of rewriting
    // an immutable export branch.
    runGit(repository, ["push", "--no-verify", "origin", `${commitSha}:refs/heads/${branch}`], {
      env: gitAuthEnv(token),
    });
    return commitSha;
  } finally {
    fs.rmSync(repository, { recursive: true, force: true });
  }
}

async function publish(options) {
  const manifest = options.noValidate ? JSON.parse(fs.readFileSync(path.join(options.exportDir, "PUBLIC_EXPORT_MANIFEST.json"), "utf8")) : validateExportTree(options.exportDir);
  const files = walkFiles(options.exportDir);
  const manifestSha256 = crypto.createHash("sha256").update(fs.readFileSync(path.join(options.exportDir, "PUBLIC_EXPORT_MANIFEST.json"))).digest("hex");
  if (!/^[a-f0-9]{64}$/.test(manifestSha256)) fail("export manifest digest could not be computed");
  const bundleSha256 = options.noValidate ? "" : agentBundleSha256(options.exportDir, options.tag);
  const initialBranch = branchName(options.tag, options.attempt);
  const planBase = { repository: options.repo, baseBranch: options.baseBranch, branch: initialBranch, tag: options.tag, fileCount: files.length, manifestSha256, ...(bundleSha256 ? { agentBundleSha256: bundleSha256 } : {}) };
  if (options.dryRun) {
    // Dry-run must still reject an accidentally inherited source/PAT token;
    // this keeps the credential boundary deterministic in local rehearsals.
    rejectFallbackCredentials();
    process.stdout.write(`${JSON.stringify({ ...planBase, dryRun: true }, null, 2)}\n`);
    return planBase;
  }

  const token = await resolveToken(options.apiBase);
  const [owner, repo] = options.repo.split("/");
  const repoInfo = await request(options.apiBase, `/repos/${owner}/${repo}`, { token });
  if (repoInfo.archived || repoInfo.disabled) fail("destination repository is unavailable");
  // GitHub's repository `permissions` summary describes the authenticated
  // principal's repository role. For an installation token on a user-owned
  // repository it may report admin/maintain/triage even when the token was
  // reduced to Contents + Pull requests + Metadata. The installation
  // permission and repository-scope endpoints are the authoritative
  // least-privilege checks for an App token; do not reject on that summary.
  await verifyDestinationInstallation(options.apiBase, token, options.repo);
  const baseRef = await request(options.apiBase, `/repos/${owner}/${repo}/git/ref/heads/${options.baseBranch}`, { token });
  const baseSha = baseRef.object?.sha;
  if (!/^[0-9a-f]{40}$/.test(baseSha ?? "")) fail("destination main ref did not return a commit SHA");

  const existingPublication = await findExistingPublication({
    apiBase: options.apiBase,
    owner,
    repo,
    baseBranch: options.baseBranch,
    tag: options.tag,
    attempt: options.attempt,
    token,
    manifestSha256,
    agentBundleSha256: bundleSha256,
  });
  const branch = existingPublication.branch;
  if (existingPublication.pr) {
    const existingSha = existingPublication.refSha;
    const result = {
      ...planBase,
      branch: existingPublication.branch,
      dryRun: false,
      reused: true,
      publicationState: existingPublication.state,
      commitSha: /^[0-9a-f]{40}$/.test(existingSha ?? "") ? existingSha : null,
      pullRequestNumber: existingPublication.pr.number ?? null,
      pullRequestUrl: existingPublication.pr.html_url ?? null,
    };
    process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
    return result;
  }

  const remoteUrl = options.gitRemote || `https://github.com/${owner}/${repo}.git`;
  const destinationPaths = destinationOwnedPaths(options.exportDir);
  const commitSha = pushProjection({
    exportDir: options.exportDir,
    remoteUrl,
    branch,
    baseSha,
    tag: options.tag,
    token,
    destinationPaths,
    destinationOwnedRoot: options.destinationOwnedRoot,
  });
  const pr = await request(options.apiBase, `/repos/${owner}/${repo}/pulls`, {
    method: "POST", token, body: {
      title: `chore(export): generated deployment projection ${options.tag}`,
      head: branch,
      base: options.baseBranch,
      body: `Generated from the protected private release pipeline for ${options.tag}.\n\nExport manifest SHA-256: ${planBase.manifestSha256}\nAgent bundle SHA-256: ${bundleSha256}\n\nThis projection is generated/read-only; the private repository remains the development authority.`,
      maintainer_can_modify: false,
    },
  });
  const result = { ...planBase, branch, dryRun: false, reused: false, commitSha, pullRequestNumber: pr.number, pullRequestUrl: pr.html_url ?? null };
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
  return result;
}

async function main() { await publish(parseArgs(process.argv.slice(2))); }

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => { process.stderr.write(`public export publication failed: ${error.message}\n`); process.exitCode = 1; });
}

export {
  agentBundleSha256,
  branchName,
  bundleShaFromPublication,
  classifyExistingPublication,
  findExistingPublication,
  destinationOwnedPaths,
  pushProjection,
  manifestShaFromPublication,
  parseArgs,
  publish,
  publicationState,
  rejectFallbackCredentials,
  verifyDestinationInstallation,
};
