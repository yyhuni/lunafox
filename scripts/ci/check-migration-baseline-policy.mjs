#!/usr/bin/env node

import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const defaultRepoRoot = path.resolve(scriptDir, "../..");
const MIGRATION_FILE_PATTERN = /^(\d{6})_([a-z0-9][a-z0-9_-]*)\.(up|down)\.sql$/;
const CHECKSUM_PATTERN = /^sha256:[a-f0-9]{64}$/;
const CHECKSUM_ALGORITHM = "sha256-canonical-pair-v1";
const BASELINE_PAIR_CHECKSUM = "sha256:7c4aaf55edfc0f403bb47e2cccd491c6876e4f611cb0d0dfc585defb508b3721";

const expectedPolicy = {
  schemaVersion: 1,
  phase: "release-candidate",
  baselineMigration: "000001",
  baselineMutable: false,
  preserveDataUpgrade: false,
  rollbackPolicy: "backup-restore-or-forward-fix",
  stableReleaseAllowed: false,
  dataRetainingDeploymentAllowed: false,
  freezeTriggers: [
    "first-public-stable-release",
    "first-data-retaining-non-disposable-deployment",
  ],
  checksumAlgorithm: CHECKSUM_ALGORITHM,
  baselineChecksum: BASELINE_PAIR_CHECKSUM,
  migrationManifest: "manifest.json",
};

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  const options = {
    repoRoot: defaultRepoRoot,
    releaseChannel: "release-candidate",
    deploymentMode: "disposable",
    releaseManifest: "",
  };

  for (let index = 0; index < argv.length; index += 1) {
    const flag = argv[index];
    const value = argv[index + 1];
    if (!["--repo-root", "--release-channel", "--deployment-mode", "--release-manifest"].includes(flag)) {
      fail(`unknown argument: ${flag}`);
    }
    if (!value) {
      fail(`missing value for ${flag}`);
    }
    if (flag === "--repo-root") options.repoRoot = path.resolve(value);
    if (flag === "--release-channel") options.releaseChannel = value;
    if (flag === "--deployment-mode") options.deploymentMode = value;
    if (flag === "--release-manifest") options.releaseManifest = path.resolve(value);
    index += 1;
  }

  if (!["development", "release-candidate", "canary", "stable"].includes(options.releaseChannel)) {
    fail(`unsupported release channel: ${options.releaseChannel}`);
  }
  if (!["disposable", "data-retaining"].includes(options.deploymentMode)) {
    fail(`unsupported deployment mode: ${options.deploymentMode}`);
  }
  return options;
}

function readJson(filePath, label) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    fail(`cannot read ${label} ${filePath}: ${error.message}`);
  }
}

function readPolicy(repoRoot) {
  const policyPath = path.join(repoRoot, "server/cmd/server/migrations/policy.json");
  const policy = readJson(policyPath, "migration policy");
  try {
    assert.deepStrictEqual(policy, expectedPolicy);
  } catch {
    fail("migration policy drift requires an explicit migration-governance change");
  }
  return policy;
}

function sha256File(filePath) {
  return `sha256:${crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex")}`;
}

function pairChecksum(migrationsDir, migration) {
  const names = [migration.down, migration.up].sort();
  const payload = names.map((name) => `${name}\n${sha256File(path.join(migrationsDir, name))}\n`).join("");
  return `sha256:${crypto.createHash("sha256").update(payload).digest("hex")}`;
}

