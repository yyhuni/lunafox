#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { validate } from "./verify-public-engine-release.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-public-engine-selftest-"));
const runtimeDigest = "sha256:" + "a".repeat(64);
const archiveDigest = "sha256:" + "b".repeat(64);
const artifactDigest = "sha256:" + "c".repeat(64);
const source = "sha256:" + "d".repeat(64);
const exportManifest = "sha256:" + "e".repeat(64);
const provenance = "sha256:" + "f".repeat(64);
const signer = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";
const engineId = "engine.lunafox.port_scan";
const dockerRuntime = `docker.io/yyhuni/lunafox-engine-runtime-port-scan@${runtimeDigest}`;
const ghcrRuntime = `ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@${runtimeDigest}`;
const dockerPackage = `docker.io/yyhuni/lunafox-engine-port-scan@${artifactDigest}`;
const ghcrPackage = `ghcr.io/yyhuni/lunafox-engine-port-scan@${artifactDigest}`;
const write = (name, value) => { const file = path.join(root, name); fs.mkdirSync(path.dirname(file), { recursive: true }); fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`); return file; };
const binding = { publicMergeCommit: "e".repeat(40), sourceRevisionDigest: source, releaseTag: "v1.2.3", publicExportManifestSha256: exportManifest, publicProvenanceSha256: provenance };
const runtime = write("runtime.json", { schemaVersion: "lunafox.engine-runtime-image-build-results.v1", mode: "production", engines: [{ engineId, indexDigest: runtimeDigest, refs: [dockerRuntime, ghcrRuntime] }] });
const packages = write("packages.json", { schemaVersion: "lunafox.engine-package-build-results.v1", mode: "production", packages: [{ engineId, packageDigest: archiveDigest, runtimeImageDigest: runtimeDigest, runtimeImageRefs: [dockerRuntime, ghcrRuntime] }] });
const runtimeEvidence = write("runtime-evidence.json", { schemaVersion: "lunafox.engine-runtime-image-publication-evidence.v1", passed: true, signerIdentity: signer, ...binding, engines: [{ engineId, indexDigest: runtimeDigest, refs: [dockerRuntime, ghcrRuntime] }] });
const packageEvidence = write("package-evidence.json", { schemaVersion: "lunafox.engine-package-v2-publication-evidence.v1", passed: true, signerIdentity: signer, ...binding, packages: [{ engineId, packageDigest: archiveDigest, artifactManifestDigest: artifactDigest }] });
const digestDir = path.join(root, "digests"); fs.mkdirSync(digestDir); fs.writeFileSync(path.join(digestDir, "port-scan.env"), `ENGINE_ID=${engineId}\nENGINE_REFS=${dockerPackage},${ghcrPackage}\n`);
const manifest = write("manifest.json", { schemaVersion: "lunafox.engine-release-manifest.v1", repository: "yyhuni/lunafox", releaseTag: "v1.2.3", publicMergeCommit: binding.publicMergeCommit, sourceRevisionDigest: source, publicExportManifestSha256: exportManifest, publicProvenanceSha256: provenance, signerIdentity: signer, engines: [{ engineId, runtimeImageDigest: runtimeDigest, runtimeImageRefs: [dockerRuntime, ghcrRuntime], packageDigest: archiveDigest, packageRuntimeImageDigest: runtimeDigest, packageArtifactManifestDigest: artifactDigest, packageRefs: [dockerPackage, ghcrPackage] }] });
const result = validate({ manifest, runtime, packages, packageDigests: digestDir, runtimeEvidence, packageEvidence, tag: "v1.2.3", merge: "e".repeat(40), source, exportManifest, provenance, workflowIdentity: signer });
assert.equal(result.passed, true);
assert.throws(() => validate({ manifest: write("bad-manifest.json", { ...JSON.parse(fs.readFileSync(manifest, "utf8")), engines: [{ ...JSON.parse(fs.readFileSync(manifest, "utf8")).engines[0], packageArtifactManifestDigest: archiveDigest }] }), runtime, packages, packageDigests: digestDir, runtimeEvidence, packageEvidence, tag: "v1.2.3", merge: binding.publicMergeCommit, source, exportManifest, provenance, workflowIdentity: signer }), /Package OCI manifest digest mismatch/);
console.log("public Engine release self-test passed");
