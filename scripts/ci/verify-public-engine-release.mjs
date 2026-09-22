#!/usr/bin/env node

/** Validate the immutable public Engine handoff consumed by private release. */

import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { validateComposition, validateCompositionPlan } from "./resolve-release-component-composition.mjs";
import { packageVersionForPair } from "./resolve-engine-release-disposition.mjs";

const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const PUBLIC_SIGNER_RE = /^https:\/\/github\.com\/yyhuni\/lunafox\/.github\/workflows\/public-validate\.yml@refs\/heads\/main$/;
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const options = { manifest: "", completeManifest: "", runtime: "", packages: "", packageDigests: "", runtimeEvidence: "", packageEvidence: "", compositionPlan: "", composition: "", tag: "", merge: "", source: "", exportManifest: "", provenance: "", workflowIdentity: "", json: false };
  const valueFlags = new Set([
    "--manifest", "--complete-manifest", "--runtime-build-results", "--package-build-results", "--package-digests-dir",
    "--runtime-evidence", "--package-evidence", "--tag", "--public-merge-commit",
    "--source-revision-digest", "--export-manifest-sha256", "--public-provenance-sha256",
    "--workflow-identity", "--composition-plan", "--composition",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") { process.stdout.write("Usage: verify-public-engine-release.mjs --composition-plan FILE --composition FILE --complete-manifest FILE --tag TAG [--manifest FILE --runtime-build-results FILE --package-build-results FILE --package-digests-dir DIR --runtime-evidence FILE --package-evidence FILE --public-merge-commit SHA --source-revision-digest DIGEST --export-manifest-sha256 DIGEST --public-provenance-sha256 DIGEST --workflow-identity ID]\n"); process.exit(0); }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    const key = {
      "--manifest": "manifest", "--complete-manifest": "completeManifest", "--runtime-build-results": "runtime", "--package-build-results": "packages",
      "--package-digests-dir": "packageDigests", "--runtime-evidence": "runtimeEvidence", "--package-evidence": "packageEvidence",
      "--tag": "tag", "--public-merge-commit": "merge", "--source-revision-digest": "source",
      "--export-manifest-sha256": "exportManifest", "--public-provenance-sha256": "provenance", "--workflow-identity": "workflowIdentity",
      "--composition-plan": "compositionPlan", "--composition": "composition",
    }[arg];
    options[key] = value;
  }
  return options;
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label} ${file}: ${error.message}`); }
}

function requireFile(file, label) {
  if (!file || !fs.statSync(file, { throwIfNoEntry: false })?.isFile()) fail(`${label} is required: ${file}`);
}

function requireDigest(value, label) {
  if (!DIGEST_RE.test(value ?? "")) fail(`${label} must be an immutable sha256 digest`);
}

function requireDigestRef(value, registry, label) {
  if (typeof value !== "string" || !value.startsWith(`${registry}/`) || !/@sha256:[a-f0-9]{64}$/.test(value)) {
    fail(`${label} must be a ${registry} digest reference`);
  }
}

function sortedUnique(values, label) {
  if (!Array.isArray(values) || values.length === 0 || new Set(values).size !== values.length) fail(`${label} must be a non-empty unique array`);
  const sorted = [...values].sort();
  if (JSON.stringify(values) !== JSON.stringify(sorted)) fail(`${label} must be canonically sorted`);
  return new Set(values);
}

function validateReceipt(receipt, schema, key, label) {
  if (receipt?.schemaVersion !== schema || receipt?.mode !== "production") fail(`${label} has an invalid schema or mode`);
  const items = receipt[key];
  if (!Array.isArray(items) || items.length === 0) fail(`${label} must contain ${key}`);
  const ids = items.map((item) => item.engineId);
  sortedUnique(ids, `${label}.${key}.engineId`);
  return items;
}

function validateEvidence(file, expectedSchema, signer, label, binding) {
  const evidence = readJson(file, label);
  if (evidence.schemaVersion !== expectedSchema || evidence.passed !== true) fail(`${label} did not pass`);
  if (evidence.signerIdentity !== signer) fail(`${label} signer identity mismatch`);
  for (const [field, expected] of Object.entries(binding)) {
    if (evidence[field] !== expected) fail(`${label} ${field} binding mismatch`);
  }
  return evidence;
}

function readPackageDigestFiles(directory) {
  if (!directory || !fs.statSync(directory, { throwIfNoEntry: false })?.isDirectory()) fail(`package digest directory is required: ${directory}`);
  const files = fs.readdirSync(directory).filter((name) => name.endsWith(".env")).sort();
  if (files.length === 0) fail("public Engine package digest directory is empty");
  const result = new Map();
  for (const name of files) {
    const payload = fs.readFileSync(path.join(directory, name), "utf8");
    const values = Object.fromEntries(payload.split(/\r?\n/).filter(Boolean).map((line) => {
      const separator = line.indexOf("=");
      return separator > 0 ? [line.slice(0, separator), line.slice(separator + 1)] : [line, ""];
    }));
    if (!values.ENGINE_ID || result.has(values.ENGINE_ID)) fail(`invalid or duplicate Engine package receipt: ${name}`);
    const refs = (values.ENGINE_REFS ?? "").split(",");
    if (refs.length !== 2) fail(`Engine package receipt must contain two refs: ${name}`);
    requireDigestRef(refs[0], "docker.io", `${name} Docker Hub ref`);
    requireDigestRef(refs[1], "ghcr.io", `${name} GHCR ref`);
    if (refs[0].split("@")[1] !== refs[1].split("@")[1]) fail(`Engine package registry digest drift: ${name}`);
    result.set(values.ENGINE_ID, refs);
  }
  return result;
}

function canonicalJson(value) {
  return JSON.stringify(value);
}

function componentComparable(component) {
  return {
    id: component.id,
    kind: component.kind,
    name: component.name,
    inputFingerprint: component.inputFingerprint,
    disposition: component.disposition,
    sourceRelease: component.sourceRelease,
    ...(component.artifact ? { artifact: component.artifact } : {}),
    ...(component.evidence ? { evidence: component.evidence } : {}),
  };
}

function engineComponents(components, label) {
  const engines = new Map();
  for (const component of components) {
    const match = /^engine\.(lunafox\.[a-z][a-z0-9_]*)\.(runtime|package)$/.exec(component.id);
    if (!match) continue;
    const [, engineName, role] = match;
    const entry = engines.get(engineName) ?? {};
    if (entry[role]) fail(`${label} has duplicate ${role} entry for engine.${engineName}`);
    entry[role] = component;
    engines.set(engineName, entry);
  }
  if (engines.size === 0) fail(`${label} does not contain Engine components`);
  for (const [engineName, pair] of engines) {
    if (!pair.runtime || !pair.package) fail(`${label} must contain both Runtime and Package components for engine.${engineName}`);
    if (pair.runtime.disposition !== pair.package.disposition) {
      fail(`${label} must not split Runtime/Package disposition for engine.${engineName}`);
    }
  }
  return engines;
}

function expectedEvidence(prefix, engineId) {
  const safe = engineId.replaceAll(".", "-");
  return {
    image: `${prefix}/${safe}.json`,
    provenance: `${prefix}/${safe}.json#provenance`,
    sbom: `${prefix}/${safe}.json#sbom`,
    signature: `${prefix}/${safe}.json#signature`,
  };
}

