#!/usr/bin/env node

import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const defaultRepoRoot = path.resolve(scriptDir, "../..");

const developmentPolicy = {
  schemaVersion: 1,
  phase: "disposable-development",
  firstStableReleaseAllowed: false,
  frozenMajorBaselines: [],
};

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  const options = {
    repoRoot: defaultRepoRoot,
    releaseChannel: "development",
    printBaseline: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const flag = argv[index];
    if (flag === "--print-baseline") {
      options.printBaseline = true;
      continue;
    }
    const value = argv[index + 1];
    if (!value) fail(`missing value for ${flag}`);
    if (flag === "--repo-root") options.repoRoot = path.resolve(value);
    else if (flag === "--release-channel") options.releaseChannel = value;
    else fail(`unknown argument: ${flag}`);
    index += 1;
  }

  if (!["development", "canary", "stable"].includes(options.releaseChannel)) {
    fail(`unsupported release channel: ${options.releaseChannel}`);
  }
  return options;
}

function readJSON(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    fail(`cannot read ${label} ${filePath}: ${error.message}`);
  }
}

function requireExactKeys(value, keys, label) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(`${label} must be an object`);
  }
  const actual = Object.keys(value).sort();
  const expected = [...keys].sort();
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    fail(`${label} has an invalid field set`);
  }
}

function readEngineAPIMajor(filePath, label) {
  let source;
  try {
    source = fs.readFileSync(filePath, "utf8");
  } catch (error) {
    fail(`cannot read ${label} ${filePath}: ${error.message}`);
  }
  const matches = [...source.matchAll(/\bEngineAPIMajor\s+uint32\s*=\s*(\d+)\b/g)];
  if (matches.length !== 1) {
    fail(`${label} must define exactly one active Engine API major`);
  }
  const major = Number(matches[0][1]);
  if (!Number.isSafeInteger(major) || major <= 0) {
    fail(`${label} has an invalid active Engine API major`);
  }
  return major;
}

function normalizeProtoSource(source) {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/\/\/[^\r\n]*/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

function listProtoFiles(root) {
  if (!fs.existsSync(root) || !fs.statSync(root).isDirectory()) {
    fail(`Engine API proto root is missing: ${root}`);
  }
  const files = [];
  const walk = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((left, right) => left.name.localeCompare(right.name))) {
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) {
        walk(entryPath);
      } else if (entry.isFile() && entry.name.endsWith(".proto")) {
        files.push(path.relative(root, entryPath).split(path.sep).join("/"));
      }
    }
  };
  walk(root);
  if (files.length === 0) fail(`Engine API proto root has no proto files: ${root}`);
  return files;
}

function calculateMajorBaseline(repoRoot, major) {
  const protoRoot = path.join(repoRoot, "proto/lunafox/engine/execution", `v${major}`);
  const protoFiles = listProtoFiles(protoRoot);
  const hash = crypto.createHash("sha256");
  for (const relativePath of protoFiles) {
    const source = fs.readFileSync(path.join(protoRoot, relativePath), "utf8");
    hash.update(relativePath);
    hash.update("\0");
    hash.update(normalizeProtoSource(source));
    hash.update("\0");
  }
  return {
    schemaVersion: 1,
    major,
    protoFiles,
    sourceSha256: hash.digest("hex"),
  };
}

function readPolicy(repoRoot) {
  const policyPath = path.join(repoRoot, "contracts/engineapi/version/engine-api-major-policy.json");
  const policy = readJSON(policyPath, "Engine API major policy");
  if (policy?.phase === "disposable-development") {
    try {
      assert.deepStrictEqual(policy, developmentPolicy);
    } catch {
      fail("Engine API disposable-development policy drift requires an explicit stable-major governance change");
    }
    return policy;
  }

  requireExactKeys(policy, ["schemaVersion", "phase", "firstStableReleaseAllowed", "frozenMajorBaselines"], "Engine API major policy");
  if (policy.schemaVersion !== 1 || policy.phase !== "stable" || policy.firstStableReleaseAllowed !== true) {
    fail("Engine API major policy must enter stable mode only through an explicit baseline freeze");
  }
  if (!Array.isArray(policy.frozenMajorBaselines) || policy.frozenMajorBaselines.length === 0) {
    fail("stable Engine API major policy requires at least one frozen major baseline");
  }
  return policy;
}

