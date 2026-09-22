#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import {
  FINGERPRINT_ALGORITHM,
  FINGERPRINT_SCHEMA_VERSION,
  buildInputFingerprint,
  compositionCorePayload,
  finalizeCompositionPlan,
  resolveComposition,
  resolveCompositionPlan,
  sha256Digest,
  validateComposition,
} from "./resolve-release-component-composition.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-runtime-composition-"));

function write(relative, content) {
  const file = path.join(root, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content);
}

function digest(letter) {
  return `sha256:${letter.repeat(64)}`;
}

function artifact(name, letter) {
  return {
    ref: `ghcr.io/yyhuni/lunafox-${name}@${digest(letter)}`,
    digest: digest(letter),
  };
}

function evidence(name) {
  return {
    image: `evidence/${name}.json`,
    provenance: `evidence/${name}.provenance.json`,
    sbom: `evidence/${name}.sbom.json`,
    signature: `evidence/${name}.sigstore.json`,
  };
}

function spec(name, letter) {
  return {
    id: `runtime.${name}`,
    kind: "runtime",
    name,
    contextPath: name,
    dockerfile: `${name}/Dockerfile`,
    dockerignore: `${name}/.dockerignore`,
    buildArgs: { NODE_ENV: "production" },
    platforms: ["linux/arm64", "linux/amd64"],
    baseImageIdentities: { "debian:bookworm-slim": "docker.io/library/debian@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" },
    builderPolicy: { provenance: "mode=max", sbom: true },
    artifact: artifact(name, letter),
    evidence: evidence(name),
  };
}

