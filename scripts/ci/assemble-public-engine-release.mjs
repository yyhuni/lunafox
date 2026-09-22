#!/usr/bin/env node

/**
 * Assemble the complete Engine inventory for a component-composed release.
 *
 * Current receipts describe only Engine pairs built by this release. Reused
 * pairs are restored from the prior release's durable evidence bundle, so a
 * release manifest can remain complete without inventing a new attestation.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { canonicalize, validateCompositionPlan } from "./resolve-release-component-composition.mjs";
import { packageVersionForPair } from "./resolve-engine-release-disposition.mjs";
import { validate as validateCurrentBuild } from "./verify-public-engine-release.mjs";
import { loadVerifiedBundle } from "./verify-component-evidence-bundle.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const ENGINE_ID_RE = /^engine\.lunafox\.[a-z][a-z0-9_]*$/;
const RELEASE_TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const COMMIT_RE = /^[0-9a-f]{40}$/;
const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const PUBLIC_SIGNER = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";

function fail(message) { throw new Error(message); }
function assert(condition, message) { if (!condition) fail(message); }

function parseArgs(argv) {
  const options = {
    plan: "", releaseContext: "", previousBundle: "", previousComposition: "", previousManifest: "", runtime: "", packages: "",
    packageDigests: "", runtimeEvidence: "", packageEvidence: "", outputDir: "", json: false,
  };
  const flags = new Set([
    "--plan", "--release-context", "--previous-bundle", "--previous-composition", "--previous-manifest", "--runtime-build-results",
    "--package-build-results", "--package-digests-dir", "--runtime-evidence",
    "--package-evidence", "--output-dir",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: assemble-public-engine-release.mjs --plan FILE --release-context FILE --output-dir DIR [--previous-bundle FILE] [--runtime-build-results FILE --package-build-results FILE --package-digests-dir DIR --runtime-evidence FILE --package-evidence FILE]\n");
      process.exit(0);
    }
    if (!flags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    const key = {
      "--plan": "plan", "--release-context": "releaseContext", "--previous-bundle": "previousBundle", "--previous-composition": "previousComposition", "--previous-manifest": "previousManifest",
      "--runtime-build-results": "runtime", "--package-build-results": "packages",
      "--package-digests-dir": "packageDigests", "--runtime-evidence": "runtimeEvidence",
      "--package-evidence": "packageEvidence", "--output-dir": "outputDir",
    }[arg];
    options[key] = path.resolve(value);
  }
  for (const [key, flag] of [["plan", "--plan"], ["releaseContext", "--release-context"], ["outputDir", "--output-dir"]]) {
    assert(options[key], `${flag} is required`);
  }
  return options;
}

function readRegular(file, label) {
  try {
    const stat = fs.lstatSync(file);
    assert(stat.isFile() && !stat.isSymbolicLink(), `${label} must be a regular non-symlink file: ${file}`);
    return fs.readFileSync(file);
  } catch (error) {
    fail(`cannot read ${label}: ${error.message}`);
  }
}

function readJson(file, label) {
  try { return JSON.parse(readRegular(file, label).toString("utf8")); }
  catch (error) { fail(`cannot parse ${label}: ${error.message}`); }
}

function writeJson(file, value) {
  const temporary = `${file}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`);
  fs.renameSync(temporary, file);
}

function sha256(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function baseRef(reference) {
  const value = String(reference ?? "");
  return value.split("#", 1)[0];
}

function safeFileName(engineID) {
  assert(ENGINE_ID_RE.test(engineID), `invalid Engine ID: ${engineID}`);
  return engineID.replaceAll(/[^A-Za-z0-9_.-]/g, "_");
}

function requiredCurrentInputs(options) {
  return [options.runtime, options.packages, options.packageDigests, options.runtimeEvidence, options.packageEvidence];
}

function enginePairs(plan) {
  const pairs = new Map();
  for (const component of plan.components) {
    const match = /^(engine\.lunafox\.[a-z][a-z0-9_]*)\.(runtime|package)$/.exec(component.id);
    if (!match) continue;
    const [, engineID, role] = match;
    const pair = pairs.get(engineID) ?? {};
    assert(!pair[role], `composition plan has duplicate ${role} component for ${engineID}`);
    pair[role] = component;
    pairs.set(engineID, pair);
  }
  assert(pairs.size > 0, "composition plan does not contain Engine pairs");
  for (const [engineID, pair] of pairs) {
    assert(pair.runtime && pair.package, `composition plan must contain Runtime and Package components for ${engineID}`);
    assert(pair.runtime.disposition === pair.package.disposition, `composition plan must not split Runtime/Package disposition for ${engineID}`);
  }
  return new Map([...pairs.entries()].sort(([left], [right]) => left.localeCompare(right)));
}

function validateContext(value, plan) {
  assert(value && value.schemaVersion === "lunafox.engine-release-context.v1", "Engine release context schema is invalid");
  assert(value.releaseTag === plan.releaseTag && RELEASE_TAG_RE.test(value.releaseTag), "Engine release context release tag does not match composition plan");
  assert(COMMIT_RE.test(value.publicMergeCommit), "Engine release context public merge commit is invalid");
  for (const field of ["sourceRevisionDigest", "publicExportManifestSha256", "publicProvenanceSha256"]) {
    assert(DIGEST_RE.test(value[field]), `Engine release context ${field} is invalid`);
  }
  assert(value.signerIdentity === PUBLIC_SIGNER, "Engine release context signer identity is invalid");
  return value;
}

function loadPreviousBundle(file, compositionFile, manifestFile) {
  assert(file, "reused Engine pairs require --previous-bundle");
  assert(compositionFile, "reused Engine pairs require --previous-composition");
  assert(manifestFile, "reused Engine pairs require --previous-manifest");
  const loaded = loadVerifiedBundle({ bundle: file, composition: compositionFile, manifest: manifestFile });
  const { bundle, assets: normalizedAssets, components } = loaded;
  assert(bundle.releaseTag && DIGEST_RE.test(bundle.compositionDigest), "previous component evidence bundle release binding is invalid");
  const assets = new Map();
  for (const [ref, asset] of normalizedAssets) assets.set(ref, asset.bytes);
  return { assets, components, compositionDigest: bundle.compositionDigest };
}

function bundleJson(assets, reference, label) {
  const ref = baseRef(reference);
  const bytes = assets.get(ref);
  assert(bytes, `${label} is missing original evidence asset ${ref}`);
  try { return JSON.parse(bytes.toString("utf8")); }
  catch (error) { fail(`${label} original evidence asset is not JSON: ${error.message}`); }
}

function expectedEvidence(prefix, engineID) {
  const safe = engineID.replaceAll(".", "-");
  return {
    image: `${prefix}/${safe}.json`,
    provenance: `${prefix}/${safe}.json#provenance`,
    sbom: `${prefix}/${safe}.json#sbom`,
    signature: `${prefix}/${safe}.json#signature`,
  };
}

function sourceReleaseWithoutCompositionDigest(sourceRelease) {
  const { compositionDigest: _compositionDigest, ...identity } = sourceRelease ?? {};
  return canonicalize(identity);
}

function requireRefs(refs, digest, label) {
  assert(Array.isArray(refs) && refs.length === 2, `${label} must contain Docker Hub and GHCR refs`);
  assert(refs[0]?.startsWith("docker.io/") && refs[1]?.startsWith("ghcr.io/"), `${label} registry order is invalid`);
  for (const ref of refs) assert(ref?.endsWith(`@${digest}`), `${label} does not bind ${digest}`);
  return [...refs];
}

function packageDigestMap(directory) {
  assert(directory && fs.statSync(directory, { throwIfNoEntry: false })?.isDirectory(), "current Engine package digest directory is required");
  const values = new Map();
  for (const name of fs.readdirSync(directory).filter((entry) => entry.endsWith(".env")).sort()) {
    const payload = readRegular(path.join(directory, name), `Engine package digest ${name}`).toString("utf8");
    const fields = Object.fromEntries(payload.split(/\r?\n/).filter(Boolean).map((line) => {
      const index = line.indexOf("=");
      return index > 0 ? [line.slice(0, index), line.slice(index + 1)] : [line, ""];
    }));
    const engineID = fields.ENGINE_ID;
    assert(ENGINE_ID_RE.test(engineID) && !values.has(engineID), `invalid or duplicate current Engine package digest ${name}`);
    const refs = fields.ENGINE_REFS?.split(",") ?? [];
    assert(refs.length === 2, `current Engine package digest ${name} has invalid refs`);
    values.set(engineID, refs);
  }
  return values;
}

function currentEntries(options, context, pairs, outputDir) {
  const runtime = readJson(options.runtime, "current Engine Runtime receipt");
  const packages = readJson(options.packages, "current Engine Package receipt");
  const runtimeEvidence = readJson(options.runtimeEvidence, "current Engine Runtime evidence");
  const packageEvidence = readJson(options.packageEvidence, "current Engine Package evidence");
  const refsByID = packageDigestMap(options.packageDigests);
  const runtimeByID = new Map((runtime.engines ?? []).map((entry) => [entry.engineId, entry]));
  const packageByID = new Map((packages.packages ?? []).map((entry) => [entry.engineId, entry]));
  const runtimeEvidenceByID = new Map((runtimeEvidence.engines ?? []).map((entry) => [entry.engineId, entry]));
  const packageEvidenceByID = new Map((packageEvidence.packages ?? []).map((entry) => [entry.engineId, entry]));
  const builtIDs = [...pairs.entries()].filter(([, pair]) => pair.runtime.disposition === "built").map(([engineID]) => engineID);
  const sameIDs = (actual, label) => {
    const sorted = [...actual.keys()].sort();
    assert(JSON.stringify(sorted) === JSON.stringify(builtIDs), `${label} must cover exactly the built Engine pairs`);
  };
  sameIDs(runtimeByID, "current Engine Runtime receipt");
  sameIDs(packageByID, "current Engine Package receipt");
  sameIDs(runtimeEvidenceByID, "current Engine Runtime evidence");
  sameIDs(packageEvidenceByID, "current Engine Package evidence");
  sameIDs(refsByID, "current Engine package digests");

  const entries = [];
  for (const engineID of builtIDs) {
    const runtimeEntry = runtimeByID.get(engineID);
    const packageEntry = packageByID.get(engineID);
    const packageEvidenceEntry = packageEvidenceByID.get(engineID);
    const plannedPair = pairs.get(engineID);
    const expectedPackageVersion = packageVersionForPair(engineID, plannedPair);
    assert(packageEntry.engineVersion === expectedPackageVersion, `${engineID} current Package version ${JSON.stringify(packageEntry.engineVersion)} does not match fingerprint-derived version ${expectedPackageVersion}`);
    const runtimeRefs = requireRefs(runtimeEntry.refs, runtimeEntry.indexDigest, `${engineID} current Runtime receipt`);
    const packageRefs = requireRefs(refsByID.get(engineID), packageEvidenceEntry.artifactManifestDigest, `${engineID} current Package receipt`);
    entries.push({
      engineId: engineID,
      runtimeImageDigest: runtimeEntry.indexDigest,
      runtimeImageRefs: runtimeRefs,
      platforms: runtimeEntry.platforms,
      packageDigest: packageEntry.packageDigest,
      packageRuntimeImageDigest: packageEntry.runtimeImageDigest,
      packageArtifactManifestDigest: packageEvidenceEntry.artifactManifestDigest,
      packageRefs,
    });
  }
  const currentManifest = manifestFor(context, entries);
  const currentManifestPath = path.join(outputDir, "current-build-manifest.json");
  writeJson(currentManifestPath, currentManifest);
  validateCurrentBuild({
    manifest: currentManifestPath,
    runtime: options.runtime,
    packages: options.packages,
    packageDigests: options.packageDigests,
    runtimeEvidence: options.runtimeEvidence,
    packageEvidence: options.packageEvidence,
    tag: context.releaseTag,
    merge: context.publicMergeCommit,
    source: context.sourceRevisionDigest,
    exportManifest: context.publicExportManifestSha256,
    provenance: context.publicProvenanceSha256,
    workflowIdentity: context.signerIdentity,
  });
  return entries;
}

function reusedEntries(pairs, previous) {
  const { assets, components, compositionDigest } = previous;
  const entries = [];
  for (const [engineID, pair] of pairs) {
    if (pair.runtime.disposition !== "reused") continue;
    for (const role of ["runtime", "package"]) {
      const bundled = components.get(`${engineID}.${role}`);
      const planned = pair[role];
      assert(bundled, `${engineID}.${role} is missing from previous component bundle`);
      assert(bundled.kind === planned.kind && (bundled.disposition === "built" || bundled.disposition === "reused"), `${engineID}.${role} previous bundle disposition is invalid`);
      assert(JSON.stringify(bundled.artifact) === JSON.stringify(planned.artifact), `${engineID}.${role} previous bundle artifact differs from composition plan`);
      assert(planned.sourceRelease?.compositionDigest === compositionDigest, `${engineID}.${role} planned source composition digest is not bound to the previous bundle`);
      assert(JSON.stringify(sourceReleaseWithoutCompositionDigest(bundled.sourceRelease)) === JSON.stringify(sourceReleaseWithoutCompositionDigest(planned.sourceRelease)), `${engineID}.${role} previous bundle source release identity differs from composition plan`);
      assert(JSON.stringify(bundled.evidence) === JSON.stringify(planned.evidence), `${engineID}.${role} previous bundle evidence reference differs from composition plan`);
    }
    assert(JSON.stringify(pair.runtime.evidence) === JSON.stringify(expectedEvidence("engine-runtime-evidence", engineID)), `${engineID} reused Runtime evidence reference is invalid`);
    assert(JSON.stringify(pair.package.evidence) === JSON.stringify(expectedEvidence("engine-package-evidence", engineID)), `${engineID} reused Package evidence reference is invalid`);
    const runtimeEvidence = bundleJson(assets, pair.runtime.evidence.image, `${engineID} Runtime`);
    const packageEvidence = bundleJson(assets, pair.package.evidence.image, `${engineID} Package`);
    const runtime = runtimeEvidence.engines?.find((entry) => entry.engineId === engineID);
    const packageRecord = packageEvidence.packages?.find((entry) => entry.engineId === engineID);
    assert(runtime && packageRecord, `${engineID} original evidence does not contain the reused pair`);
    const runtimeRefs = requireRefs(runtime.refs, pair.runtime.artifact.digest, `${engineID} original Runtime evidence`);
    assert(runtime.indexDigest === pair.runtime.artifact.digest && runtimeRefs.includes(pair.runtime.artifact.ref), `${engineID} original Runtime evidence does not match composition plan`);
    const packageRefs = requireRefs((packageRecord.candidates ?? []).map((entry) => entry.ref), pair.package.artifact.digest, `${engineID} original Package evidence`);
    assert(DIGEST_RE.test(packageRecord.packageDigest) && packageRecord.artifactManifestDigest === pair.package.artifact.digest && packageRefs.includes(pair.package.artifact.ref), `${engineID} original Package evidence does not match composition plan`);
    entries.push({
      engineId: engineID,
      runtimeImageDigest: runtime.indexDigest,
      runtimeImageRefs: runtimeRefs,
      platforms: runtime.platforms,
      packageDigest: packageRecord.packageDigest,
      packageRuntimeImageDigest: runtime.indexDigest,
      packageArtifactManifestDigest: packageRecord.artifactManifestDigest,
      packageRefs,
    });
  }
  return entries;
}

function manifestFor(context, entries) {
  return {
    schemaVersion: "lunafox.engine-release-manifest.v1",
    repository: PUBLIC_REPOSITORY,
    releaseTag: context.releaseTag,
    publicMergeCommit: context.publicMergeCommit,
    sourceRevisionDigest: context.sourceRevisionDigest,
    publicExportManifestSha256: context.publicExportManifestSha256,
    publicProvenanceSha256: context.publicProvenanceSha256,
    signerIdentity: context.signerIdentity,
    engines: [...entries].sort((left, right) => left.engineId.localeCompare(right.engineId)),
  };
}

function copyCurrentInputs(options, outputDir) {
  for (const [source, target] of [
    [options.runtime, "runtime-build-results.json"],
    [options.packages, "package-build-results.json"],
    [options.runtimeEvidence, "runtime-evidence.json"],
    [options.packageEvidence, "package-evidence.json"],
  ]) fs.copyFileSync(source, path.join(outputDir, target));
}

function writePackageDigests(entries, outputDir) {
  const directory = path.join(outputDir, "public-engine-package-digests");
  fs.mkdirSync(directory);
  for (const entry of entries) {
    fs.writeFileSync(path.join(directory, `${safeFileName(entry.engineId)}.env`), `ENGINE_ID=${entry.engineId}\nENGINE_REFS=${entry.packageRefs.join(",")}\n`);
  }
}

function assemble(options) {
  assert(!fs.existsSync(options.outputDir), `--output-dir must not already exist: ${options.outputDir}`);
  const plan = validateCompositionPlan(readJson(options.plan, "Engine composition plan"));
  const context = validateContext(readJson(options.releaseContext, "Engine release context"), plan);
  const pairs = enginePairs(plan);
  const builtEngineIds = [...pairs.entries()].filter(([, pair]) => pair.runtime.disposition === "built").map(([engineID]) => engineID);
  const reusedEngineIds = [...pairs.entries()].filter(([, pair]) => pair.runtime.disposition === "reused").map(([engineID]) => engineID);
  fs.mkdirSync(options.outputDir, { recursive: true });

  let entries = [];
  if (builtEngineIds.length > 0) {
    assert(requiredCurrentInputs(options).every(Boolean), "built Engine pairs require current Runtime, Package, digest, and evidence inputs");
    entries = currentEntries(options, context, pairs, options.outputDir);
    copyCurrentInputs(options, options.outputDir);
  } else {
    assert(requiredCurrentInputs(options).every((value) => !value), "reused-only Engine release must not provide current-release build evidence");
  }
  if (reusedEngineIds.length > 0) entries.push(...reusedEntries(pairs, loadPreviousBundle(options.previousBundle, options.previousComposition, options.previousManifest)));
  const expectedIDs = [...pairs.keys()];
  assert(JSON.stringify(entries.map((entry) => entry.engineId).sort()) === JSON.stringify(expectedIDs), "assembled Engine manifest inventory is incomplete");
  const manifest = manifestFor(context, entries);
  writeJson(path.join(options.outputDir, "manifest.json"), manifest);
  writeJson(path.join(options.outputDir, "engine-release-manifest.json"), manifest);
  writePackageDigests(manifest.engines, options.outputDir);
  const mode = {
    schemaVersion: 1,
    kind: "lunafox.engine-release-composition-handoff.v1",
    releaseTag: context.releaseTag,
    planDigest: plan.planDigest,
    builtEngineIds,
    reusedEngineIds,
    currentBuildEvidence: builtEngineIds.length > 0,
  };
  writeJson(path.join(options.outputDir, "mode.json"), mode);
  const verification = {
    schemaVersion: 1,
    passed: true,
    releaseTag: context.releaseTag,
    engineCount: expectedIDs.length,
    builtEngineIds,
    reusedEngineIds,
    manifestSha256: sha256(readRegular(path.join(options.outputDir, "manifest.json"), "assembled Engine manifest")),
  };
  writeJson(path.join(options.outputDir, "verification.json"), verification);
  return verification;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = assemble(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `assembled Engine release inventory: ${result.engineCount} Engines`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) { process.stderr.write(`assemble public Engine release failed: ${error.message}\n`); process.exitCode = 1; }
}

export { assemble, enginePairs, parseArgs, reusedEntries };
