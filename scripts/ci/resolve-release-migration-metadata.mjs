#!/usr/bin/env node

/**
 * Resolve release-level database migration metadata from audited migration
 * manifests. The current manifest is checked against its SQL pair; a previous
 * release manifest is treated as the immutable historical boundary.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DEFAULT_CURRENT_MANIFEST = path.join(DEFAULT_ROOT, "server/cmd/server/migrations/manifest.json");

export const RESOLVER_SCHEMA_VERSION = 1;
export const RESOLVER_KIND = "lunafox.release-migration-metadata.v1";
export const MIGRATION_SCHEMA_VERSION = 1;
export const CHECKSUM_ALGORITHM = "sha256-canonical-pair-v1";

const ID_RE = /^\d{6}$/;
const SLUG_RE = /^[a-z0-9][a-z0-9_-]*$/;
const CHECKSUM_RE = /^sha256:[a-f0-9]{64}$/;
const MANIFEST_KEYS = ["schemaVersion", "checksumAlgorithm", "baselineMigration", "migrations"];
const ENTRY_KEYS = ["id", "slug", "up", "down", "checksum"];

function fail(message) {
  throw new Error(message);
}

function assert(condition, message) {
  if (!condition) fail(message);
}

function isObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function assertExactKeys(value, expected, label) {
  assert(isObject(value), `${label} must be an object`);
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  assert(JSON.stringify(actual) === JSON.stringify(wanted), `${label} has an invalid field set`);
}

function readJson(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    fail(`cannot read ${label} ${filePath}: ${error.message}`);
  }
}

function sha256File(filePath) {
  return `sha256:${crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex")}`;
}

export function pairChecksum(migrationsRoot, migration) {
  const names = [migration.down, migration.up].sort();
  const payload = names.map((name) => `${name}\n${sha256File(path.join(migrationsRoot, name))}\n`).join("");
  return `sha256:${crypto.createHash("sha256").update(payload).digest("hex")}`;
}

function migrationRootFor(manifestPath) {
  return path.dirname(manifestPath);
}

function existingSqlFiles(root) {
  if (!fs.existsSync(root)) return [];
  return fs.readdirSync(root).filter((name) => name.endsWith(".up.sql") || name.endsWith(".down.sql")).sort();
}

function validateEntry(raw, index, label, migrationsRoot, requireFiles) {
  assertExactKeys(raw, ENTRY_KEYS, `${label}.migrations[${index}]`);
  const id = String(raw.id ?? "");
  const slug = String(raw.slug ?? "");
  const up = String(raw.up ?? "");
  const down = String(raw.down ?? "");
  const checksum = String(raw.checksum ?? "");
  assert(ID_RE.test(id), `${label} migration id must be six digits: ${id}`);
  assert(SLUG_RE.test(slug), `${label} migration slug is invalid: ${slug}`);
  assert(up === `${id}_${slug}.up.sql`, `${label} migration ${id} up filename is not canonical`);
  assert(down === `${id}_${slug}.down.sql`, `${label} migration ${id} down filename is not canonical`);
  assert(CHECKSUM_RE.test(checksum), `${label} migration ${id} checksum must be sha256`);

  const files = [up, down];
  const present = files.map((file) => fs.existsSync(path.join(migrationsRoot, file)));
  if (requireFiles) {
    for (const [file, exists] of files.map((file, index) => [file, present[index]])) {
      assert(exists, `${label} migration file is missing: ${file}`);
      assert(fs.statSync(path.join(migrationsRoot, file)).isFile(), `${label} migration file is not a regular file: ${file}`);
    }
    assert(checksum === pairChecksum(migrationsRoot, { up, down }), `${label} migration checksum mismatch: ${id}`);
  } else if (present.some(Boolean)) {
    assert(present.every(Boolean), `${label} migration pair is only partially available: ${id}`);
    assert(checksum === pairChecksum(migrationsRoot, { up, down }), `${label} migration checksum mismatch: ${id}`);
  }

  return Object.freeze({ id, slug, up, down, checksum });
}

function validateManifest(manifestPath, label, { requireFiles }) {
  const absolutePath = path.resolve(manifestPath);
  const value = readJson(absolutePath, label);
  assertExactKeys(value, MANIFEST_KEYS, label);
  assert(value.schemaVersion === MIGRATION_SCHEMA_VERSION, `${label} schemaVersion must be ${MIGRATION_SCHEMA_VERSION}`);
  assert(value.checksumAlgorithm === CHECKSUM_ALGORITHM, `${label} checksum algorithm is unsupported`);
  const baselineMigration = String(value.baselineMigration ?? "");
  assert(ID_RE.test(baselineMigration), `${label} baselineMigration must be six digits`);
  assert(Array.isArray(value.migrations) && value.migrations.length > 0, `${label} migrations must be non-empty`);

  const migrationsRoot = migrationRootFor(absolutePath);
  const entries = value.migrations.map((entry, index) => validateEntry(entry, index, label, migrationsRoot, requireFiles));
  const ids = new Set();
  const baselineNumber = Number.parseInt(baselineMigration, 10);
  entries.forEach((entry, index) => {
    assert(!ids.has(entry.id), `${label} contains duplicate migration id: ${entry.id}`);
    ids.add(entry.id);
    const expected = String(baselineNumber + index).padStart(6, "0");
    assert(entry.id === expected, `${label} numbering must be contiguous; expected ${expected}, got ${entry.id}`);
  });
  assert(entries[0].id === baselineMigration, `${label} must begin at baseline migration ${baselineMigration}`);

  if (requireFiles) {
    const listedFiles = new Set(entries.flatMap((entry) => [entry.up, entry.down]));
    const actualFiles = existingSqlFiles(migrationsRoot);
    assert(JSON.stringify([...listedFiles].sort()) === JSON.stringify(actualFiles), `${label} must list exactly every up/down SQL file`);
  }

  return Object.freeze({
    path: absolutePath,
    schemaVersion: value.schemaVersion,
    checksumAlgorithm: value.checksumAlgorithm,
    baselineMigration,
    entries: Object.freeze(entries),
  });
}

function compareEntry(previous, current, index) {
  for (const field of ["id", "slug", "checksum"]) {
    if (previous[field] !== current[field]) {
      fail(`migration boundary drift at index ${index}: ${field} changed from ${previous[field]} to ${current[field]}`);
    }
  }
}

export function resolveReleaseMigrationMetadata(currentManifestPath = DEFAULT_CURRENT_MANIFEST, previousManifestPath = "") {
  const current = validateManifest(currentManifestPath, "current migration manifest", { requireFiles: true });
  let previous = null;
  if (previousManifestPath) {
    previous = validateManifest(previousManifestPath, "previous migration manifest", { requireFiles: false });
    assert(previous.schemaVersion === current.schemaVersion, "migration manifest schemaVersion differs across release boundary");
    assert(previous.checksumAlgorithm === current.checksumAlgorithm, "migration checksum algorithm differs across release boundary");
    assert(previous.baselineMigration === current.baselineMigration, "migration baseline differs across release boundary");
    assert(previous.entries.length <= current.entries.length, "current migration manifest is shorter than the previous boundary");
    previous.entries.forEach((entry, index) => compareEntry(entry, current.entries[index], index));
  } else if (current.entries.length !== 1 || current.entries[0].id !== current.baselineMigration) {
    fail("previous migration manifest is required when the current manifest is beyond the frozen baseline");
  }

  const previousCount = previous?.entries.length ?? 0;
  // The frozen baseline is the only valid no-boundary case. It is the
  // initial state, not a release that newly applies migration 000001.
  const additions = previous ? current.entries.slice(previousCount) : [];
  assert(additions.length <= 1, `migration boundary contains ${additions.length} new migrations; exactly one is supported per release`);
  if (additions.length === 1 && previous) {
    const expected = String(Number.parseInt(previous.entries.at(-1).id, 10) + 1).padStart(6, "0");
    assert(additions[0].id === expected, `new migration must continue the previous boundary at ${expected}`);
  }

  const migration = additions[0] ?? null;
  return Object.freeze({
    schemaVersion: RESOLVER_SCHEMA_VERSION,
    kind: RESOLVER_KIND,
    boundary: previous ? "verified-previous-manifest" : "frozen-baseline",
    hasDatabaseMigration: Boolean(migration),
    migrationType: migration ? "" : "none",
    migrationId: migration ? `${migration.id}_${migration.slug}` : "",
    checksum: migration?.checksum ?? "",
    currentMigrationCount: current.entries.length,
    previousMigrationCount: previous?.entries.length ?? 0,
    previousMigrationId: previous?.entries.at(-1)?.id ?? "",
  });
}

function writeJson(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const temporary = `${filePath}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o644 });
  fs.renameSync(temporary, filePath);
}

function parseArgs(argv) {
  const options = { current: DEFAULT_CURRENT_MANIFEST, previous: "", output: "", json: false };
  const aliases = new Map([
    ["--current", "current"],
    ["--current-manifest", "current"],
    ["--current-migration-manifest", "current"],
    ["--previous", "previous"],
    ["--previous-manifest", "previous"],
    ["--previous-migration-manifest", "previous"],
    ["--output", "output"],
  ]);
  const seen = new Set();
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      options.json = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: resolve-release-migration-metadata.mjs [--current-manifest FILE] [--previous-manifest FILE] [--output FILE] [--json]\n");
      process.exit(0);
    }
    const key = aliases.get(arg);
    if (!key) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    assert(!seen.has(key), `${arg} must not be repeated`);
    seen.add(key);
    options[key] = path.resolve(value);
  }
  return options;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = resolveReleaseMigrationMetadata(options.current, options.previous);
  if (options.output) writeJson(options.output, result);
  if (options.json || !options.output) {
    process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
  } else {
    process.stdout.write(`release migration metadata resolved: ${result.hasDatabaseMigration ? result.migrationId : "none"}\n`);
  }
}

try {
  if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) main();
} catch (error) {
  process.stderr.write(`release migration metadata resolution failed: ${error.message}\n`);
  process.exitCode = 1;
}
