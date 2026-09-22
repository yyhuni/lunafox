#!/usr/bin/env node

import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { assemble } from "./assemble-public-engine-release.mjs";
import { compositionCorePayload, FINGERPRINT_SCHEMA_VERSION, sha256Digest } from "./resolve-release-component-composition.mjs";
import { packageVersionForPair } from "./resolve-engine-release-disposition.mjs";
import { loadVerifiedBundle } from "./verify-component-evidence-bundle.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-engine-assembly-selftest-"));
const signer = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";
const currentTag = "v1.2.3";
const previousTag = "v1.2.2";
const merge = "e".repeat(40);
const source = "sha256:" + "d".repeat(64);
const exportManifest = "sha256:" + "f".repeat(64);
const provenance = "sha256:" + "a".repeat(64);
const builtEngine = "engine.lunafox.port_scan";
const reusedEngine = "engine.lunafox.http_probe";
const builtRuntimeDigest = "sha256:" + "1".repeat(64);
const builtPackageDigest = "sha256:" + "2".repeat(64);
const builtArchiveDigest = "sha256:" + "3".repeat(64);
const reusedRuntimeDigest = "sha256:" + "4".repeat(64);
const reusedPackageDigest = "sha256:" + "5".repeat(64);
const reusedArchiveDigest = "sha256:" + "6".repeat(64);

function write(file, value) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
  return file;
}

function writeText(file, value) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, value);
  return file;
}

