#!/usr/bin/env node

/** Secretless validator for the public schema-v3 Compose release-channel branch. */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const FIRST_TAG = "v0.0.1-alpha.57";
const LEGACY_RE = /(?:alpha\.46|SCHEMA_VERSION=2|IMAGE_REGISTRY=|IMAGE_NAMESPACE=|WORKER_IMAGE|lunafox-installer|checksums\.txt)/;
const ALLOWED_KEYS = new Set(["SCHEMA_VERSION", "VERSION", "RELEASE_MANIFEST", "RELEASE_MANIFEST_SHA256"]);

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const args = { root: DEFAULT_ROOT, first: false, allowEmpty: false, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--root-dir") {
      const value = argv[++index];
      if (!value || value.startsWith("--")) fail("--root-dir requires a value");
      args.root = path.resolve(value);
    } else if (arg === "--require-first-release") args.first = true;
    else if (arg === "--allow-empty") args.allowEmpty = true;
    else if (arg === "--json") args.json = true;
    else if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/check-public-channel.mjs [--root-dir <dir>] [--allow-empty] [--require-first-release] [--json]\\n");
      process.exit(0);
    } else fail(`unknown argument: ${arg}`);
  }
  return args;
}

function sha256(bytes) { return crypto.createHash("sha256").update(bytes).digest("hex"); }

function parseEnv(filePath) {
  const values = new Map();
  for (const line of fs.readFileSync(filePath, "utf8").split(/\r?\n/)) {
    if (!line || line.startsWith("#")) continue;
    const index = line.indexOf("=");
    if (index <= 0 || values.has(line.slice(0, index))) fail(`${filePath} has duplicate/invalid environment key`);
    values.set(line.slice(0, index), line.slice(index + 1));
  }
  return values;
}

function validateVersionEnv(filePath, expectedVersion) {
  const values = parseEnv(filePath);
  if (values.get("SCHEMA_VERSION") !== "3") fail(`${filePath} must use SCHEMA_VERSION=3`);
  if (values.get("VERSION") !== expectedVersion) fail(`${filePath} VERSION must be ${expectedVersion}`);
  if (values.get("RELEASE_MANIFEST") !== `manifests/${expectedVersion}.yaml`) fail(`${filePath} has invalid RELEASE_MANIFEST`);
  if (!/^[a-f0-9]{64}$/.test(values.get("RELEASE_MANIFEST_SHA256") ?? "")) fail(`${filePath} has invalid RELEASE_MANIFEST_SHA256`);
  for (const key of values.keys()) if (!ALLOWED_KEYS.has(key)) fail(`${filePath} contains unsupported key ${key}`);
  return values;
}

function listFiles(root) {
  const output = [];
  for (const top of ["channels", "manifests"]) {
    const directory = path.join(root, top);
    if (!fs.existsSync(directory)) continue;
    const visit = (current) => {
      for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
        const full = path.join(current, entry.name);
        if (entry.isDirectory()) visit(full);
        else if (entry.isFile()) output.push(path.relative(root, full).split(path.sep).join("/"));
      }
    };
    visit(directory);
  }
  return output.sort();
}

function validate(root, requireFirst, allowEmpty = false) {
  const channels = path.join(root, "channels");
  const manifests = path.join(root, "manifests");
  const channelsPresent = fs.existsSync(channels);
  const manifestsPresent = fs.existsSync(manifests);
  if (!channelsPresent && !manifestsPresent) {
    if (requireFirst) fail("first release requires both channels and manifests directories");
    if (allowEmpty) {
      return { schemaVersion: 3, passed: true, files: [], firstRelease: false, empty: true };
    }
    fail("release-channel is empty");
  }
  if (!channelsPresent || !manifestsPresent || !fs.statSync(channels).isDirectory() || !fs.statSync(manifests).isDirectory()) {
    fail("channels and manifests must be both present directories or both absent");
  }
  const files = listFiles(root);
  if (files.length === 0) fail("release-channel is empty");
  for (const relative of files) {
    const full = path.join(root, ...relative.split("/"));
    const text = fs.readFileSync(full, "utf8");
    if (LEGACY_RE.test(text) || /0\.0\.0-dev/.test(text)) fail(`retired/development content found: ${relative}`);
    if (relative.startsWith("channels/") && relative.endsWith(".env")) {
      const filename = relative.slice("channels/".length, -".env".length);
      const version = /^(?:canary|stable)$/.test(filename) ? (parseEnv(full).get("VERSION") ?? "") : filename;
      if (!/^v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?$/.test(version)) fail(`${relative} has invalid VERSION`);
      validateVersionEnv(full, version);
    }
    if (relative.startsWith("manifests/") && !/^manifests\/v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?\.yaml$/.test(relative)) fail(`invalid manifest filename: ${relative}`);
  }
  for (const relative of files.filter((file) => file.startsWith("channels/") && file.endsWith(".env"))) {
    const values = parseEnv(path.join(root, ...relative.split("/")));
    const version = values.get("VERSION") ?? "";
    const manifestPath = path.join(manifests, `${version}.yaml`);
    if (!fs.existsSync(manifestPath)) fail(`${relative} points to missing manifests/${version}.yaml`);
    if (sha256(fs.readFileSync(manifestPath)) !== values.get("RELEASE_MANIFEST_SHA256")) fail(`${relative} manifest digest does not match ${version}.yaml`);
  }
  if (requireFirst) {
    const expected = [`channels/${FIRST_TAG}.env`, "channels/canary.env", `manifests/${FIRST_TAG}.yaml`];
    for (const relative of expected) if (!files.includes(relative)) fail(`first-release channel file missing: ${relative}`);
    if (files.includes("channels/stable.env")) fail("stable.env must not exist for first canary");
    const canary = validateVersionEnv(path.join(channels, "canary.env"), FIRST_TAG);
    const version = validateVersionEnv(path.join(channels, `${FIRST_TAG}.env`), FIRST_TAG);
    if (canary.get("RELEASE_MANIFEST_SHA256") !== version.get("RELEASE_MANIFEST_SHA256")) fail("canary alias digest differs from alpha.57");
  }
  return { schemaVersion: 3, passed: true, files, firstRelease: files.includes(`channels/${FIRST_TAG}.env`) };
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const result = validate(fs.realpathSync(args.root), args.first, args.allowEmpty);
  process.stdout.write(`${args.json ? JSON.stringify(result, null, 2) : `public Compose channel verified: ${result.files.length} files`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { process.stderr.write(`public channel verification failed: ${error.message}\n`); process.exitCode = 1; }
}

export { parseEnv, validate, validateVersionEnv };
