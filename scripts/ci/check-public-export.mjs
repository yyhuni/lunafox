#!/usr/bin/env node

/**
 * Secretless verifier executed by the public repository. It rechecks the
 * exact manifest emitted by the private exporter without accessing private
 * source, release credentials, or external artifact builders.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
  buildFileManifest,
  compilePolicy,
  PUBLIC_MANIFEST_PATH,
  scanExportTree,
  validateFreshGitHistory,
} from "./export-public-repository.mjs";
import { validateReleaseNotes } from "./validate-public-release-notes.mjs";
import { validatePublicDocumentation } from "./validate-public-documentation.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_REPO_ROOT = path.resolve(SCRIPT_DIR, "../..");
const AGENT_BUNDLE_FILES = Object.freeze([
  "lunafox-agent-linux-amd64",
  "lunafox-agent-linux-arm64",
  "lunafox-engine-mount-preflight-linux-amd64",
  "lunafox-engine-mount-preflight-linux-arm64",
  "agent-bundle.json",
  "agent-bundle.sha256",
  "agent-bundle.sigstore.json",
]);
const AGENT_ARTIFACT_ID_RE = /^sha256-[a-f0-9]{64}$/;

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  const options = {
    repoRoot: DEFAULT_REPO_ROOT,
    policy: "",
    manifest: "",
    requireAgentBundle: false,
    json: false,
  };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      options.json = true;
      continue;
    }
    if (arg === "--require-agent-bundle") {
      options.requireAgentBundle = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write(usage());
      process.exit(0);
    }
    if (!["--repo-root", "--policy", "--manifest"].includes(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--repo-root") options.repoRoot = path.resolve(value);
    if (arg === "--policy") options.policy = path.resolve(value);
    if (arg === "--manifest") options.manifest = path.resolve(value);
    index += 1;
  }
  if (!options.policy) options.policy = path.join(options.repoRoot, "scripts/ci/public-export-policy.json");
  if (!options.manifest) options.manifest = path.join(options.repoRoot, PUBLIC_MANIFEST_PATH);
  return options;
}

function usage() {
  return "Usage: node scripts/ci/check-public-export.mjs [--repo-root <dir>] [--policy <file>] [--manifest <file>] [--require-agent-bundle] [--json]\\n";
}

function readJson(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    fail(`cannot read ${label} ${filePath}: ${error.message}`);
  }
}

function assertExactKeys(actual, expected, label) {
  const actualKeys = Object.keys(actual ?? {}).sort();
  const expectedKeys = [...expected].sort();
  if (JSON.stringify(actualKeys) !== JSON.stringify(expectedKeys)) {
    fail(`${label} has an invalid field set`);
  }
}

function isAllowed(relPath, compiled) {
  if (compiled.exact.has(relPath)) return true;
  return compiled.prefixes.some((prefix) => relPath.startsWith(prefix));
}

function walk(root) {
  const results = [];
  const visit = (directory, relative = "") => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((left, right) => left.name.localeCompare(right.name))) {
      if (entry.name === ".git") continue;
      const relPath = path.posix.join(relative, entry.name);
      const absolute = path.join(directory, entry.name);
      if (entry.isDirectory()) visit(absolute, relPath);
      else results.push(relPath);
    }
  };
  visit(root);
  return results.sort();
}

function isContainedManifestPath(filePath) {
  if (typeof filePath !== "string" || !filePath || filePath.startsWith("/") || filePath.includes("\0") || filePath.includes("\\")) {
    return false;
  }

  return filePath === path.posix.normalize(filePath) &&
    !filePath.split("/").some((segment) => segment === "." || segment === "..");
}

function validateManifest(manifest, policy) {
  assertExactKeys(manifest, [
    "schemaVersion", "exporterVersion", "sourceRevisionDigest", "releaseTag",
    "generatedReadOnly", "excludedPaths", "files",
  ], "public export manifest");
  if (manifest.schemaVersion !== 1) fail("public export manifest schemaVersion must be 1");
  if (manifest.generatedReadOnly !== true) fail("public export manifest must be generated/read-only");
  if (typeof manifest.exporterVersion !== "string" || !manifest.exporterVersion) fail("public export manifest exporterVersion is required");
  if (typeof manifest.sourceRevisionDigest !== "string" || !/^sha256:[a-f0-9]{64}$/.test(manifest.sourceRevisionDigest)) {
    fail("public export manifest sourceRevisionDigest is invalid");
  }
  if (typeof manifest.releaseTag !== "string" || !/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(manifest.releaseTag)) {
    fail("public export manifest releaseTag is invalid");
  }
  const destinationOwned = [...new Set(policy.destinationOwnedExact ?? [])].sort();
  const expectedExcludedPaths = [PUBLIC_MANIFEST_PATH, ...destinationOwned].sort();
  if (JSON.stringify(manifest.excludedPaths) !== JSON.stringify(expectedExcludedPaths)) {
    fail("public export manifest must exclude only its own path and declared destination-owned paths");
  }
  if (!Array.isArray(manifest.files) || manifest.files.length === 0) fail("public export manifest files must be non-empty");
  let previous = "";
  for (const file of manifest.files) {
    assertExactKeys(file, ["path", "type", "mode", "size", "sha256"], "public export manifest file");
    if (!isContainedManifestPath(file.path) || file.path <= previous) {
      fail("public export manifest paths must be sorted, relative, and contained");
    }
    previous = file.path;
    if (file.type !== "file") fail(`public export manifest has unsupported file type: ${file.path}`);
    if (typeof file.mode !== "string" || !/^[0-7]{4}$/.test(file.mode)) fail(`public export manifest mode is invalid: ${file.path}`);
    if (!Number.isInteger(file.size) || file.size < 0) fail(`public export manifest size is invalid: ${file.path}`);
    if (typeof file.sha256 !== "string" || !/^[a-f0-9]{64}$/.test(file.sha256)) fail(`public export manifest sha256 is invalid: ${file.path}`);
  }
}

function validateImmutableAgentBundle(manifest, repoRoot) {
  const paths = manifest.files.filter((file) => file.path.startsWith("agent/")).map((file) => file.path).sort();
  const artifactIds = [...new Set(paths.map((file) => /^agent\/bin\/(sha256-[a-f0-9]{64})\//.exec(file)?.[1]).filter(Boolean))];
  if (artifactIds.length !== 1 || !AGENT_ARTIFACT_ID_RE.test(artifactIds[0] ?? "")) {
    fail("public export must contain exactly one immutable Agent artifact directory");
  }
  const artifactId = artifactIds[0];
  const prefix = `agent/bin/${artifactId}/`;
  const expected = AGENT_BUNDLE_FILES.map((name) => `${prefix}${name}`).sort();
  if (JSON.stringify(paths) !== JSON.stringify(expected)) {
    fail("public export must contain exactly one immutable seven-file Agent bundle");
  }
  const bundle = readJson(path.join(repoRoot, ...`${prefix}agent-bundle.json`.split("/")), "public Agent bundle manifest");
  if (bundle.schemaVersion !== "lunafox.agent-bundle.v2" || bundle.artifactId !== artifactId ||
      bundle.inputFingerprint?.version !== 1 || bundle.inputFingerprint?.algorithm !== "sha256-canonical-json-v1" ||
      !/^sha256:[a-f0-9]{64}$/.test(bundle.inputFingerprint?.value ?? "") || `sha256-${bundle.inputFingerprint.value.slice("sha256:".length)}` !== artifactId ||
      bundle.publicTreePath !== `agent/bin/${artifactId}`) {
    fail("public Agent bundle manifest is not bound to its immutable artifact identity");
  }
}

function validatePublicExport(options) {
  const repoRoot = fs.realpathSync(options.repoRoot);
  const policy = readJson(options.policy, "public export policy");
  const compiled = compilePolicy(policy);
  const manifest = readJson(options.manifest, "public export manifest");
  validateManifest(manifest, policy);
  const documentation = validatePublicDocumentation({ rootDir: repoRoot });
  const releaseNotes = validateReleaseNotes({ rootDir: repoRoot, tag: manifest.releaseTag });
  if (options.requireAgentBundle) validateImmutableAgentBundle(manifest, repoRoot);

  const actualManifest = buildFileManifest(repoRoot, {
    sourceRevisionDigest: manifest.sourceRevisionDigest,
    releaseTag: manifest.releaseTag,
  }, { excludePaths: manifest.excludedPaths });
  if (JSON.stringify(actualManifest.files) !== JSON.stringify(manifest.files)) {
    fail("public export tree differs from PUBLIC_EXPORT_MANIFEST.json");
  }

  const manifestPaths = new Set(manifest.files.map((file) => file.path));
  if (!manifestPaths.has(releaseNotes.notesPath)) {
    fail(`public export manifest is missing release notes: ${releaseNotes.notesPath}`);
  }
  const destinationOwned = new Set(policy.destinationOwnedExact ?? []);
  for (const relPath of walk(repoRoot)) {
    if (!isAllowed(relPath, compiled)) fail(`public repository contains a non-allowlisted path: ${relPath}`);
    if (compiled.deny.some((pattern) => pattern.test(relPath))) fail(`public repository contains a denied path: ${relPath}`);
    if (relPath !== PUBLIC_MANIFEST_PATH && !manifestPaths.has(relPath) && !destinationOwned.has(relPath)) {
      fail(`public repository path is missing from the exact manifest: ${relPath}`);
    }
  }
  for (const required of policy.requiredPaths ?? []) {
    if (!manifestPaths.has(required)) fail(`public repository is missing required deployment path: ${required}`);
  }
  for (const group of compiled.destinationOwnedGroups) {
    const sentinel = path.join(repoRoot, ...group.sentinel.split("/"));
    const present = group.paths.filter((groupPath) => fs.existsSync(path.join(repoRoot, ...groupPath.split("/"))));
    // The private exporter validates the source-only tree before publication;
    // the destination validates the complete group after projection.
    if (!fs.existsSync(sentinel) && present.length === 0) continue;
    if (!fs.existsSync(sentinel) || present.length !== group.paths.length) {
      fail(`public repository contains a partial destination-owned group: ${group.sentinel}`);
    }
    for (const groupPath of group.paths) {
      if (manifestPaths.has(groupPath)) fail(`destination-owned path must be excluded from the exact manifest: ${groupPath}`);
    }
  }
  for (const marker of policy.generatedMarkers ?? []) {
    const text = fs.readFileSync(path.join(repoRoot, marker), "utf8");
    if (!/GENERATED/i.test(text) || !/READ[- ]ONLY/i.test(text)) fail(`generated/read-only marker is missing from ${marker}`);
  }
  const provenance = readJson(path.join(repoRoot, "PUBLIC_PROVENANCE.json"), "public provenance");
  if (provenance.sourceRevisionDigest !== manifest.sourceRevisionDigest || provenance.releaseTag !== manifest.releaseTag || provenance.generatedReadOnly !== true) {
    fail("PUBLIC_PROVENANCE.json does not match the exact public export manifest");
  }
  const findings = scanExportTree(repoRoot, policy);
  if (findings.length > 0) fail(`public secret scan failed: ${JSON.stringify(findings)}`);
  validateFreshGitHistory(repoRoot, policy.git, manifest.releaseTag, { allowExistingPublicTags: true });

  const result = {
    schemaVersion: 1,
    passed: true,
    releaseTag: manifest.releaseTag,
    sourceRevisionDigest: manifest.sourceRevisionDigest,
    fileCount: manifest.files.length,
    documentation,
  };
  if (options.json) process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
  else process.stdout.write(`public export verified: ${result.fileCount} files (${result.releaseTag})\n`);
  return result;
}

function main() {
  validatePublicExport(parseArgs(process.argv.slice(2)));
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`public export verification failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export { AGENT_ARTIFACT_ID_RE, AGENT_BUNDLE_FILES, validatePublicExport };
