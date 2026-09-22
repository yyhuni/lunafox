#!/usr/bin/env node

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const PRIVATE_REPOSITORY = "yyhuni/lunafox-private";
const PRIVATE_SIGNER_ISSUER = "https://token.actions.githubusercontent.com";
const DEFAULT_PRIVATE_SIGNER = "https://github.com/yyhuni/lunafox-private/.github/workflows/release.yml@refs/tags/*";
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const ARTIFACT_ID_RE = /^sha256-[a-f0-9]{64}$/;
const RELEASE_TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const COMMIT_RE = /^[0-9a-f]{40}$/;
const FIXED_MEMBERS = Object.freeze([
  { name: "lunafox-agent-linux-amd64", kind: "agent", platform: "linux/amd64" },
  { name: "lunafox-agent-linux-arm64", kind: "agent", platform: "linux/arm64" },
  { name: "lunafox-engine-mount-preflight-linux-amd64", kind: "mount-preflight", platform: "linux/amd64" },
  { name: "lunafox-engine-mount-preflight-linux-arm64", kind: "mount-preflight", platform: "linux/arm64" },
]);
const FIXED_ASSETS = new Set([...FIXED_MEMBERS.map((item) => item.name), "agent-bundle.json", "agent-bundle.sha256", "agent-bundle.sigstore.json"]);

function fail(message) {
  throw new Error(message);
}

function assert(condition, message) {
  if (!condition) fail(message);
}

function assertExactKeys(value, keys, label) {
  assert(value && typeof value === "object" && !Array.isArray(value), `${label} must be an object`);
  const actual = Object.keys(value).sort();
  const expected = [...keys].sort();
  assert(JSON.stringify(actual) === JSON.stringify(expected), `${label} has unknown or missing fields`);
}

function parseArgs(argv) {
  const options = {
    bundleDir: "", artifactId: "", inputFingerprint: "", sourceReleaseTag: "",
    signerIdentity: DEFAULT_PRIVATE_SIGNER, allowUnsigned: false, json: false,
  };
  const valueFlags = new Set(["--bundle-dir", "--artifact-id", "--input-fingerprint", "--source-release-tag", "--signer-identity"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      options.json = true;
      continue;
    }
    if (arg === "--allow-unsigned") {
      options.allowUnsigned = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-agent-binary-bundle.mjs --bundle-dir <dir> --artifact-id <sha256-...> --input-fingerprint <sha256:...> [--source-release-tag <vX.Y.Z>] [--signer-identity <identity>] [--allow-unsigned] [--json]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--bundle-dir") options.bundleDir = path.resolve(value);
    else if (arg === "--artifact-id") options.artifactId = value;
    else if (arg === "--input-fingerprint") options.inputFingerprint = value;
    else if (arg === "--source-release-tag") options.sourceReleaseTag = value;
    else options.signerIdentity = value;
  }
  if (!options.bundleDir) fail("--bundle-dir is required");
  if (!ARTIFACT_ID_RE.test(options.artifactId)) fail("--artifact-id must be sha256- followed by a lowercase digest");
  if (!DIGEST_RE.test(options.inputFingerprint)) fail("--input-fingerprint must be a sha256 digest");
  if (options.artifactId !== `sha256-${options.inputFingerprint.slice("sha256:".length)}`) fail("--artifact-id must derive from --input-fingerprint");
  if (options.sourceReleaseTag && !RELEASE_TAG_RE.test(options.sourceReleaseTag)) fail("--source-release-tag is invalid");
  if (options.signerIdentity !== DEFAULT_PRIVATE_SIGNER) fail("signer identity must be the private release workflow tag identity");
  return options;
}

function readJson(file, label) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    fail(`cannot read ${label}: ${error.message}`);
  }
}

function sha256File(file) {
  return `sha256:${crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex")}`;
}

function requireDigest(value, label) {
  if (!DIGEST_RE.test(String(value ?? ""))) fail(`${label} must be an immutable sha256 digest`);
}

