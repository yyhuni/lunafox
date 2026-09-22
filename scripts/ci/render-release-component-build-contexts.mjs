#!/usr/bin/env node

/**
 * Render BuildKit named image contexts for one component build.
 *
 * A composition fingerprint records resolved base-image identities. BuildKit
 * otherwise resolves a tag again when it evaluates FROM, leaving a window for
 * the tag to move between planning and publication. A same-name named context
 * makes BuildKit consume the recorded digest instead. This renderer replays
 * the source closure before emitting anything, so a stale plan, altered
 * Dockerfile, or mismatched base-image map fails before a publish action.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
  buildInputFingerprint,
  canonicalJson,
  FINGERPRINT_SCHEMA_VERSION,
  validateCompositionPlan,
} from "./resolve-release-component-composition.mjs";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const IMAGE_REFERENCE_RE = /^[A-Za-z0-9._/:@-]+$/;

function fail(message) {
  throw new Error(message);
}

function assert(condition, message) {
  if (!condition) fail(message);
}

function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function assertExactKeys(value, expected, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  assert(JSON.stringify(actual) === JSON.stringify(wanted), `${label} has an unsupported schema`);
}

function readJson(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    fail(`cannot read ${label}: ${error.message}`);
  }
}

function digestFromIdentity(identity, label) {
  const text = String(identity ?? "");
  const digest = text.includes("@") ? text.slice(text.lastIndexOf("@") + 1) : text;
  assert(DIGEST_RE.test(digest), `${label} must resolve to a sha256 digest`);
  return digest;
}

function normalizeBaseImages(value) {
  assert(isPlainObject(value), "base image identities must be an object");
  return Object.fromEntries(Object.keys(value).sort().map((reference) => {
    assert(IMAGE_REFERENCE_RE.test(reference), `base image reference is unsafe: ${reference}`);
    const digest = String(value[reference] ?? "");
    assert(DIGEST_RE.test(digest), `base image identity is invalid for ${reference}`);
    return [reference, digest];
  }));
}

function componentSpecFromInputs(componentID, fingerprint) {
  assertExactKeys(fingerprint, ["version", "algorithm", "digest", "baseImagesResolved", "inputs"], `component ${componentID} inputFingerprint`);
  const inputs = fingerprint.inputs;
  assertExactKeys(inputs, [
    "schemaVersion", "componentId", "kind", "contextPath", "dockerfile", "dockerignore", "files",
    "namedContexts", "buildArgs", "platforms", "baseImages", "baseImagesResolved", "builderPolicy", "generatedInputs",
  ], `component ${componentID} inputFingerprint.inputs`);
  assert(inputs.schemaVersion === FINGERPRINT_SCHEMA_VERSION, `component ${componentID} inputFingerprint inputs schemaVersion is unsupported`);
  assert(inputs.componentId === componentID, `component ${componentID} inputFingerprint inputs do not match the component`);
  assert(Array.isArray(inputs.baseImages), `component ${componentID} inputFingerprint inputs baseImages must be an array`);
  const baseImageIdentities = {};
  for (const [index, entry] of inputs.baseImages.entries()) {
    assertExactKeys(entry, ["ref", "identity", "resolved"], `component ${componentID} baseImages[${index}]`);
    const reference = String(entry.ref ?? "");
    assert(IMAGE_REFERENCE_RE.test(reference), `component ${componentID} baseImages[${index}].ref is unsafe`);
    assert(entry.resolved === true, `component ${componentID} has an unresolved base image: ${reference}`);
    baseImageIdentities[reference] = String(entry.identity ?? "");
  }
  assert(Object.keys(baseImageIdentities).length === inputs.baseImages.length, `component ${componentID} inputFingerprint has duplicate base-image references`);
  return {
    inputs,
    spec: {
      id: componentID,
      kind: inputs.kind,
      contextPath: inputs.contextPath,
      dockerfile: inputs.dockerfile,
      dockerignore: inputs.dockerignore,
      namedContexts: inputs.namedContexts,
      buildArgs: inputs.buildArgs,
      platforms: inputs.platforms,
      baseImageIdentities,
      builderPolicy: inputs.builderPolicy,
      generatedInputs: inputs.generatedInputs,
    },
  };
}

/**
 * Returns one `name=docker-image://reference@digest` entry per mutable
 * external image reference. The result is intentionally ready for both the
 * build-push-action `build-contexts` input and `docker buildx --build-context`.
 */
