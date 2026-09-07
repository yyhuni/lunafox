#!/usr/bin/env node

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { buildFileManifest } from "./export-public-repository.mjs";

const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const PRIVATE_REPOSITORY = "yyhuni/lunafox-private";
const PRIVATE_SIGNER_ISSUER = "https://token.actions.githubusercontent.com";
const DEFAULT_PRIVATE_SIGNER = "https://github.com/yyhuni/lunafox-private/.github/workflows/release.yml@refs/tags/*";
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const COMMIT_RE = /^[0-9a-f]{40}$/;
const FIXED_MEMBERS = Object.freeze([
  { name: "lunafox-agent-linux-amd64", kind: "agent", platform: "linux/amd64" },
  { name: "lunafox-agent-linux-arm64", kind: "agent", platform: "linux/arm64" },
  { name: "lunafox-engine-mount-preflight-linux-amd64", kind: "mount-preflight", platform: "linux/amd64" },
  { name: "lunafox-engine-mount-preflight-linux-arm64", kind: "mount-preflight", platform: "linux/arm64" },
]);
const FIXED_ASSETS = new Set([...FIXED_MEMBERS.map((item) => item.name), "agent-bundle.json", "agent-bundle.sha256", "agent-bundle.sigstore.json"]);
const PUBLIC_EXPORT_POLICY_PATH = path.join("scripts", "ci", "public-export-policy.json");

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const options = {
    bundleDir: "", publicExportRoot: "", tag: "", sourceRevisionDigest: "", exportManifestSha256: "", publicProvenanceSha256: "",
    signerIdentity: DEFAULT_PRIVATE_SIGNER, allowUnsigned: false, json: false,
  };
  const valueFlags = new Set(["--bundle-dir", "--public-export-root", "--tag", "--source-revision-digest", "--export-manifest-sha256", "--public-provenance-sha256", "--signer-identity"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--allow-unsigned") { options.allowUnsigned = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-agent-binary-bundle.mjs --bundle-dir <dir> --tag <vX.Y.Z> --source-revision-digest <sha256:...> [--export-manifest-sha256 <sha256:...> | --public-export-root <dir>] --public-provenance-sha256 <sha256:...> [--signer-identity <identity>] [--allow-unsigned] [--json]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--bundle-dir") options.bundleDir = path.resolve(value);
    if (arg === "--public-export-root") options.publicExportRoot = path.resolve(value);
    if (arg === "--tag") options.tag = value;
    if (arg === "--source-revision-digest") options.sourceRevisionDigest = value;
    if (arg === "--export-manifest-sha256") options.exportManifestSha256 = value;
    if (arg === "--public-provenance-sha256") options.publicProvenanceSha256 = value;
    if (arg === "--signer-identity") options.signerIdentity = value;
  }
  if (!options.bundleDir) fail("--bundle-dir is required");
  if (!TAG_RE.test(options.tag)) fail(`invalid release tag: ${options.tag}`);
  if (!options.exportManifestSha256 && !options.publicExportRoot) fail("one of --export-manifest-sha256 or --public-export-root is required");
  for (const [value, label] of [[options.sourceRevisionDigest, "source revision digest"], [options.publicProvenanceSha256, "public provenance digest"]]) {
    if (!DIGEST_RE.test(value)) fail(`${label} must be a sha256 digest`);
  }
  if (options.signerIdentity !== DEFAULT_PRIVATE_SIGNER) fail("signer identity must be the private release workflow tag identity");
  return options;
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function sha256File(file) {
  return `sha256:${crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex")}`;
}

function requireDigest(value, label) { if (!DIGEST_RE.test(String(value ?? ""))) fail(`${label} must be an immutable sha256 digest`); }

function assertRegularMember(file, name) {
  if (!fs.existsSync(file) || !fs.statSync(file).isFile()) fail(`missing bundle member: ${name}`);
  const stat = fs.statSync(file);
  if ((stat.mode & 0o111) === 0) fail(`bundle member is not executable: ${name}`);
  const header = fs.readFileSync(file).subarray(0, 4);
  if (header.length !== 4 || header.toString("hex") !== "7f454c46") fail(`bundle member is not an ELF executable: ${name}`);
}