function validateMigrationManifest(repoRoot, policy) {
  const migrationsDir = path.join(repoRoot, "server/cmd/server/migrations");
  const manifestPath = path.join(migrationsDir, policy.migrationManifest);
  const manifest = readJson(manifestPath, "migration manifest");
  assert.equal(manifest.schemaVersion, policy.schemaVersion, "migration manifest schemaVersion must match policy");
  assert.equal(manifest.checksumAlgorithm, CHECKSUM_ALGORITHM, "migration manifest checksum algorithm is unsupported");
  assert.equal(manifest.baselineMigration, policy.baselineMigration, "migration manifest baselineMigration must match policy");
  assert.ok(Array.isArray(manifest.migrations) && manifest.migrations.length > 0, "migration manifest must contain migrations");

  const files = fs.readdirSync(migrationsDir)
    .filter((name) => name.endsWith(".up.sql") || name.endsWith(".down.sql"))
    .sort();
  const entriesById = new Map();
  const listedFiles = new Set();
  for (const migration of manifest.migrations) {
    assert.equal(typeof migration.id, "string", "migration manifest id must be a string");
    assert.equal(typeof migration.slug, "string", `migration ${migration.id} slug must be a string`);
    assert.match(migration.id, /^\d{6}$/, `migration id must be six digits: ${migration.id}`);
    assert.match(migration.slug, /^[a-z0-9][a-z0-9_-]*$/, `migration slug is invalid: ${migration.slug}`);
    assert.equal(typeof migration.up, "string", `migration ${migration.id} up file is required`);
    assert.equal(typeof migration.down, "string", `migration ${migration.id} down file is required`);
    assert.match(migration.checksum, CHECKSUM_PATTERN, `migration ${migration.id} checksum must be sha256`);
    assert.equal(path.basename(migration.up), `${migration.id}_${migration.slug}.up.sql`, `migration ${migration.id} up filename is not canonical`);
    assert.equal(path.basename(migration.down), `${migration.id}_${migration.slug}.down.sql`, `migration ${migration.id} down filename is not canonical`);
    assert.ok(!entriesById.has(migration.id), `duplicate migration id: ${migration.id}`);
    entriesById.set(migration.id, migration);
    for (const filename of [migration.up, migration.down]) {
      assert.equal(path.basename(filename), filename, `migration path must stay inside migrations directory: ${filename}`);
      assert.ok(fs.existsSync(path.join(migrationsDir, filename)), `migration file is missing: ${filename}`);
      assert.ok(!listedFiles.has(filename), `migration file is listed more than once: ${filename}`);
      listedFiles.add(filename);
    }
    const downSql = fs.readFileSync(path.join(migrationsDir, migration.down), "utf8");
    assert.ok(downSql.startsWith("-- DESTRUCTIVE TEST TEARDOWN ONLY."), `${migration.down} must be test-teardown-only`);
    assert.equal(migration.checksum, pairChecksum(migrationsDir, migration), `migration checksum mismatch: ${migration.id}`);
  }

  const expectedIds = [...entriesById.keys()].sort();
  for (let index = 0; index < expectedIds.length; index += 1) {
    const expected = String(index + 1).padStart(6, "0");
    assert.equal(expectedIds[index], expected, `migration numbering must be contiguous; expected ${expected}, got ${expectedIds[index]}`);
  }
  assert.deepStrictEqual([...listedFiles].sort(), files, "migration manifest must list exactly every up/down SQL file");
  const baseline = entriesById.get(policy.baselineMigration);
  assert.ok(baseline, `baseline migration ${policy.baselineMigration} is missing from manifest`);
  assert.equal(baseline.slug, "init_schema", "000001 baseline slug is immutable");
  assert.equal(baseline.checksum, policy.baselineChecksum, "000001 baseline checksum does not match policy");
  assert.equal(baseline.checksum, BASELINE_PAIR_CHECKSUM, "000001 baseline checksum changed");
  return { manifest, entriesById };
}