function hasCurrentBuildHandoff(options) {
  return [options.manifest, options.runtime, options.packages, options.packageDigests, options.runtimeEvidence, options.packageEvidence]
    .some((value) => Boolean(value));
}

function validateManifestBinding(manifest, options) {
  const signer = options.workflowIdentity || manifest.signerIdentity;
  if (!PUBLIC_SIGNER_RE.test(signer)) fail("Engine signer must be the protected public-validate.yml identity");
  if (manifest.schemaVersion !== "lunafox.engine-release-manifest.v1") fail("unsupported public Engine manifest schema");
  if (manifest.repository !== PUBLIC_REPOSITORY || manifest.releaseTag !== options.tag || manifest.publicMergeCommit !== options.merge || manifest.sourceRevisionDigest !== options.source || manifest.publicExportManifestSha256 !== options.exportManifest || manifest.publicProvenanceSha256 !== options.provenance || manifest.signerIdentity !== signer) fail("public Engine manifest provenance binding mismatch");
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(options.tag) || !/^[0-9a-f]{40}$/.test(options.merge)) fail("release tag or public merge commit is invalid");
  for (const [value, label] of [[options.source, "source revision"], [options.exportManifest, "export manifest"], [options.provenance, "public provenance"]]) requireDigest(value, label);
  return signer;
}

