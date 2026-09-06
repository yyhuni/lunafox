#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const defaultRepoRoot = path.resolve(scriptDir, "../..");

const expectedPolicy = {
  schemaVersion: 1,
  phase: "disposable-development",
  baselineMigration: "000001",
  baselineMutable: true,
  preserveDataUpgrade: false,
  developmentRollback: "git-history-and-empty-state-fresh-install",
  stableReleaseAllowed: false,
  dataRetainingDeploymentAllowed: false,
  freezeTriggers: [
    "first-public-stable-release",
    "first-data-retaining-non-disposable-deployment",
  ],
};

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  const options = {
    repoRoot: defaultRepoRoot,
    releaseChannel: "development",
    deploymentMode: "disposable",
  };

  for (let index = 0; index < argv.length; index += 1) {
    const flag = argv[index];
    const value = argv[index + 1];
    if (!["--repo-root", "--release-channel", "--deployment-mode"].includes(flag)) {
      fail(`unknown argument: ${flag}`);
    }
    if (!value) {
      fail(`missing value for ${flag}`);
    }
    if (flag === "--repo-root") options.repoRoot = path.resolve(value);
    if (flag === "--release-channel") options.releaseChannel = value;
    if (flag === "--deployment-mode") options.deploymentMode = value;
    index += 1;
  }

  if (!["development", "canary", "stable"].includes(options.releaseChannel)) {
    fail(`unsupported release channel: ${options.releaseChannel}`);
  }
  if (!["disposable", "data-retaining"].includes(options.deploymentMode)) {
    fail(`unsupported deployment mode: ${options.deploymentMode}`);
  }
  return options;
}

function readPolicy(repoRoot) {
  const policyPath = path.join(repoRoot, "server/cmd/server/migrations/policy.json");
  let policy;
  try {
    policy = JSON.parse(fs.readFileSync(policyPath, "utf8"));
  } catch (error) {
    fail(`cannot read migration policy ${policyPath}: ${error.message}`);
  }

  try {
    assert.deepStrictEqual(policy, expectedPolicy);
  } catch {
    fail("migration policy drift requires an explicit baseline-freeze governance change");
  }
  return policy;
}

function validateMigrationFiles(repoRoot) {
  const migrationsDir = path.join(repoRoot, "server/cmd/server/migrations");
  const migrationFiles = fs.readdirSync(migrationsDir)
    .filter((name) => name.endsWith(".up.sql") || name.endsWith(".down.sql"))
    .sort();
  assert.deepStrictEqual(migrationFiles, [
    "000001_init_schema.down.sql",
    "000001_init_schema.up.sql",
  ], "disposable development must keep exactly one squashed 000001 migration pair");

  const downPath = path.join(migrationsDir, "000001_init_schema.down.sql");
  const downSql = fs.readFileSync(downPath, "utf8");
  assert.ok(
    downSql.startsWith("-- DESTRUCTIVE DEVELOPMENT/TEST TEARDOWN ONLY."),
    "000001 down migration must declare its development/test-only destructive purpose",
  );
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const policy = readPolicy(options.repoRoot);
  validateMigrationFiles(options.repoRoot);

  if (options.releaseChannel === "stable" && !policy.stableReleaseAllowed) {
    fail("migration baseline freeze required before the first public stable release");
  }
  if (options.deploymentMode === "data-retaining" && !policy.dataRetainingDeploymentAllowed) {
    fail("migration baseline freeze required before the first data-retaining deployment");
  }

  process.stdout.write(
    `migration baseline policy verified: phase=${policy.phase} channel=${options.releaseChannel} deployment=${options.deploymentMode}\n`,
  );
}

try {
  main();
} catch (error) {
  process.stderr.write(`migration baseline policy check failed: ${error.message}\n`);
  process.exitCode = 1;
}
