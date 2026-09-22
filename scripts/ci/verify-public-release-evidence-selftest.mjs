#!/usr/bin/env node

import assert from "node:assert/strict";

import { validateReleaseProvenance } from "./verify-public-release-evidence.mjs";

const digest = (letter) => `sha256:${letter.repeat(64)}`;
const facts = {
  tag: "v1.2.3",
  manifestSHA256: digest("a"),
  compositionDigest: digest("b"),
  compositionSHA256: digest("c"),
  bundleSHA256: digest("d"),
};
const valid = {
  schemaVersion: 1,
  status: "published",
  releaseTag: facts.tag,
  artifactDigest: facts.manifestSHA256,
  runtimeComposition: { asset: "runtime-composition.json", digest: facts.compositionDigest, assetSha256: facts.compositionSHA256 },
  componentEvidence: { asset: "component-evidence.json", assetSha256: facts.bundleSHA256 },
  privateSourceRevision: digest("e"),
  publicSourceCommit: "f".repeat(40),
  deploymentSnapshotCommit: "1".repeat(40),
  workflowIdentity: "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main",
  builderRun: "https://github.com/yyhuni/lunafox/actions/runs/123",
  exportManifest: digest("2"),
  publicProvenance: digest("3"),
  sbom: true,
  scanEvidence: { runtimeImages: true, engineHandoff: true, signatures: true },
};

validateReleaseProvenance(valid, facts);
const swappedComposition = structuredClone(valid);
swappedComposition.runtimeComposition.assetSha256 = digest("4");
assert.throws(() => validateReleaseProvenance(swappedComposition, facts), /composition asset digest does not match bytes/);
const unknownField = structuredClone(valid);
unknownField.unbound = true;
assert.throws(() => validateReleaseProvenance(unknownField, facts), /unexpected field set/);
const incompleteEvidence = structuredClone(valid);
incompleteEvidence.scanEvidence.signatures = false;
assert.throws(() => validateReleaseProvenance(incompleteEvidence, facts), /scan evidence is incomplete/);
process.stdout.write("ok - public release provenance binds manifest, composition, and component evidence bytes\n");
