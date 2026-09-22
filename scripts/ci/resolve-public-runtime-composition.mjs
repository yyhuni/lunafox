#!/usr/bin/env node

/**
 * Compose the protected public release's component plan and final evidence.
 *
 * The workflow owns registry publication; this helper only turns verified
 * source/build receipts into the canonical composition contract.  A missing
 * previous release is treated as a first publication (all components build),
 * while a malformed previous composition is rejected instead of granting
 * reuse on incomplete evidence.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
  bindCompositionToManifest,
  finalizeCompositionPlan,
  resolveCompositionPlan,
  validateComposition,
} from "./resolve-release-component-composition.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const RELEASE_TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const COMMIT_RE = /^[0-9a-f]{40}$/;
const RUNTIME_COMPONENTS = Object.freeze([
  { id: "runtime.server", kind: "runtime", name: "server", contextPath: ".", dockerfile: "server/Dockerfile", dockerignore: ".dockerignore" },
  { id: "runtime.frontend", kind: "runtime", name: "frontend", contextPath: "frontend", dockerfile: "frontend/Dockerfile", dockerignore: "frontend/.dockerignore" },
  { id: "runtime.nginx", kind: "runtime", name: "nginx", contextPath: "docker/nginx", dockerfile: "docker/nginx/Dockerfile", dockerignore: "" },
  { id: "runtime.bootstrap", kind: "runtime", name: "bootstrap", contextPath: ".", dockerfile: "docker/bootstrap/Dockerfile", dockerignore: ".dockerignore" },
  { id: "runtime.agent", kind: "runtime", name: "agent", contextPath: ".", dockerfile: "docker/agent/Dockerfile", dockerignore: ".dockerignore" },
]);

function fail(message) { throw new Error(message); }
function assert(condition, message) { if (!condition) fail(message); }
function isObject(value) { return value !== null && typeof value === "object" && !Array.isArray(value); }

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function writeJson(file, value) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  const temp = `${file}.${process.pid}.tmp`;
  fs.writeFileSync(temp, `${JSON.stringify(value, null, 2)}\n`);
  fs.renameSync(temp, file);
}

function parseArgs(argv) {
  const options = {
    mode: "plan", root: DEFAULT_ROOT, releaseTag: "", sourceRevisionDigest: "",
    publicMergeCommit: "", previous: "", plan: "", output: "", receipts: "",
    runtimeEvidenceDir: "", engineRuntimeEvidence: "", enginePackageEvidence: "",
    engineManifest: "", engineDiscovery: "", baseImages: "", manifestDigest: "", json: false,
  };
  const valueFlags = new Set([
    "--mode", "--root-dir", "--release-tag", "--source-revision-digest", "--public-merge-commit",
    "--previous", "--plan", "--output", "--receipts", "--runtime-evidence-dir",
    "--engine-runtime-evidence", "--engine-package-evidence", "--engine-manifest",
    "--engine-discovery", "--base-images", "--manifest-digest",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: resolve-public-runtime-composition.mjs --mode plan|final|bind --output FILE [options]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--mode") options.mode = value;
    else if (arg === "--root-dir") options.root = path.resolve(value);
    else if (arg === "--release-tag") options.releaseTag = value;
    else if (arg === "--source-revision-digest") options.sourceRevisionDigest = value;
    else if (arg === "--public-merge-commit") options.publicMergeCommit = value;
    else if (arg === "--previous") options.previous = path.resolve(value);
    else if (arg === "--plan") options.plan = path.resolve(value);
    else if (arg === "--output") options.output = path.resolve(value);
    else if (arg === "--receipts") options.receipts = path.resolve(value);
    else if (arg === "--runtime-evidence-dir") options.runtimeEvidenceDir = path.resolve(value);
    else if (arg === "--engine-runtime-evidence") options.engineRuntimeEvidence = path.resolve(value);
    else if (arg === "--engine-package-evidence") options.enginePackageEvidence = path.resolve(value);
    else if (arg === "--engine-manifest") options.engineManifest = path.resolve(value);
    else if (arg === "--engine-discovery") options.engineDiscovery = path.resolve(value);
    else if (arg === "--base-images") options.baseImages = path.resolve(value);
    else if (arg === "--manifest-digest") options.manifestDigest = value;
  }
  assert(["plan", "final", "bind"].includes(options.mode), "--mode must be plan, final, or bind");
  assert(options.output, "--output is required");
  if (options.mode !== "bind") {
    assert(RELEASE_TAG_RE.test(options.releaseTag), `invalid release tag: ${options.releaseTag}`);
    assert(DIGEST_RE.test(options.sourceRevisionDigest), "source revision digest is required");
    assert(COMMIT_RE.test(options.publicMergeCommit), "public merge commit is required");
  } else {
    assert(options.manifestDigest && DIGEST_RE.test(options.manifestDigest), "--manifest-digest is required for bind mode");
    assert(options.plan, "--plan must point to a composition for bind mode");
  }
  if (options.mode === "final") {
    assert(options.plan, "final mode requires --plan");
    assert(options.receipts || options.runtimeEvidenceDir || options.engineRuntimeEvidence || options.enginePackageEvidence,
      "final mode requires built component receipts or evidence inputs");
  }
  return options;
}

function findAgentArtifactId(root) {
  const directory = path.join(root, "agent", "bin");
  assert(fs.existsSync(directory) && fs.statSync(directory).isDirectory(), "public Agent bundle directory is missing");
  const entries = fs.readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && /^sha256-[a-f0-9]{64}$/.test(entry.name))
    .map((entry) => entry.name)
    .sort();
  assert(entries.length === 1, "public Agent tree must contain exactly one immutable artifact directory");
  return entries[0];
}

function baseImageMap(file) {
  if (!file) return {};
  const value = readJson(file, "base image identities");
  assert(isObject(value), "base image identities must be an object");
  return value;
}

function runtimeSpecs(root, baseImages) {
  const artifactId = findAgentArtifactId(root);
  return RUNTIME_COMPONENTS.map((component) => ({
    ...component,
    buildArgs: component.id === "runtime.agent" ? { AGENT_ARTIFACT_ID: artifactId } : {},
    baseImageIdentities: baseImages,
    platforms: ["linux/amd64", "linux/arm64"],
    builderPolicy: { provenance: "mode=max", sbom: true, source: "public-validate" },
  }));
}

function engineSpecs(root, discovery, releaseTag, baseImages) {
  if (!discovery) return [];
  const value = readJson(discovery, "Engine discovery");
  assert(Array.isArray(value.engines) && value.engines.length > 0, "Engine discovery is empty");
  return value.engines.flatMap((engine) => {
    const engineId = String(engine.engineId ?? "");
    const directory = String(engine.directory ?? "");
    const dockerfile = String(engine.dockerfile ?? "");
    const contextPath = "extensions/engines";
    assert(/^engine\.lunafox\.[a-z][a-z0-9_]*$/.test(engineId), `invalid discovered Engine ID: ${engineId}`);
    assert(/^[a-z][a-z0-9_]*$/.test(directory) && dockerfile === `${directory}/Dockerfile`, `invalid Engine source closure: ${engineId}`);
    const common = {
      kind: "engine", name: directory, contextPath, dockerfile: `extensions/engines/${dockerfile}`,
      dockerignore: "", namedContexts: { contracts: { path: "contracts" }, "engine-go": { path: "engine-go" } },
      // Product release metadata is published in the manifest/provenance, not
      // injected into the image graph. A stable label value lets an unchanged
      // Engine fingerprint be promoted across release tags.
      buildArgs: { ENGINE_IMAGE_VERSION: "immutable", ENGINE_IMAGE_SOURCE: "https://github.com/yyhuni/lunafox" },
      baseImageIdentities: baseImages, platforms: ["linux/amd64", "linux/arm64"],
      builderPolicy: { provenance: "mode=max", sbom: true, source: "public-engine-release" },
      generatedInputs: [
        { name: `${directory}/engine.json`, file: `extensions/engines/${directory}/engine.json` },
        { name: "contracts/go.sum", file: "contracts/go.sum" },
        { name: "engine-go/go.sum", file: "engine-go/go.sum" },
      ],
    };
    return [
      { ...common, id: `${engineId}.runtime` },
      { ...common, id: `${engineId}.package`, builderPolicy: { ...common.builderPolicy, artifactKind: "engine-package" } },
    ];
  });
}

function compositionSpecs(options) {
  const baseImages = baseImageMap(options.baseImages);
  return [...runtimeSpecs(options.root, baseImages), ...engineSpecs(options.root, options.engineDiscovery, options.releaseTag, baseImages)];
}

function evidenceReference(prefix, component) {
  const safe = component.replaceAll(".", "-");
  return {
    image: `${prefix}/${safe}.json`,
    provenance: `${prefix}/${safe}.json#provenance`,
    sbom: `${prefix}/${safe}.json#sbom`,
    signature: `${prefix}/${safe}.json#signature`,
  };
}

function runtimeReceipts(options) {
  const receipts = {};
  if (!options.runtimeEvidenceDir) return receipts;
  assert(fs.existsSync(options.runtimeEvidenceDir), `Runtime evidence directory is missing: ${options.runtimeEvidenceDir}`);
  for (const file of fs.readdirSync(options.runtimeEvidenceDir).filter((name) => name.endsWith(".json")).sort()) {
    const record = readJson(path.join(options.runtimeEvidenceDir, file), `Runtime evidence ${file}`);
    const component = String(record.component ?? "");
    // Promotion receipts use stable composition IDs and intentionally stay out
    // of finalization: reused artifact/evidence remain the verified plan facts.
    if (record.kind === "lunafox.component-promotion-receipt.v1") continue;
    if (!RUNTIME_COMPONENTS.some((entry) => entry.name === component)) continue;
    assert(DIGEST_RE.test(record.digest) && typeof record.image === "string" && record.image.endsWith(`@${record.digest}`), `invalid Runtime evidence artifact: ${file}`);
    receipts[`runtime.${component}`] = {
      artifact: { ref: record.image, digest: record.digest, platforms: record.platforms },
      evidence: evidenceReference("runtime-evidence", component),
    };
  }
  return receipts;
}

function engineReceipts(options) {
  const receipts = {};
  if (!options.engineRuntimeEvidence && !options.enginePackageEvidence) return receipts;
  assert(options.engineRuntimeEvidence && options.enginePackageEvidence, "Engine Runtime and Package evidence must be supplied together");
  const runtime = readJson(options.engineRuntimeEvidence, "Engine Runtime evidence");
  const packages = readJson(options.enginePackageEvidence, "Engine Package evidence");
  const runtimeById = new Map((runtime.engines ?? []).map((entry) => [entry.engineId, entry]));
  const packageById = new Map((packages.packages ?? []).map((entry) => [entry.engineId, entry]));
  assert(runtimeById.size > 0 && runtimeById.size === packageById.size, "Engine evidence inventory is incomplete");
  for (const [engineId, runtimeEntry] of runtimeById) {
    const packageEntry = packageById.get(engineId);
    assert(Array.isArray(runtimeEntry.refs) && runtimeEntry.refs.length > 0, `Engine Runtime refs missing: ${engineId}`);
    assert(DIGEST_RE.test(runtimeEntry.indexDigest), `Engine Runtime digest missing: ${engineId}`);
    const runtimeRef = runtimeEntry.refs.find((ref) => ref.endsWith(`@${runtimeEntry.indexDigest}`)) ?? runtimeEntry.refs[0];
    assert(runtimeRef.endsWith(`@${runtimeEntry.indexDigest}`), `Engine Runtime ref digest mismatch: ${engineId}`);
    receipts[`${engineId}.runtime`] = {
      artifact: { ref: runtimeRef, digest: runtimeEntry.indexDigest, platforms: runtimeEntry.platforms },
      evidence: evidenceReference("engine-runtime-evidence", engineId),
    };
    const packageCandidate = (packageEntry?.candidates ?? []).find((entry) => String(entry.ref ?? "").endsWith(`@${packageEntry.artifactManifestDigest}`));
    assert(packageCandidate && DIGEST_RE.test(packageEntry.artifactManifestDigest), `Engine Package artifact evidence missing: ${engineId}`);
    receipts[`${engineId}.package`] = {
      artifact: { ref: packageCandidate.ref, digest: packageEntry.artifactManifestDigest },
      evidence: evidenceReference("engine-package-evidence", engineId),
    };
  }
  return receipts;
}

function buildReceipts(options, plan = null) {
  const receipts = { ...runtimeReceipts(options), ...engineReceipts(options) };
  if (options.receipts) {
    const supplied = readJson(options.receipts, "built component receipts");
    assert(isObject(supplied), "built component receipts must be an object");
    Object.assign(receipts, supplied);
  }
  if (plan) {
    const builtIds = new Set(plan.components
      .filter((component) => component.disposition === "built")
      .map((component) => component.id));
    return Object.fromEntries(Object.entries(receipts).filter(([id]) => builtIds.has(id)));
  }
  return receipts;
}

function resolvePlan(options) {
  const previous = options.previous ? readJson(options.previous, "previous runtime composition") : null;
  return resolveCompositionPlan({
    root: options.root,
    releaseTag: options.releaseTag,
    sourceRevisionDigest: options.sourceRevisionDigest,
    publicMergeCommit: options.publicMergeCommit,
    previous,
    components: compositionSpecs(options),
    capabilities: { dynamicFrontendUpstream: true, runtimeCompositionSchemaVersion: 1 },
  });
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  let result;
  if (options.mode === "bind") {
    result = bindCompositionToManifest(readJson(options.plan, "runtime composition"), options.manifestDigest);
  } else if (options.mode === "plan") {
    result = resolvePlan(options);
  } else {
    const plan = readJson(options.plan, "runtime composition plan");
    result = finalizeCompositionPlan(plan, buildReceipts(options, plan));
  }
  if (options.mode === "final" || options.mode === "bind") result = validateComposition(result);
  writeJson(options.output, result);
  const summary = {
    mode: options.mode,
    compositionDigest: result.compositionDigest ?? null,
    planDigest: result.planDigest ?? null,
    built: (result.components ?? []).filter((entry) => entry.disposition === "built").map((entry) => entry.id),
    reused: (result.components ?? []).filter((entry) => entry.disposition === "reused").map((entry) => entry.id),
  };
  process.stdout.write(`${options.json ? JSON.stringify({ ...summary, result }, null, 2) : `runtime composition ${options.mode}: ${summary.built.length} built, ${summary.reused.length} reused`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) { process.stderr.write(`public runtime composition failed: ${error.message}\n`); process.exitCode = 1; }
}

export {
  RUNTIME_COMPONENTS,
  buildReceipts,
  compositionSpecs,
  engineReceipts,
  parseArgs,
  resolvePlan,
  runtimeReceipts,
};
