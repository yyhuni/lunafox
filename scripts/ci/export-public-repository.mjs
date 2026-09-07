#!/usr/bin/env node

/**
 * Build a fresh-history public deployment projection from a frozen Git
 * revision.  This script deliberately has no npm dependencies so the private
 * release job can run it before a destination credential is requested.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";

const EXPORTER_VERSION = "public-exporter/v1";
const PUBLIC_MANIFEST_PATH = "PUBLIC_EXPORT_MANIFEST.json";
const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_REPO_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DEFAULT_POLICY = path.join(SCRIPT_DIR, "public-export-policy.json");

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  const options = {
    sourceRoot: DEFAULT_REPO_ROOT,
    output: "",
    policy: DEFAULT_POLICY,
    revision: "refs/heads/main",
    tag: "v0.0.1-alpha.57",
    overlayDir: "",
    manifestOut: "",
    privateEvidenceOut: "",
    dryRun: false,
    json: false,
    keepTemp: false,
  };

  const valueFlags = new Set([
    "--source-root", "--output", "--policy", "--revision", "--tag",
    "--overlay-dir", "--manifest-out", "--private-evidence-out",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--dry-run") {
      options.dryRun = true;
      continue;
    }
    if (arg === "--json") {
      options.json = true;
      continue;
    }
    if (arg === "--keep-temp") {
      options.keepTemp = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write(usage());
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--source-root") options.sourceRoot = path.resolve(value);
    else if (arg === "--output") options.output = path.resolve(value);
    else if (arg === "--policy") options.policy = path.resolve(value);
    else if (arg === "--revision") options.revision = value;
    else if (arg === "--tag") options.tag = value;
    else if (arg === "--overlay-dir") options.overlayDir = path.resolve(value);
    else if (arg === "--manifest-out") options.manifestOut = path.resolve(value);
    else if (arg === "--private-evidence-out") options.privateEvidenceOut = path.resolve(value);
    index += 1;
  }

  if (!options.output && !options.dryRun) fail("--output is required unless --dry-run is used");
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(options.tag)) {
    fail(`invalid release tag: ${options.tag}`);
  }
  return options;
}

function usage() {
  return `Usage: node scripts/ci/export-public-repository.mjs [options]\n\n` +
    `  --source-root <dir>          frozen private Git repository (default: repo root)\n` +
    `  --revision <ref>             allowed ref to freeze (default: refs/heads/main)\n` +
    `  --tag <tag>                  public release tag (default: v0.0.1-alpha.57)\n` +
    `  --output <dir>               fresh public tree destination\n` +
    `  --overlay-dir <dir>          generated channels/manifests to add\n` +
    `  --manifest-out <file>        exact exported-file manifest\n` +
    `  --private-evidence-out <f>   full private provenance (never in public tree)\n` +
    `  --dry-run                    validate and report without creating Git history\n` +
    `  --json                       print the result as JSON\n`;
}

function readJson(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    fail(`cannot read ${label} ${filePath}: ${error.message}`);
  }
}

function runGit(repoRoot, args, options = {}) {
  try {
    return execFileSync("git", ["-C", repoRoot, ...args], {
      encoding: "utf8",
      stdio: options.stdio ?? ["ignore", "pipe", "pipe"],
      maxBuffer: 16 * 1024 * 1024,
    });
  } catch (error) {
    const stderr = error.stderr ? String(error.stderr).trim() : "";
    fail(`git ${args.join(" ")} failed${stderr ? `: ${stderr}` : ""}`);
  }
}

function sha256(value) {
  return crypto.createHash("sha256").update(value).digest("hex");
}

function normalizeRelative(value) {
  const normalized = value.split(path.sep).join("/");
  if (!normalized || normalized === "." || normalized.startsWith("../") || normalized.includes("/../") || path.posix.isAbsolute(normalized)) {
    fail(`unsafe relative path: ${value}`);
  }
  return normalized;
}

function globToRegExp(glob) {
  let source = "^";
  for (let index = 0; index < glob.length; index += 1) {
    const char = glob[index];
    if (char === "*") {
      if (glob[index + 1] === "*") {
        source += ".*";
        index += 1;
      } else {
        source += "[^/]*";
      }
    } else if (char === "?") {
      source += "[^/]";
    } else {
      source += char.replace(/[\\^$+?.()|{}\[\]]/g, "\\$&");
    }
  }
  return new RegExp(`${source}$`);
}

function compilePolicy(policy) {
  if (policy?.schemaVersion !== 1) fail("public export policy schemaVersion must be 1");
  if (policy.sourceRepository !== "yyhuni/lunafox-private") fail("policy sourceRepository must be yyhuni/lunafox-private");
  if (policy.destinationRepository !== "yyhuni/lunafox") fail("policy destinationRepository must be yyhuni/lunafox");
  validateReviewedSecretExceptions(policy);
  validateApprovedMaintenanceCommits(policy.git);
  const allow = policy.allowlist ?? {};
  const exact = new Set([...(allow.exact ?? []), ...(allow.generatedExact ?? [])]);
  const prefixes = [...(allow.prefixes ?? []), ...(allow.generatedPrefixes ?? [])];
  const deny = (policy.denylist ?? []).map((pattern) => new RegExp(pattern));
  const destinationOwned = new Set(policy.destinationOwnedExact ?? []);
  for (const destinationPath of destinationOwned) {
    if (!exact.has(destinationPath)) fail(`destination-owned path must be allowlisted: ${destinationPath}`);
    if (deny.some((pattern) => pattern.test(destinationPath))) fail(`destination-owned path is denied: ${destinationPath}`);
  }
  return { policy, exact, prefixes, deny, destinationOwned };
}

function isAllowed(relPath, compiled, { includeGenerated = true } = {}) {
  if (compiled.exact.has(relPath)) return true;
  const generatedPrefixes = compiled.policy.allowlist?.generatedPrefixes ?? [];
  if (!includeGenerated && generatedPrefixes.some((prefix) => relPath.startsWith(prefix))) return false;
  return compiled.prefixes.some((prefix) => relPath.startsWith(prefix));
}

function isDenied(relPath, compiled) {
  return compiled.deny.some((pattern) => pattern.test(relPath));
}

function listGitTree(repoRoot, revision) {
  const raw = runGit(repoRoot, ["ls-tree", "-r", "-z", "--full-tree", revision]);
  const entries = [];
  for (const record of raw.split("\0")) {
    if (!record) continue;
    const tab = record.indexOf("\t");
    if (tab < 0) fail("malformed git ls-tree record");
    const [mode, type, object] = record.slice(0, tab).split(" ");
    const relPath = normalizeRelative(record.slice(tab + 1));
    entries.push({ mode, type, object, path: relPath });
  }
  return entries;
}

function resolveRevision(repoRoot, requested) {
  if (!requested || /[\0\n]/.test(requested) || requested.startsWith("-") || requested.includes("..")) {
    fail(`unsafe Git revision: ${requested}`);
  }
  const commit = runGit(repoRoot, ["rev-parse", "--verify", `${requested}^{commit}`]).trim();
  if (!/^[0-9a-f]{40}$/.test(commit)) fail(`revision is not a commit: ${requested}`);
  return commit;
}

function assertAllowedRef(requested, policy, tag) {
  const sourceRefs = policy.sourceRefs ?? {};
  const allowedRefs = new Set(sourceRefs.allowed ?? []);
  const tagPattern = new RegExp(sourceRefs.tagPattern ?? "^$");
  const expectedTagRef = `refs/tags/${tag}`;

  if (!tagPattern.test(expectedTagRef)) {
    fail(`release tag is not allowed by policy: ${tag}`);
  }
  if (requested === expectedTagRef) return;
  if (requested.startsWith("refs/tags/")) {
    fail(`source tag must match release tag: expected ${expectedTagRef}, got ${requested}`);
  }
  if (!allowedRefs.has(requested)) {
    fail(`source ref must be an explicitly allowed branch or the release tag: ${requested}`);
  }
}

function readGitBlob(repoRoot, revision, relPath) {
  try {
    return execFileSync("git", ["-C", repoRoot, "show", `${revision}:${relPath}`], {
      encoding: "buffer",
      maxBuffer: 128 * 1024 * 1024,
    });
  } catch (error) {
    fail(`cannot read frozen source path ${relPath}: ${error.message}`);
  }
}

function modeFromGit(mode) {
  const numeric = Number.parseInt(mode, 8);
  if (!Number.isInteger(numeric)) fail(`invalid Git mode: ${mode}`);
  return numeric & 0o111 ? (numeric & 0o7777) : (numeric & 0o7777);
}

function ensureContainedSymlink(root, destination, target) {
  const resolved = path.resolve(path.dirname(destination), target);
  const rootResolved = path.resolve(root) + path.sep;
  if (resolved !== path.resolve(root) && !resolved.startsWith(rootResolved)) {
    fail(`symlink escapes export root: ${path.relative(root, destination)} -> ${target}`);
  }
}

function walkFilesystem(root) {
  const result = [];
  const visit = (directory, relative = "") => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((left, right) => left.name.localeCompare(right.name))) {
      if (entry.name === ".git") continue;
      const absolute = path.join(directory, entry.name);
      const relPath = normalizeRelative(path.posix.join(relative.split(path.sep).join("/"), entry.name));
      if (entry.isDirectory()) {
        visit(absolute, relPath);
      } else {
        result.push({ path: relPath, absolute, type: entry.isSymbolicLink() ? "symlink" : "file" });
      }
    }
  };
  visit(root);
  return result;
}

function validateReviewedSecretExceptions(policy) {
  const exceptions = policy.secretScan?.reviewedExceptions ?? [];
  if (!Array.isArray(exceptions)) fail("secret scan reviewedExceptions must be an array");
  const seen = new Set();
  for (const exception of exceptions) {
    if (!exception || typeof exception !== "object" ||
      JSON.stringify(Object.keys(exception).sort()) !== JSON.stringify(["match", "path", "reason", "rule"])) {
      fail("secret scan exception must declare only path, rule, match, and reason");
    }
    if (typeof exception.path !== "string" || !exception.path ||
      typeof exception.rule !== "string" || !exception.rule ||
      typeof exception.match !== "string" || !exception.match ||
      typeof exception.reason !== "string" || !exception.reason.trim()) {
      fail("secret scan exception fields must be non-empty strings");
    }
    normalizeRelative(exception.path);
    const key = `${exception.path}\u0000${exception.rule}\u0000${exception.match}`;
    if (seen.has(key)) fail(`duplicate secret scan exception: ${exception.path}`);
    seen.add(key);
  }
}

function validateApprovedMaintenanceCommits(policy) {
  const commits = policy?.approvedMaintenanceCommits ?? [];
  if (!Array.isArray(commits)) fail("git approvedMaintenanceCommits must be an array");
  const seen = new Set();
  const expectedKeys = [
    "authorEmail",
    "authorName",
    "committerEmail",
    "committerName",
    "hash",
    "parentCount",
    "subject",
  ];
  for (const commit of commits) {
    if (!commit || typeof commit !== "object" ||
      JSON.stringify(Object.keys(commit).sort()) !== JSON.stringify(expectedKeys)) {
      fail("approved maintenance commit must declare only hash, subject, author, committer, and parentCount");
    }
    if (!/^[0-9a-f]{40}$/.test(commit.hash)) {
      fail(`approved maintenance commit hash is invalid: ${commit.hash}`);
    }
    if (seen.has(commit.hash)) fail(`duplicate approved maintenance commit: ${commit.hash}`);
    seen.add(commit.hash);
    for (const field of ["subject", "authorName", "authorEmail", "committerName", "committerEmail"]) {
      if (typeof commit[field] !== "string" || !commit[field]) {
        fail(`approved maintenance commit ${field} must be a non-empty string`);
      }
    }
    if (!Number.isInteger(commit.parentCount) || commit.parentCount < 0) {
      fail(`approved maintenance commit parentCount is invalid: ${commit.hash}`);
    }
  }
  return commits;
}

function readException(policy, relPath, ruleId, match) {
  // An approved fixture must match the complete scanned token. Path- or
  // pattern-level exemptions could hide a newly introduced credential.
  return (policy.secretScan?.reviewedExceptions ?? []).some((exception) =>
    exception.path === relPath && exception.rule === ruleId && exception.match === match
  );
}

const SECRET_RULES = [
  { id: "private-key", regex: /-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----/g },
  { id: "github-token", regex: /\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\b/g },
  { id: "cloud-credential", regex: /\b(?:AKIA|ASIA)[A-Z0-9]{16}\b|\bxox[baprs]-[A-Za-z0-9-]{16,}\b/gi },
  { id: "generic-password-assignment", regex: /\b(?:password|passwd|secret|api[_-]?key)\s*[:=]\s*["']?([A-Za-z0-9/+_=-]{20,})/gi },
  { id: "bearer-token-literal", regex: /\bBearer\s+[A-Za-z0-9._~+/=-]{24,}\b/g },
];

function scanText(relPath, text, policy) {
  const findings = [];
  for (const rule of SECRET_RULES) {
    rule.regex.lastIndex = 0;
    for (const match of text.matchAll(rule.regex)) {
      const value = match[0];
      if (/change[-_ ]?me|replace[-_ ]?me|example|your[-_ ]|<[^>]+>|\$\{|REDACTED|placeholder/i.test(value)) continue;
      if (!readException(policy, relPath, rule.id, value)) {
        findings.push({ path: relPath, rule: rule.id, match: value.slice(0, 80) });
      }
    }
  }
  return findings;
}

function scanExportTree(root, policy) {
  const findings = [];
  for (const entry of walkFilesystem(root)) {
    if (entry.type === "symlink") {
      const target = fs.readlinkSync(entry.absolute);
      ensureContainedSymlink(root, entry.absolute, target);
      if (policy.symlinks?.allow !== true) findings.push({ path: entry.path, rule: "symlink", match: target });
      continue;
    }
    const bytes = fs.readFileSync(entry.absolute);
    const text = bytes.includes(0) ? "" : bytes.toString("utf8");
    if (text) findings.push(...scanText(entry.path, text, policy));
  }
  return findings;
}

function copyEntry(outputRoot, relPath, bytes, mode, type = "file") {
  const destination = path.join(outputRoot, ...relPath.split("/"));
  fs.mkdirSync(path.dirname(destination), { recursive: true, mode: 0o755 });
  if (type === "symlink") {
    const target = bytes.toString("utf8");
    ensureContainedSymlink(outputRoot, destination, target);
    fs.symlinkSync(target, destination);
  } else {
    fs.writeFileSync(destination, bytes, { mode });
    fs.chmodSync(destination, mode);
  }
}

function copyFrozenFiles(repoRoot, revision, entries, outputRoot, compiled) {
  const copied = new Set();
  for (const entry of entries) {
    // The public repository owns its workflow runner definition. It is
    // validated as an allowlisted destination path, but the private snapshot
    // must never overwrite it.
    if (compiled.destinationOwned.has(entry.path)) continue;
    // Source deny rules are explicit exclusions: tracked build output and
    // private build contexts must be omitted from the projection, even when a
    // broad approved source prefix (such as frontend/) contains them.
    if (isDenied(entry.path, compiled)) {
      if (compiled.exact.has(entry.path)) fail(`allowlist/denylist conflict for source path: ${entry.path}`);
      continue;
    }
    if (!isAllowed(entry.path, compiled, { includeGenerated: false })) continue;
    if (entry.type !== "blob") fail(`unsupported Git tree entry (submodules are not exportable): ${entry.path}`);
    if (entry.mode === "120000") {
      const target = readGitBlob(repoRoot, revision, entry.path);
      if (compiled.policy.symlinks?.allow !== true) fail(`symlink is not allowed in public export: ${entry.path}`);
      copyEntry(outputRoot, entry.path, target, 0o777, "symlink");
    } else {
      copyEntry(outputRoot, entry.path, readGitBlob(repoRoot, revision, entry.path), modeFromGit(entry.mode));
    }
    copied.add(entry.path);
  }
  return copied;
}

function copyOverlayFiles(overlayRoot, outputRoot, compiled) {
  if (!overlayRoot) return new Set();
  if (!fs.existsSync(overlayRoot) || !fs.statSync(overlayRoot).isDirectory()) fail(`overlay directory is missing: ${overlayRoot}`);
  const copied = new Set();
  for (const entry of walkFilesystem(overlayRoot)) {
    if (compiled.destinationOwned.has(entry.path)) fail(`overlay cannot overwrite destination-owned path: ${entry.path}`);
    if (!isAllowed(entry.path, compiled)) fail(`overlay path is outside the generated allowlist: ${entry.path}`);
    if (isDenied(entry.path, compiled)) fail(`overlay path is denied: ${entry.path}`);
    if (entry.type === "symlink") fail(`overlay symlink is not allowed: ${entry.path}`);
    const stat = fs.statSync(entry.absolute);
    copyEntry(outputRoot, entry.path, fs.readFileSync(entry.absolute), stat.mode & 0o777);
    copied.add(entry.path);
  }
  return copied;
}

function writePublicProvenance(outputRoot, policy, revision, tag) {
  const sourceDigest = sha256(revision);
  const payload = {
    schemaVersion: 1,
    generatedReadOnly: true,
    exporterVersion: EXPORTER_VERSION,
    sourceRepository: policy.sourceRepository,
    destinationRepository: policy.destinationRepository,
    releaseTag: tag,
    sourceRevisionDigest: `sha256:${sourceDigest}`,
    privateProvenance: "retained in the protected private release evidence; not present in public Git",
  };
  const destination = path.join(outputRoot, "PUBLIC_PROVENANCE.json");
  fs.writeFileSync(destination, `${JSON.stringify(payload, null, 2)}\n`, { mode: 0o644 });
  fs.chmodSync(destination, 0o644);
  return payload;
}

function buildFileManifest(root, metadata, { excludePaths = [] } = {}) {
  const excluded = new Set(excludePaths);
  const files = [];
  for (const entry of walkFilesystem(root)) {
    if (excluded.has(entry.path)) continue;
    const stat = fs.lstatSync(entry.absolute);
    if (entry.type === "symlink") {
      files.push({
        path: entry.path,
        type: "symlink",
        mode: (stat.mode & 0o7777).toString(8).padStart(4, "0"),
        size: Buffer.byteLength(fs.readlinkSync(entry.absolute)),
        sha256: sha256(fs.readlinkSync(entry.absolute)),
      });
      continue;
    }
    const bytes = fs.readFileSync(entry.absolute);
    files.push({
      path: entry.path,
      type: "file",
      mode: (stat.mode & 0o7777).toString(8).padStart(4, "0"),
      size: bytes.length,
      sha256: sha256(bytes),
    });
  }
  // Use a locale-independent bytewise order so every verifier (including
  // shell/JSON consumers) observes the same deterministic manifest ordering.
  files.sort((left, right) => (left.path < right.path ? -1 : left.path > right.path ? 1 : 0));
  return {
    schemaVersion: 1,
    exporterVersion: EXPORTER_VERSION,
    sourceRevisionDigest: metadata.sourceRevisionDigest,
    releaseTag: metadata.releaseTag,
    generatedReadOnly: true,
    excludedPaths: [...excluded].sort(),
    files,
  };
}

function validateTree(root, compiled, manifest, policy, { requireGit = false } = {}) {
  const entries = walkFilesystem(root);
  const paths = new Set(entries.map((entry) => entry.path));
  for (const required of policy.requiredPaths ?? []) {
    if (!paths.has(required)) fail(`public export is missing required path: ${required}`);
  }
  if (manifest) {
    const expected = JSON.stringify(manifest.files);
    const actual = JSON.stringify(buildFileManifest(root, {
      sourceRevisionDigest: manifest.sourceRevisionDigest,
      releaseTag: manifest.releaseTag,
    }, { excludePaths: manifest.excludedPaths ?? [] }).files);
    if (expected !== actual) fail("exported tree does not match the exact file manifest");
  }
  for (const entry of entries) {
    if (entry.path === ".git" || entry.path.startsWith(".git/")) fail("public export contains Git metadata");
    if (!isAllowed(entry.path, compiled)) fail(`public export contains a non-allowlisted path: ${entry.path}`);
    if (isDenied(entry.path, compiled)) fail(`public export contains denied path: ${entry.path}`);
    if (entry.type === "symlink") {
      ensureContainedSymlink(root, entry.absolute, fs.readlinkSync(entry.absolute));
      if (policy.symlinks?.allow !== true) fail(`public export contains a symlink: ${entry.path}`);
    }
  }
  for (const marker of policy.generatedMarkers ?? []) {
    const markerPath = path.join(root, marker);
    if (!fs.existsSync(markerPath)) fail(`generated marker is missing: ${marker}`);
    const text = fs.readFileSync(markerPath, "utf8");
    if (!/GENERATED/i.test(text) || !/READ[- ]ONLY/i.test(text)) fail(`generated marker is incomplete: ${marker}`);
  }
  const findings = scanExportTree(root, policy);
  if (findings.length > 0) fail(`secret scan failed: ${JSON.stringify(findings)}`);
  if (requireGit) validateFreshGitHistory(root, compiled.policy.git, manifest?.releaseTag);
}

function historyRecordKind(record, policy) {
  const {
    hash,
    parents,
    authorName,
    authorEmail,
    committerName,
    committerEmail,
    subject,
  } = record;
  const parentList = parents ? parents.trim().split(/\s+/).filter(Boolean) : [];
  const approvedMaintenance = (policy.approvedMaintenanceCommits ?? []).find((commit) => commit.hash === hash);
  if (approvedMaintenance) {
    const metadataMatches = approvedMaintenance.subject === subject &&
      approvedMaintenance.authorName === authorName &&
      approvedMaintenance.authorEmail === authorEmail &&
      approvedMaintenance.committerName === committerName &&
      approvedMaintenance.committerEmail === committerEmail &&
      approvedMaintenance.parentCount === parentList.length;
    if (!metadataMatches) {
      fail(`approved maintenance commit metadata does not match policy: ${hash}`);
    }
    return { kind: "maintenance", tag: "" };
  }
  const generatedPattern = new RegExp(policy.publicCommitMessagePattern);
  const squashPattern = new RegExp(
    policy.publicSquashCommitMessagePattern ??
      `${String(policy.publicCommitMessagePattern).replace(/\$$/, "")} \\(#[0-9]+\\)$`,
  );
  const bootstrapPattern = new RegExp(
    policy.bootstrapCommitMessagePattern ??
      "^chore\\(bootstrap\\): initialize public deployment projection$",
  );
  const mergePattern = new RegExp(
    policy.publicMergeCommitMessagePattern ??
      "^Merge pull request #[0-9]+ from [A-Za-z0-9_.-]+/export/v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[A-Za-z0-9.-]+)?(?:-retry-[0-9]+)?$",
  );
  const generatedTagPattern = /^chore\(export\): generated deployment projection (v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?)$/;
  const squashTagPattern = /^chore\(export\): generated deployment projection (v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?) \(#\d+\)$/;
  const mergePartsPattern = /^Merge pull request #\d+ from ([A-Za-z0-9_.-]+)\/export\/(v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?)$/;

  if (bootstrapPattern.test(subject)) {
    if (parentList.length !== 0) {
      fail(`bootstrap commit must be a root commit: ${hash}`);
    }
    return { kind: "bootstrap", tag: "" };
  }

  if (mergePattern.test(subject)) {
    if (parentList.length !== 2) {
      fail(`public GitHub merge commit must have exactly two parents: ${hash}`);
    }
    // GitHub creates this commit after review. Its actor metadata is not the
    // projection identity, so the exact merge subject/parent shape is the
    // controlled boundary; generated commits below still require both fixed
    // author and committer identities.
    if (!authorName || !authorEmail || !committerName || !committerEmail) {
      fail(`public merge commit metadata is incomplete: ${hash}`);
    }
    const mergeParts = subject.match(mergePartsPattern);
    const mergeOwner = mergeParts?.[1] ?? "";
    if (policy.publicMergeHeadOwner && mergeOwner !== policy.publicMergeHeadOwner) {
      fail(`public merge commit head owner is not authorized: ${mergeOwner}`);
    }
    const branchTag = mergeParts?.[2] ?? "";
    return { kind: "merge", tag: branchTag.replace(/-retry-\d+$/, "") };
  }

  if (squashPattern.test(subject)) {
    if (parentList.length !== 1) fail(`public squash commit must have exactly one parent: ${hash}`);
    if (!authorName || !authorEmail || !committerName || !committerEmail) {
      fail(`public squash commit metadata is incomplete: ${hash}`);
    }
    return { kind: "squash", tag: subject.match(squashTagPattern)?.[1] ?? "" };
  }

  if (!generatedPattern.test(subject)) return { kind: "unknown", tag: "" };
  if (authorName !== policy.publicAuthorName || authorEmail !== policy.publicAuthorEmail ||
      committerName !== policy.publicAuthorName || committerEmail !== policy.publicAuthorEmail) {
    fail("public generated commit metadata is not the sanitized projection identity");
  }
  return { kind: "generated", tag: subject.match(generatedTagPattern)?.[1] ?? "" };
}

function validateFreshGitHistory(root, policy, expectedTag = "") {
  const log = runGit(root, [
    "log",
    "--format=%H%x00%P%x00%an%x00%ae%x00%cn%x00%ce%x00%s",
  ]);
  const records = log.split("\n").filter(Boolean).map((line) => {
    const [hash, parents, authorName, authorEmail, committerName, committerEmail, subject] = line.split("\0");
    return { hash, parents, authorName, authorEmail, committerName, committerEmail, subject };
  });
  if (records.length === 0) fail("public export history is empty");
  const forbiddenPattern = new RegExp(
    policy.forbiddenCommitMessagePattern ??
      "(?:alpha\\.46|0\\.0\\.0-dev|SCHEMA_VERSION=2|IMAGE_REGISTRY=|IMAGE_NAMESPACE=|WORKER_IMAGE)",
  );
  let bootstrapCount = 0;
  let rootCount = 0;
  for (const [index, record] of records.entries()) {
    if (forbiddenPattern.test(record.subject ?? "")) {
      fail(`public history contains a retired or development identity: ${record.subject}`);
    }
    const classification = historyRecordKind(record, policy);
    if (classification.kind === "unknown") {
      fail(`public history contains an unapproved commit message: ${record.subject}`);
    }
    if (classification.kind === "bootstrap") {
      bootstrapCount += 1;
      // The bootstrap is a manually/administratively controlled root. It may
      // carry the maintainer's GitHub identity, but it cannot hide a private
      // parent history.
    }
    const parentCount = record.parents ? record.parents.trim().split(/\s+/).filter(Boolean).length : 0;
    if (parentCount === 0) rootCount += 1;
    if (index === 0 && expectedTag && classification.kind !== "bootstrap" && classification.tag !== expectedTag) {
      fail(`public history HEAD does not describe the exported release tag: ${expectedTag}`);
    }
  }
  if (bootstrapCount > 1) fail("public history contains multiple bootstrap roots");
  if (rootCount !== 1) fail(`public history must contain exactly one root commit, found ${rootCount}`);
  const refs = runGit(root, ["for-each-ref", "--format=%(refname)"]).trim().split("\n").filter(Boolean);
  for (const ref of refs) {
    if (ref.startsWith("refs/tags/") || ref.startsWith("refs/heads/private/")) {
      fail(`public export contains private ref: ${ref}`);
    }
  }
}

function writeAtomic(filePath, content, mode = 0o644) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const temp = path.join(path.dirname(filePath), `.${path.basename(filePath)}.${process.pid}.tmp`);
  fs.writeFileSync(temp, content, { mode });
  fs.chmodSync(temp, mode);
  fs.renameSync(temp, filePath);
}

function createFreshHistory(outputRoot, gitPolicy, tag) {
  runGit(outputRoot, ["init", "--initial-branch=main"], { stdio: "ignore" });
  runGit(outputRoot, ["config", "user.name", gitPolicy.publicAuthorName]);
  runGit(outputRoot, ["config", "user.email", gitPolicy.publicAuthorEmail]);
  runGit(outputRoot, ["add", "--all"]);
  runGit(outputRoot, ["commit", "--no-verify", "-m", `chore(export): generated deployment projection ${tag}`], { stdio: "ignore" });
  // A projection must not carry source tags, remotes, or branch metadata.
  const remotes = runGit(outputRoot, ["remote"]).trim();
  if (remotes) runGit(outputRoot, ["remote", "remove", "origin"], { stdio: "ignore" });
}

function writePrivateEvidence(filePath, payload) {
  if (!filePath) return;
  writeAtomic(filePath, `${JSON.stringify(payload, null, 2)}\n`, 0o600);
}

function exportProjection(options) {
  const sourceRoot = fs.realpathSync(options.sourceRoot);
  const policy = readJson(options.policy, "public export policy");
  const compiled = compilePolicy(policy);
  if (!fs.existsSync(path.join(sourceRoot, ".git"))) fail(`source root is not a Git repository: ${sourceRoot}`);
  assertAllowedRef(options.revision, policy, options.tag);
  const revision = resolveRevision(sourceRoot, options.revision);
  const entries = listGitTree(sourceRoot, revision);

  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-public-export-"));
  let outputRoot = tempRoot;
  try {
    const copiedSource = copyFrozenFiles(sourceRoot, revision, entries, outputRoot, compiled);
    const copiedOverlay = copyOverlayFiles(options.overlayDir, outputRoot, compiled);
    if (options.tag === policy.sourceRefs?.firstPublicTag) {
      for (const required of policy.allowlist?.firstReleaseRequired ?? []) {
        if (!copiedSource.has(required) && !copiedOverlay.has(required)) fail(`first release is missing generated path: ${required}`);
      }
    }
    const provenance = writePublicProvenance(outputRoot, policy, revision, options.tag);
    const manifest = buildFileManifest(outputRoot, provenance, {
      excludePaths: [PUBLIC_MANIFEST_PATH, ...compiled.destinationOwned],
    });
    writeAtomic(path.join(outputRoot, PUBLIC_MANIFEST_PATH), `${JSON.stringify(manifest, null, 2)}\n`, 0o644);
    validateTree(outputRoot, compiled, manifest, policy);

    const manifestText = `${JSON.stringify(manifest, null, 2)}\n`;
    const manifestOut = options.manifestOut || (options.output ? path.join(path.dirname(options.output), `${path.basename(options.output)}.manifest.json`) : "");
    if (manifestOut) writeAtomic(manifestOut, manifestText, 0o644);

    const privateEvidence = {
      schemaVersion: 1,
      exporterVersion: EXPORTER_VERSION,
      sourceRepository: policy.sourceRepository,
      destinationRepository: policy.destinationRepository,
      releaseTag: options.tag,
      sourceRevision: revision,
      sourceRevisionDigest: provenance.sourceRevisionDigest,
      exportManifestSha256: sha256(manifestText),
      exportedFiles: manifest.files,
      secretScan: { passed: true, reviewedExceptions: policy.secretScan?.reviewedExceptions ?? [] },
    };
    writePrivateEvidence(options.privateEvidenceOut, privateEvidence);

    if (!options.dryRun) {
      if (fs.existsSync(options.output)) {
        const existing = fs.readdirSync(options.output);
        if (existing.length > 0) fail(`output directory must be empty or absent: ${options.output}`);
        fs.rmSync(options.output, { recursive: true, force: true });
      }
      fs.mkdirSync(options.output, { recursive: true });
      // Copy the validated tree without its temporary filesystem metadata.
      for (const entry of walkFilesystem(outputRoot)) {
        const destination = path.join(options.output, ...entry.path.split("/"));
        fs.mkdirSync(path.dirname(destination), { recursive: true });
        if (entry.type === "symlink") fs.symlinkSync(fs.readlinkSync(entry.absolute), destination);
        else {
          fs.copyFileSync(entry.absolute, destination);
          fs.chmodSync(destination, fs.lstatSync(entry.absolute).mode & 0o7777);
        }
      }
      createFreshHistory(options.output, policy.git, options.tag);
      validateFreshGitHistory(options.output, policy.git, options.tag);
      validateTree(options.output, compiled, null, policy, { requireGit: true });
    }

    const result = {
      schemaVersion: 1,
      exporterVersion: EXPORTER_VERSION,
      dryRun: options.dryRun,
      sourceRevisionDigest: provenance.sourceRevisionDigest,
      releaseTag: options.tag,
      fileCount: manifest.files.length,
      manifestPath: manifestOut || null,
      privateEvidencePath: options.privateEvidenceOut || null,
      output: options.dryRun ? null : options.output,
      freshHistory: !options.dryRun,
    };
    if (options.json) process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
    else process.stdout.write(`public export ${options.dryRun ? "validated" : "created"}: ${manifest.files.length} files (${options.tag})\n`);
    return result;
  } finally {
    if (!options.keepTemp) fs.rmSync(tempRoot, { recursive: true, force: true });
  }
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  exportProjection(options);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`public export failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export {
  EXPORTER_VERSION,
  buildFileManifest,
  compilePolicy,
  exportProjection,
  PUBLIC_MANIFEST_PATH,
  scanExportTree,
  validateFreshGitHistory,
};
