#!/usr/bin/env node

/** Validate a canonical runtime-composition.json release asset. */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { validateComposition } from "./resolve-release-component-composition.mjs";
import { parseEngineBlocks, parseRuntimeBlocks, validateManifest } from "./verify-public-release.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_POLICY = path.join(SCRIPT_DIR, "public-release-policy.json");
const RELEASE_TAG_RE = /^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  const options = { composition: "", expectedComponents: [], manifest: "", policy: "", json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      options.json = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-release-component-composition.mjs --composition <runtime-composition.json> [--expected-components <id,id,...>] [--manifest <release.manifest.yaml> [--policy <public-release-policy.json>]] [--json]\n");
      process.exit(0);
    }
    if (!["--composition", "--expected-components", "--manifest", "--policy"].includes(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--composition") options.composition = path.resolve(value);
    else if (arg === "--expected-components") options.expectedComponents = value.split(",").map((item) => item.trim()).filter(Boolean);
    else if (arg === "--manifest") options.manifest = path.resolve(value);
    else options.policy = path.resolve(value);
  }
  if (!options.composition) fail("--composition is required");
  if (options.policy && !options.manifest) fail("--policy requires --manifest");
  return options;
}

function readRegularFile(filePath, label) {
  let info;
  try {
    info = fs.lstatSync(filePath);
  } catch (error) {
    fail(`cannot read ${label}: ${error.message}`);
  }
  if (!info.isFile() || info.isSymbolicLink()) fail(`${label} must be a regular file`);
  try { return fs.readFileSync(filePath); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function readComposition(filePath) {
  try { return JSON.parse(readRegularFile(filePath, "runtime composition").toString("utf8")); }
  catch (error) { fail(`cannot read runtime composition: ${error.message}`); }
}

function readPolicy(filePath) {
  try { return JSON.parse(readRegularFile(filePath, "public release policy").toString("utf8")); }
  catch (error) { fail(`cannot read public release policy: ${error.message}`); }
}

function canonicalReleaseTag(value, label) {
  const tag = String(value ?? "");
  if (!RELEASE_TAG_RE.test(tag)) fail(`${label} is invalid`);
  return tag.startsWith("v") ? tag : `v${tag}`;
}

function assertExactComponentInventory(composition, manifestText, policy) {
  const runtime = parseRuntimeBlocks(manifestText);
  const packages = parseEngineBlocks(manifestText, policy);
  const byID = new Map(composition.components.map((component) => [component.id, component]));
  const expectedIDs = new Set();

  for (const entry of runtime) {
    const id = `runtime.${entry.name}`;
    const component = byID.get(id);
    if (!component || component.kind !== "runtime" || component.artifact.digest !== entry.refs[0].digest) {
      fail(`runtime composition ${id} artifact does not match the release manifest`);
    }
    expectedIDs.add(id);
  }

  const packageComponents = composition.components.filter((component) => component.id.startsWith("engine.") && component.id.endsWith(".package"));
  if (packageComponents.length !== packages.length) fail("runtime composition Engine package inventory does not match the release manifest");
  const packageByDigest = new Map();
  for (const component of packageComponents) {
    if (packageByDigest.has(component.artifact.digest)) fail(`runtime composition has duplicate Engine package artifact ${component.artifact.digest}`);
    packageByDigest.set(component.artifact.digest, component);
  }

  for (const refs of packages) {
    const component = packageByDigest.get(refs[0].digest);
    if (!component) fail(`runtime composition is missing Engine package ${refs[0].digest}`);
    const runtimeID = component.id.slice(0, -".package".length) + ".runtime";
    const runtimeComponent = byID.get(runtimeID);
    if (!runtimeComponent || runtimeComponent.kind !== "engine") {
      fail(`runtime composition is missing Engine runtime pair ${runtimeID}`);
    }
    expectedIDs.add(component.id);
    expectedIDs.add(runtimeID);
  }

  const actualIDs = composition.components.map((component) => component.id).sort();
  const expected = [...expectedIDs].sort();
  if (JSON.stringify(actualIDs) !== JSON.stringify(expected)) {
    fail("runtime composition component inventory contains an unknown or incomplete component");
  }
}

function validateManifestBinding(composition, manifestPath, policyPath) {
  const manifestBytes = readRegularFile(manifestPath, "release manifest");
  const policy = readPolicy(policyPath || DEFAULT_POLICY);
  const tag = canonicalReleaseTag(composition.releaseTag, "runtime composition releaseTag");
  const manifest = validateManifest(manifestPath, policy, tag);
  if (composition.compositionDigest !== manifest.runtimeComposition.sha256) {
    fail(`runtime composition canonical digest does not match manifest: expected ${manifest.runtimeComposition.sha256}, got ${composition.compositionDigest}`);
  }
  const manifestDigest = `sha256:${manifest.sha256}`;
  if (composition.manifestBinding?.manifestDigest !== manifestDigest) {
    fail("runtime composition manifest binding does not match the release manifest bytes");
  }
  assertExactComponentInventory(composition, manifestBytes.toString("utf8"), policy);
  return { manifestDigest, releaseTag: tag };
}

export function verify({ composition, expectedComponents = [], manifest = "", policy = "" }) {
  const value = typeof composition === "string" ? readComposition(path.resolve(composition)) : composition;
  const normalized = validateComposition(value, {
    ...(expectedComponents.length ? { expectedComponentIds: expectedComponents } : {}),
    ...(manifest ? { requireManifestBinding: true } : {}),
  });
  const binding = manifest ? validateManifestBinding(normalized, path.resolve(manifest), policy ? path.resolve(policy) : DEFAULT_POLICY) : null;
  return {
    schemaVersion: normalized.schemaVersion,
    kind: normalized.kind,
    releaseTag: normalized.releaseTag,
    passed: true,
    compositionDigest: normalized.compositionDigest ?? null,
    componentCount: normalized.components.length,
    reusedComponents: normalized.components.filter((component) => component.disposition === "reused").map((component) => component.id),
    ...(binding ? { manifestDigest: binding.manifestDigest, manifestReleaseTag: binding.releaseTag } : {}),
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = verify(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `runtime composition verified: ${result.componentCount} components`}\n`);
}

// macOS exposes /var through a /private/var symlink. Compare canonical file
// paths so copied public fixtures still execute the CLI entrypoint instead of
// silently importing this module as a library.
if (process.argv[1] && fileURLToPath(import.meta.url) === fs.realpathSync(path.resolve(process.argv[1]))) {
  try { main(); }
  catch (error) { process.stderr.write(`runtime component composition verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { parseArgs };
