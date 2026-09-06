#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { COMPONENTS, verify } from "./verify-public-runtime-image-evidence.mjs";

const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-runtime-evidence-selftest-"));
const evidenceDir = path.join(root, "evidence");
fs.mkdirSync(evidenceDir, { recursive: true });
const tag = "v0.0.1-alpha.57";
const sourceRevisionDigest = `sha256:${"a".repeat(64)}`;
const publicMergeCommit = "b".repeat(40);
const exportManifestSha256 = `sha256:${"c".repeat(64)}`;
const publicProvenanceSha256 = `sha256:${"d".repeat(64)}`;
const workflowRunId = "12345";

function digest(letter) { return `sha256:${letter.repeat(64)}`; }

function record(component, overrides = {}) {
  const descriptor = COMPONENTS[component];
  const imageDigest = digest(component === "server" ? "1" : component === "frontend" ? "2" : component === "nginx" ? "3" : "4");
  const value = {
    schemaVersion: 1,
    status: "published",
    component,
    sourceRepository: "yyhuni/lunafox-private",
    destinationRepository: "yyhuni/lunafox",
    sourceRevisionDigest,
    publicMergeCommit,
    releaseTag: tag,
    workflowRun: `https://github.com/yyhuni/lunafox/actions/runs/${workflowRunId}`,
    workflowIdentity: "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main",
    image: `ghcr.io/yyhuni/${descriptor.repository}@${imageDigest}`,
    digest: imageDigest,
    publicExportManifestSha256: exportManifestSha256,
    publicProvenanceSha256,
    dockerfileSha256: digest("e"),
    platforms: ["linux/amd64", "linux/arm64"],
    sbom: true,
    provenance: true,
    attestation: true,
    attestationSubject: `ghcr.io/yyhuni/${descriptor.repository}@${imageDigest}`,
    attestationPredicateType: "https://slsa.dev/provenance/v1",
    dockerHubImage: `docker.io/yyhuni/${descriptor.repository}@${imageDigest}`,
    ...(descriptor.lockfile ? { lockfileSha256: digest("f") } : {}),
    ...(descriptor.binary ? {
      binaryBundleDigest: digest("9"),
      binarySourceRevisionDigest: sourceRevisionDigest,
      binaryExportManifestSha256: exportManifestSha256,
      binaryProvenanceSha256: publicProvenanceSha256,
      binaryTreePath: `agent/bin/${tag}`,
      binaryTreeBaseManifestSha256: digest("8"),
      binarySignerIssuer: "https://token.actions.githubusercontent.com",
      binarySignerIdentity: "https://github.com/yyhuni/lunafox-private/.github/workflows/release.yml@refs/tags/*",
      binaryMembers: [
        "lunafox-agent-linux-amd64",
        "lunafox-agent-linux-arm64",
        "lunafox-engine-mount-preflight-linux-amd64",
        "lunafox-engine-mount-preflight-linux-arm64",
      ],
    } : {}),
    ...overrides,
  };
  return value;
}

function write(component, overrides = {}) {
  fs.writeFileSync(path.join(evidenceDir, `${component}.json`), `${JSON.stringify(record(component, overrides), null, 2)}\n`);
}

try {
  for (const component of Object.keys(COMPONENTS)) write(component);
  const result = verify({
    evidenceDir,
    components: Object.keys(COMPONENTS),
    tag,
    sourceRevisionDigest,
    publicMergeCommit,
    workflowRunId,
    exportManifestSha256,
    publicProvenanceSha256,
    dockerfileSha256: digest("e"),
    lockfileSha256: digest("f"),
  });
  assert.equal(result.passed, true);
  assert.deepEqual(result.components.map((item) => item.component).sort(), Object.keys(COMPONENTS).sort());

  write("nginx", { digest: "latest" });
  assert.throws(() => verify({
    evidence: path.join(evidenceDir, "nginx.json"),
    component: "nginx",
    tag,
    sourceRevisionDigest,
    publicMergeCommit,
    workflowRunId,
    exportManifestSha256,
    publicProvenanceSha256,
    dockerfileSha256: digest("e"),
  }), /immutable sha256 digest/);

  write("bootstrap", { platforms: ["linux/amd64"] });
  assert.throws(() => verify({
    evidence: path.join(evidenceDir, "bootstrap.json"),
    component: "bootstrap",
    tag,
    sourceRevisionDigest,
    publicMergeCommit,
    workflowRunId,
    exportManifestSha256,
    publicProvenanceSha256,
    dockerfileSha256: digest("e"),
  }), /must list linux\/amd64 and linux\/arm64 exactly/);

  write("agent", { binaryStagingIdentity: `agent-input-${tag}` });
  assert.throws(() => verify({
    evidence: path.join(evidenceDir, "agent.json"),
    component: "agent",
    tag,
    sourceRevisionDigest,
    publicMergeCommit,
    workflowRunId,
    exportManifestSha256,
    publicProvenanceSha256,
    dockerfileSha256: digest("e"),
  }), /must not retain staging Asset provenance/);

  process.stdout.write("ok - public Runtime image evidence binds all components, provenance, platforms, and immutable digests\n");
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