function assertRegularMember(file, name) {
  if (!fs.existsSync(file) || !fs.statSync(file).isFile()) fail(`missing bundle member: ${name}`);
  const stat = fs.statSync(file);
  if ((stat.mode & 0o111) === 0) fail(`bundle member is not executable: ${name}`);
  const header = fs.readFileSync(file).subarray(0, 4);
  if (header.length !== 4 || header.toString("hex") !== "7f454c46") fail(`bundle member is not an ELF executable: ${name}`);
}

function parseChecksums(file) {
  if (!fs.existsSync(file)) fail("agent-bundle.sha256 is missing");
  const lines = fs.readFileSync(file, "utf8").trimEnd().split(/\r?\n/).filter(Boolean);
  if (lines.length !== FIXED_MEMBERS.length) fail("agent-bundle.sha256 must contain exactly four entries");
  const entries = new Map();
  for (const line of lines) {
    const match = /^(sha256:[a-f0-9]{64}|[a-f0-9]{64})  (.+)$/.exec(line);
    if (!match) fail(`invalid checksum entry: ${line}`);
    const digest = match[1].startsWith("sha256:") ? match[1] : `sha256:${match[1]}`;
    if (entries.has(match[2])) fail(`duplicate checksum entry: ${match[2]}`);
    entries.set(match[2], digest);
  }
  return entries;
}

