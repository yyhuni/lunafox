#!/usr/bin/env node

/** Validate durable component evidence published with a release. */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { validateComposition } from "./resolve-release-component-composition.mjs";
import { verify as verifyComposition } from "./verify-release-component-composition.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_POLICY = path.join(SCRIPT_DIR, "public-release-policy.json");
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const PUBLIC_SIGNER = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";
const PRIVATE_REPOSITORY = "yyhuni/lunafox-private";
const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const MAX_EVIDENCE_ASSET_BYTES = 8 << 20;

function fail(message) { throw new Error(message); }
function assert(condition, message) { if (!condition) fail(message); }

function parseArgs(argv) {
  const options = { bundle: "", composition: "", manifest: "", policy: "", json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: verify-component-evidence-bundle.mjs --bundle FILE --composition FILE [--manifest FILE] [--policy FILE] [--json]\n");
      process.exit(0);
    }
    if (!["--bundle", "--composition", "--manifest", "--policy"].includes(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--bundle") options.bundle = path.resolve(value);
    else if (arg === "--composition") options.composition = path.resolve(value);
    else if (arg === "--manifest") options.manifest = path.resolve(value);
    else options.policy = path.resolve(value);
  }
  assert(options.bundle, "--bundle is required");
  assert(options.composition, "--composition is required");
  return options;
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function sha256(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function canonical(value) { return JSON.stringify(value); }

function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function assertAllowedKeys(value, keys, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  const unexpected = Object.keys(value).filter((key) => !keys.has(key));
  assert(unexpected.length === 0, `${label} contains unknown fields: ${unexpected.sort().join(", ")}`);
}

function safeRelative(value, label) {
  assert(typeof value === "string", `${label} is required`);
  const normalized = value.replaceAll("\\", "/");
  assert(normalized && !normalized.startsWith("/") && !normalized.split("/").includes(".."), `${label} is unsafe`);
  return normalized;
}

function decodeCanonicalBase64(value, label) {
  assert(typeof value === "string" && value.length > 0, `${label} must be non-empty base64`);
  assert(/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(value), `${label} is not canonical base64`);
  const bytes = Buffer.from(value, "base64");
  assert(bytes.length > 0 && bytes.length <= MAX_EVIDENCE_ASSET_BYTES, `${label} decoded size is invalid`);
  assert(bytes.toString("base64") === value, `${label} is not canonical base64`);
  return bytes;
}

function normalizeAsset(asset, label) {
  assertAllowedKeys(asset, new Set(["ref", "sha256", "mediaType", "byteLength", "contentBase64"]), label);
  const ref = safeRelative(asset.ref, `${label}.ref`);
  assert(!ref.includes("#") && ref.endsWith(".json"), `${label}.ref must be a JSON asset path`);
  assert(typeof asset.sha256 === "string" && DIGEST_RE.test(asset.sha256), `${label}.sha256 is invalid`);
  assert(asset.mediaType === "application/json", `${label}.mediaType must be application/json`);
  assert(Number.isSafeInteger(asset.byteLength) && asset.byteLength > 0 && asset.byteLength <= MAX_EVIDENCE_ASSET_BYTES, `${label}.byteLength is invalid`);
  const bytes = decodeCanonicalBase64(asset.contentBase64, `${label}.contentBase64`);
  assert(bytes.length === asset.byteLength, `${label}.byteLength does not match decoded content`);
  assert(sha256(bytes) === asset.sha256, `${label}.sha256 does not match decoded content`);
  let value;
  try { value = JSON.parse(bytes.toString("utf8")); }
  catch (error) { fail(`${label}.contentBase64 does not contain JSON: ${error.message}`); }
  return { ref, bytes, value };
}

function evidenceReferences(value, label) {
  const references = Array.isArray(value) ? value : [value];
  assert(references.length > 0, `${label} must contain at least one reference`);
  return references.map((reference, index) => {
    assert(typeof reference === "string" && reference.trim() === reference && reference, `${label}[${index}] is invalid`);
    return reference;
  });
}

function parseFragment(reference) {
  assert(typeof reference === "string" && reference, "evidence reference is invalid");
  const value = reference;
  const separator = value.indexOf("#");
  const asset = safeRelative(separator < 0 ? value : value.slice(0, separator), "evidence asset reference");
  assert(!asset.includes("#") && asset.endsWith(".json"), "evidence asset reference must be a JSON asset path");
  return { asset, fragment: separator < 0 ? "" : value.slice(separator + 1) };
}

function pointerValue(value, fragment, label) {
  if (!fragment) return value;
  const pointer = fragment.startsWith("/") ? fragment : `/${fragment}`;
  let current = value;
  for (const raw of pointer.split("/").slice(1)) {
    const key = raw.replaceAll("~1", "/").replaceAll("~0", "~");
    // Engine aggregate evidence uses the same stable four-reference shape as
    // image evidence, while provenance/SBOM/signature are verified by the
    // aggregate's signed records rather than top-level JSON properties.
    if ((key === "provenance" || key === "sbom" || key === "signature") &&
        (current === value || (current && typeof current === "object" && !Object.hasOwn(current, key)))) {
      continue;
    }
    assert(current !== null && current !== undefined && Object.hasOwn(Object(current), key), `${label} fragment does not resolve: #${fragment}`);
    current = current[key];
  }
  return current;
}

function runtimeRecord(component, value, label) {
  assert(value && typeof value === "object" && !Array.isArray(value), `${label} must contain a JSON object`);
  assert(!value.kind || value.kind !== "lunafox.component-promotion-receipt.v1", `${label} must retain original evidence, not a promotion receipt`);
  assert(value.status === "published" && value.sbom === true && value.provenance === true && value.attestation === true, `${label} is missing original Runtime proof fields`);
  assert(value.sourceRepository === PRIVATE_REPOSITORY && value.destinationRepository === PUBLIC_REPOSITORY, `${label} repository binding is invalid`);
  assert(value.workflowIdentity === PUBLIC_SIGNER, `${label} signer workflow identity is invalid`);
  assert(value.component === component.id.slice("runtime.".length), `${label} component identity does not match the composition component`);
  assert(DIGEST_RE.test(value.digest), `${label}.digest is invalid`);
  assert(value.image === component.artifact.ref || value.image === `ghcr.io/yyhuni/${value.component === "agent" ? "lunafox-agent" : `lunafox-${value.component}`}@${value.digest}`, `${label} image does not match its component artifact`);
  assert(value.digest === component.artifact.digest, `${label} digest does not match the composition artifact`);
  assert(value.attestationSubject === value.image, `${label} attestation subject is not bound to the image`);
  assert(value.attestationPredicateType === "https://slsa.dev/provenance/v1", `${label} attestation predicate type is invalid`);
  assert(value.releaseTag === component.sourceRelease.tag, `${label} release tag does not match source release`);
  return value;
}

function engineRecord(component, value, label) {
  assert(value && typeof value === "object" && !Array.isArray(value), `${label} must contain a JSON object`);
  assert(value.passed === true && value.signerIdentity === PUBLIC_SIGNER, `${label} is missing verified Engine publication proof`);
  assert(value.releaseTag === component.sourceRelease.tag, `${label} release tag does not match source release`);
  const engineID = component.id.slice(0, -(`.${component.id.endsWith(".runtime") ? "runtime" : "package"}`).length);
  if (component.id.endsWith(".runtime")) {
    const entry = value.engines?.find((item) => item.engineId === engineID);
    assert(entry && entry.indexDigest === component.artifact.digest && entry.refs?.includes(component.artifact.ref), `${label} Runtime entry does not match the composition artifact`);
  } else {
    const entry = value.packages?.find((item) => item.engineId === engineID);
    assert(entry && entry.artifactManifestDigest === component.artifact.digest, `${label} Package entry does not match the composition artifact (engine ${engineID}, keys ${Object.keys(value).join(",")}, expected ${component.artifact.digest}, got ${entry?.artifactManifestDigest ?? "missing"})`);
    assert(entry.candidates?.some((candidate) => candidate.ref === component.artifact.ref), `${label} Package registry ref does not match the composition artifact (expected ${component.artifact.ref})`);
  }
  return value;
}

// Bundle consumers (including release assembly) must share this strict
// decoder; no caller may fall back to a permissive base64/JSON parser.
function loadBundle(bundleFile) {
  const bundle = readJson(bundleFile, "component evidence bundle");
  assertAllowedKeys(bundle, new Set(["schemaVersion", "kind", "releaseTag", "compositionDigest", "components", "assets"]), "component evidence bundle");
  assert(bundle.schemaVersion === 1 && bundle.kind === "lunafox.component-evidence-bundle.v1", "component evidence bundle schema is invalid");
  assert(Array.isArray(bundle.components) && Array.isArray(bundle.assets), "component evidence bundle inventory is invalid");
  const components = new Map(bundle.components.map((component, index) => {
    assertAllowedKeys(component, new Set(["id", "kind", "disposition", "artifact", "sourceRelease", "evidence"]), `component evidence bundle.components[${index}]`);
    assert(typeof component.id === "string" && component.id, `component evidence bundle.components[${index}].id is invalid`);
    return [component.id, component];
  }));
  const assets = new Map(bundle.assets.map((asset, index) => {
    const normalized = normalizeAsset(asset, `component evidence bundle.assets[${index}]`);
    return [normalized.ref, normalized];
  }));
  assert(components.size === bundle.components.length && assets.size === bundle.assets.length, "component evidence bundle contains duplicate IDs or assets");
  const referencedAssets = new Set();
  for (const component of bundle.components) {
    assert(isPlainObject(component.evidence), `${component.id} evidence is invalid`);
    for (const [proof, value] of Object.entries(component.evidence)) {
      for (const reference of evidenceReferences(value, `${component.id}.evidence.${proof}`)) {
        const parsed = parseFragment(reference);
        const asset = assets.get(parsed.asset);
        assert(asset, `${component.id} ${proof} asset is missing: ${parsed.asset}`);
        referencedAssets.add(parsed.asset);
        pointerValue(asset.value, parsed.fragment, `${component.id} ${proof}`);
      }
    }
  }
  assert(referencedAssets.size === assets.size, "component evidence bundle contains orphan assets");
  return { bundle, components, assets };
}

// Load and fully validate a bundle for callers that need to carry original
// evidence bytes forward. Returning the decoded assets keeps those callers
// from reimplementing base64, digest, and component-chain validation.
function loadVerifiedBundle(options) {
  assert(options && options.bundle && options.composition, "bundle and composition are required");
  const compositionInput = typeof options.composition === "string"
    ? readJson(options.composition, "runtime composition")
    : options.composition;
  const composition = validateComposition(
    compositionInput,
    { requireManifestBinding: Boolean(options.manifest) },
  );
  if (options.manifest) {
    verifyComposition({
      composition: compositionInput,
      manifest: options.manifest,
      policy: options.policy || DEFAULT_POLICY,
    });
  }
  const loaded = loadBundle(typeof options.bundle === "string" ? options.bundle : options.bundleFile);
  const { bundle } = loaded;
  assert(bundle.releaseTag === composition.releaseTag, "component evidence bundle release tag does not match composition");
  assert(bundle.compositionDigest === composition.compositionDigest, "component evidence bundle composition digest does not match composition");
  const { components, assets } = loaded;
  assert(components.size === composition.components.length, "component evidence bundle component inventory is incomplete");
  const referencedAssets = new Set();
  for (const component of composition.components) {
    const bundled = components.get(component.id);
    assert(bundled, `component evidence bundle is missing ${component.id}`);
    assert(bundled.kind === component.kind && bundled.disposition === component.disposition, `${component.id} evidence metadata disposition drifted`);
    assert(canonical(bundled.artifact) === canonical(component.artifact), `${component.id} evidence artifact drifted`);
    assert(canonical(bundled.sourceRelease) === canonical(component.sourceRelease), `${component.id} evidence source release drifted`);
    assert(canonical(bundled.evidence) === canonical(component.evidence), `${component.id} evidence references drifted`);
    for (const [proof, value] of Object.entries(component.evidence)) {
      for (const reference of evidenceReferences(value, `${component.id}.evidence.${proof}`)) {
        const parsed = parseFragment(reference);
        const asset = assets.get(parsed.asset);
        assert(asset, `${component.id} ${proof} asset is missing: ${parsed.asset}`);
        referencedAssets.add(parsed.asset);
        pointerValue(asset.value, parsed.fragment, `${component.id} ${proof}`);
      }
    }
    assert(typeof component.evidence.image === "string", `${component.id} image evidence must be a single original artifact`);
    const imageRef = parseFragment(component.evidence.image);
    const imageAsset = assets.get(imageRef.asset);
    const imageProof = pointerValue(imageAsset.value, imageRef.fragment, `${component.id} image`);
    if (component.kind === "engine") engineRecord(component, imageProof, `${component.id} evidence`);
    else runtimeRecord(component, imageProof, `${component.id} evidence`);
  }
  assert(referencedAssets.size === assets.size, "component evidence bundle contains orphan assets");
  return { bundle, composition, components, assets };
}

function verify(options) {
  const loaded = loadVerifiedBundle({
    ...options,
    compositionFile: options.composition,
    bundleFile: options.bundle,
  });
  const { bundle, composition } = loaded;
  return {
    schemaVersion: 1,
    passed: true,
    releaseTag: composition.releaseTag,
    compositionDigest: composition.compositionDigest,
    componentCount: composition.components.length,
    assetCount: bundle.assets.length,
    reusedComponents: composition.components.filter((component) => component.disposition === "reused").map((component) => component.id),
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `component evidence bundle verified: ${result.componentCount} components`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) { process.stderr.write(`component evidence bundle verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { loadBundle, loadVerifiedBundle, parseArgs, verify };
