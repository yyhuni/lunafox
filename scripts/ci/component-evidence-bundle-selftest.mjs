#!/usr/bin/env node

import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { compositionCorePayload, FINGERPRINT_SCHEMA_VERSION, sha256Digest } from "./resolve-release-component-composition.mjs";
import { build } from "./build-component-evidence-bundle.mjs";
import { verify } from "./verify-component-evidence-bundle.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-component-evidence-"));
const digest = (letter) => `sha256:${letter.repeat(64)}`;
const signer = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";
const write = (relative, value) => {
  const file = path.join(root, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, typeof value === "string" ? value : `${JSON.stringify(value, null, 2)}\n`);
  return file;
};
const runtimeEvidence = (component, artifactDigest, tag) => ({
  schemaVersion: 1,
  status: "published",
  component,
  sourceRepository: "yyhuni/lunafox-private",
  destinationRepository: "yyhuni/lunafox",
  sourceRevisionDigest: digest("a"),
  publicMergeCommit: "b".repeat(40),
  releaseTag: tag,
  workflowRun: "https://github.com/yyhuni/lunafox/actions/runs/123",
  workflowIdentity: signer,
  image: `ghcr.io/yyhuni/lunafox-${component}@${artifactDigest}`,
  dockerHubImage: `docker.io/yyhuni/lunafox-${component}@${artifactDigest}`,
  digest: artifactDigest,
  publicExportManifestSha256: digest("c"),
  publicProvenanceSha256: digest("d"),
  dockerfileSha256: digest("e"),
  platforms: ["linux/amd64", "linux/arm64"],
  sbom: true,
  provenance: true,
  attestation: true,
  attestationSubject: `ghcr.io/yyhuni/lunafox-${component}@${artifactDigest}`,
  attestationPredicateType: "https://slsa.dev/provenance/v1",
});

const fingerprint = (componentId) => {
  const kind = componentId.split(".")[0];
  const inputs = {
    schemaVersion: FINGERPRINT_SCHEMA_VERSION, componentId, kind, contextPath: ".",
    dockerfile: `${componentId.replaceAll(".", "/")}/Dockerfile`, dockerignore: "",
    files: [], namedContexts: {}, buildArgs: {}, platforms: ["linux/amd64", "linux/arm64"],
    baseImages: [], baseImagesResolved: true, builderPolicy: {}, generatedInputs: [],
  };
  return { version: FINGERPRINT_SCHEMA_VERSION, algorithm: "sha256-canonical-json-v1", digest: sha256Digest(inputs), baseImagesResolved: true, inputs };
};

function component(id, kind, name, artifactDigest, evidence, disposition = "built", sourceTag = "v1.0.0") {
  return {
    id, kind, name,
    inputFingerprint: fingerprint(id),
    artifact: { ref: `ghcr.io/yyhuni/${kind === "engine" ? (id.endsWith(".package") ? "lunafox-engine-runtime-port-scan" : "lunafox-engine-runtime-port-scan") : `lunafox-${name}`}@${artifactDigest}`, digest: artifactDigest },
    disposition,
    sourceRelease: { tag: sourceTag, ...(disposition === "reused" ? { compositionDigest: digest("9") } : {}) },
    evidence,
  };
}

