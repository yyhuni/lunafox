#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { resolveCompositionPlan } from "./resolve-release-component-composition.mjs";
import { renderBuildContexts, transportReference } from "./render-release-component-build-contexts.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-build-contexts-"));
const digest = (letter) => `sha256:${letter.repeat(64)}`;
const write = (relative, content) => {
  const file = path.join(root, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content);
};

function planFor(dockerfile, baseImageIdentities) {
  return resolveCompositionPlan({
    root,
    releaseTag: "v1.0.0",
    sourceRevisionDigest: digest("1"),
    publicMergeCommit: "a".repeat(40),
    components: [{
      id: "runtime.demo",
      kind: "runtime",
      name: "demo",
      contextPath: "runtime",
      dockerfile,
      dockerignore: "runtime/.dockerignore",
      namedContexts: { contracts: { path: "contracts" } },
      buildArgs: {},
      platforms: ["linux/amd64", "linux/arm64"],
      baseImageIdentities,
      builderPolicy: { provenance: "mode=max", sbom: true },
    }],
  });
}

try {
  write("contracts/module.go", "package contracts\n");
  write("runtime/.dockerignore", "ignored\n");
  write("runtime/app", "v1\n");
  write("runtime/ignored", "ignored\n");
  write("runtime/Dockerfile", [
    "ARG BASE=alpine:3.20",
    "FROM ${BASE} AS builder",
    "COPY app /app",
    "FROM builder AS final",
  ].join("\n"));

  const mutableDigest = digest("b");
  const mutablePlan = planFor("runtime/Dockerfile", { "alpine:3.20": mutableDigest });
  assert.deepEqual(renderBuildContexts({
    root,
    plan: mutablePlan,
    componentID: "runtime.demo",
    baseImages: { "alpine:3.20": mutableDigest },
  }), [`alpine:3.20=docker-image://alpine:3.20@${mutableDigest}`]);

  write("runtime/Dockerfile", [
    "ARG BASE=public.ecr.aws/docker/library/alpine:3.20",
    "FROM ${BASE} AS builder",
    "COPY app /app",
    "FROM builder AS final",
  ].join("\n"));
  const ecrPlan = planFor("runtime/Dockerfile", { "public.ecr.aws/docker/library/alpine:3.20": mutableDigest });
  assert.equal(
    transportReference("public.ecr.aws/docker/library/alpine:3.20", "dockerhub"),
    "docker.io/library/alpine:3.20",
  );
  assert.deepEqual(renderBuildContexts({
    root,
    plan: ecrPlan,
    componentID: "runtime.demo",
    baseImages: { "public.ecr.aws/docker/library/alpine:3.20": mutableDigest },
    transport: "dockerhub",
  }), [`public.ecr.aws/docker/library/alpine:3.20=docker-image://docker.io/library/alpine:3.20@${mutableDigest}`]);
  assert.throws(
    () => transportReference("public.ecr.aws/other/alpine:3.20", "dockerhub"),
    /unsupported Public ECR transport mapping/,
  );
  assert.throws(
    () => renderBuildContexts({
      root,
      plan: ecrPlan,
      componentID: "runtime.demo",
      baseImages: { "public.ecr.aws/docker/library/alpine:3.20": digest("c") },
      transport: "dockerhub",
    }),
    /does not match the verified plan/,
  );

  const pinnedEcrDigest = digest("e");
  write("runtime/Dockerfile", `FROM public.ecr.aws/docker/library/alpine:3.20@${pinnedEcrDigest}\nCOPY app /app\n`);
  const pinnedEcrPlan = planFor("runtime/Dockerfile", {});
  assert.deepEqual(renderBuildContexts({
    root,
    plan: pinnedEcrPlan,
    componentID: "runtime.demo",
    baseImages: {},
    transport: "dockerhub",
  }), [`public.ecr.aws/docker/library/alpine:3.20@${pinnedEcrDigest}=docker-image://docker.io/library/alpine:3.20@${pinnedEcrDigest}`]);

  write("runtime/Dockerfile", [
    "ARG BASE=alpine:3.20",
    "FROM ${BASE} AS builder",
    "COPY app /app",
    "FROM builder AS final",
  ].join("\n"));
  assert.throws(
    () => renderBuildContexts({ root, plan: mutablePlan, componentID: "runtime.demo", baseImages: {} }),
    /base image identity is missing/,
  );
  assert.throws(
    () => renderBuildContexts({ root, plan: mutablePlan, componentID: "runtime.demo", baseImages: { "alpine:3.20": digest("c") } }),
    /does not match the verified plan/,
  );

  // The renderer reparses the on-disk Dockerfile instead of trusting only the
  // stored plan, so a changed tag is rejected before BuildKit starts.
  write("runtime/Dockerfile", "FROM alpine:3.21\nCOPY app /app\n");
  assert.throws(
    () => renderBuildContexts({ root, plan: mutablePlan, componentID: "runtime.demo", baseImages: { "alpine:3.20": mutableDigest } }),
    /build inputs no longer match/,
  );

  const pinnedDigest = digest("d");
  write("runtime/Dockerfile", `FROM alpine@${pinnedDigest}\nCOPY app /app\n`);
  const pinnedPlan = planFor("runtime/Dockerfile", {});
  assert.deepEqual(renderBuildContexts({ root, plan: pinnedPlan, componentID: "runtime.demo", baseImages: {} }), []);

  process.stdout.write("ok - release component BuildKit contexts are plan-bound, Dockerfile-bound, and digest-pinned\n");
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
