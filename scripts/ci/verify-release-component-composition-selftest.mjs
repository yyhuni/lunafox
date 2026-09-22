#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { bindCompositionToManifest, compositionCorePayload, FINGERPRINT_SCHEMA_VERSION, sha256Digest, validateComposition } from "./resolve-release-component-composition.mjs";
import { verify } from "./verify-release-component-composition.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-composition-verify-"));
const file = path.join(root, "runtime-composition.json");
const digest = (letter) => `sha256:${letter.repeat(64)}`;

function fingerprint(componentId) {
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
    builderPolicy: {},
    generatedInputs: [],
  };
  return { version: FINGERPRINT_SCHEMA_VERSION, algorithm: "sha256-canonical-json-v1", digest: sha256Digest(inputs), baseImagesResolved: true, inputs };
}

function component(id, kind, name, artifactDigest) {
  const repository = kind === "runtime" ? `lunafox-${name}` : `lunafox-engine-runtime-${name}`;
  return {
    id,
    kind,
    name,
    inputFingerprint: fingerprint(id),
    artifact: { ref: `ghcr.io/yyhuni/${repository}@${artifactDigest}`, digest: artifactDigest },
    disposition: "built",
    sourceRelease: { tag: "v1.2.3" },
    evidence: { image: "image.json", provenance: "provenance.json", sbom: "sbom.json", signature: "signature.json" },
  };
}

const valid = {
  schemaVersion: 1,
  kind: "lunafox.runtime-composition",
  releaseTag: "v1.2.3",
  components: [
    {
      id: "runtime.frontend",
      kind: "runtime",
      name: "frontend",
      inputFingerprint: fingerprint("runtime.frontend"),
      artifact: { ref: `ghcr.io/yyhuni/lunafox-frontend@${digest("b")}`, digest: digest("b") },
      disposition: "built",
      sourceRelease: { tag: "v1.2.3" },
      evidence: { image: "image.json", provenance: "provenance.json", sbom: "sbom.json", signature: "signature.json" },
    },
  ],
  capabilities: { dynamicFrontendUpstream: true },
};
valid.compositionDigest = sha256Digest(compositionCorePayload(valid));

try {
  fs.writeFileSync(file, `${JSON.stringify(valid, null, 2)}\n`);
  const result = verify({ composition: file, expectedComponents: ["runtime.frontend"] });
  assert.equal(result.passed, true);
  assert.equal(result.componentCount, 1);

  const missingClosure = structuredClone(valid);
  delete missingClosure.components[0].inputFingerprint.inputs;
  missingClosure.compositionDigest = sha256Digest(compositionCorePayload(missingClosure));
  assert.throws(() => validateComposition(missingClosure), /inputFingerprint\.inputs must be an object/);

  const incompleteClosure = structuredClone(valid);
  delete incompleteClosure.components[0].inputFingerprint.inputs.generatedInputs;
  incompleteClosure.components[0].inputFingerprint.digest = sha256Digest(incompleteClosure.components[0].inputFingerprint.inputs);
  incompleteClosure.compositionDigest = sha256Digest(compositionCorePayload(incompleteClosure));
  assert.throws(() => validateComposition(incompleteClosure), /inputs\.generatedInputs is required/);

  const inconsistentFingerprint = structuredClone(valid);
  inconsistentFingerprint.components[0].inputFingerprint.inputs.files.push({ path: "frontend/app", digest: digest("f"), size: 1, mode: 0o644 });
  inconsistentFingerprint.compositionDigest = sha256Digest(compositionCorePayload(inconsistentFingerprint));
  assert.throws(() => validateComposition(inconsistentFingerprint), /inputFingerprint\.digest does not match inputs/);

  const bound = bindCompositionToManifest(valid, digest("c"));
  assert.equal(bound.compositionDigest, valid.compositionDigest);
  assert.equal(bound.manifestBinding.manifestDigest, digest("c"));

  const duplicate = structuredClone(valid);
  duplicate.components.push(structuredClone(duplicate.components[0]));
  fs.writeFileSync(file, `${JSON.stringify(duplicate)}\n`);
  assert.throws(() => verify({ composition: file }), /duplicate component/);

  const mutable = structuredClone(valid);
  mutable.components[0].artifact.ref = "ghcr.io/yyhuni/lunafox-frontend:latest";
  fs.writeFileSync(file, `${JSON.stringify(mutable)}\n`);
  assert.throws(() => verify({ composition: file }), /immutable digest-qualified OCI reference/);

  const manifest = path.join(root, "release.manifest.yaml");
  const runtime = [
    ["server", "b"], ["frontend", "c"], ["nginx", "d"], ["agent", "e"], ["bootstrap", "f"],
  ];
  const manifestRuntime = runtime.map(([name, letter]) => `  - name: ${name}\n    refs:\n      - docker.io/yyhuni/lunafox-${name}@${digest(letter)}\n      - ghcr.io/yyhuni/lunafox-${name}@${digest(letter)}\n`).join("");
  fs.writeFileSync(manifest, `releaseVersion: "1.2.3"\nreleaseNotes:\n  digest: "sha256:4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188"\n  body: |\n    ## English\n\n    - Test release notes.\n\n    ## 简体中文\n\n    - 测试发布说明。\nruntimeImages:\n${manifestRuntime}enginePackages:\n  - refs:\n      - docker.io/yyhuni/lunafox-engine-runtime-port-scan@${digest("a")}\n      - ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@${digest("a")}\nruntimeComposition:\n  schemaVersion: 1\n  asset: "runtime-composition.json"\n  sha256: "PLACEHOLDER"\n`);
  const manifestComposition = {
    schemaVersion: 1,
    kind: "lunafox.runtime-composition",
    releaseTag: "v1.2.3",
    components: [
      ...runtime.map(([name, letter]) => component(`runtime.${name}`, "runtime", name, digest(letter))),
      component("engine.port-scan.runtime", "engine", "port-scan", digest("9")),
      component("engine.port-scan.package", "engine", "port-scan", digest("a")),
    ],
    capabilities: { dynamicFrontendUpstream: true },
  };
  manifestComposition.components.sort((left, right) => left.id.localeCompare(right.id));
  manifestComposition.compositionDigest = sha256Digest(compositionCorePayload(manifestComposition));
  fs.writeFileSync(manifest, fs.readFileSync(manifest, "utf8").replace("PLACEHOLDER", manifestComposition.compositionDigest));
  const manifestBound = bindCompositionToManifest(manifestComposition, sha256Digest(fs.readFileSync(manifest)));
  fs.writeFileSync(file, `${JSON.stringify(manifestBound, null, 2)}\n`);
  const boundResult = verify({ composition: file, manifest });
  assert.equal(boundResult.manifestDigest, sha256Digest(fs.readFileSync(manifest)));

  const mismatchedBinding = structuredClone(manifestBound);
  mismatchedBinding.manifestBinding.manifestDigest = digest("0");
  fs.writeFileSync(file, `${JSON.stringify(mismatchedBinding)}\n`);
  assert.throws(() => verify({ composition: file, manifest }), /manifest binding does not match/);

  process.stdout.write("ok - runtime composition validator rejects duplicate, mutable, and manifest-inconsistent component evidence\n");
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
