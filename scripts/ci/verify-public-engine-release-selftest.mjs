#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { validate, validateCompositionHandoff } from "./verify-public-engine-release.mjs";
import {
  compositionCorePayload,
  finalizeCompositionPlan,
  FINGERPRINT_SCHEMA_VERSION,
  sha256Digest,
} from "./resolve-release-component-composition.mjs";
import { packageVersionForPair } from "./resolve-engine-release-disposition.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-public-engine-selftest-"));
const runtimeDigest = "sha256:" + "a".repeat(64);
const archiveDigest = "sha256:" + "b".repeat(64);
const artifactDigest = "sha256:" + "c".repeat(64);
const source = "sha256:" + "d".repeat(64);
const exportManifest = "sha256:" + "e".repeat(64);
const provenance = "sha256:" + "f".repeat(64);
const signer = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";
const engineId = "engine.lunafox.port_scan";
const fingerprint = (letter, componentId) => {
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
};
const dockerRuntime = `docker.io/yyhuni/lunafox-engine-runtime-port-scan@${runtimeDigest}`;
const ghcrRuntime = `ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@${runtimeDigest}`;
const dockerPackage = `docker.io/yyhuni/lunafox-engine-port-scan@${artifactDigest}`;
const ghcrPackage = `ghcr.io/yyhuni/lunafox-engine-port-scan@${artifactDigest}`;
const write = (name, value) => { const file = path.join(root, name); fs.mkdirSync(path.dirname(file), { recursive: true }); fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`); return file; };
const binding = { publicMergeCommit: "e".repeat(40), sourceRevisionDigest: source, releaseTag: "v1.2.3", publicExportManifestSha256: exportManifest, publicProvenanceSha256: provenance };
const runtime = write("runtime.json", { schemaVersion: "lunafox.engine-runtime-image-build-results.v1", mode: "production", engines: [{ engineId, indexDigest: runtimeDigest, refs: [dockerRuntime, ghcrRuntime] }] });
const packages = write("packages.json", { schemaVersion: "lunafox.engine-package-build-results.v1", mode: "production", packages: [{ engineId, engineVersion: packageVersionForPair(engineId, { runtime: { inputFingerprint: fingerprint("3", `${engineId}.runtime`) }, package: { inputFingerprint: fingerprint("4", `${engineId}.package`) } }), packageDigest: archiveDigest, runtimeImageDigest: runtimeDigest, runtimeImageRefs: [dockerRuntime, ghcrRuntime] }] });
const runtimeEvidence = write("runtime-evidence.json", { schemaVersion: "lunafox.engine-runtime-image-publication-evidence.v1", passed: true, signerIdentity: signer, ...binding, engines: [{ engineId, indexDigest: runtimeDigest, refs: [dockerRuntime, ghcrRuntime] }] });
const packageEvidence = write("package-evidence.json", { schemaVersion: "lunafox.engine-package-v2-publication-evidence.v1", passed: true, signerIdentity: signer, ...binding, packages: [{ engineId, packageDigest: archiveDigest, artifactManifestDigest: artifactDigest }] });
const digestDir = path.join(root, "digests"); fs.mkdirSync(digestDir); fs.writeFileSync(path.join(digestDir, "port-scan.env"), `ENGINE_ID=${engineId}\nENGINE_REFS=${dockerPackage},${ghcrPackage}\n`);
const manifest = write("manifest.json", { schemaVersion: "lunafox.engine-release-manifest.v1", repository: "yyhuni/lunafox", releaseTag: "v1.2.3", publicMergeCommit: binding.publicMergeCommit, sourceRevisionDigest: source, publicExportManifestSha256: exportManifest, publicProvenanceSha256: provenance, signerIdentity: signer, engines: [{ engineId, runtimeImageDigest: runtimeDigest, runtimeImageRefs: [dockerRuntime, ghcrRuntime], packageDigest: archiveDigest, packageRuntimeImageDigest: runtimeDigest, packageArtifactManifestDigest: artifactDigest, packageRefs: [dockerPackage, ghcrPackage] }] });
const result = validate({ manifest, runtime, packages, packageDigests: digestDir, runtimeEvidence, packageEvidence, tag: "v1.2.3", merge: "e".repeat(40), source, exportManifest, provenance, workflowIdentity: signer });
assert.equal(result.passed, true);
assert.throws(() => validate({ manifest: write("bad-manifest.json", { ...JSON.parse(fs.readFileSync(manifest, "utf8")), engines: [{ ...JSON.parse(fs.readFileSync(manifest, "utf8")).engines[0], packageArtifactManifestDigest: archiveDigest }] }), runtime, packages, packageDigests: digestDir, runtimeEvidence, packageEvidence, tag: "v1.2.3", merge: binding.publicMergeCommit, source, exportManifest, provenance, workflowIdentity: signer }), /Package OCI manifest digest mismatch/);

const previousTag = "v1.2.2";
const previousComposition = "sha256:" + "9".repeat(64);
const reusedRuntimeDigest = "sha256:" + "1".repeat(64);
const reusedPackageDigest = "sha256:" + "2".repeat(64);
const evidence = (prefix, id) => {
  const safe = id.replaceAll(".", "-");
  return { image: `${prefix}/${safe}.json`, provenance: `${prefix}/${safe}.json#provenance`, sbom: `${prefix}/${safe}.json#sbom`, signature: `${prefix}/${safe}.json#signature` };
};
const reusedEngine = "engine.lunafox.http_probe";
const plan = {
  schemaVersion: 1,
  kind: "lunafox.runtime-composition-plan",
  releaseTag: binding.releaseTag,
  sourceRevisionDigest: source,
  publicMergeCommit: binding.publicMergeCommit,
  capabilities: { dynamicFrontendUpstream: true },
  components: [
    { id: `${engineId}.runtime`, kind: "engine", name: "port_scan", inputFingerprint: fingerprint("3", `${engineId}.runtime`), disposition: "built", sourceRelease: { tag: binding.releaseTag, publicMergeCommit: binding.publicMergeCommit } },
    { id: `${engineId}.package`, kind: "engine", name: "port_scan", inputFingerprint: fingerprint("4", `${engineId}.package`), disposition: "built", sourceRelease: { tag: binding.releaseTag, publicMergeCommit: binding.publicMergeCommit } },
    { id: `${reusedEngine}.runtime`, kind: "engine", name: "http_probe", inputFingerprint: fingerprint("5", `${reusedEngine}.runtime`), disposition: "reused", sourceRelease: { tag: previousTag, compositionDigest: previousComposition }, artifact: { ref: `docker.io/yyhuni/lunafox-engine-runtime-http-probe@${reusedRuntimeDigest}`, digest: reusedRuntimeDigest }, evidence: evidence("previous-runtime", reusedEngine) },
    { id: `${reusedEngine}.package`, kind: "engine", name: "http_probe", inputFingerprint: fingerprint("6", `${reusedEngine}.package`), disposition: "reused", sourceRelease: { tag: previousTag, compositionDigest: previousComposition }, artifact: { ref: `docker.io/yyhuni/lunafox-engine-http-probe@${reusedPackageDigest}`, digest: reusedPackageDigest }, evidence: evidence("previous-package", reusedEngine) },
  ],
};
plan.components.sort((left, right) => left.id.localeCompare(right.id));
plan.planDigest = sha256Digest(((value) => { const { planDigest: _digest, ...payload } = value; return payload; })(plan));
const composition = finalizeCompositionPlan(plan, {
  [`${engineId}.runtime`]: { artifact: { ref: dockerRuntime, digest: runtimeDigest }, evidence: evidence("engine-runtime-evidence", engineId) },
  [`${engineId}.package`]: { artifact: { ref: dockerPackage, digest: artifactDigest }, evidence: evidence("engine-package-evidence", engineId) },
});
const planFile = write("composition-plan.json", plan);
const compositionFile = write("runtime-composition.json", composition);
const reusedRuntimeDocker = `docker.io/yyhuni/lunafox-engine-runtime-http-probe@${reusedRuntimeDigest}`;
const reusedRuntimeGHCR = `ghcr.io/yyhuni/lunafox-engine-runtime-http-probe@${reusedRuntimeDigest}`;
const reusedPackageDocker = `docker.io/yyhuni/lunafox-engine-http-probe@${reusedPackageDigest}`;
const reusedPackageGHCR = `ghcr.io/yyhuni/lunafox-engine-http-probe@${reusedPackageDigest}`;
const completeManifest = write("complete-manifest.json", {
  ...JSON.parse(fs.readFileSync(manifest, "utf8")),
  engines: [
    ...JSON.parse(fs.readFileSync(manifest, "utf8")).engines,
    {
      engineId: reusedEngine,
      runtimeImageDigest: reusedRuntimeDigest,
      runtimeImageRefs: [reusedRuntimeDocker, reusedRuntimeGHCR],
      packageDigest: "sha256:" + "7".repeat(64),
      packageRuntimeImageDigest: reusedRuntimeDigest,
      packageArtifactManifestDigest: reusedPackageDigest,
      packageRefs: [reusedPackageDocker, reusedPackageGHCR],
    },
  ].sort((left, right) => left.engineId.localeCompare(right.engineId)),
});
const mixed = validateCompositionHandoff({
  manifest, runtime, packages, packageDigests: digestDir, runtimeEvidence, packageEvidence,
  completeManifest, compositionPlan: planFile, composition: compositionFile,
  tag: binding.releaseTag, merge: binding.publicMergeCommit, source, exportManifest, provenance, workflowIdentity: signer,
});
assert.deepEqual(mixed.builtEngineIds, [engineId]);
assert.deepEqual(mixed.reusedEngineIds, [reusedEngine]);