function validateCompleteManifest(options, planned, final) {
  requireFile(options.completeManifest, "complete Engine manifest");
  const manifest = readJson(options.completeManifest, "complete Engine manifest");
  validateManifestBinding(manifest, options);
  if (!Array.isArray(manifest.engines) || manifest.engines.length !== planned.size) fail("complete Engine manifest inventory does not match composition plan");
  const engineIds = manifest.engines.map((entry) => entry.engineId);
  sortedUnique(engineIds, "complete Engine manifest engineId");
  const expectedIDs = [...planned.keys()].map((engineID) => `engine.${engineID}`).sort();
  if (canonicalJson(engineIds) !== canonicalJson(expectedIDs)) fail("complete Engine manifest Engine IDs do not match composition plan");
  const byID = new Map(manifest.engines.map((entry) => [entry.engineId, entry]));
  for (const [engineName] of planned) {
    const engineID = `engine.${engineName}`;
    const item = byID.get(engineID);
    const finalPair = final.get(engineName);
    requireDigest(item.runtimeImageDigest, `${engineID} Runtime Image digest`);
    requireDigest(item.packageDigest, `${engineID} Package digest`);
    requireDigest(item.packageRuntimeImageDigest, `${engineID} Package Runtime Image digest`);
    requireDigest(item.packageArtifactManifestDigest, `${engineID} Package OCI manifest digest`);
    if (item.packageRuntimeImageDigest !== item.runtimeImageDigest) fail(`${engineID} Package Runtime Image digest does not match Runtime Image digest`);
    requireDigestRef(item.runtimeImageRefs?.[0], "docker.io", `${engineID} Runtime Docker Hub ref`);
    requireDigestRef(item.runtimeImageRefs?.[1], "ghcr.io", `${engineID} Runtime GHCR ref`);
    requireDigestRef(item.packageRefs?.[0], "docker.io", `${engineID} Package Docker Hub ref`);
    requireDigestRef(item.packageRefs?.[1], "ghcr.io", `${engineID} Package GHCR ref`);
    if (item.runtimeImageRefs.length !== 2 || item.packageRefs.length !== 2) fail(`${engineID} complete manifest must retain exactly two registry refs per artifact`);
    if (item.runtimeImageRefs.some((ref) => !ref.endsWith(`@${item.runtimeImageDigest}`))) fail(`${engineID} Runtime refs do not bind Runtime digest`);
    if (item.packageRefs.some((ref) => !ref.endsWith(`@${item.packageArtifactManifestDigest}`))) fail(`${engineID} Package refs do not bind Package artifact digest`);
    if (item.runtimeImageDigest !== finalPair.runtime.artifact.digest || !item.runtimeImageRefs.includes(finalPair.runtime.artifact.ref)) fail(`${engineID} complete manifest Runtime artifact does not match final composition`);
    if (item.packageArtifactManifestDigest !== finalPair.package.artifact.digest || !item.packageRefs.includes(finalPair.package.artifact.ref)) fail(`${engineID} complete manifest Package artifact does not match final composition`);
  }
  return manifest;
}