function readDestinationOwnedPaths(publicExportRoot) {
  const policyPath = path.join(publicExportRoot, PUBLIC_EXPORT_POLICY_PATH);
  let policy;
  try {
    policy = JSON.parse(fs.readFileSync(policyPath, "utf8"));
  } catch (error) {
    fail(`cannot read public export policy for Agent bundle verification: ${error.message}`);
  }
  const paths = [...new Set(policy.destinationOwnedExact ?? [])].sort();
  for (const relativePath of paths) {
    if (!relativePath || relativePath.startsWith("/") || relativePath.includes("\\") || relativePath.split("/").some((part) => !part || part === "." || part === "..")) {
      fail(`invalid destination-owned path in public export policy: ${relativePath}`);
    }
  }
  return paths;
}

function deriveBaseExportManifestSha256(publicExportRoot, tag, sourceRevisionDigest) {
  if (!fs.existsSync(publicExportRoot) || !fs.statSync(publicExportRoot).isDirectory()) fail(`public export root is missing: ${publicExportRoot}`);
  const destinationOwned = readDestinationOwnedPaths(publicExportRoot);
  const excludedPaths = ["PUBLIC_EXPORT_MANIFEST.json", ...destinationOwned].sort();
  const manifest = buildFileManifest(publicExportRoot, { sourceRevisionDigest, releaseTag: tag }, { excludePaths: excludedPaths });
  const prefix = `agent/bin/${tag}/`;
  manifest.files = manifest.files.filter((file) => !file.path.startsWith(prefix));
  const serialized = `${JSON.stringify({ ...manifest, excludedPaths }, null, 2)}\n`;
  return `sha256:${crypto.createHash("sha256").update(serialized).digest("hex")}`;
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
  if (manifest.schemaVersion !== "lunafox.agent-bundle.v1") fail("unsupported Agent bundle schema");
  if (manifest.releaseTag !== options.tag || manifest.version !== options.tag.slice(1)) fail("bundle release tag/version mismatch");
  if (manifest.sourceRepository !== PRIVATE_REPOSITORY || manifest.destinationRepository !== PUBLIC_REPOSITORY) fail("bundle repository identity mismatch");
  if (manifest.sourceRevisionDigest !== options.sourceRevisionDigest) fail("bundle source revision digest mismatch");
  const expectedExportManifestSha256 = options.exportManifestSha256 || deriveBaseExportManifestSha256(options.publicExportRoot, options.tag, options.sourceRevisionDigest);
  if (manifest.publicExportManifestSha256 !== expectedExportManifestSha256) fail("bundle public export manifest digest mismatch");
  if (manifest.publicProvenanceSha256 !== options.publicProvenanceSha256) fail("bundle public provenance digest mismatch");
  if (!COMMIT_RE.test(manifest.sourceRevision ?? "")) fail("bundle source revision is invalid");
  if (JSON.stringify(manifest.platforms) !== JSON.stringify(["linux/amd64", "linux/arm64"])) fail("bundle platform set is invalid");
  if (manifest.publicTreePath !== `agent/bin/${options.tag}`) fail("bundle public tree path is not versioned by the release tag");
  if (manifest.signerIssuer !== PRIVATE_SIGNER_ISSUER || manifest.signerIdentity !== options.signerIdentity) fail("bundle signer identity mismatch");
  if (!Array.isArray(manifest.members) || manifest.members.length !== FIXED_MEMBERS.length) fail("bundle must contain exactly four members");
  const manifestNames = manifest.members.map((item) => item?.name);
  if (new Set(manifestNames).size !== manifestNames.length) fail("duplicate member in manifest");
  if (manifestNames.some((name) => !FIXED_MEMBERS.some((item) => item.name === name))) fail("manifest contains an unexpected member");
  const checksums = parseChecksums(path.join(root, "agent-bundle.sha256"));
  for (const name of checksums.keys()) {
    if (!FIXED_MEMBERS.some((item) => item.name === name)) fail(`checksum list contains an unexpected member: ${name}`);
  }
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
  if (sigstore.schemaVersion !== "lunafox.agent-bundle.sigstore.v1" || sigstore.subject !== "agent-bundle.json" || sigstore.signerIssuer !== PRIVATE_SIGNER_ISSUER || sigstore.signerIdentity !== options.signerIdentity) fail("Sigstore bundle identity mismatch");
  if (sigstore.signed !== true && !options.allowUnsigned) fail("Agent bundle is not signed");
  if (sigstore.manifestSha256 !== sha256File(path.join(root, "agent-bundle.json"))) fail("Sigstore bundle does not bind agent-bundle.json");
  return { schemaVersion: 1, passed: true, tag: options.tag, sourceRevision: manifest.sourceRevision, publicTreePath: manifest.publicTreePath, members };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : "Agent binary bundle verified"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { process.stderr.write(`Agent binary bundle verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { FIXED_MEMBERS, deriveBaseExportManifestSha256, parseArgs, verify };