try {
  const runtimeDir = path.join(root, "runtime");
  fs.mkdirSync(runtimeDir, { recursive: true });
  const runtimeDigest = digest("1");
  const agentDigest = digest("2");
  const engineRuntimeDigest = digest("3");
  const enginePackageDigest = digest("4");
  write("runtime/public-runtime-image-evidence-server.json", runtimeEvidence("server", runtimeDigest, "v1.0.0"));
  write("runtime/public-runtime-image-evidence-agent.json", runtimeEvidence("agent", agentDigest, "v1.0.0"));
  const engineRuntime = write("engine-runtime.json", {
    schemaVersion: "lunafox.engine-runtime-image-publication-evidence.v1",
    passed: true, signerIdentity: signer, releaseTag: "v1.0.0", engines: [{ engineId: "engine.lunafox.port_scan", indexDigest: engineRuntimeDigest, refs: [`ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@${engineRuntimeDigest}`] }],
  });
  const enginePackage = write("engine-package.json", {
    schemaVersion: "lunafox.engine-package-v2-publication-evidence.v1",
    passed: true, signerIdentity: signer, releaseTag: "v1.0.0", packages: [{ engineId: "engine.lunafox.port_scan", artifactManifestDigest: enginePackageDigest, candidates: [{ ref: `ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@${enginePackageDigest}` }] }],
  });
  const evidence = { image: "runtime-evidence/server.json", provenance: "runtime-evidence/server.json#provenance", sbom: "runtime-evidence/server.json#sbom", signature: "runtime-evidence/server.json#signature" };
  const agentEvidence = { image: "runtime-evidence/agent.json", provenance: "runtime-evidence/agent.json#provenance", sbom: "runtime-evidence/agent.json#sbom", signature: "runtime-evidence/agent.json#signature" };
  const engineRuntimeEvidence = { image: "engine-runtime-evidence/engine-lunafox-port_scan.json", provenance: "engine-runtime-evidence/engine-lunafox-port_scan.json#provenance", sbom: "engine-runtime-evidence/engine-lunafox-port_scan.json#sbom", signature: "engine-runtime-evidence/engine-lunafox-port_scan.json#signature" };
  const enginePackageEvidence = { image: "engine-package-evidence/engine-lunafox-port_scan.json", provenance: "engine-package-evidence/engine-lunafox-port_scan.json#provenance", sbom: "engine-package-evidence/engine-lunafox-port_scan.json#sbom", signature: "engine-package-evidence/engine-lunafox-port_scan.json#signature" };
  const composition = {
    schemaVersion: 1, kind: "lunafox.runtime-composition", releaseTag: "v1.0.0", components: [
      component("runtime.server", "runtime", "server", runtimeDigest, evidence),
      component("runtime.agent", "runtime", "agent", agentDigest, agentEvidence),
      component("engine.lunafox.port_scan.runtime", "engine", "port_scan", engineRuntimeDigest, engineRuntimeEvidence),
      component("engine.lunafox.port_scan.package", "engine", "port_scan", enginePackageDigest, enginePackageEvidence),
    ], capabilities: { dynamicFrontendUpstream: true },
  };
  composition.components.sort((left, right) => left.id.localeCompare(right.id));
  composition.compositionDigest = sha256Digest(compositionCorePayload(composition));
  composition.manifestBinding = { manifestDigest: digest("8") };
  const compositionPath = write("composition.json", composition);
  const bundlePath = path.join(root, "component-evidence.json");
  build({ composition: compositionPath, runtimeEvidenceDir: runtimeDir, engineRuntimeEvidence: engineRuntime, enginePackageEvidence: enginePackage, output: bundlePath });
  const verified = verify({ bundle: bundlePath, composition: compositionPath });
  assert.equal(verified.passed, true);
  assert.equal(verified.componentCount, 4);

  const reused = structuredClone(composition);
  reused.releaseTag = "v1.0.1";
  reused.components = reused.components.map((entry) => ({ ...entry, disposition: "reused", sourceRelease: { tag: "v1.0.0", compositionDigest: composition.compositionDigest } }));
  reused.components.sort((left, right) => left.id.localeCompare(right.id));
  reused.compositionDigest = sha256Digest(compositionCorePayload(reused));
  reused.manifestBinding = { manifestDigest: digest("7") };
  const reusedCompositionPath = write("reused-composition.json", reused);
  write("runtime/public-runtime-image-evidence-server.json", { schemaVersion: 1, kind: "lunafox.component-promotion-receipt.v1" });
  const reusedBundlePath = path.join(root, "reused-component-evidence.json");
  build({ composition: reusedCompositionPath, runtimeEvidenceDir: runtimeDir, previousBundle: bundlePath, output: reusedBundlePath });
  const reusedVerified = verify({ bundle: reusedBundlePath, composition: reusedCompositionPath });
  assert.deepEqual(reusedVerified.reusedComponents.sort(), reused.components.map((entry) => entry.id).sort());

  const tampered = JSON.parse(fs.readFileSync(reusedBundlePath, "utf8"));
  const tamperedBytes = Buffer.from("tampered");
  tampered.assets[0].contentBase64 = tamperedBytes.toString("base64");
  tampered.assets[0].byteLength = tamperedBytes.length;
  fs.writeFileSync(reusedBundlePath, `${JSON.stringify(tampered)}\n`);
  assert.throws(() => verify({ bundle: reusedBundlePath, composition: reusedCompositionPath }), /sha256 does not match decoded content/);

  const orphaned = JSON.parse(fs.readFileSync(bundlePath, "utf8"));
  const orphanBytes = Buffer.from('{"orphan":true}\n');
  orphaned.assets.push({
    ref: "evidence/orphan.json",
    sha256: `sha256:${crypto.createHash("sha256").update(orphanBytes).digest("hex")}`,
    mediaType: "application/json",
    byteLength: orphanBytes.length,
    contentBase64: orphanBytes.toString("base64"),
  });
  fs.writeFileSync(bundlePath, `${JSON.stringify(orphaned)}\n`);
  assert.throws(() => verify({ bundle: bundlePath, composition: compositionPath }), /orphan assets/);
  process.stdout.write("ok - component evidence bundle preserves original Runtime/Agent/Engine proof and rejects promotion, tampering, and orphan evidence\n");
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
