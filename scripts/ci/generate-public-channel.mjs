#!/usr/bin/env node

/** Build append-only schema-v3 records for the Compose-first public channel. */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { parseRuntimeComposition } from "./verify-public-release.mjs";
import { validateComposition } from "./resolve-release-component-composition.mjs";
import { canonicalPreheatManifestBytes, validatePreheatManifest } from "./preheat-manifest.mjs";
import { canonicalThirdPartyPolicyBytes, validatePolicyAgainstComposeTemplate } from "./third-party-image-policy.mjs";
import {
  ReleaseCompatibilityProfileAlpha164Bridge,
  assertReleaseCompatibilityProfile,
} from "./release-compatibility-profile.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const FIRST_TAG = "v0.0.1-alpha.57";
const SCHEMA_VERSION = "3";
const RUNTIME_COMPOSITION_ASSET = "runtime-composition.json";
const PREHEAT_MANIFEST_ASSET = "preheat-manifest.json";
const COMPOSE_ASSET = "compose.yaml";
const THIRD_PARTY_POLICY_ASSET = "third-party-image-policy.json";
const LEGACY_RE = /(?:alpha\.46|SCHEMA_VERSION=2|IMAGE_REGISTRY=|IMAGE_NAMESPACE=|WORKER_IMAGE|lunafox-installer|checksums\.txt)/;
const CHANNEL_KEYS = new Set(["SCHEMA_VERSION", "VERSION", "RELEASE_MANIFEST", "RELEASE_MANIFEST_SHA256"]);

function fail(message) { throw new Error(message); }

function usage() {
  return "Usage: node scripts/ci/generate-public-channel.mjs --tag <tag> --channel <canary|stable> --manifest <file> --runtime-composition <file> [--preheat-manifest <file> --compose <file> --third-party-policy <file>] --output-dir <dir> [--release-profile <modern|alpha164-bridge>] [--publication-complete] [--dry-run]\\n";
}

