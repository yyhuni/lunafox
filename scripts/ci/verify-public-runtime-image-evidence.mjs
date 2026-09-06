#!/usr/bin/env node

/**
 * Verify the immutable evidence emitted by the public Runtime Image lane.
 *
 * The verifier is deliberately registry-free: it validates the source,
 * workflow, manifest, and digest bindings in an evidence record. Registry
 * pullability and cross-registry equality remain separate release gates.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const PRIVATE_REPOSITORY = "yyhuni/lunafox-private";
const PUBLIC_WORKFLOW_IDENTITY = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";
const PRIVATE_AGENT_SIGNER_IDENTITY = "https://github.com/yyhuni/lunafox-private/.github/workflows/release.yml@refs/tags/*";
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const COMMIT_RE = /^[0-9a-f]{40}$/;
const RELEASE_TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const PLATFORM_SET = ["linux/amd64", "linux/arm64"];
const COMPONENTS = Object.freeze({
  server: { repository: "lunafox-server", dockerfile: "server/Dockerfile", lockfile: "" },
  frontend: { repository: "lunafox-frontend", dockerfile: "frontend/Dockerfile", lockfile: "frontend/pnpm-lock.yaml" },
  nginx: { repository: "lunafox-nginx", dockerfile: "docker/nginx/Dockerfile", lockfile: "" },
  agent: { repository: "lunafox-agent", dockerfile: "docker/agent/Dockerfile", lockfile: "", binary: true },
  bootstrap: { repository: "lunafox-bootstrap", dockerfile: "docker/bootstrap/Dockerfile", lockfile: "" },
});
const AGENT_BINARY_MEMBERS = Object.freeze([
  "lunafox-agent-linux-amd64",
  "lunafox-agent-linux-arm64",
  "lunafox-engine-mount-preflight-linux-amd64",
  "lunafox-engine-mount-preflight-linux-arm64",
]);

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const options = {
    evidence: "",
    evidenceDir: "",
    components: Object.keys(COMPONENTS),
    component: "",
    tag: "",
    sourceRevisionDigest: "",
    publicMergeCommit: "",
    workflowRunId: "",
    exportManifestSha256: "",
    publicProvenanceSha256: "",
    dockerfileSha256: "",
    lockfileSha256: "",
    json: false,
  };
  const valueFlags = new Set([
    "--evidence", "--evidence-dir", "--components", "--component", "--tag",
    "--source-revision-digest", "--public-merge-commit", "--workflow-run-id",
    "--export-manifest-sha256", "--public-provenance-sha256", "--dockerfile-sha256",
    "--lockfile-sha256",
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-public-runtime-image-evidence.mjs --evidence <file>|--evidence-dir <dir> --tag <tag> --source-revision-digest <digest> --public-merge-commit <sha> --workflow-run-id <id> --export-manifest-sha256 <digest> --public-provenance-sha256 <digest> --dockerfile-sha256 <digest> [--lockfile-sha256 <digest>] [--component <name>] [--json]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    switch (arg) {
      case "--evidence": options.evidence = path.resolve(value); break;
      case "--evidence-dir": options.evidenceDir = path.resolve(value); break;
      case "--components": options.components = value.split(",").map((item) => item.trim()).filter(Boolean); break;
      case "--component": options.component = value; break;
      case "--tag": options.tag = value; break;
      case "--source-revision-digest": options.sourceRevisionDigest = value; break;
      case "--public-merge-commit": options.publicMergeCommit = value; break;
      case "--workflow-run-id": options.workflowRunId = value; break;
      case "--export-manifest-sha256": options.exportManifestSha256 = value; break;
      case "--public-provenance-sha256": options.publicProvenanceSha256 = value; break;
      case "--dockerfile-sha256": options.dockerfileSha256 = value; break;
      case "--lockfile-sha256": options.lockfileSha256 = value; break;
      default: fail(`unknown argument: ${arg}`);
    }
  }
  if (Boolean(options.evidence) === Boolean(options.evidenceDir)) fail("exactly one of --evidence or --evidence-dir is required");
  if (!RELEASE_TAG_RE.test(options.tag)) fail(`invalid release tag: ${options.tag}`);
  if (options.component && !Object.hasOwn(COMPONENTS, options.component)) fail(`unsupported Runtime Image component: ${options.component}`);
  options.components = options.component ? [options.component] : options.components;
  if (options.components.length === 0 || options.components.some((name) => !Object.hasOwn(COMPONENTS, name))) {
    fail("--components must contain only server, frontend, nginx, agent, bootstrap");
  }
  if (!DIGEST_RE.test(options.sourceRevisionDigest)) fail("--source-revision-digest must be a sha256 digest");
  if (!DIGEST_RE.test(options.exportManifestSha256)) fail("--export-manifest-sha256 must be a sha256 digest");
  if (!DIGEST_RE.test(options.publicProvenanceSha256)) fail("--public-provenance-sha256 must be a sha256 digest");
  if (!DIGEST_RE.test(options.dockerfileSha256)) fail("--dockerfile-sha256 must be a sha256 digest");
  if (options.lockfileSha256 && !DIGEST_RE.test(options.lockfileSha256)) fail("--lockfile-sha256 must be a sha256 digest");
  if (!COMMIT_RE.test(options.publicMergeCommit)) fail("--public-merge-commit must be a 40-character commit SHA");
  if (!/^\d+$/.test(options.workflowRunId)) fail("--workflow-run-id must be numeric");
  return options;
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function sha256File(file) {
  return `sha256:${crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex")}`;
}

function assertDigest(value, field) {
  if (!DIGEST_RE.test(String(value ?? ""))) fail(`${field} must be an immutable sha256 digest`);
}

function expectedForComponent(component) {
  const descriptor = COMPONENTS[component];
  if (!descriptor) fail(`unsupported Runtime Image component: ${component}`);
  return descriptor;
}

function verifyRecord(record, expected) {
  if (!record || typeof record !== "object" || Array.isArray(record)) fail("public Runtime image evidence must be a JSON object");
  const component = String(record.component ?? expected.component ?? "");
  if (!Object.hasOwn(COMPONENTS, component)) fail(`evidence has an unsupported component: ${component}`);
  const descriptor = expectedForComponent(component);
  if (expected.component && component !== expected.component) fail(`evidence component does not match ${expected.component}`);
  const required = [
    "schemaVersion", "status", "component", "sourceRepository", "destinationRepository",
    "sourceRevisionDigest", "publicMergeCommit", "releaseTag", "workflowRun", "workflowIdentity",
    "image", "digest", "publicExportManifestSha256", "publicProvenanceSha256", "dockerfileSha256",
    "dockerHubImage", "platforms", "sbom", "provenance", "attestation", "attestationSubject", "attestationPredicateType",
  ];
  if (descriptor.lockfile) required.push("lockfileSha256");
  for (const field of required) if (record[field] === undefined || record[field] === null || record[field] === "") fail(`evidence is missing ${field} for ${component}`);
  if (record.schemaVersion !== 1 || record.status !== "published") fail(`public Runtime image evidence schema/status is invalid for ${component}`);
  if (record.component !== component) fail(`evidence component field is invalid for ${component}`);
  if (record.sourceRepository !== PRIVATE_REPOSITORY) fail(`${component} evidence source repository drifted`);
  if (record.destinationRepository !== PUBLIC_REPOSITORY) fail(`${component} evidence destination repository drifted`);
  if (expected.tag && record.releaseTag !== expected.tag) fail(`${component} evidence release tag does not match ${expected.tag}`);
  if (expected.sourceRevisionDigest && record.sourceRevisionDigest !== expected.sourceRevisionDigest) fail(`${component} evidence source revision does not match the private export`);
  if (expected.publicMergeCommit && record.publicMergeCommit !== expected.publicMergeCommit) fail(`${component} evidence merge commit does not match the protected public merge`);
  if (record.workflowIdentity !== PUBLIC_WORKFLOW_IDENTITY) fail(`${component} evidence workflow identity is not the canonical public main workflow`);
  if (!COMMIT_RE.test(record.publicMergeCommit)) fail(`${component} evidence publicMergeCommit is invalid`);
  const expectedRun = `https://github.com/${PUBLIC_REPOSITORY}/actions/runs/${expected.workflowRunId}`;
  if (expected.workflowRunId && record.workflowRun !== expectedRun) fail(`${component} evidence workflow run does not match the retrieved public workflow run`);
  assertDigest(record.sourceRevisionDigest, `${component}.sourceRevisionDigest`);
  assertDigest(record.digest, `${component}.digest`);
  assertDigest(record.publicExportManifestSha256, `${component}.publicExportManifestSha256`);
  assertDigest(record.publicProvenanceSha256, `${component}.publicProvenanceSha256`);
  assertDigest(record.dockerfileSha256, `${component}.dockerfileSha256`);
  if (expected.exportManifestSha256 && record.publicExportManifestSha256 !== expected.exportManifestSha256) fail(`${component} evidence manifest digest does not match the private export`);
  if (expected.publicProvenanceSha256 && record.publicProvenanceSha256 !== expected.publicProvenanceSha256) fail(`${component} evidence provenance digest does not match the private export`);
  if (expected.dockerfileSha256 && record.dockerfileSha256 !== expected.dockerfileSha256) fail(`${component} evidence Dockerfile digest does not match the public export`);
  if (descriptor.lockfile) {
    assertDigest(record.lockfileSha256, `${component}.lockfileSha256`);
    if (expected.lockfileSha256 && record.lockfileSha256 !== expected.lockfileSha256) fail(`${component} evidence lockfile digest does not match the public export`);
  } else if (record.lockfileSha256 !== undefined) {
    assertDigest(record.lockfileSha256, `${component}.lockfileSha256`);
  }
  if (record.image !== `ghcr.io/yyhuni/${descriptor.repository}@${record.digest}`) fail(`${component} evidence image is not digest-qualified or has the wrong repository`);
  if (record.dockerHubImage !== `docker.io/yyhuni/${descriptor.repository}@${record.digest}`) fail(`${component} evidence Docker Hub image is not digest-qualified or has the wrong repository`);
  if (record.attestationSubject !== record.image) fail(`${component} evidence attestation subject does not match image`);
  if (record.attestationPredicateType !== "https://slsa.dev/provenance/v1") fail(`${component} evidence attestation predicate type is invalid`);
  if (!Array.isArray(record.platforms) || JSON.stringify([...record.platforms].sort()) !== JSON.stringify(PLATFORM_SET)) fail(`${component} evidence must list linux/amd64 and linux/arm64 exactly`);
  for (const field of ["sbom", "provenance", "attestation"]) if (record[field] !== true) fail(`${component} evidence must include ${field}`);
  if (descriptor.binary) {
    if (record.binaryStagingIdentity !== undefined || record.binaryAsset !== undefined) {
      fail(`${component} evidence must not retain staging Asset provenance`);
    }
    for (const field of ["binaryBundleDigest", "binarySourceRevisionDigest", "binaryExportManifestSha256", "binaryProvenanceSha256", "binaryTreeBaseManifestSha256"]) {
      assertDigest(record[field], `${component}.${field}`);
    }
    if (record.binarySourceRevisionDigest !== record.sourceRevisionDigest ||
        record.binaryExportManifestSha256 !== record.publicExportManifestSha256 ||
        record.binaryProvenanceSha256 !== record.publicProvenanceSha256) {
      fail(`${component} binary provenance is not bound to the public export`);
    }
    if (record.binaryTreePath !== `agent/bin/${record.releaseTag}`) fail(`${component} binary tree path is not bound to the release tag`);
    if (record.binarySignerIssuer !== "https://token.actions.githubusercontent.com" ||
        record.binarySignerIdentity !== PRIVATE_AGENT_SIGNER_IDENTITY) {
      fail(`${component} binary signer identity is invalid`);
    }
    if (JSON.stringify(record.binaryMembers) !== JSON.stringify(AGENT_BINARY_MEMBERS)) {
      fail(`${component} evidence must enumerate the fixed Agent bundle members`);
    }
  }
  return {
    schemaVersion: 1,
    passed: true,
    component,
    repository: PUBLIC_REPOSITORY,
    tag: record.releaseTag,
    publicMergeCommit: record.publicMergeCommit,
    sourceRevisionDigest: record.sourceRevisionDigest,
    digest: record.digest,
    image: record.image,
    workflowRun: record.workflowRun,
  };
}

function expectedFromOptions(options, component) {
  return {
    component,
    tag: options.tag,
    sourceRevisionDigest: options.sourceRevisionDigest,
    publicMergeCommit: options.publicMergeCommit,
    workflowRunId: options.workflowRunId,
    exportManifestSha256: options.exportManifestSha256,
    publicProvenanceSha256: options.publicProvenanceSha256,
    dockerfileSha256: options.dockerfileSha256,
    lockfileSha256: options.lockfileSha256,
  };
}

function recordsFromInput(options) {
  if (options.evidence) return [{ file: options.evidence, record: readJson(options.evidence, "public Runtime image evidence") }];
  if (!fs.existsSync(options.evidenceDir) || !fs.statSync(options.evidenceDir).isDirectory()) fail(`evidence directory is missing: ${options.evidenceDir}`);
  const files = fs.readdirSync(options.evidenceDir).filter((name) => /\.json$/.test(name)).sort();
  if (files.length === 0) fail("evidence directory contains no JSON records");
  return files.map((name) => ({ file: path.join(options.evidenceDir, name), record: readJson(path.join(options.evidenceDir, name), "public Runtime image evidence") }));
}

function verify(options) {
  const inputs = recordsFromInput(options);
  const records = [];
  for (const input of inputs) {
    const candidates = Array.isArray(input.record)
      ? input.record
      : Array.isArray(input.record.records) ? input.record.records
        : Array.isArray(input.record.components) ? input.record.components
          : [input.record];
    for (const record of candidates) records.push({ file: input.file, result: verifyRecord(record, expectedFromOptions(options, record?.component ?? "")), record });
  }
  const expectedComponents = [...new Set(options.components)];
  const selected = records.filter(({ record }) => expectedComponents.includes(record.component));
  if (options.evidence && selected.length !== 1) fail("single evidence input must contain exactly one requested component");
  const seen = new Set();
  for (const item of selected) {
    if (seen.has(item.record.component)) fail(`duplicate public Runtime image evidence: ${item.record.component}`);
    seen.add(item.record.component);
  }
  for (const component of expectedComponents) if (!seen.has(component)) fail(`missing public Runtime image evidence: ${component}`);
  return {
    schemaVersion: 1,
    passed: true,
    components: selected.map(({ result }) => result),
    evidence: selected.map(({ file, record }) => ({ component: record.component, file, evidenceSha256: sha256File(file) })),
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `public Runtime image evidence verified: ${result.components.map((item) => item.component).join(", ")}`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); }
  catch (error) { process.stderr.write(`public Runtime image evidence verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { COMPONENTS, PLATFORM_SET, parseArgs, verify, verifyRecord };
