#!/usr/bin/env node

/** Build append-only schema-v3 records for the Compose-first public channel. */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { parseRuntimeComposition } from "./verify-public-release.mjs";
import { validateComposition } from "./resolve-release-component-composition.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const FIRST_TAG = "v0.0.1-alpha.57";
const SCHEMA_VERSION = "3";
const RUNTIME_COMPOSITION_ASSET = "runtime-composition.json";
const LEGACY_RE = /(?:alpha\.46|SCHEMA_VERSION=2|IMAGE_REGISTRY=|IMAGE_NAMESPACE=|WORKER_IMAGE|lunafox-installer|checksums\.txt)/;
const CHANNEL_KEYS = new Set(["SCHEMA_VERSION", "VERSION", "RELEASE_MANIFEST", "RELEASE_MANIFEST_SHA256"]);

function fail(message) { throw new Error(message); }

function usage() {
  return "Usage: node scripts/ci/generate-public-channel.mjs --tag <tag> --channel <canary|stable> --manifest <file> --runtime-composition <file> --output-dir <dir> [--publication-complete] [--dry-run]\\n";
}

function parseArgs(argv) {
  const args = { tag: "", channel: "", manifest: "", runtimeComposition: "", outputDir: "", publicationComplete: false, dryRun: false };
  const values = new Set(["--tag", "--channel", "--manifest", "--runtime-composition", "--output-dir"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--publication-complete") { args.publicationComplete = true; continue; }
    if (arg === "--dry-run") { args.dryRun = true; continue; }
    if (arg === "--help" || arg === "-h") { process.stdout.write(usage()); process.exit(0); }
    if (!values.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--tag") args.tag = value;
    else if (arg === "--channel") args.channel = value;
    else if (arg === "--manifest") args.manifest = path.resolve(value);
    else if (arg === "--runtime-composition") args.runtimeComposition = path.resolve(value);
    else args.outputDir = path.resolve(value);
  }
  if (!/^v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?$/.test(args.tag)) fail(`invalid release tag: ${args.tag || "(missing)"}`);
  if (!/^(?:canary|stable)$/.test(args.channel)) fail(`unsupported channel: ${args.channel}`);
  if (!args.manifest) fail("--manifest is required");
  if (!args.runtimeComposition) fail("--runtime-composition is required");
  if (!args.outputDir && !args.dryRun) fail("--output-dir is required unless --dry-run is used");
  return args;
}

function sha256(bytes) { return crypto.createHash("sha256").update(bytes).digest("hex"); }

function runtimeBlockExists(text, name) {
  return new RegExp(`^  - name: ${name}\\n    refs:\\n(?:      - [^\\n]+\\n){2}`, "m").test(text);
}

function validateManifest(manifestPath, tag) {
  if (!fs.existsSync(manifestPath)) fail(`release manifest is missing: ${manifestPath}`);
  const raw = fs.readFileSync(manifestPath, "utf8");
  if (LEGACY_RE.test(raw) || /0\.0\.0-dev/.test(raw)) fail("release manifest contains retired or development content");
  const releaseVersion = raw.match(/^releaseVersion:\s*["']?([^"'\s]+)["']?/m)?.[1] ?? "";
  if (`v${releaseVersion}` !== tag) fail(`release manifest releaseVersion does not match ${tag}`);
  for (const name of ["server", "frontend", "nginx", "agent", "bootstrap"]) {
    if (!runtimeBlockExists(raw, name)) fail(`release manifest is missing complete runtime image block: ${name}`);
  }
  if (!/^enginePackages:\s*$/m.test(raw)) fail("release manifest is missing enginePackages");
  return { raw, digest: sha256(raw), runtimeComposition: parseRuntimeComposition(raw) };
}

function readRuntimeComposition(filePath, expectedDigest, expectedManifestDigest, expectedTag) {
  if (!filePath || !fs.existsSync(filePath)) fail(`runtime composition asset is missing: ${filePath || "(missing)"}`);
  const info = fs.lstatSync(filePath);
  if (!info.isFile() || info.isSymbolicLink()) fail(`runtime composition asset must be a regular file: ${filePath}`);
  const bytes = fs.readFileSync(filePath);
  if (!bytes.length) fail(`runtime composition asset is empty: ${filePath}`);
  let value;
  try { value = JSON.parse(bytes.toString("utf8")); }
  catch (error) { fail(`runtime composition asset is not valid JSON: ${error.message}`); }
  let normalized;
  try { normalized = validateComposition(value, { requireManifestBinding: true }); }
  catch (error) { fail(`runtime composition asset is invalid: ${error.message}`); }
  if (String(normalized.releaseTag).replace(/^v/, "") !== expectedTag.replace(/^v/, "")) fail("runtime composition asset release tag does not match manifest");
  if (normalized.compositionDigest !== expectedDigest) fail(`runtime composition canonical digest does not match manifest: expected ${expectedDigest}, got ${normalized.compositionDigest}`);
  if (normalized.manifestBinding.manifestDigest !== `sha256:${expectedManifestDigest}`) fail("runtime composition manifest binding does not match manifest bytes");
  return { bytes, digest: normalized.compositionDigest, rawSha256: sha256(bytes) };
}

function parseEnvText(text, label) {
  const values = new Map();
  for (const line of text.split(/\r?\n/)) {
    if (!line || line.startsWith("#")) continue;
    const split = line.indexOf("=");
    if (split <= 0 || values.has(line.slice(0, split))) fail(`${label} has duplicate/invalid environment key`);
    values.set(line.slice(0, split), line.slice(split + 1));
  }
  return values;
}

function validateVersionEnvText(text, expectedVersion, label) {
  const values = parseEnvText(text, label);
  if (values.get("SCHEMA_VERSION") !== SCHEMA_VERSION) fail(`${label} must use schema v3`);
  if (values.get("VERSION") !== expectedVersion) fail(`${label} VERSION must be ${expectedVersion}`);
  if (values.get("RELEASE_MANIFEST") !== `manifests/${expectedVersion}.yaml`) fail(`${label} has mismatched RELEASE_MANIFEST`);
  if (!/^[a-f0-9]{64}$/.test(values.get("RELEASE_MANIFEST_SHA256") ?? "")) fail(`${label} has invalid RELEASE_MANIFEST_SHA256`);
  for (const key of values.keys()) if (!CHANNEL_KEYS.has(key)) fail(`${label} contains unsupported key ${key}`);
  return values;
}

function buildVersionEnv(tag, manifestDigest) {
  return [
    `SCHEMA_VERSION=${SCHEMA_VERSION}`,
    `VERSION=${tag}`,
    `RELEASE_MANIFEST=manifests/${tag}.yaml`,
    `RELEASE_MANIFEST_SHA256=${manifestDigest}`,
    "",
  ].join("\n");
}

function readExisting(outputDir) {
  const files = new Map();
  if (!outputDir || !fs.existsSync(outputDir)) return files;
  const visit = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const full = path.join(directory, entry.name);
      if (entry.isDirectory()) visit(full);
      else if (entry.isFile()) files.set(path.relative(outputDir, full).split(path.sep).join("/"), fs.readFileSync(full));
    }
  };
  visit(outputDir);
  return files;
}

