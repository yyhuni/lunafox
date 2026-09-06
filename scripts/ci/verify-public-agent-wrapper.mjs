#!/usr/bin/env node

/** Validate the public Agent Docker wrapper without needing private binaries. */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const MEMBERS = Object.freeze([
  "lunafox-agent-linux-amd64",
  "lunafox-agent-linux-arm64",
  "lunafox-engine-mount-preflight-linux-amd64",
  "lunafox-engine-mount-preflight-linux-arm64",
]);

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const options = { root: DEFAULT_ROOT, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-public-agent-wrapper.mjs [--root-dir <dir>] [--json]\n");
      process.exit(0);
    }
    if (arg !== "--root-dir") fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail("--root-dir requires a value");
    options.root = path.resolve(value);
  }
  return options;
}

function validate(options) {
  const dockerfilePath = path.join(options.root, "docker/agent/Dockerfile");
  if (!fs.existsSync(dockerfilePath)) fail("public Agent wrapper Dockerfile is missing");
  const text = fs.readFileSync(dockerfilePath, "utf8");
  if (!/^FROM\s+debian:[^\s]+/m.test(text)) fail("public Agent wrapper must use a pinned Debian runtime base");
  if (!/^COPY\s+\.\s+\/agent-assets\//m.test(text)) fail("public Agent wrapper must copy only the prepared binary context");
  if (/agent\/(?:cmd|internal|go\.mod)|go\.mod|go\.sum|private|secret|credential/i.test(text)) {
    fail("public Agent wrapper must not reference Agent source or credentials");
  }
  for (const member of MEMBERS) if (!text.includes(member)) fail(`public Agent wrapper does not validate ${member}`);
  if (!text.includes("TARGETARCH") || !text.includes("linux-${suffix}")) fail("public Agent wrapper must select binaries by TARGETARCH");
  return { schemaVersion: 1, passed: true, dockerfile: "docker/agent/Dockerfile", binaryContextOnly: true, members: [...MEMBERS] };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = validate(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : "public Agent wrapper verified"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); }
  catch (error) { process.stderr.write(`public Agent wrapper verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { MEMBERS, parseArgs, validate };