function sha(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function refs(name, digest) {
  return [
    `docker.io/yyhuni/${name}@${digest}`,
    `ghcr.io/yyhuni/${name}@${digest}`,
  ];
}

function evidenceRef(prefix, engineID) {
  const safe = engineID.replaceAll(".", "-");
  return {
    image: `${prefix}/${safe}.json`,
    provenance: `${prefix}/${safe}.json#provenance`,
    sbom: `${prefix}/${safe}.json#sbom`,
    signature: `${prefix}/${safe}.json#signature`,
  };
}

function runtimeEvidence(component, ref, digest) {
  return {
    status: "published",
    sbom: true,
    provenance: true,
    attestation: true,
    sourceRepository: "yyhuni/lunafox-private",
    destinationRepository: "yyhuni/lunafox",
    workflowIdentity: signer,
    digest,
    image: ref,
    attestationSubject: ref,
    attestationPredicateType: "https://slsa.dev/provenance/v1",
    releaseTag: previousTag,
    component,
  };
}

function fingerprint(letter, componentId) {
  const kind = componentId.split(".")[0];
  const inputs = {
    schemaVersion: FINGERPRINT_SCHEMA_VERSION,
    componentId,
    kind,
    contextPath: ".",
    dockerfile: `${componentId.replaceAll(".", "/")}/Dockerfile`,
    dockerignore: "",
    files: [],
    namedContexts: {},
    buildArgs: {},
    platforms: ["linux/amd64", "linux/arm64"],
    baseImages: [],
    baseImagesResolved: true,
    builderPolicy: { fixture: letter },
    generatedInputs: [],
  };
  return {
    version: FINGERPRINT_SCHEMA_VERSION,
    algorithm: "sha256-canonical-json-v1",
    digest: sha256Digest(inputs),
    baseImagesResolved: true,
    inputs,
  };
}

function planFor(dispositions, reusedCompositionDigest = "sha256:" + "c".repeat(64)) {
  const components = [
    {
      id: `${builtEngine}.runtime`, kind: "engine", name: "port_scan", inputFingerprint: fingerprint("7", `${builtEngine}.runtime`),
      disposition: dispositions.built, sourceRelease: { tag: dispositions.built === "built" ? currentTag : previousTag, ...(dispositions.built === "reused" ? { compositionDigest: reusedCompositionDigest } : {}) },
      ...(dispositions.built === "reused" ? { artifact: { ref: refs("lunafox-engine-runtime-port-scan", builtRuntimeDigest)[0], digest: builtRuntimeDigest }, evidence: evidenceRef("engine-runtime-evidence", builtEngine) } : {}),
    },
    {
      id: `${builtEngine}.package`, kind: "engine", name: "port_scan", inputFingerprint: fingerprint("8", `${builtEngine}.package`),
      disposition: dispositions.built, sourceRelease: { tag: dispositions.built === "built" ? currentTag : previousTag, ...(dispositions.built === "reused" ? { compositionDigest: reusedCompositionDigest } : {}) },
      ...(dispositions.built === "reused" ? { artifact: { ref: refs("lunafox-engine-runtime-port-scan", builtPackageDigest)[0], digest: builtPackageDigest }, evidence: evidenceRef("engine-package-evidence", builtEngine) } : {}),
    },
    {
      id: `${reusedEngine}.runtime`, kind: "engine", name: "http_probe", inputFingerprint: fingerprint("9", `${reusedEngine}.runtime`),
      disposition: dispositions.reused, sourceRelease: { tag: previousTag, compositionDigest: reusedCompositionDigest },
      artifact: { ref: refs("lunafox-engine-runtime-http-probe", reusedRuntimeDigest)[0], digest: reusedRuntimeDigest }, evidence: evidenceRef("engine-runtime-evidence", reusedEngine),
    },
    {
      id: `${reusedEngine}.package`, kind: "engine", name: "http_probe", inputFingerprint: fingerprint("0", `${reusedEngine}.package`),
      disposition: dispositions.reused, sourceRelease: { tag: previousTag, compositionDigest: reusedCompositionDigest },
      artifact: { ref: refs("lunafox-engine-runtime-http-probe", reusedPackageDigest)[0], digest: reusedPackageDigest }, evidence: evidenceRef("engine-package-evidence", reusedEngine),
    },
  ].sort((left, right) => left.id.localeCompare(right.id));
  const plan = {
    schemaVersion: 1,
    kind: "lunafox.runtime-composition-plan",
    releaseTag: currentTag,
    sourceRevisionDigest: source,
    publicMergeCommit: merge,
    capabilities: { dynamicFrontendUpstream: true },
    components,
  };
  plan.planDigest = sha256Digest(plan);
  return plan;
}

function context() {
  return {
    schemaVersion: "lunafox.engine-release-context.v1",
    releaseTag: currentTag,
    publicMergeCommit: merge,
    sourceRevisionDigest: source,
    publicExportManifestSha256: exportManifest,
    publicProvenanceSha256: provenance,
    signerIdentity: signer,
  };
}

function aggregateEvidence(tag, runtimeEntries, packageEntries) {
  const binding = {
    publicMergeCommit: merge,
    sourceRevisionDigest: source,
    releaseTag: tag,
    publicExportManifestSha256: exportManifest,
    publicProvenanceSha256: provenance,
  };
  return {
    runtime: { schemaVersion: "lunafox.engine-runtime-image-publication-evidence.v1", passed: true, signerIdentity: signer, ...binding, engines: runtimeEntries },
    packages: { schemaVersion: "lunafox.engine-package-v2-publication-evidence.v1", passed: true, signerIdentity: signer, ...binding, packages: packageEntries },
  };
}

function bundleAsset(ref, value) {
  const bytes = Buffer.from(`${JSON.stringify(value, null, 2)}\n`);
  return { ref, sha256: sha(bytes), mediaType: "application/json", byteLength: bytes.length, contentBase64: bytes.toString("base64") };
}

function previousRuntimeComponents() {
  return ["server", "frontend", "nginx", "agent", "bootstrap"].map((name, index) => {
    const digest = `sha256:${"789ab"[index].repeat(64)}`;
    const ref = refs(`lunafox-${name}`, digest)[0];
    return {
      id: `runtime.${name}`,
      kind: "runtime",
      name,
      disposition: "built",
      inputFingerprint: fingerprint(String(index + 1), `runtime.${name}`),
      artifact: { ref, digest },
      sourceRelease: { tag: previousTag },
      evidence: evidenceRef("runtime-evidence", name),
      proof: runtimeEvidence(name, ref, digest),
    };
  });
}

function previousBundle(compositionDigest) {
  const builtRuntime = { engineId: builtEngine, indexDigest: builtRuntimeDigest, refs: refs("lunafox-engine-runtime-port-scan", builtRuntimeDigest), platforms: ["linux/amd64", "linux/arm64"] };
  const reusedRuntime = { engineId: reusedEngine, indexDigest: reusedRuntimeDigest, refs: refs("lunafox-engine-runtime-http-probe", reusedRuntimeDigest), platforms: ["linux/amd64", "linux/arm64"] };
  const builtPackage = { engineId: builtEngine, packageDigest: builtArchiveDigest, artifactManifestDigest: builtPackageDigest, candidates: refs("lunafox-engine-runtime-port-scan", builtPackageDigest).map((ref) => ({ ref })) };
  const reusedPackage = { engineId: reusedEngine, packageDigest: reusedArchiveDigest, artifactManifestDigest: reusedPackageDigest, candidates: refs("lunafox-engine-runtime-http-probe", reusedPackageDigest).map((ref) => ({ ref })) };
  const aggregate = aggregateEvidence(previousTag, [builtRuntime, reusedRuntime], [builtPackage, reusedPackage]);
  const builtSourceRelease = { tag: previousTag };
  const reusedSourceRelease = { tag: previousTag };
  const components = [
    { id: `${builtEngine}.runtime`, kind: "engine", disposition: "built", artifact: { ref: refs("lunafox-engine-runtime-port-scan", builtRuntimeDigest)[0], digest: builtRuntimeDigest }, sourceRelease: builtSourceRelease, evidence: evidenceRef("engine-runtime-evidence", builtEngine) },
    { id: `${builtEngine}.package`, kind: "engine", disposition: "built", artifact: { ref: refs("lunafox-engine-runtime-port-scan", builtPackageDigest)[0], digest: builtPackageDigest }, sourceRelease: builtSourceRelease, evidence: evidenceRef("engine-package-evidence", builtEngine) },
    { id: `${reusedEngine}.runtime`, kind: "engine", disposition: "built", artifact: { ref: refs("lunafox-engine-runtime-http-probe", reusedRuntimeDigest)[0], digest: reusedRuntimeDigest }, sourceRelease: reusedSourceRelease, evidence: evidenceRef("engine-runtime-evidence", reusedEngine) },
    { id: `${reusedEngine}.package`, kind: "engine", disposition: "built", artifact: { ref: refs("lunafox-engine-runtime-http-probe", reusedPackageDigest)[0], digest: reusedPackageDigest }, sourceRelease: reusedSourceRelease, evidence: evidenceRef("engine-package-evidence", reusedEngine) },
  ];
  const runtimeComponents = previousRuntimeComponents();
  components.push(...runtimeComponents.map(({ id, kind, disposition, artifact, sourceRelease, evidence }) => ({
    id, kind, disposition, artifact, sourceRelease, evidence,
  })));
  const runtimeAssets = runtimeComponents.map(({ id, proof }) => bundleAsset(evidenceRef("runtime-evidence", id.slice("runtime.".length)).image, proof));
  return {
    schemaVersion: 1,
    kind: "lunafox.component-evidence-bundle.v1",
    releaseTag: previousTag,
    compositionDigest,
    components,
    assets: [
      bundleAsset(evidenceRef("engine-runtime-evidence", builtEngine).image, aggregate.runtime),
      bundleAsset(evidenceRef("engine-package-evidence", builtEngine).image, aggregate.packages),
      bundleAsset(evidenceRef("engine-runtime-evidence", reusedEngine).image, aggregate.runtime),
      bundleAsset(evidenceRef("engine-package-evidence", reusedEngine).image, aggregate.packages),
      ...runtimeAssets,
    ],
  };
}

function previousManifest(composition, manifestDigest = "") {
  const runtimeComponents = composition.components.filter((component) => component.id.startsWith("runtime."));
  const packageComponents = composition.components.filter((component) => component.id.endsWith(".package"));
  // Reuse loads the previous manifest through the public verifier, which
  // rejects a manifest that has no bound bilingual release notes.
  const notesBody = "## English\n\n- Previous release.\n\n## 简体中文\n\n- 上一版本。\n";
  const notesDigest = crypto.createHash("sha256").update(Buffer.from(notesBody, "utf8")).digest("hex");
  const lines = [
    `releaseVersion: "${previousTag.slice(1)}"`,
    "releaseNotes:",
    `  digest: "sha256:${notesDigest}"`,
    "  body: |",
    ...notesBody.replace(/\n$/, "").split("\n").map((line) => (line === "" ? "" : `    ${line}`)),
    "runtimeImages:",
    ...runtimeComponents.map((component) => `  - name: ${component.id.slice("runtime.".length)}\n    refs: ["${component.artifact.ref}", "${component.artifact.ref.replace("docker.io/", "ghcr.io/")}\"]`),
    "enginePackages:",
    ...packageComponents.map((component) => `  - refs: ["${component.artifact.ref}", "${component.artifact.ref.replace("docker.io/", "ghcr.io/")}\"]`),
    "runtimeComposition:",
    "  schemaVersion: 1",
    '  asset: "runtime-composition.json"',
    `  sha256: "${composition.compositionDigest}"`,
    "upgrade:",
    '  manifestId: "lunafox-1.2.2"',
    '  deploymentMode: "single-node-compose"',
    '  compatibilityRange: ">=0.0.0 <1.0.0"',
    "  maintenanceWindowMinutes: 15",
    "  requiresAdminConfirmation: true",
    "  databaseMigration:",
    "    hasDatabaseMigration: false",
    '    migrationType: "none"',
    '    migrationId: ""',
    '    checksum: ""',
    "    policyVersion: 1",
  ];
  if (manifestDigest) {
    // The composition digest excludes manifestBinding, so this reverse binding
    // can be added after the manifest bytes have been hashed.
    return { text: `${lines.join("\n")}\n`, digest: manifestDigest };
  }
  return { text: `${lines.join("\n")}\n` };
}

function currentInputs() {
  const currentRuntime = { engineId: builtEngine, indexDigest: builtRuntimeDigest, refs: refs("lunafox-engine-runtime-port-scan", builtRuntimeDigest), platforms: ["linux/amd64", "linux/arm64"] };
  const currentPackage = { engineId: builtEngine, engineVersion: packageVersionForPair(builtEngine, { runtime: { inputFingerprint: fingerprint("7", `${builtEngine}.runtime`) }, package: { inputFingerprint: fingerprint("8", `${builtEngine}.package`) } }), packageDigest: builtArchiveDigest, runtimeImageDigest: builtRuntimeDigest, runtimeImageRefs: currentRuntime.refs };
  const aggregate = aggregateEvidence(currentTag, [currentRuntime], [{ engineId: builtEngine, packageDigest: builtArchiveDigest, artifactManifestDigest: builtPackageDigest, candidates: refs("lunafox-engine-runtime-port-scan", builtPackageDigest).map((ref) => ({ ref })) }]);
  const directory = path.join(root, "current");
  const runtime = write(path.join(directory, "runtime.json"), { schemaVersion: "lunafox.engine-runtime-image-build-results.v1", mode: "production", engines: [currentRuntime] });
  const packages = write(path.join(directory, "packages.json"), { schemaVersion: "lunafox.engine-package-build-results.v1", mode: "production", packages: [currentPackage] });
  const runtimeEvidence = write(path.join(directory, "runtime-evidence.json"), aggregate.runtime);
  const packageEvidence = write(path.join(directory, "package-evidence.json"), aggregate.packages);
  const digests = path.join(directory, "digests");
  fs.mkdirSync(digests, { recursive: true });
  fs.writeFileSync(path.join(digests, "port-scan.env"), `ENGINE_ID=${builtEngine}\nENGINE_REFS=${refs("lunafox-engine-runtime-port-scan", builtPackageDigest).join(",")}\n`);
  return { runtime, packages, runtimeEvidence, packageEvidence, packageDigests: digests };
}

try {
  const contextPath = write(path.join(root, "context.json"), context());
  // A prior release composition is complete: even components that were built
  // there retain their immutable artifact/evidence bindings for later reuse.
  const previousPlan = planFor({ built: "reused", reused: "reused" });
  previousPlan.components = previousPlan.components.map((component) => ({
    ...component,
    disposition: "built",
    sourceRelease: { tag: component.sourceRelease.tag },
  }));
  const previousComposition = {
    schemaVersion: 1,
    kind: "lunafox.runtime-composition",
    releaseTag: previousTag,
    sourceRevisionDigest: source,
    publicMergeCommit: merge,
    capabilities: previousPlan.capabilities,
    components: [...previousPlan.components, ...previousRuntimeComponents().map(({ proof: _proof, ...component }) => component)]
      .sort((left, right) => left.id.localeCompare(right.id)),
  };
  // Normalize the historical component inventory before calculating its
  // digest; the production validator sorts components and strips optional
  // fields, so hashing the fixture's construction order would be invalid.
  const normalizedPreviousCore = {
    ...previousComposition,
    components: [...previousComposition.components].sort((left, right) => left.id.localeCompare(right.id)),
    capabilities: { ...previousComposition.capabilities },
  };
  previousComposition.compositionDigest = sha256Digest(compositionCorePayload(normalizedPreviousCore));
  const previousCompositionPath = write(path.join(root, "previous-composition.json"), previousComposition);
  const bundlePath = write(path.join(root, "previous-bundle.json"), previousBundle(previousComposition.compositionDigest));
  const manifestDraft = previousManifest(previousComposition).text;
  const manifestDigest = sha(Buffer.from(manifestDraft));
  previousComposition.manifestBinding = { manifestDigest };
  write(previousCompositionPath, previousComposition);
  writeText(path.join(root, "previous-manifest.yaml"), manifestDraft);
  const objectCompositionBundle = loadVerifiedBundle({
    bundle: bundlePath,
    composition: previousComposition,
    manifest: path.join(root, "previous-manifest.yaml"),
  });
  assert.equal(objectCompositionBundle.composition.compositionDigest, previousComposition.compositionDigest);
  const mixedPlan = write(path.join(root, "mixed-plan.json"), planFor({ built: "built", reused: "reused" }, previousComposition.compositionDigest));
  const mixedOutput = path.join(root, "mixed-output");
  const previousManifestPath = path.join(root, "previous-manifest.yaml");
  const mixed = assemble({ plan: mixedPlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: mixedOutput, ...currentInputs() });
  assert.deepEqual(mixed.builtEngineIds, [builtEngine]);
  assert.deepEqual(mixed.reusedEngineIds, [reusedEngine]);
  assert.equal(JSON.parse(fs.readFileSync(path.join(mixedOutput, "manifest.json"), "utf8")).engines.length, 2);
  assert.equal(fs.readdirSync(path.join(mixedOutput, "public-engine-package-digests")).length, 2);
  const emptyOutput = path.join(root, "empty-precreated-output");
  fs.mkdirSync(emptyOutput);
  const precreated = assemble({ plan: mixedPlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: emptyOutput, ...currentInputs() });
  assert.equal(precreated.engineCount, 2);
  const transcriptOutput = path.join(root, "transcript-precreated-output");
  fs.mkdirSync(transcriptOutput);
  fs.writeFileSync(path.join(transcriptOutput, "assembly-verification.json"), "");
  const withTranscript = assemble({ plan: mixedPlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: transcriptOutput, ...currentInputs() });
  assert.equal(withTranscript.engineCount, 2);
  const dirtyOutput = path.join(root, "dirty-precreated-output");
  fs.mkdirSync(dirtyOutput);
  fs.writeFileSync(path.join(dirtyOutput, "stale.json"), "{}\n");
  assert.throws(
    () => assemble({ plan: mixedPlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: dirtyOutput, ...currentInputs() }),
    /must not already exist/,
  );

  const mismatchedPackageInputs = currentInputs();
  const mismatchedPackage = JSON.parse(fs.readFileSync(mismatchedPackageInputs.packages, "utf8"));
  mismatchedPackage.packages[0].engineVersion = `0.0.0+${"f".repeat(64)}`;
  write(mismatchedPackageInputs.packages, mismatchedPackage);
  assert.throws(
    () => assemble({ plan: mixedPlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: path.join(root, "mismatched-package-version-output"), ...mismatchedPackageInputs }),
    /does not match fingerprint-derived version/,
  );

  const reusePlan = write(path.join(root, "reuse-plan.json"), planFor({ built: "reused", reused: "reused" }, previousComposition.compositionDigest));
  const reuseOutput = path.join(root, "reuse-output");
  const reused = assemble({ plan: reusePlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: reuseOutput });
  assert.deepEqual(reused.builtEngineIds, []);
  assert.deepEqual(reused.reusedEngineIds, [reusedEngine, builtEngine]);
  assert.equal(fs.existsSync(path.join(reuseOutput, "current-build-manifest.json")), false);
  assert.equal(fs.existsSync(path.join(reuseOutput, "runtime-evidence.json")), false);
  assert.equal(JSON.parse(fs.readFileSync(path.join(reuseOutput, "mode.json"), "utf8")).currentBuildEvidence, false);

  const missingSourceDigestPlanValue = planFor({ built: "reused", reused: "reused" }, previousComposition.compositionDigest);
  missingSourceDigestPlanValue.components = missingSourceDigestPlanValue.components.map((component) => {
    const { compositionDigest: _compositionDigest, ...sourceRelease } = component.sourceRelease;
    return { ...component, sourceRelease };
  });
  const { planDigest: _stalePlanDigest, ...missingSourceDigestPayload } = missingSourceDigestPlanValue;
  missingSourceDigestPlanValue.planDigest = sha256Digest(missingSourceDigestPayload);
  const missingSourceDigestPlan = write(path.join(root, "missing-source-composition-digest-plan.json"), missingSourceDigestPlanValue);
  assert.throws(
    () => assemble({ plan: missingSourceDigestPlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: path.join(root, "missing-source-composition-digest-output") }),
    /planned source composition digest is not bound to the previous bundle/,
  );

  const driftedSourceReleasePlanValue = planFor({ built: "reused", reused: "reused" }, previousComposition.compositionDigest);
  driftedSourceReleasePlanValue.components = driftedSourceReleasePlanValue.components.map((component) => component.id === `${reusedEngine}.runtime`
    ? { ...component, sourceRelease: { ...component.sourceRelease, workflowRun: "unexpected" } }
    : component);
  const { planDigest: _staleDriftedPlanDigest, ...driftedSourceReleasePayload } = driftedSourceReleasePlanValue;
  driftedSourceReleasePlanValue.planDigest = sha256Digest(driftedSourceReleasePayload);
  const driftedSourceReleasePlan = write(path.join(root, "drifted-source-release-plan.json"), driftedSourceReleasePlanValue);
  assert.throws(
    () => assemble({ plan: driftedSourceReleasePlan, releaseContext: contextPath, previousBundle: bundlePath, previousComposition: previousCompositionPath, previousManifest: previousManifestPath, outputDir: path.join(root, "drifted-source-release-output") }),
    /previous bundle source release identity differs from composition plan/,
  );
  console.log("public Engine release assembly self-test passed");
} finally {
  fs.rmSync(root, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
}
