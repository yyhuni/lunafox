#!/usr/bin/env node

/**
 * Derive the Engine Runtime/Package publication selection from the verified
 * release composition plan. Engine pairs are an atomic deployment unit: a
 * Runtime image cannot be rebuilt while its Package is promoted, or vice
 * versa. This resolver is the only workflow input that may narrow the Engine
 * fleet to a built subset.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
  sha256Digest,
  validateCompositionPlan,
} from "./resolve-release-component-composition.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DISCOVERY_SCHEMA_VERSION = "lunafox.engine-release-discovery.v1";
const SELECTION_SCHEMA_VERSION = 1;
const SELECTION_KIND = "lunafox.engine-release-selection.v1";
const ENGINE_ID_RE = /^engine\.lunafox\.[a-z][a-z0-9_]*$/;
const COMPONENT_ID_RE = /^(engine\.lunafox\.[a-z][a-z0-9_]*)\.(runtime|package)$/;
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const SEMVER_RE = /^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.+-]+)?$/;

function fail(message) { throw new Error(message); }
function assert(condition, message) { if (!condition) fail(message); }
function isObject(value) { return value !== null && typeof value === "object" && !Array.isArray(value); }

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function writeJson(file, value) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  const temporary = `${file}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`);
  fs.renameSync(temporary, file);
}

function parseArgs(argv) {
  const options = { plan: "", discovery: "", output: "", json: false };
  const valueFlags = new Set(["--plan", "--discovery", "--output"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: resolve-engine-release-disposition.mjs --plan FILE --discovery FILE --output FILE [--json]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--plan") options.plan = path.resolve(value);
    else if (arg === "--discovery") options.discovery = path.resolve(value);
    else options.output = path.resolve(value);
  }
  assert(options.plan, "--plan is required");
  assert(options.discovery, "--discovery is required");
  assert(options.output, "--output is required");
  return options;
}

function assertExactKeys(value, expected, label) {
  assert(isObject(value), `${label} must be an object`);
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  assert(JSON.stringify(actual) === JSON.stringify(wanted), `${label} has an invalid field set`);
}

function normalizeDiscovery(value) {
  assertExactKeys(value, ["schemaVersion", "engineRoot", "engines"], "Engine discovery");
  assert(value.schemaVersion === DISCOVERY_SCHEMA_VERSION, `unsupported Engine discovery schemaVersion ${JSON.stringify(value.schemaVersion)}`);
  assert(typeof value.engineRoot === "string" && value.engineRoot, "Engine discovery engineRoot is required");
  assert(Array.isArray(value.engines) && value.engines.length > 0, "Engine discovery engines must be non-empty");
  let previousID = "";
  return value.engines.map((entry, index) => {
    assertExactKeys(entry, ["engineId", "directory", "dockerfile", "buildContext", "repository"], `Engine discovery engines[${index}]`);
    const engineId = String(entry.engineId ?? "");
    const directory = String(entry.directory ?? "");
    assert(ENGINE_ID_RE.test(engineId), `Engine discovery engines[${index}].engineId is invalid`);
    assert(directory === engineId.slice("engine.lunafox.".length), `Engine discovery ${engineId} directory does not match its canonical identity`);
    assert(entry.dockerfile === `${directory}/Dockerfile`, `Engine discovery ${engineId} Dockerfile is invalid`);
    assert(entry.buildContext === ".", `Engine discovery ${engineId} buildContext is invalid`);
    // OCI repository names cannot keep the Engine ID's underscores; discovery
    // derives lunafox-engine-runtime-<hyphenated-local-name>.
    assert(entry.repository === `lunafox-engine-runtime-${directory.replaceAll("_", "-")}`, `Engine discovery ${engineId} repository is invalid`);
    assert(!previousID || engineId > previousID, "Engine discovery engines must be ordered and unique by engineId");
    previousID = engineId;
    return { engineId, directory, dockerfile: entry.dockerfile, buildContext: entry.buildContext, repository: entry.repository };
  });
}

function engineComponentPairs(plan) {
  const pairs = new Map();
  for (const component of plan.components) {
    if (component.kind !== "engine") continue;
    const match = COMPONENT_ID_RE.exec(component.id);
    assert(match, `Engine composition component id is invalid: ${component.id}`);
    const [, engineId, role] = match;
    const pair = pairs.get(engineId) ?? {};
    assert(!pair[role], `Engine composition plan contains duplicate ${role} component for ${engineId}`);
    pair[role] = component;
    pairs.set(engineId, pair);
  }
  assert(pairs.size > 0, "Engine composition plan contains no Engine components");
  for (const [engineId, pair] of pairs) {
    assert(pair.runtime && pair.package, `Engine composition plan must contain Runtime and Package components for ${engineId}`);
    assert(pair.runtime.disposition === pair.package.disposition, `Engine Runtime/Package disposition must match for ${engineId}`);
    assert(pair.runtime.name === engineId.slice("engine.lunafox.".length) && pair.package.name === pair.runtime.name, `Engine composition names are invalid for ${engineId}`);
    for (const component of [pair.runtime, pair.package]) {
      assert(DIGEST_RE.test(component.inputFingerprint?.digest ?? ""), `${component.id} input fingerprint is invalid`);
      if (component.disposition === "built") {
        assert(component.sourceRelease?.tag === plan.releaseTag, `${component.id} built source release must be current`);
      } else {
        assert(component.sourceRelease?.tag !== plan.releaseTag && DIGEST_RE.test(component.sourceRelease?.compositionDigest ?? ""), `${component.id} reused source release must retain original composition evidence`);
      }
    }
  }
  return pairs;
}

// Package version is content-addressed from the exact Runtime/Package input
// pair, rather than from the product release tag. A frontend-only release
// consequently cannot manufacture a new Engine Package archive. It remains a
// valid SemVer because the package contract currently requires that field.
function packageVersionForPair(engineId, pair) {
  const identity = {
    schemaVersion: 1,
    engineId,
    runtimeInputFingerprint: pair.runtime.inputFingerprint.digest,
    packageInputFingerprint: pair.package.inputFingerprint.digest,
  };
  const digest = sha256Digest(identity);
  const version = `0.0.0+${digest.slice("sha256:".length)}`;
  assert(SEMVER_RE.test(version), `derived Engine Package version is invalid for ${engineId}`);
  return version;
}

function resolveSelection(planValue, discoveryValue) {
  const plan = validateCompositionPlan(planValue);
  const discovery = normalizeDiscovery(discoveryValue);
  const pairs = engineComponentPairs(plan);
  assert(pairs.size === discovery.length, "Engine composition plan inventory does not match source discovery");
  const engines = discovery.map((source) => {
    const pair = pairs.get(source.engineId);
    assert(pair, `Engine composition plan is missing discovered Engine ${source.engineId}`);
    const disposition = pair.runtime.disposition;
    return {
      ...source,
      disposition,
      runtimeComponentId: pair.runtime.id,
      packageComponentId: pair.package.id,
      packageVersion: packageVersionForPair(source.engineId, pair),
    };
  });
  for (const engineId of pairs.keys()) assert(discovery.some((source) => source.engineId === engineId), `Engine composition plan contains unknown Engine ${engineId}`);
  const builtEngineIds = engines.filter((entry) => entry.disposition === "built").map((entry) => entry.engineId);
  const reusedEngineIds = engines.filter((entry) => entry.disposition === "reused").map((entry) => entry.engineId);
  return {
    schemaVersion: SELECTION_SCHEMA_VERSION,
    kind: SELECTION_KIND,
    releaseTag: plan.releaseTag,
    planDigest: plan.planDigest,
    engines,
    builtEngineIds,
    reusedEngineIds,
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const selection = resolveSelection(readJson(options.plan, "runtime composition plan"), readJson(options.discovery, "Engine discovery"));
  writeJson(options.output, selection);
  const summary = {
    schemaVersion: selection.schemaVersion,
    passed: true,
    planDigest: selection.planDigest,
    builtEngineIds: selection.builtEngineIds,
    reusedEngineIds: selection.reusedEngineIds,
  };
  process.stdout.write(`${options.json ? JSON.stringify(summary, null, 2) : `Engine selection: ${summary.builtEngineIds.length} built, ${summary.reusedEngineIds.length} reused`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) { process.stderr.write(`resolve Engine release disposition failed: ${error.message}\n`); process.exitCode = 1; }
}

export {
  ENGINE_ID_RE,
  SELECTION_KIND,
  SELECTION_SCHEMA_VERSION,
  normalizeDiscovery,
  packageVersionForPair,
  resolveSelection,
};
