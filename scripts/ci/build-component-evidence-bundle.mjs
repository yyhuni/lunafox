#!/usr/bin/env node

/**
 * Build the durable release asset that carries original component evidence.
 *
 * The bundle stores the exact evidence bytes (base64) instead of a promotion
 * receipt.  A reused component may therefore be copied into a later release
 * without claiming that the later workflow built or attested the artifact.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { canonicalJson, validateComposition } from "./resolve-release-component-composition.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const MAX_EVIDENCE_ASSET_BYTES = 8 << 20;

function fail(message) { throw new Error(message); }
function assert(condition, message) { if (!condition) fail(message); }

function parseArgs(argv) {
  const options = {
    composition: "",
    runtimeEvidenceDir: "",
    engineRuntimeEvidence: "",
    enginePackageEvidence: "",
    previousBundle: "",
    output: "",
    root: DEFAULT_ROOT,
    json: false,
  };
  const valueFlags = new Set([
    "--composition", "--runtime-evidence-dir", "--engine-runtime-evidence",
    "--engine-package-evidence", "--previous-bundle", "--output", "--root-dir",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: build-component-evidence-bundle.mjs --composition FILE --output FILE [--runtime-evidence-dir DIR] [--engine-runtime-evidence FILE] [--engine-package-evidence FILE] [--previous-bundle FILE]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--composition") options.composition = path.resolve(value);
    else if (arg === "--runtime-evidence-dir") options.runtimeEvidenceDir = path.resolve(value);
    else if (arg === "--engine-runtime-evidence") options.engineRuntimeEvidence = path.resolve(value);
    else if (arg === "--engine-package-evidence") options.enginePackageEvidence = path.resolve(value);
    else if (arg === "--previous-bundle") options.previousBundle = path.resolve(value);
    else if (arg === "--output") options.output = path.resolve(value);
    else if (arg === "--root-dir") options.root = path.resolve(value);
  }
  assert(options.composition, "--composition is required");
  assert(options.output, "--output is required");
  return options;
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function readBytes(file, label) {
  try {
    const stat = fs.lstatSync(file);
    assert(stat.isFile() && !stat.isSymbolicLink(), `${label} must be a regular file`);
    return fs.readFileSync(file);
  } catch (error) {
    fail(`cannot read ${label}: ${error.message}`);
  }
}

function sha256(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function assertAllowedKeys(value, keys, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  const unexpected = Object.keys(value).filter((key) => !keys.has(key));
  assert(unexpected.length === 0, `${label} contains unknown fields: ${unexpected.sort().join(", ")}`);
}

function decodeCanonicalBase64(value, label) {
  assert(typeof value === "string" && value.length > 0, `${label} must be non-empty base64`);
  assert(/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(value), `${label} is not canonical base64`);
  const bytes = Buffer.from(value, "base64");
  assert(bytes.length > 0 && bytes.length <= MAX_EVIDENCE_ASSET_BYTES, `${label} decoded size is invalid`);
  assert(bytes.toString("base64") === value, `${label} is not canonical base64`);
  return bytes;
}

function baseRef(reference) {
  const value = String(reference ?? "");
  const separator = value.indexOf("#");
  return separator < 0 ? value : value.slice(0, separator);
}

function safeRelative(value, label) {
  const normalized = String(value ?? "").replaceAll("\\", "/");
  assert(normalized && !normalized.startsWith("/") && !normalized.split("/").includes(".."), `${label} is unsafe`);
  return normalized;
}

function normalizeAsset(asset, label) {
  assertAllowedKeys(asset, new Set(["ref", "sha256", "mediaType", "byteLength", "contentBase64"]), label);
  assert(typeof asset.ref === "string", `${label}.ref is required`);
  const ref = safeRelative(asset.ref, `${label}.ref`);
  assert(!ref.includes("#") && ref.endsWith(".json"), `${label}.ref must be a JSON asset path`);
  assert(typeof asset.sha256 === "string" && DIGEST_RE.test(asset.sha256), `${label}.sha256 is invalid`);
  assert(asset.mediaType === "application/json", `${label}.mediaType must be application/json`);
  assert(Number.isSafeInteger(asset.byteLength) && asset.byteLength > 0 && asset.byteLength <= MAX_EVIDENCE_ASSET_BYTES, `${label}.byteLength is invalid`);
  const bytes = decodeCanonicalBase64(asset.contentBase64, `${label}.contentBase64`);
  assert(bytes.length === asset.byteLength, `${label}.byteLength does not match decoded content`);
  assert(sha256(bytes) === asset.sha256, `${label}.sha256 does not match decoded content`);
  return { ref, sha256: asset.sha256, mediaType: asset.mediaType, byteLength: asset.byteLength, contentBase64: asset.contentBase64, bytes };
}

function loadPreviousBundle(file) {
  if (!file) return null;
  const value = readJson(file, "previous component evidence bundle");
  assertAllowedKeys(value, new Set(["schemaVersion", "kind", "releaseTag", "compositionDigest", "components", "assets"]), "previous component evidence bundle");
  assert(value.schemaVersion === 1 && value.kind === "lunafox.component-evidence-bundle.v1", "previous component evidence bundle schema is invalid");
  assert(Array.isArray(value.components) && Array.isArray(value.assets), "previous component evidence bundle inventory is invalid");
  const normalizedAssets = value.assets.map((asset, index) => normalizeAsset(asset, `previous component evidence bundle.assets[${index}]`));
  const assets = new Map(normalizedAssets.map((asset) => [asset.ref, asset]));
  assert(assets.size === normalizedAssets.length, "previous component evidence bundle contains duplicate assets");
  const components = new Map(value.components.map((component, index) => {
    assertAllowedKeys(component, new Set(["id", "kind", "disposition", "artifact", "sourceRelease", "evidence"]), `previous component evidence bundle.components[${index}]`);
    assert(typeof component.id === "string" && component.id, `previous component evidence bundle.components[${index}].id is invalid`);
    return [component.id, component];
  }));
  assert(components.size === value.components.length, "previous component evidence bundle contains duplicate component IDs");
  return { value, assets, components };
}

function findRuntimeEvidence(directory, component) {
  if (!directory) return null;
  const file = path.join(directory, `public-runtime-image-evidence-${component}.json`);
  return fs.existsSync(file) ? file : null;
}

function findEngineEvidence(file, engineId) {
  if (!file) return null;
  const value = readJson(file, `Engine evidence ${file}`);
  // Keep the complete aggregate as the original proof. The component ref is
  // a stable locator into this asset; the verifier checks the matching entry.
  assert(value && typeof value === "object", `Engine evidence ${file} must be an object`);
  return { file, value, engineId };
}

function componentEngineId(id) {
  const match = /^engine\.(lunafox\.[a-z][a-z0-9_]*)\.(runtime|package)$/.exec(id);
  return match ? match[1] : "";
}

function evidenceReferences(value, label) {
  const references = Array.isArray(value) ? value : [value];
  assert(references.length > 0, `${label} must contain at least one reference`);
  return references.map((reference, index) => {
    assert(typeof reference === "string" && reference.trim() === reference && reference, `${label}[${index}] is invalid`);
    return reference;
  });
}

function sourceForReference(component, reference, options, previous) {
  const ref = baseRef(reference);
  const priorAsset = previous?.assets.get(ref);
  // A reused component must always carry the original source-release proof.
  // Even a well-formed current aggregate is not an acceptable substitute.
  if (component.disposition === "reused" && priorAsset) {
    assert(previous?.value?.compositionDigest && component.sourceRelease.compositionDigest === previous.value.compositionDigest,
      `${component.id} reused source release composition does not match the previous evidence bundle`);
    return { bytes: priorAsset.bytes, source: "source-release" };
  }
  if (component.disposition === "reused") {
    assert(component.sourceRelease.compositionDigest, `${component.id} reused source release composition digest is required`);
    fail(`${component.id} has no source-release evidence asset for ${ref}`);
  }
  const runtimeMatch = /^runtime-evidence\/([a-z][a-z0-9_-]*)\.json$/.exec(ref);
  if (runtimeMatch) {
    const file = findRuntimeEvidence(options.runtimeEvidenceDir, runtimeMatch[1]);
    if (file) {
      const bytes = readBytes(file, `Runtime evidence ${runtimeMatch[1]}`);
      // The workflow emits a promotion receipt for a reused component so the
      // final job can prove the plan was followed. It is not original proof;
      // fall through to the source-release bundle instead.
      try {
        const value = JSON.parse(bytes.toString("utf8"));
        if (value?.kind !== "lunafox.component-promotion-receipt.v1") return { bytes, source: "current" };
      } catch {
        return { bytes, source: "current" };
      }
    }
  }
  const engineId = componentEngineId(component.id);
  if (engineId && ref.startsWith("engine-runtime-evidence/") && options.engineRuntimeEvidence) {
    return { bytes: readBytes(options.engineRuntimeEvidence, "Engine Runtime evidence"), source: "current" };
  }
  if (engineId && ref.startsWith("engine-package-evidence/") && options.enginePackageEvidence) {
    return { bytes: readBytes(options.enginePackageEvidence, "Engine Package evidence"), source: "current" };
  }
  if (priorAsset) {
    return { bytes: priorAsset.bytes, source: "source-release" };
  }
  return null;
}

function build(options) {
  const composition = validateComposition(readJson(options.composition, "runtime composition"), { requireManifestBinding: true });
  const previous = loadPreviousBundle(options.previousBundle);
  if (previous) {
    assert(typeof previous.value.releaseTag === "string" && previous.value.releaseTag, "previous component evidence bundle releaseTag is required");
    assert(typeof previous.value.compositionDigest === "string" && DIGEST_RE.test(previous.value.compositionDigest), "previous component evidence bundle compositionDigest is invalid");
  }
  const refs = new Map();
  const components = composition.components.map((component) => {
    if (component.disposition === "reused") {
      const priorComponent = previous?.components.get(component.id);
      assert(priorComponent, `${component.id} has no source-release component evidence`);
      assert(canonicalJson(priorComponent.artifact) === canonicalJson(component.artifact), `${component.id} source-release artifact drifted`);
      assert(canonicalJson({ ...priorComponent.sourceRelease, compositionDigest: previous.value.compositionDigest }) === canonicalJson(component.sourceRelease), `${component.id} source-release metadata drifted`);
      assert(canonicalJson(priorComponent.evidence) === canonicalJson(component.evidence), `${component.id} source-release evidence references drifted`);
    }
    const evidence = {};
    for (const [kind, value] of Object.entries(component.evidence)) {
      for (const reference of evidenceReferences(value, `${component.id}.evidence.${kind}`)) {
        const source = sourceForReference(component, reference, options, previous);
        assert(source, `${component.id} evidence is not available: ${reference}`);
        const assetRef = safeRelative(baseRef(reference), `${component.id}.evidence.${kind}`);
        assert(!assetRef.includes("#") && assetRef.endsWith(".json"), `${component.id}.evidence.${kind} must reference a JSON asset`);
        const bytesDigest = sha256(source.bytes);
        const existing = refs.get(assetRef);
        if (existing && existing.sha256 !== bytesDigest) fail(`evidence asset bytes drift for ${assetRef}`);
        if (!existing) {
          refs.set(assetRef, {
            ref: assetRef,
            sha256: bytesDigest,
            mediaType: "application/json",
            byteLength: source.bytes.length,
            contentBase64: source.bytes.toString("base64"),
          });
        }
      }
      evidence[kind] = value;
    }
    return {
      id: component.id,
      kind: component.kind,
      disposition: component.disposition,
      artifact: component.artifact,
      sourceRelease: component.sourceRelease,
      evidence,
    };
  });
  const bundle = {
    schemaVersion: 1,
    kind: "lunafox.component-evidence-bundle.v1",
    releaseTag: composition.releaseTag,
    compositionDigest: composition.compositionDigest,
    components,
    assets: [...refs.values()].sort((left, right) => left.ref.localeCompare(right.ref)),
  };
  fs.mkdirSync(path.dirname(options.output), { recursive: true });
  const temp = `${options.output}.${process.pid}.tmp`;
  fs.writeFileSync(temp, `${JSON.stringify(bundle, null, 2)}\n`);
  fs.renameSync(temp, options.output);
  return bundle;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const bundle = build(options);
  process.stdout.write(`${options.json ? JSON.stringify({ schemaVersion: 1, passed: true, releaseTag: bundle.releaseTag, compositionDigest: bundle.compositionDigest, componentCount: bundle.components.length, assetCount: bundle.assets.length }, null, 2) : `component evidence bundle built: ${bundle.components.length} components, ${bundle.assets.length} assets`}\n`);
}

if (process.argv[1] && fileURLToPath(import.meta.url) === fs.realpathSync(path.resolve(process.argv[1]))) {
  try { main(); }
  catch (error) { process.stderr.write(`component evidence bundle build failed: ${error.message}\n`); process.exitCode = 1; }
}

export { baseRef, build, parseArgs };