// validateCompositionHandoff verifies the complete Engine inventory against
// the build/reuse plan. Current-release receipts are required only for built
// pairs; reused pairs must be byte-for-byte promotions of the verified plan,
// so no current-release attestation can be substituted for their provenance.
function validateCompositionHandoff(options) {
  if (!options.compositionPlan || !options.composition) {
    fail("composition plan and final composition are required together");
  }
  const plan = validateCompositionPlan(readJson(options.compositionPlan, "Engine composition plan"));
  const composition = validateComposition(readJson(options.composition, "final runtime composition"));
  if (composition.releaseTag !== plan.releaseTag || composition.releaseTag !== options.tag) {
    fail("Engine composition release tag does not match the requested release");
  }
  const planned = engineComponents(plan.components, "Engine composition plan");
  const final = engineComponents(composition.components, "final runtime composition");
  if (canonicalJson([...planned.keys()].sort()) !== canonicalJson([...final.keys()].sort())) {
    fail("final runtime composition Engine inventory does not match the plan");
  }
  if (!options.completeManifest) fail("composition handoff requires a complete Engine manifest");
  validateCompleteManifest(options, planned, final);

  const builtEngineIds = [...planned.entries()]
    .filter(([, pair]) => pair.runtime.disposition === "built")
    .map(([engineId]) => `engine.${engineId}`)
    .sort();
  let builtReceipt = null;
  if (builtEngineIds.length > 0) {
    const required = [options.manifest, options.runtime, options.packages, options.packageDigests, options.runtimeEvidence, options.packageEvidence];
    if (required.some((value) => !value)) fail("built Engine pairs require current-release Runtime, Package, and evidence receipts");
    builtReceipt = validate(options);
    if (canonicalJson(builtReceipt.engineIds) !== canonicalJson(builtEngineIds)) {
      fail("current-release Engine receipts must cover exactly the built Engine pairs");
    }
  } else if (hasCurrentBuildHandoff(options)) {
    fail("reused-only Engine release must not provide current-release build evidence");
  }

  const builtById = builtReceipt?.enginesByID ?? new Map();
  for (const [engineName, plannedPair] of planned) {
    const engineId = `engine.${engineName}`;
    const finalPair = final.get(engineName);
    for (const role of ["runtime", "package"]) {
      const plannedComponent = plannedPair[role];
      const finalComponent = finalPair[role];
      if (plannedComponent.disposition === "reused") {
        if (plannedComponent.sourceRelease.tag === options.tag || !plannedComponent.sourceRelease.compositionDigest) {
          fail(`${engineId}.${role} reused source release must retain a prior composition digest`);
        }
        if (canonicalJson(componentComparable(finalComponent)) !== canonicalJson(componentComparable(plannedComponent))) {
          fail(`${engineId}.${role} reused component differs from its original verified plan evidence`);
        }
        continue;
      }
      if (role === "package") {
        const expectedPackageVersion = packageVersionForPair(engineId, plannedPair);
        const actualPackageVersion = builtReceipt?.packageVersionsByID?.get(engineId);
        if (actualPackageVersion !== expectedPackageVersion) {
          fail(`${engineId}.package engineVersion ${JSON.stringify(actualPackageVersion)} does not match fingerprint-derived version ${expectedPackageVersion}`);
        }
      }
      if (canonicalJson(componentComparable({ ...finalComponent, artifact: undefined, evidence: undefined })) !== canonicalJson(componentComparable(plannedComponent))) {
        fail(`${engineId}.${role} built component drifted from the planned input identity`);
      }
      const current = builtById.get(engineId);
      if (!current) fail(`${engineId}.${role} is missing current-release receipt evidence`);
      if (role === "runtime") {
        if (finalComponent.artifact.digest !== current.runtimeImageDigest || !current.runtimeImageRefs.includes(finalComponent.artifact.ref)) {
          fail(`${engineId}.runtime artifact does not match the current-release Runtime receipt`);
        }
        if (canonicalJson(finalComponent.evidence) !== canonicalJson(expectedEvidence("engine-runtime-evidence", engineId))) {
          fail(`${engineId}.runtime evidence does not bind the current Runtime receipt`);
        }
      } else {
        if (finalComponent.artifact.digest !== current.packageArtifactManifestDigest || !current.packageRefs.includes(finalComponent.artifact.ref)) {
          fail(`${engineId}.package artifact does not match the current-release Package receipt`);
        }
        if (canonicalJson(finalComponent.evidence) !== canonicalJson(expectedEvidence("engine-package-evidence", engineId))) {
          fail(`${engineId}.package evidence does not bind the current Package receipt`);
        }
      }
    }
  }
  return {
    schemaVersion: 2,
    passed: true,
    repository: PUBLIC_REPOSITORY,
    releaseTag: options.tag,
    compositionDigest: composition.compositionDigest,
    engineCount: planned.size,
    builtEngineIds,
    reusedEngineIds: [...planned.entries()].filter(([, pair]) => pair.runtime.disposition === "reused").map(([engineId]) => `engine.${engineId}`).sort(),
  };
}