function parseArgs(argv) {
  const args = { tag: "", channel: "", manifest: "", runtimeComposition: "", preheatManifest: "", compose: "", thirdPartyPolicy: "", outputDir: "", releaseProfile: "", publicationComplete: false, dryRun: false };
  const values = new Set(["--tag", "--channel", "--manifest", "--runtime-composition", "--preheat-manifest", "--compose", "--third-party-policy", "--output-dir", "--release-profile"]);
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
    else if (arg === "--preheat-manifest") args.preheatManifest = path.resolve(value);
    else if (arg === "--compose") args.compose = path.resolve(value);
    else if (arg === "--third-party-policy") args.thirdPartyPolicy = path.resolve(value);
    else if (arg === "--output-dir") args.outputDir = path.resolve(value);
    else args.releaseProfile = value;
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

function validateManifest(manifestPath, tag, requestedReleaseProfile = "") {
  if (!fs.existsSync(manifestPath)) fail(`release manifest is missing: ${manifestPath}`);
  const raw = fs.readFileSync(manifestPath, "utf8");
  if (LEGACY_RE.test(raw) || /0\.0\.0-dev/.test(raw)) fail("release manifest contains retired or development content");
  const releaseVersion = raw.match(/^releaseVersion:\s*["']?([^"'\s]+)["']?/m)?.[1] ?? "";
  if (`v${releaseVersion}` !== tag) fail(`release manifest releaseVersion does not match ${tag}`);
  const releaseProfile = assertReleaseCompatibilityProfile(releaseVersion, requestedReleaseProfile);
  const bridge = releaseProfile === ReleaseCompatibilityProfileAlpha164Bridge;
  const hasReleaseNotes = /^releaseNotes:[ \t]*$/m.test(raw);
  const hasRuntimeComposition = /^runtimeComposition:[ \t]*$/m.test(raw);
  if (bridge && (hasReleaseNotes || hasRuntimeComposition)) {
    fail("alpha164-bridge manifest must omit both releaseNotes and runtimeComposition");
  }
  if (!bridge && (!hasReleaseNotes || !hasRuntimeComposition)) {
    fail("modern release manifest must contain releaseNotes and runtimeComposition");
  }
  for (const name of ["server", "frontend", "nginx", "agent", "bootstrap"]) {
    if (!runtimeBlockExists(raw, name)) fail(`release manifest is missing complete runtime image block: ${name}`);
  }
  if (!/^enginePackages:\s*$/m.test(raw)) fail("release manifest is missing enginePackages");
  return { raw, digest: sha256(raw), releaseProfile, runtimeComposition: bridge ? null : parseRuntimeComposition(raw) };
}

function readPreheatManifest(filePath, tag, releaseManifestDigest, compositionDigest) {
  if (!filePath || !fs.existsSync(filePath)) fail(`preheat manifest asset is missing: ${filePath || "(missing)"}`);
  const info = fs.lstatSync(filePath);
  if (!info.isFile() || info.isSymbolicLink()) fail(`preheat manifest asset must be a regular file: ${filePath}`);
  const bytes = fs.readFileSync(filePath);
  let value;
  try { value = JSON.parse(bytes.toString("utf8")); }
  catch (error) { fail(`preheat manifest asset is not valid JSON: ${error.message}`); }
  let normalized;
  try { normalized = validatePreheatManifest(value); }
  catch (error) { fail(`preheat manifest asset is invalid: ${error.message}`); }
  if (!bytes.equals(canonicalPreheatManifestBytes(normalized))) fail("preheat manifest asset must use canonical bytes");
  if (normalized.release.tag !== tag || normalized.release.manifestDigest !== `sha256:${releaseManifestDigest}` || normalized.release.compositionDigest !== compositionDigest) {
    fail("preheat manifest asset is not bound to the release manifest and runtime composition");
  }
  return { bytes, digest: normalized.manifestDigest, manifest: normalized };
}

function readDeploymentAssets(options, preheat) {
  if (!options.compose || !fs.existsSync(options.compose)) fail(`deployment Compose asset is missing: ${options.compose || "(missing)"}`);
  if (!options.thirdPartyPolicy || !fs.existsSync(options.thirdPartyPolicy)) fail(`third-party policy asset is missing: ${options.thirdPartyPolicy || "(missing)"}`);
  const composeInfo = fs.lstatSync(options.compose);
  const policyInfo = fs.lstatSync(options.thirdPartyPolicy);
  if (!composeInfo.isFile() || composeInfo.isSymbolicLink()) fail(`deployment Compose asset must be a regular file: ${options.compose}`);
  if (!policyInfo.isFile() || policyInfo.isSymbolicLink()) fail(`third-party policy asset must be a regular file: ${options.thirdPartyPolicy}`);
  const compose = fs.readFileSync(options.compose);
  const policyBytes = fs.readFileSync(options.thirdPartyPolicy);
  let policy;
  try { policy = JSON.parse(policyBytes.toString("utf8")); }
  catch (error) { fail(`third-party policy asset is not valid JSON: ${error.message}`); }
  try {
    if (!policyBytes.equals(canonicalThirdPartyPolicyBytes(policy))) fail("third-party policy asset must use canonical bytes");
    validatePolicyAgainstComposeTemplate(policy, compose.toString("utf8"));
  } catch (error) {
    fail(`third-party policy asset is invalid: ${error.message}`);
  }
  const digest = (bytes) => `sha256:${sha256(bytes)}`;
  if (preheat.release.composeDigest !== digest(compose) || preheat.release.thirdPartyPolicyDigest !== digest(policyBytes)) {
    fail("deployment assets are not bound to the preheat manifest");
  }
  return { compose, policy: policyBytes, composeDigest: digest(compose), policyDigest: digest(policyBytes) };
}

function requiresPreheatManifest(tag, releaseProfile) {
  // The first public channel record and the registered alpha.183 bridge are
  // immutable historical publications. They predate this asset and must not
  // be rewritten as if they had modern preheat support.
  return releaseProfile !== ReleaseCompatibilityProfileAlpha164Bridge && tag !== FIRST_TAG;
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
  if (expectedDigest && normalized.compositionDigest !== expectedDigest) fail(`runtime composition canonical digest does not match manifest: expected ${expectedDigest}, got ${normalized.compositionDigest}`);
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
        !new RegExp(`^manifests\\/v\\d+\\.\\d+\\.\\d+(?:-(?:alpha|beta|rc)\\.\\d+)?\\/(?:${RUNTIME_COMPOSITION_ASSET}|${PREHEAT_MANIFEST_ASSET}|${COMPOSE_ASSET}|${THIRD_PARTY_POLICY_ASSET})$`).test(name)) {
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
  const manifest = validateManifest(manifestPath, options.tag, options.releaseProfile);
  const compositionPath = options.runtimeComposition || path.join(path.dirname(manifestPath), RUNTIME_COMPOSITION_ASSET);
  const composition = readRuntimeComposition(compositionPath, manifest.runtimeComposition?.sha256 ?? "", manifest.digest, options.tag);
  const needsPreheat = requiresPreheatManifest(options.tag, manifest.releaseProfile);
  if (!needsPreheat && (options.preheatManifest || options.compose || options.thirdPartyPolicy)) {
    fail(`preheat manifest is not supported for historical release profile ${manifest.releaseProfile}`);
  }
  const preheat = needsPreheat
    ? readPreheatManifest(options.preheatManifest, options.tag, manifest.digest, composition.digest)
    : null;
  const deploymentAssets = needsPreheat ? readDeploymentAssets(options, preheat.manifest) : null;
  const existing = readExisting(options.outputDir);
  validateExisting(existing);
  assertFirstInventory(options.tag, existing, options.publicationComplete, options.dryRun);
  if (options.tag === FIRST_TAG && options.channel !== "canary") fail(`first ${FIRST_TAG} release may advance only the canary channel`);

  const versionPath = `channels/${options.tag}.env`;
  const manifestPathInChannel = `manifests/${options.tag}.yaml`;
  const compositionPathInChannel = `manifests/${options.tag}/${RUNTIME_COMPOSITION_ASSET}`;
  const preheatPathInChannel = `manifests/${options.tag}/${PREHEAT_MANIFEST_ASSET}`;
  const composePathInChannel = `manifests/${options.tag}/${COMPOSE_ASSET}`;
  const policyPathInChannel = `manifests/${options.tag}/${THIRD_PARTY_POLICY_ASSET}`;
  const envText = buildVersionEnv(options.tag, manifest.digest);
  const proposed = new Map(existing);
  if (!needsPreheat && proposed.has(preheatPathInChannel)) {
    fail(`historical release must not publish a preheat manifest: ${preheatPathInChannel}`);
  }
  if (proposed.has(versionPath) && !proposed.get(versionPath).equals(Buffer.from(envText))) fail(`immutable channel record already exists with different bytes: ${versionPath}`);
  if (proposed.has(manifestPathInChannel) && !proposed.get(manifestPathInChannel).equals(Buffer.from(manifest.raw))) fail(`immutable manifest record already exists with different bytes: ${manifestPathInChannel}`);
  if (proposed.has(compositionPathInChannel) && !proposed.get(compositionPathInChannel).equals(composition.bytes)) fail(`immutable runtime composition record already exists with different bytes: ${compositionPathInChannel}`);
  if (preheat && proposed.has(preheatPathInChannel) && !proposed.get(preheatPathInChannel).equals(preheat.bytes)) fail(`immutable preheat manifest record already exists with different bytes: ${preheatPathInChannel}`);
  if (deploymentAssets && proposed.has(composePathInChannel) && !proposed.get(composePathInChannel).equals(deploymentAssets.compose)) fail(`immutable deployment Compose record already exists with different bytes: ${composePathInChannel}`);
  if (deploymentAssets && proposed.has(policyPathInChannel) && !proposed.get(policyPathInChannel).equals(deploymentAssets.policy)) fail(`immutable deployment policy record already exists with different bytes: ${policyPathInChannel}`);
  proposed.set(versionPath, Buffer.from(envText));
  proposed.set(manifestPathInChannel, Buffer.from(manifest.raw));
  proposed.set(compositionPathInChannel, composition.bytes);
  if (preheat) proposed.set(preheatPathInChannel, preheat.bytes);
  if (deploymentAssets) {
    proposed.set(composePathInChannel, deploymentAssets.compose);
    proposed.set(policyPathInChannel, deploymentAssets.policy);
  }
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
    releaseProfile: manifest.releaseProfile,
    firstRelease: options.tag === FIRST_TAG,
    publicationComplete: options.publicationComplete,
    files: [...proposed.keys()].sort(),
    manifestSha256: manifest.digest,
    runtimeCompositionSha256: composition.digest,
    preheatManifestSha256: preheat?.digest ?? null,
    deploymentComposeSha256: deploymentAssets?.composeDigest ?? null,
    thirdPartyPolicySha256: deploymentAssets?.policyDigest ?? null,
  };
}

function main() { process.stdout.write(`${JSON.stringify(generate(parseArgs(process.argv.slice(2))), null, 2)}\n`); }

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { process.stderr.write(`public channel generation failed: ${error.message}\n`); process.exitCode = 1; }
}

export { FIRST_TAG, buildVersionEnv, generate, parseEnvText, requiresPreheatManifest, validateExisting, validateManifest, validateVersionEnvText };