function validateStableBaselines(repoRoot, activeMajor, baselines) {
  const seenMajors = new Set();
  let largestFrozenMajor = 0;
  for (const baseline of baselines) {
    requireExactKeys(baseline, ["schemaVersion", "major", "protoFiles", "sourceSha256"], "frozen Engine API major baseline");
    if (baseline.schemaVersion !== 1 || !Number.isSafeInteger(baseline.major) || baseline.major <= 0) {
      fail("frozen Engine API major baseline has an invalid major");
    }
    if (seenMajors.has(baseline.major)) fail(`duplicate frozen Engine API major baseline: v${baseline.major}`);
    seenMajors.add(baseline.major);
    largestFrozenMajor = Math.max(largestFrozenMajor, baseline.major);
    if (!Array.isArray(baseline.protoFiles) || baseline.protoFiles.length === 0 ||
      !baseline.protoFiles.every((file) => typeof file === "string" && file.endsWith(".proto")) ||
      JSON.stringify([...baseline.protoFiles].sort()) !== JSON.stringify(baseline.protoFiles)) {
      fail(`frozen Engine API major v${baseline.major} baseline has an invalid proto file set`);
    }
    if (typeof baseline.sourceSha256 !== "string" || !/^[a-f0-9]{64}$/.test(baseline.sourceSha256)) {
      fail(`frozen Engine API major v${baseline.major} baseline has an invalid source hash`);
    }

    const actual = calculateMajorBaseline(repoRoot, baseline.major);
    if (JSON.stringify(actual.protoFiles) !== JSON.stringify(baseline.protoFiles) || actual.sourceSha256 !== baseline.sourceSha256) {
      fail(`frozen Engine API major v${baseline.major} changed; define a new Engine API major instead`);
    }
  }
  if (activeMajor < largestFrozenMajor) {
    fail(`active Engine API major v${activeMajor} cannot be older than frozen major v${largestFrozenMajor}`);
  }
}

function validateEngineAPIMajorPolicy(repoRoot, releaseChannel = "development") {
  const protocolMajor = readEngineAPIMajor(path.join(repoRoot, "engine-go/protocol/abi.go"), "engine-go protocol");
  const contractMajor = readEngineAPIMajor(path.join(repoRoot, "contracts/engineapi/version/major.go"), "Engine API version contract");
  if (protocolMajor !== contractMajor) {
    fail(`Engine API major drift: engine-go=v${protocolMajor}, contracts=v${contractMajor}`);
  }
  const activeContextPath = path.join(repoRoot, "proto/lunafox/engine/execution", `v${protocolMajor}`, "engine_execution_context.proto");
  if (!fs.existsSync(activeContextPath)) {
    fail(`active Engine API major v${protocolMajor} is missing its EngineExecutionContext proto`);
  }

  const policy = readPolicy(repoRoot);
  if (releaseChannel === "stable" && !policy.firstStableReleaseAllowed) {
    fail("Engine API major baseline freeze required before the first stable release");
  }
  if (policy.phase === "stable") {
    validateStableBaselines(repoRoot, protocolMajor, policy.frozenMajorBaselines);
  }
  return { activeMajor: protocolMajor, policy };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.printBaseline) {
    const protocolMajor = readEngineAPIMajor(path.join(options.repoRoot, "engine-go/protocol/abi.go"), "engine-go protocol");
    process.stdout.write(`${JSON.stringify(calculateMajorBaseline(options.repoRoot, protocolMajor), null, 2)}\n`);
    return;
  }
  const result = validateEngineAPIMajorPolicy(options.repoRoot, options.releaseChannel);
  process.stdout.write(`Engine API major policy verified: phase=${result.policy.phase} active=v${result.activeMajor} channel=${options.releaseChannel}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`Engine API major policy check failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export { calculateMajorBaseline, validateEngineAPIMajorPolicy };