function parseScalar(value) {
  const trimmed = value.trim();
  if (trimmed === "true") return true;
  if (trimmed === "false") return false;
  if (/^-?\d+$/.test(trimmed)) return Number(trimmed);
  if (/^\".*\"$/.test(trimmed) || /^'.*'$/.test(trimmed)) return trimmed.slice(1, -1);
  return trimmed;
}

function readReleaseMigrationMetadata(manifestPath) {
  let lines;
  try {
    lines = fs.readFileSync(manifestPath, "utf8").split(/\r?\n/);
  } catch (error) {
    fail(`cannot read release manifest ${manifestPath}: ${error.message}`);
  }
  const start = lines.findIndex((line) => /^\s+databaseMigration:\s*$/.test(line));
  if (start < 0) fail("release manifest is missing upgrade.databaseMigration");
  const values = {};
  for (let index = start + 1; index < lines.length; index += 1) {
    const match = /^(\s+)([A-Za-z][A-Za-z0-9]*):\s*(.*)$/.exec(lines[index]);
    if (!match) continue;
    if (match[1].length <= 2) break;
    values[match[2]] = parseScalar(match[3]);
  }
  for (const key of ["hasDatabaseMigration", "migrationType", "migrationId", "checksum", "policyVersion"]) {
    if (!Object.hasOwn(values, key)) fail(`release manifest migration metadata is missing ${key}`);
  }
  return values;
}

function validateReleaseManifest(manifestPath, policy, migrationManifest) {
  if (!manifestPath) return;
  const metadata = readReleaseMigrationMetadata(manifestPath);
  assert.equal(metadata.policyVersion, policy.schemaVersion, "release manifest migration policyVersion does not match policy");
  if (!metadata.hasDatabaseMigration) {
    assert.equal(metadata.migrationType, "none", "release without migration must use migrationType none");
    assert.equal(metadata.migrationId, "", "release without migration must not claim a migration id");
    assert.equal(metadata.checksum, "", "release without migration must not claim a migration checksum");
    return;
  }
  assert.ok(["compatible", "preserve-data", "destructive"].includes(metadata.migrationType), "release migrationType is unsupported");
  assert.match(metadata.migrationId, /^\d{6}_[a-z0-9][a-z0-9_-]*$/, "release migrationId must be a canonical migration stem");
  assert.match(metadata.checksum, CHECKSUM_PATTERN, "release migration checksum must be sha256");
  const entry = [...migrationManifest.entriesById.values()].find((candidate) => `${candidate.id}_${candidate.slug}` === metadata.migrationId);
  assert.ok(entry, `release references unknown migration ${metadata.migrationId}`);
  assert.equal(metadata.checksum, entry.checksum, "release migration checksum does not match migration manifest");
}

function validateImageDelivery(repoRoot) {
  const defaultsPath = path.join(repoRoot, "server/internal/config/defaults.go");
  let defaults;
  try {
    defaults = fs.readFileSync(defaultsPath, "utf8");
  } catch (error) {
    fail(`cannot read Server defaults ${defaultsPath}: ${error.message}`);
  }
  const declared = [...defaults.matchAll(/v\.SetDefault\(\s*"MIGRATION_POLICY_PATH"\s*,\s*"([^"]+)"\s*\)/g)];
  if (declared.length !== 1) {
    fail(`Server defaults must declare exactly one MIGRATION_POLICY_PATH default, found ${declared.length}`);
  }
  const policyPath = declared[0][1];
  const dockerfilePath = path.join(repoRoot, "server/Dockerfile");
  let dockerfile;
  try {
    dockerfile = fs.readFileSync(dockerfilePath, "utf8");
  } catch (error) {
    fail(`cannot read the Server image definition ${dockerfilePath}: ${error.message}`);
  }
  const escapedPath = policyPath.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const copiesPolicy = new RegExp(`^COPY\\s+--from=\\S+\\s+/src/server/cmd/server/migrations/policy\\.json\\s+${escapedPath}\\s*$`, "m");
  if (!copiesPolicy.test(dockerfile)) fail(`Server image must copy the reviewed migration policy to ${policyPath}`);
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const policy = readPolicy(options.repoRoot);
  const migrationManifest = validateMigrationManifest(options.repoRoot, policy);
  validateReleaseManifest(options.releaseManifest, policy, migrationManifest);
  validateImageDelivery(options.repoRoot);

  if (options.releaseChannel === "stable" && !policy.stableReleaseAllowed) {
    fail("stable release remains blocked until backup, restore, compatibility-window, and upgrade rehearsal governance is approved");
  }
  if (options.deploymentMode === "data-retaining" && !policy.dataRetainingDeploymentAllowed) {
    fail("data-retaining deployment remains blocked until backup, restore, compatibility-window, and upgrade rehearsal governance is approved");
  }

  process.stdout.write(`migration governance verified: phase=${policy.phase} channel=${options.releaseChannel} deployment=${options.deploymentMode} migrations=${migrationManifest.manifest.migrations.length}\n`);
}

try {
  main();
} catch (error) {
  process.stderr.write(`migration governance check failed: ${error.message}\n`);
  process.exitCode = 1;
}
