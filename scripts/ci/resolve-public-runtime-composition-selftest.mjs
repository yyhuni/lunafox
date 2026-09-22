#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { bindCompositionToManifest, finalizeCompositionPlan, sha256Digest } from "./resolve-release-component-composition.mjs";
import {
  buildReceipts,
  resolvePlan,
} from "./resolve-public-runtime-composition.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-public-composition-"));
const digest = (letter) => `sha256:${letter.repeat(64)}`;
const write = (relative, content) => {
  const file = path.join(root, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content);
};

try {
  write(".dockerignore", "node_modules\n");
  write("frontend/.dockerignore", "node_modules\n");
  write("server/Dockerfile", "FROM alpine@sha256:" + "a".repeat(64) + "\nCOPY server/app /app\n");
  write("server/app", "server-v1\n");
  write("frontend/Dockerfile", "FROM alpine@sha256:" + "a".repeat(64) + "\nCOPY . /app\n");
  write("frontend/app", "frontend-v1\n");
  write("docker/nginx/Dockerfile", "FROM alpine@sha256:" + "a".repeat(64) + "\nCOPY nginx.conf /etc/nginx/nginx.conf\n");
  write("docker/nginx/nginx.conf", "events {}\n");
  write("docker/bootstrap/Dockerfile", "FROM alpine@sha256:" + "a".repeat(64) + "\nCOPY docker/bootstrap/bootstrap.sh /bootstrap.sh\n");
  write("docker/bootstrap/bootstrap.sh", "#!/bin/sh\n");
  write("docker/agent/Dockerfile", "FROM alpine@sha256:" + "a".repeat(64) + "\nARG AGENT_ARTIFACT_ID\nCOPY agent/bin/${AGENT_ARTIFACT_ID}/lunafox-agent-linux-amd64 /agent\n");
  const artifactId = `sha256-${"b".repeat(64)}`;
  write(`agent/bin/${artifactId}/lunafox-agent-linux-amd64`, "agent\n");

  const evidenceDir = path.join(root, "evidence");
  fs.mkdirSync(evidenceDir);
  for (const [name, letter] of [["server", "1"], ["frontend", "2"], ["nginx", "3"], ["bootstrap", "4"], ["agent", "5"]]) {
    const value = digest(letter);
    fs.writeFileSync(path.join(evidenceDir, `public-runtime-image-evidence-${name}.json`), JSON.stringify({
      component: name,
      image: `ghcr.io/yyhuni/lunafox-${name}@${value}`,
      digest: value,
      platforms: ["linux/amd64", "linux/arm64"],
    }));
  }

  const base = { root, releaseTag: "v1.0.0", sourceRevisionDigest: digest("c"), publicMergeCommit: "d".repeat(40), engineDiscovery: "", baseImages: "" };
  const firstPlan = resolvePlan(base);
  assert.equal(firstPlan.components.length, 5);
  assert.equal(firstPlan.components.every((component) => component.disposition === "built"), true);
  const firstServerFingerprint = firstPlan.components.find((component) => component.id === "runtime.server").inputFingerprint;
  assert.deepEqual(firstServerFingerprint.inputs.baseImages.map((entry) => entry.ref), ["alpine@sha256:" + "a".repeat(64)]);
  assert.equal(firstServerFingerprint.digest, sha256Digest(firstServerFingerprint.inputs));
  assert.equal(Object.hasOwn(firstServerFingerprint, "baseImageRefs"), false);
  const first = finalizeCompositionPlan(firstPlan, buildReceipts({ ...base, runtimeEvidenceDir: evidenceDir }, firstPlan));
  assert.equal(first.components.length, 5);

  // Engine Dockerfiles use global ARG declarations before their first FROM and
  // consume repository named contexts. The public resolver must fingerprint
  // that same BuildKit shape instead of rejecting it as a stage instruction.
  write("contracts/go.sum", "contracts-v1\n");
  write("engine-go/go.sum", "engine-go-v1\n");
  write("extensions/engines/demo/engine.json", '{"engineId":"engine.lunafox.demo"}\n');
  write("extensions/engines/demo/Dockerfile", [
    "ARG TOOLS_GO_VERSION=1.26",
    "FROM --platform=$BUILDPLATFORM alpine:${TOOLS_GO_VERSION} AS builder",
    "COPY --from=contracts go.sum /contracts/go.sum",
    "COPY --from=engine-go go.sum /engine-go/go.sum",
    "FROM builder AS final",
  ].join("\n"));
  const engineDiscovery = path.join(root, "engine-discovery.json");
  fs.writeFileSync(engineDiscovery, JSON.stringify({
    engines: [{ engineId: "engine.lunafox.demo", directory: "demo", dockerfile: "demo/Dockerfile" }],
  }));
  const baseImages = path.join(root, "base-images.json");
  fs.writeFileSync(baseImages, JSON.stringify({ "alpine:1.26": digest("8") }));
  const enginePlan = resolvePlan({ ...base, engineDiscovery, baseImages });
  assert.equal(enginePlan.components.length, 7);
  const engineRuntime = enginePlan.components.find((component) => component.id === "engine.lunafox.demo.runtime");
  assert.deepEqual(engineRuntime.inputFingerprint.inputs.baseImages.map((entry) => entry.ref), ["alpine:1.26"]);
  assert.deepEqual(Object.keys(engineRuntime.inputFingerprint.inputs.namedContexts), ["contracts", "engine-go"]);
  assert.equal(Object.hasOwn(engineRuntime.inputFingerprint, "baseImageRefs"), false);

  write("frontend/app", "frontend-v2\n");
  const previous = path.join(root, "previous.json");
  fs.writeFileSync(previous, `${JSON.stringify(first)}\n`);
  const secondOptions = {
    ...base,
    releaseTag: "v1.0.1",
    sourceRevisionDigest: digest("e"),
    publicMergeCommit: "f".repeat(40),
    previous,
  };
  const secondPlan = resolvePlan(secondOptions);
  const byId = new Map(secondPlan.components.map((component) => [component.id, component]));
  assert.equal(byId.get("runtime.server").disposition, "reused");
  assert.equal(byId.get("runtime.frontend").disposition, "built");
  const reusedServer = byId.get("runtime.server");
  fs.writeFileSync(path.join(evidenceDir, "public-runtime-image-evidence-server.json"), JSON.stringify({
    schemaVersion: 1,
    kind: "lunafox.component-promotion-receipt.v1",
    // A promotion receipt is deliberately not a Runtime build receipt. Keep
    // the human component name here to prove finalization ignores it even if
    // an older producer used that field shape.
    component: "server",
    artifact: reusedServer.artifact,
    evidence: reusedServer.evidence,
    sourceRelease: reusedServer.sourceRelease,
    copiedWithoutRebuild: true,
  }));
  const secondReceipts = buildReceipts({ ...secondOptions, runtimeEvidenceDir: evidenceDir }, secondPlan);
  assert.deepEqual(Object.keys(secondReceipts), ["runtime.frontend"]);
  const second = finalizeCompositionPlan(secondPlan, secondReceipts);
  assert.equal(second.components.find((component) => component.id === "runtime.server").sourceRelease.tag, "v1.0.0");
  const bound = bindCompositionToManifest(second, digest("9"));
  assert.equal(bound.manifestBinding.manifestDigest, digest("9"));
  assert.equal(bound.compositionDigest, second.compositionDigest);
  process.stdout.write("ok - public runtime composition plan/final/bind orchestration and prior reuse\n");
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