export function renderBuildContexts({ root = DEFAULT_ROOT, plan, componentID, baseImages }) {
  const normalizedPlan = validateCompositionPlan(plan);
  const id = String(componentID ?? "");
  const matching = normalizedPlan.components.filter((component) => component.id === id);
  assert(matching.length === 1, `runtime composition plan must contain exactly one component ${id}`);
  const component = matching[0];
  const { inputs, spec } = componentSpecFromInputs(id, component.inputFingerprint);
  const actual = buildInputFingerprint(path.resolve(root), spec);
  assert(actual.digest === component.inputFingerprint.digest, `component ${id} build inputs no longer match the verified composition plan`);
  assert(canonicalJson(actual.inputs) === canonicalJson(inputs), `component ${id} Dockerfile base-image references no longer match the verified composition plan`);
  assert(component.inputFingerprint.baseImagesResolved === true && actual.inputs.baseImagesResolved === true,
    `component ${id} has unresolved base images`);

  const identities = normalizeBaseImages(baseImages);
  const contexts = [];
  for (const entry of actual.inputs.baseImages) {
    const reference = entry.ref;
    const digest = digestFromIdentity(entry.identity, `component ${id} base image ${reference}`);
    if (reference.includes("@sha256:")) {
      assert(reference.endsWith(`@${digest}`), `component ${id} pinned base image does not match its recorded identity: ${reference}`);
      continue;
    }
    const mappedDigest = identities[reference];
    assert(mappedDigest !== undefined, `base image identity is missing for component ${id}: ${reference}`);
    assert(mappedDigest === digest, `base image identity does not match the verified plan for component ${id}: ${reference}`);
    contexts.push(`${reference}=docker-image://${reference}@${digest}`);
  }
  return [...new Set(contexts)].sort();
}

export function parseArgs(argv) {
  const options = { root: DEFAULT_ROOT, plan: "", componentID: "", baseImages: "", output: "", json: false };
  const valueFlags = new Set(["--root-dir", "--plan", "--component-id", "--base-images", "--output"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/render-release-component-build-contexts.mjs --plan FILE --component-id ID --base-images FILE --output FILE [--root-dir DIR] [--json]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--root-dir") options.root = path.resolve(value);
    else if (arg === "--plan") options.plan = path.resolve(value);
    else if (arg === "--component-id") options.componentID = value;
    else if (arg === "--base-images") options.baseImages = path.resolve(value);
    else options.output = path.resolve(value);
  }
  for (const [name, value] of Object.entries({ plan: options.plan, componentID: options.componentID, baseImages: options.baseImages, output: options.output })) {
    assert(value, `--${name.replace(/[A-Z]/g, (character) => `-${character.toLowerCase()}`)} is required`);
  }
  return options;
}

function writeOutput(filePath, contexts) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const temporary = `${filePath}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, contexts.length ? `${contexts.join("\n")}\n` : "", { mode: 0o600 });
  fs.renameSync(temporary, filePath);
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const contexts = renderBuildContexts({
    root: options.root,
    plan: readJson(options.plan, "runtime composition plan"),
    componentID: options.componentID,
    baseImages: readJson(options.baseImages, "base image identities"),
  });
  writeOutput(options.output, contexts);
  process.stdout.write(options.json
    ? `${JSON.stringify({ componentID: options.componentID, contexts }, null, 2)}\n`
    : `rendered ${contexts.length} pinned BuildKit image context(s) for ${options.componentID}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) { process.stderr.write(`release component build-context rendering failed: ${error.message}\n`); process.exitCode = 1; }
}