function verify(options) {
  const root = options.bundleDir;
  if (!fs.existsSync(root) || !fs.statSync(root).isDirectory()) fail(`bundle directory is missing: ${root}`);
  const names = fs.readdirSync(root).filter((name) => !name.startsWith("."));
  const unexpected = names.filter((name) => !FIXED_ASSETS.has(name));
  if (unexpected.length) fail(`bundle contains unexpected assets: ${unexpected.join(", ")}`);
  for (const asset of FIXED_ASSETS) if (!names.includes(asset)) fail(`bundle is missing fixed asset: ${asset}`);

  const manifest = readJson(path.join(root, "agent-bundle.json"), "agent-bundle.json");
  assertExactKeys(manifest, [
    "artifactId", "destinationRepository", "inputFingerprint", "members", "platforms", "publicTreePath",
    "schemaVersion", "signerIdentity", "signerIssuer", "sourceRelease", "sourceRepository", "sourceRevision", "sourceRevisionDigest",
  ], "Agent bundle manifest");
  if (manifest.schemaVersion !== "lunafox.agent-bundle.v2") fail("unsupported Agent bundle schema");
  if (manifest.artifactId !== options.artifactId) fail("bundle artifact identity mismatch");
  assertExactKeys(manifest.inputFingerprint, ["algorithm", "value", "version"], "Agent bundle inputFingerprint");
  if (manifest.inputFingerprint.version !== 1 || manifest.inputFingerprint.algorithm !== "sha256-canonical-json-v1" || manifest.inputFingerprint.value !== options.inputFingerprint) {
    fail("bundle input fingerprint mismatch");
  }
  if (manifest.artifactId !== `sha256-${manifest.inputFingerprint.value.slice("sha256:".length)}`) fail("bundle artifact identity does not derive from its input fingerprint");
  assertExactKeys(manifest.sourceRelease, ["tag"], "Agent bundle sourceRelease");
  if (!RELEASE_TAG_RE.test(manifest.sourceRelease.tag)) fail("bundle source release tag is invalid");
  if (options.sourceReleaseTag && manifest.sourceRelease.tag !== options.sourceReleaseTag) fail("bundle source release tag mismatch");
  if (manifest.sourceRepository !== PRIVATE_REPOSITORY || manifest.destinationRepository !== PUBLIC_REPOSITORY) fail("bundle repository identity mismatch");
  if (!COMMIT_RE.test(manifest.sourceRevision ?? "")) fail("bundle source revision is invalid");
  requireDigest(manifest.sourceRevisionDigest, "bundle source revision digest");
  if (JSON.stringify(manifest.platforms) !== JSON.stringify(["linux/amd64", "linux/arm64"])) fail("bundle platform set is invalid");
  if (manifest.publicTreePath !== `agent/bin/${options.artifactId}`) fail("bundle public tree path is not bound to immutable artifact identity");
  if (manifest.signerIssuer !== PRIVATE_SIGNER_ISSUER || manifest.signerIdentity !== options.signerIdentity) fail("bundle signer identity mismatch");
  if (!Array.isArray(manifest.members) || manifest.members.length !== FIXED_MEMBERS.length) fail("bundle must contain exactly four members");
  const manifestNames = manifest.members.map((item) => item?.name);
  if (new Set(manifestNames).size !== manifestNames.length) fail("duplicate member in manifest");
  if (manifestNames.some((name) => !FIXED_MEMBERS.some((item) => item.name === name))) fail("manifest contains an unexpected member");

  const checksums = parseChecksums(path.join(root, "agent-bundle.sha256"));
  for (const name of checksums.keys()) if (!FIXED_MEMBERS.some((item) => item.name === name)) fail(`checksum list contains an unexpected member: ${name}`);
  const seenNames = new Set();
  const seenPlatforms = new Set();
  const members = [];
  for (const expected of FIXED_MEMBERS) {
    const record = manifest.members.find((item) => item?.name === expected.name);
    if (!record) fail(`manifest is missing member: ${expected.name}`);
    if (seenNames.has(record.name)) fail(`duplicate member: ${record.name}`);
    seenNames.add(record.name);
    if (record.kind !== expected.kind || record.platform !== expected.platform || record.path !== expected.name) fail(`member provenance mismatch: ${expected.name}`);
    if (seenPlatforms.has(record.platform) && record.kind === "agent") fail(`duplicate Agent platform: ${record.platform}`);
    if (record.kind === "agent") seenPlatforms.add(record.platform);
    requireDigest(record.sha256, `${expected.name}.sha256`);
    const file = path.join(root, expected.name);
    assertRegularMember(file, expected.name);
    const actual = sha256File(file);
    if (actual !== record.sha256) fail(`member digest mismatch: ${expected.name}`);
    if (checksums.get(expected.name) !== actual) fail(`checksum list mismatch: ${expected.name}`);
    if (record.size !== fs.statSync(file).size) fail(`member size mismatch: ${expected.name}`);
    members.push({ name: expected.name, platform: expected.platform, sha256: actual });
  }
  if (seenNames.size !== FIXED_MEMBERS.length || checksums.size !== FIXED_MEMBERS.length) fail("bundle has duplicate or extra members");

  const sigstore = readJson(path.join(root, "agent-bundle.sigstore.json"), "agent-bundle.sigstore.json");
  if (sigstore.schemaVersion !== "lunafox.agent-bundle.sigstore.v1" || sigstore.subject !== "agent-bundle.json" || sigstore.signerIssuer !== PRIVATE_SIGNER_ISSUER || sigstore.signerIdentity !== options.signerIdentity) {
    fail("Sigstore bundle identity mismatch");
  }
  if (sigstore.signed !== true && !options.allowUnsigned) fail("Agent bundle is not signed");
  if (sigstore.manifestSha256 !== sha256File(path.join(root, "agent-bundle.json"))) fail("Sigstore bundle does not bind agent-bundle.json");
  return {
    schemaVersion: 2,
    passed: true,
    artifactId: manifest.artifactId,
    inputFingerprint: manifest.inputFingerprint.value,
    sourceRelease: manifest.sourceRelease,
    sourceRevision: manifest.sourceRevision,
    publicTreePath: manifest.publicTreePath,
    members,
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : "Agent binary bundle verified"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`Agent binary bundle verification failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export { FIXED_MEMBERS, parseArgs, verify };