const forged = structuredClone(composition);
forged.components.find((component) => component.id === `${reusedEngine}.runtime`).evidence = evidence("engine-runtime-evidence", reusedEngine);
forged.compositionDigest = sha256Digest(compositionCorePayload(forged));
const forgedFile = write("forged-runtime-composition.json", forged);
assert.throws(() => validateCompositionHandoff({
  manifest, runtime, packages, packageDigests: digestDir, runtimeEvidence, packageEvidence,
  completeManifest, compositionPlan: planFile, composition: forgedFile,
  tag: binding.releaseTag, merge: binding.publicMergeCommit, source, exportManifest, provenance, workflowIdentity: signer,
}), /reused component differs from its original verified plan evidence/);

const mismatchedPackageReceipt = JSON.parse(fs.readFileSync(packages, "utf8"));
mismatchedPackageReceipt.packages[0].engineVersion = `0.0.0+${"f".repeat(64)}`;
const mismatchedPackagePath = write("mismatched-packages.json", mismatchedPackageReceipt);
assert.throws(() => validateCompositionHandoff({
  manifest, runtime, packages: mismatchedPackagePath, packageDigests: digestDir, runtimeEvidence, packageEvidence,
  completeManifest, compositionPlan: planFile, composition: compositionFile,
  tag: binding.releaseTag, merge: binding.publicMergeCommit, source, exportManifest, provenance, workflowIdentity: signer,
}), /does not match fingerprint-derived version/);
console.log("public Engine release self-test passed");