try {
  write("server/Dockerfile", "FROM debian:bookworm-slim\nCOPY app /app\n");
  write("server/.dockerignore", "ignored.txt\n");
  write("server/app", "server-v1\n");
  write("server/unused.txt", "unused-v1\n");
  write("server/ignored.txt", "ignored-v1\n");
  write("frontend/Dockerfile", "FROM debian:bookworm-slim\nCOPY app /app\n");
  write("frontend/.dockerignore", "ignored.txt\n");
  write("frontend/app", "frontend-v1\n");
  write("frontend/ignored.txt", "ignored-v1\n");

  const first = resolveComposition({
    root,
    releaseTag: "v1.0.0",
    sourceRevisionDigest: digest("1"),
    publicMergeCommit: "b".repeat(40),
    capabilities: { dynamicFrontendUpstream: true },
    components: [spec("server", "2"), spec("frontend", "3")],
  });
  assert.equal(first.components.every((component) => component.disposition === "built"), true);
  assert.match(first.compositionDigest, /^sha256:[a-f0-9]{64}$/);
  assert.equal(validateComposition(first).components.length, 2);
  assert.equal(first.components[0].inputFingerprint.version, FINGERPRINT_SCHEMA_VERSION);

  // A frontend-only source change produces a plan where only the frontend
  // still needs a new artifact/evidence receipt.
  write("frontend/app", "frontend-v2\n");
  const buildPlan = resolveCompositionPlan({
    root,
    releaseTag: "v1.0.1",
    sourceRevisionDigest: digest("4"),
    publicMergeCommit: "c".repeat(40),
    capabilities: { dynamicFrontendUpstream: true },
    previous: first,
    components: [spec("server", "9"), spec("frontend", "a")],
  });
  const planned = new Map(buildPlan.components.map((component) => [component.id, component]));
  assert.equal(planned.get("runtime.server").disposition, "reused");
  assert.equal(Object.hasOwn(planned.get("runtime.frontend"), "artifact"), false);
  assert.equal(Object.hasOwn(planned.get("runtime.frontend"), "evidence"), false);
  const finalized = finalizeCompositionPlan(buildPlan, {
    "runtime.frontend": { artifact: artifact("frontend", "a"), evidence: evidence("frontend") },
  });
  assert.equal(finalized.components.find((component) => component.id === "runtime.server").disposition, "reused");
  assert.equal(finalized.components.find((component) => component.id === "runtime.frontend").disposition, "built");
  const finalizedFrontendFingerprint = finalized.components.find((component) => component.id === "runtime.frontend").inputFingerprint;
  assert.equal(Object.hasOwn(finalizedFrontendFingerprint, "inputs"), true);
  assert.equal(Object.hasOwn(finalizedFrontendFingerprint, "baseImageRefs"), false);
  assert.equal(finalizedFrontendFingerprint.digest, sha256Digest(finalizedFrontendFingerprint.inputs));

  // The final composition carries the original server artifact/evidence
  // forward rather than manufacturing a current-release build receipt.
  const second = resolveComposition({
    root,
    releaseTag: "v1.0.1",
    sourceRevisionDigest: digest("4"),
    publicMergeCommit: "c".repeat(40),
    capabilities: { dynamicFrontendUpstream: true },
    previous: first,
    components: [spec("server", "9"), spec("frontend", "a")],
  });
  const secondById = new Map(second.components.map((component) => [component.id, component]));
  assert.equal(secondById.get("runtime.server").disposition, "reused");
  assert.equal(secondById.get("runtime.server").artifact.digest, artifact("server", "2").digest);
  assert.equal(secondById.get("runtime.server").sourceRelease.tag, "v1.0.0");
  assert.equal(secondById.get("runtime.server").sourceRelease.compositionDigest, first.compositionDigest);
  assert.equal(secondById.get("runtime.frontend").disposition, "built");

  const missingReuseBinding = structuredClone(second);
  delete missingReuseBinding.components.find((component) => component.id === "runtime.server").sourceRelease.compositionDigest;
  missingReuseBinding.compositionDigest = sha256Digest(compositionCorePayload(missingReuseBinding));
  assert.throws(() => validateComposition(missingReuseBinding), /reused artifact source composition digest is required/);

  // Docker-ignored content is not part of the effective closure.
  const ignoredBefore = buildInputFingerprint(root, spec("server", "9")).digest;
  write("server/ignored.txt", "ignored-v2\n");
  assert.equal(buildInputFingerprint(root, spec("server", "9")).digest, ignoredBefore);

  // Docker COPY preserves executable permission bits. A mode-only change must
  // therefore invalidate the component fingerprint even when file bytes stay
  // identical.
  const modePath = path.join(root, "server", "app");
  fs.chmodSync(modePath, 0o644);
  const nonExecutable = buildInputFingerprint(root, spec("server", "9"));
  fs.chmodSync(modePath, 0o755);
  const executable = buildInputFingerprint(root, spec("server", "9"));
  assert.equal(nonExecutable.inputs.files.find((file) => file.path === "server/app").mode, 0o644);
  assert.equal(executable.inputs.files.find((file) => file.path === "server/app").mode, 0o755);
  assert.notEqual(nonExecutable.digest, executable.digest);
  fs.chmodSync(modePath, 0o644);

  // A resolved base-image identity is part of the fingerprint.
  const baseChanged = { ...spec("server", "9"), baseImageIdentities: { "debian:bookworm-slim": "docker.io/library/debian@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" } };
  assert.notEqual(buildInputFingerprint(root, spec("server", "9")).digest, buildInputFingerprint(root, baseChanged).digest);

  // A shared generated input changes every consumer that declares it.
  write("shared/generated.json", "v1\n");
  const withShared = { ...spec("server", "9"), generatedInputs: [{ name: "shared/generated.json", file: "shared/generated.json" }] };
  const sharedBefore = buildInputFingerprint(root, withShared).digest;
  write("shared/generated.json", "v2\n");
  assert.notEqual(sharedBefore, buildInputFingerprint(root, withShared).digest);

  // A non-ignored file that is outside the Docker COPY closure must not
  // classify the component as changed, while a referenced file must.
  const closureBefore = buildInputFingerprint(root, spec("server", "9")).digest;
  write("server/unused.txt", "unused-v2\n");
  assert.equal(buildInputFingerprint(root, spec("server", "9")).digest, closureBefore);
  write("server/app", "server-v2\n");
  assert.notEqual(buildInputFingerprint(root, spec("server", "9")).digest, closureBefore);

  // ADD and named build contexts are part of the same effective closure.
  write("add/Dockerfile", "FROM debian:bookworm-slim\nADD payload /payload\n");
  write("add/.dockerignore", "ignored.txt\n");
  write("add/payload", "payload-v1\n");
  write("add/ignored.txt", "ignored-v1\n");
  const addSpec = spec("add", "b");
  const addBefore = buildInputFingerprint(root, addSpec).digest;
  write("add/ignored.txt", "ignored-v2\n");
  assert.equal(buildInputFingerprint(root, addSpec).digest, addBefore);
  write("add/payload", "payload-v2\n");
  assert.notEqual(buildInputFingerprint(root, addSpec).digest, addBefore);

  write("named/Dockerfile", "FROM debian:bookworm-slim\nCOPY --from=assets data /data\n");
  write("named/.dockerignore", "\n");
  write("assets/data", "asset-v1\n");
  write("assets/.dockerignore", "ignored.txt\n");
  write("assets/ignored.txt", "ignored-v1\n");
  const namedSpec = {
    ...spec("named", "c"),
    dockerfile: "named/Dockerfile",
    namedContexts: { assets: { path: "assets", dockerignore: "assets/.dockerignore" } },
  };
  const namedBefore = buildInputFingerprint(root, namedSpec).digest;
  write("assets/ignored.txt", "ignored-v2\n");
  assert.equal(buildInputFingerprint(root, namedSpec).digest, namedBefore);
  write("assets/data", "asset-v2\n");
  assert.notEqual(buildInputFingerprint(root, namedSpec).digest, namedBefore);
  assert.throws(() => buildInputFingerprint(root, {
    ...namedSpec,
    namedContexts: { assets: { path: "missing-assets" } },
  }), /build context does not exist/);
  write("named/Dockerfile", "FROM debian:bookworm-slim\nCOPY missing /data\n");
  assert.throws(() => buildInputFingerprint(root, namedSpec), /does not resolve to an effective build-context file/);
  write("named/Dockerfile", "FROM debian:bookworm-slim\nADD https://example.invalid/payload /payload\n");
  assert.throws(() => buildInputFingerprint(root, namedSpec), /uses a remote source/);

  assert.throws(() => resolveComposition({
    root,
    releaseTag: "v1.0.2",
    components: [{ ...spec("server", "9"), artifact: { ref: "ghcr.io/yyhuni/lunafox-server:latest", digest: digest("9") } }],
  }), /immutable digest-qualified OCI reference/);

  const fabricatedBuiltSource = structuredClone(first);
  fabricatedBuiltSource.releaseTag = "v1.0.2";
  fabricatedBuiltSource.components = fabricatedBuiltSource.components.map((entry) => ({
    ...entry,
    disposition: "built",
    sourceRelease: { tag: "v1.0.0" },
  }));
  fabricatedBuiltSource.compositionDigest = sha256Digest(compositionCorePayload(fabricatedBuiltSource));
  assert.throws(() => validateComposition(fabricatedBuiltSource), /built component must originate from the current release/);

  const brokenPrevious = structuredClone(first);
  brokenPrevious.components[0].evidence = { image: "only-image.json" };
  assert.throws(() => resolveComposition({
    root,
    releaseTag: "v1.0.2",
    previous: brokenPrevious,
    components: [spec("server", "9"), spec("frontend", "a")],
  }), /evidence\.(provenance|sbom|signature) is required/);

  const duplicated = structuredClone(first);
  duplicated.components.push(structuredClone(duplicated.components[0]));
  assert.throws(() => validateComposition(duplicated), /duplicate component/);

  assert.equal(FINGERPRINT_ALGORITHM, "sha256-canonical-json-v1");
  assert.match(sha256Digest({ b: 2, a: 1 }), /^sha256:[a-f0-9]{64}$/);
  process.stdout.write("ok - runtime component composition fingerprints, reuse, evidence, and fail-closed validation\n");
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
