#!/usr/bin/env node

/** Resolve the exact release Manifest layout from the shared version registry. */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_REGISTRY = path.resolve(SCRIPT_DIR, "../../contracts/releasemanifest/release_compatibility_profiles.json");
const RELEASE_VERSION_RE = /^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.+-]+)?$/;

const ReleaseCompatibilityProfileModern = "modern";
const ReleaseCompatibilityProfileAlpha164Bridge = "alpha164-bridge";

function fail(message) {
  throw new Error(message);
}

function assertExactKeys(value, keys, label) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) fail(`${label} must be an object`);
  const actual = Object.keys(value).sort();
  const expected = [...keys].sort();
  if (JSON.stringify(actual) !== JSON.stringify(expected)) fail(`${label} has an unexpected field set`);
}

function assertReleaseVersion(releaseVersion) {
  if (typeof releaseVersion !== "string" || !RELEASE_VERSION_RE.test(releaseVersion)) {
    fail("releaseVersion must be a canonical bare semantic version");
  }
}

// Read the registry independently of the Go package so public release tooling
// and the runtime parser use the same reviewed version-to-shape mapping.
function readReleaseCompatibilityProfileRegistry(registryPath = DEFAULT_REGISTRY) {
  let value;
  try {
    value = JSON.parse(fs.readFileSync(registryPath, "utf8"));
  } catch (error) {
    fail(`cannot read release compatibility profile registry: ${error.message}`);
  }
  assertExactKeys(value, ["schemaVersion", "profiles"], "release compatibility profile registry");
  if (value.schemaVersion !== 1 || !Array.isArray(value.profiles)) {
    fail("release compatibility profile registry schema is invalid");
  }

  const profiles = new Map();
  for (const [index, entry] of value.profiles.entries()) {
    assertExactKeys(entry, ["releaseVersion", "profile"], `release compatibility profile registry profiles[${index}]`);
    assertReleaseVersion(entry.releaseVersion);
    if (entry.profile !== ReleaseCompatibilityProfileAlpha164Bridge) {
      fail(`release compatibility profile registry profiles[${index}].profile is not supported`);
    }
    if (profiles.has(entry.releaseVersion)) {
      fail(`release compatibility profile registry has duplicate releaseVersion ${entry.releaseVersion}`);
    }
    profiles.set(entry.releaseVersion, entry.profile);
  }
  return profiles;
}

function resolveReleaseCompatibilityProfile(releaseVersion, { registryPath = DEFAULT_REGISTRY } = {}) {
  assertReleaseVersion(releaseVersion);
  return readReleaseCompatibilityProfileRegistry(registryPath).get(releaseVersion) ?? ReleaseCompatibilityProfileModern;
}

function assertReleaseCompatibilityProfile(releaseVersion, requestedProfile = "", options = {}) {
  const resolvedProfile = resolveReleaseCompatibilityProfile(releaseVersion, options);
  if (requestedProfile !== "" && requestedProfile !== ReleaseCompatibilityProfileModern && requestedProfile !== ReleaseCompatibilityProfileAlpha164Bridge) {
    fail(`unsupported release compatibility profile: ${requestedProfile}`);
  }
  if (requestedProfile !== "" && requestedProfile !== resolvedProfile) {
    fail(`release compatibility profile ${requestedProfile} does not match registered profile ${resolvedProfile} for ${releaseVersion}`);
  }
  return resolvedProfile;
}

function parseArgs(argv) {
  const options = { releaseVersion: "", releaseProfile: "", registryPath: DEFAULT_REGISTRY, json: false };
  const values = new Set(["--release-version", "--release-profile", "--registry"]);
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--json") {
      options.json = true;
      continue;
    }
    if (argument === "--help" || argument === "-h") {
      process.stdout.write("Usage: node scripts/ci/release-compatibility-profile.mjs --release-version <bare-version> [--release-profile <modern|alpha164-bridge>] [--registry <file>] [--json]\n");
      process.exit(0);
    }
    if (!values.has(argument)) fail(`unknown argument: ${argument}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${argument} requires a value`);
    if (argument === "--release-version") options.releaseVersion = value;
    else if (argument === "--release-profile") options.releaseProfile = value;
    else options.registryPath = path.resolve(value);
  }
  if (!options.releaseVersion) fail("--release-version is required");
  return options;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const profile = assertReleaseCompatibilityProfile(options.releaseVersion, options.releaseProfile, options);
  process.stdout.write(options.json
    ? `${JSON.stringify({ releaseVersion: options.releaseVersion, profile }, null, 2)}\n`
    : `${profile}\n`);
}

const invokedScript = process.argv[1];
if (
  invokedScript &&
  invokedScript !== "-" &&
  fs.existsSync(invokedScript) &&
  fileURLToPath(import.meta.url) === fs.realpathSync(path.resolve(invokedScript))
) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`release compatibility profile resolution failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export {
  DEFAULT_REGISTRY,
  ReleaseCompatibilityProfileAlpha164Bridge,
  ReleaseCompatibilityProfileModern,
  assertReleaseCompatibilityProfile,
  readReleaseCompatibilityProfileRegistry,
  resolveReleaseCompatibilityProfile,
};