function validate(options) {
  for (const [file, label] of [[options.manifest, "Engine manifest"], [options.runtime, "Runtime receipt"], [options.packages, "Package receipt"], [options.runtimeEvidence, "Runtime evidence"], [options.packageEvidence, "Package evidence"]]) requireFile(file, label);
  const manifest = readJson(options.manifest, "Engine manifest");
  const runtime = readJson(options.runtime, "Runtime receipt");
  const packages = readJson(options.packages, "Package receipt");
  const runtimeItems = validateReceipt(runtime, "lunafox.engine-runtime-image-build-results.v1", "engines", "Runtime receipt");
  const packageItems = validateReceipt(packages, "lunafox.engine-package-build-results.v1", "packages", "Package receipt");
  const packageRefs = readPackageDigestFiles(options.packageDigests);
  const signer = validateManifestBinding(manifest, options);
  const evidenceBinding = {
    publicMergeCommit: options.merge,
    sourceRevisionDigest: options.source,
    releaseTag: options.tag,
    publicExportManifestSha256: options.exportManifest,
    publicProvenanceSha256: options.provenance,
  };
  const runtimeEvidence = validateEvidence(options.runtimeEvidence, "lunafox.engine-runtime-image-publication-evidence.v1", signer, "Runtime evidence", evidenceBinding);
  const packageEvidence = validateEvidence(options.packageEvidence, "lunafox.engine-package-v2-publication-evidence.v1", signer, "Package evidence", evidenceBinding);
  if (!Array.isArray(manifest.engines) || manifest.engines.length !== runtimeItems.length || manifest.engines.length !== packageItems.length) fail("public Engine manifest engine count does not match receipts");
  const runtimeById = new Map(runtimeItems.map((item) => [item.engineId, item]));
  const packageById = new Map(packageItems.map((item) => [item.engineId, item]));
  const manifestIds = manifest.engines.map((item) => item.engineId);
  sortedUnique(manifestIds, "public Engine manifest engineId");
  for (const item of manifest.engines) {
    const runtimeItem = runtimeById.get(item.engineId);
    const packageItem = packageById.get(item.engineId);
    const refs = packageRefs.get(item.engineId);
    if (!runtimeItem || !packageItem || !refs) fail(`public Engine manifest is missing receipt ${item.engineId}`);
    requireDigest(item.runtimeImageDigest, `${item.engineId} Runtime Image digest`);
    requireDigest(item.packageDigest, `${item.engineId} Package digest`);
    requireDigest(item.packageArtifactManifestDigest, `${item.engineId} Package OCI manifest digest`);
    if (item.runtimeImageDigest !== runtimeItem.indexDigest || item.packageRuntimeImageDigest !== runtimeItem.indexDigest || item.packageDigest !== packageItem.packageDigest) fail(`public Engine digest binding mismatch: ${item.engineId}`);
    if (JSON.stringify(item.runtimeImageRefs) !== JSON.stringify(runtimeItem.refs) || JSON.stringify(item.packageRefs) !== JSON.stringify(refs)) fail(`public Engine registry refs mismatch: ${item.engineId}`);
    requireDigestRef(item.runtimeImageRefs[0], "docker.io", `${item.engineId} Runtime Docker Hub ref`);
    requireDigestRef(item.runtimeImageRefs[1], "ghcr.io", `${item.engineId} Runtime GHCR ref`);
    requireDigestRef(item.packageRefs[0], "docker.io", `${item.engineId} Package Docker Hub ref`);
    requireDigestRef(item.packageRefs[1], "ghcr.io", `${item.engineId} Package GHCR ref`);
    if (item.packageRefs[0].split("@")[1] !== item.packageArtifactManifestDigest || item.packageRefs[1].split("@")[1] !== item.packageArtifactManifestDigest) {
      fail(`public Engine Package OCI manifest digest mismatch: ${item.engineId}`);
    }
    const packageRecord = packageEvidence.packages?.find((record) => record.engineId === item.engineId);
    if (!packageRecord || packageRecord.packageDigest !== item.packageDigest || packageRecord.artifactManifestDigest !== item.packageArtifactManifestDigest) {
      fail(`public Engine package evidence binding mismatch: ${item.engineId}`);
    }
    const runtimeRecord = runtimeEvidence.engines?.find((record) => record.engineId === item.engineId);
    if (!runtimeRecord || runtimeRecord.indexDigest !== item.runtimeImageDigest || JSON.stringify(runtimeRecord.refs) !== JSON.stringify(item.runtimeImageRefs)) {
      fail(`public Engine Runtime evidence binding mismatch: ${item.engineId}`);
    }
  }
  const enginesByID = new Map(manifest.engines.map((item) => [item.engineId, item]));
  return {
    schemaVersion: 1,
    passed: true,
    repository: PUBLIC_REPOSITORY,
    signerIdentity: signer,
    releaseTag: options.tag,
    engineCount: manifest.engines.length,
    engineIds: [...enginesByID.keys()].sort(),
    enginesByID,
    packageVersionsByID: new Map(packageItems.map((item) => [item.engineId, item.engineVersion])),
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = options.compositionPlan || options.composition
    ? validateCompositionHandoff(options)
    : validate(options);
  const output = { ...result };
  delete output.enginesByID;
  delete output.packageVersionsByID;
  process.stdout.write(`${options.json ? JSON.stringify(output, null, 2) : "public Engine release verified"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { process.stderr.write(`public Engine release verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { validate, validateCompositionHandoff, parseArgs };