function validateExisting(files) {
  for (const [name, bytes] of files) {
    if (!/^(?:channels|manifests)\//.test(name)) fail(`channel tree contains unauthorized path: ${name}`);
    if (name.startsWith("manifests/") &&
        !/^manifests\/v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?\.yaml$/.test(name) &&
        !new RegExp(`^manifests\\/v\\d+\\.\\d+\\.\\d+(?:-(?:alpha|beta|rc)\\.\\d+)?\\/${RUNTIME_COMPOSITION_ASSET}$`).test(name)) {
      fail(`channel tree contains unauthorized manifest asset: ${name}`);
    }
    const text = bytes.toString("utf8");
    if (LEGACY_RE.test(text) || /0\.0\.0-dev/.test(text)) fail(`channel tree contains retired/development content: ${name}`);
    if (name.startsWith("channels/") && name.endsWith(".env")) {
      const version = parseEnvText(text, name).get("VERSION") ?? "";
      validateVersionEnvText(text, version, name);
    }
  }
}

function assertFirstInventory(tag, files, publicationComplete, dryRun) {
  if (tag !== FIRST_TAG) return;
  if (!publicationComplete && !dryRun) fail(`first ${FIRST_TAG} channel publication requires --publication-complete after all gates`);
  const expected = new Set([
    `channels/${FIRST_TAG}.env`,
    "channels/canary.env",
    `manifests/${FIRST_TAG}.yaml`,
    `manifests/${FIRST_TAG}/${RUNTIME_COMPOSITION_ASSET}`,
  ]);
  for (const name of files.keys()) if (!expected.has(name)) fail(`first channel must contain only alpha.57 files; found ${name}`);
}

function generate(options) {
  const manifestPath = fs.realpathSync(options.manifest);
  const manifest = validateManifest(manifestPath, options.tag);
  const compositionPath = options.runtimeComposition || path.join(path.dirname(manifestPath), RUNTIME_COMPOSITION_ASSET);
  const composition = readRuntimeComposition(compositionPath, manifest.runtimeComposition.sha256, manifest.digest, options.tag);
  const existing = readExisting(options.outputDir);
  validateExisting(existing);
  assertFirstInventory(options.tag, existing, options.publicationComplete, options.dryRun);
  if (options.tag === FIRST_TAG && options.channel !== "canary") fail(`first ${FIRST_TAG} release may advance only the canary channel`);

  const versionPath = `channels/${options.tag}.env`;
  const manifestPathInChannel = `manifests/${options.tag}.yaml`;
  const compositionPathInChannel = `manifests/${options.tag}/${RUNTIME_COMPOSITION_ASSET}`;
  const envText = buildVersionEnv(options.tag, manifest.digest);
  const proposed = new Map(existing);
  if (proposed.has(versionPath) && !proposed.get(versionPath).equals(Buffer.from(envText))) fail(`immutable channel record already exists with different bytes: ${versionPath}`);
  if (proposed.has(manifestPathInChannel) && !proposed.get(manifestPathInChannel).equals(Buffer.from(manifest.raw))) fail(`immutable manifest record already exists with different bytes: ${manifestPathInChannel}`);
  if (proposed.has(compositionPathInChannel) && !proposed.get(compositionPathInChannel).equals(composition.bytes)) fail(`immutable runtime composition record already exists with different bytes: ${compositionPathInChannel}`);
  proposed.set(versionPath, Buffer.from(envText));
  proposed.set(manifestPathInChannel, Buffer.from(manifest.raw));
  proposed.set(compositionPathInChannel, composition.bytes);
  if (options.publicationComplete) proposed.set(`channels/${options.channel}.env`, Buffer.from(envText));
  if (options.tag === FIRST_TAG && options.dryRun && !proposed.has("channels/canary.env")) proposed.set("channels/canary.env", Buffer.from(envText));
  if (options.tag === FIRST_TAG) {
    const expected = new Set([versionPath, "channels/canary.env", manifestPathInChannel, compositionPathInChannel]);
    for (const name of proposed.keys()) if (!expected.has(name)) fail(`first channel contains unexpected file: ${name}`);
    if (proposed.has("channels/stable.env")) fail("stable.env is forbidden for the first canary");
    if (!proposed.has("channels/canary.env")) fail("first canary alias is missing");
  }
  if (!options.dryRun) {
    for (const [name, bytes] of proposed) {
      const destination = path.join(options.outputDir, ...name.split("/"));
      fs.mkdirSync(path.dirname(destination), { recursive: true });
      const temporary = `${destination}.${process.pid}.tmp`;
      fs.writeFileSync(temporary, bytes, { mode: 0o644 });
      fs.renameSync(temporary, destination);
    }
  }
  return {
    schemaVersion: 3,
    tag: options.tag,
    channel: options.channel,
    firstRelease: options.tag === FIRST_TAG,
    publicationComplete: options.publicationComplete,
    files: [...proposed.keys()].sort(),
    manifestSha256: manifest.digest,
    runtimeCompositionSha256: composition.digest,
  };
}

function main() { process.stdout.write(`${JSON.stringify(generate(parseArgs(process.argv.slice(2))), null, 2)}\n`); }

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { process.stderr.write(`public channel generation failed: ${error.message}\n`); process.exitCode = 1; }
}

export { FIRST_TAG, buildVersionEnv, generate, parseEnvText, validateExisting, validateManifest, validateVersionEnvText };
