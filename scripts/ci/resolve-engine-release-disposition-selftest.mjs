#!/usr/bin/env node

import assert from "node:assert/strict";

import { FINGERPRINT_SCHEMA_VERSION, sha256Digest } from "./resolve-release-component-composition.mjs";
import { packageVersionForPair, resolveSelection } from "./resolve-engine-release-disposition.mjs";

const digest = (letter) => `sha256:${letter.repeat(64)}`;
const releaseTag = "v1.2.3";
const engineIDs = ["engine.lunafox.port_scan", "engine.lunafox.website_discovery"];

function fingerprint(componentId, letter) {
  const inputs = {
    schemaVersion: FINGERPRINT_SCHEMA_VERSION,
    componentId,
    kind: "engine",
    contextPath: ".",
    dockerfile: `${componentId.replaceAll(".", "/")}/Dockerfile`,
    dockerignore: "",
    files: [],
    namedContexts: {},
    buildArgs: {},
    platforms: ["linux/amd64", "linux/arm64"],
    baseImages: [],
    baseImagesResolved: true,
    builderPolicy: { fixture: letter },
    generatedInputs: [],
  };
  return {
    version: FINGERPRINT_SCHEMA_VERSION,
    algorithm: "sha256-canonical-json-v1",
    digest: sha256Digest(inputs),
    baseImagesResolved: true,
    inputs,
  };
}

function component(engineId, role, disposition, fingerprintLetter, sourceTag = releaseTag) {
  const componentId = `${engineId}.${role}`;
  const value = {
    id: componentId,
    kind: "engine",
    name: engineId.slice("engine.lunafox.".length),
    inputFingerprint: fingerprint(componentId, fingerprintLetter),
    disposition,
    sourceRelease: disposition === "built"
      ? { tag: sourceTag, publicMergeCommit: "a".repeat(40) }
      : { tag: sourceTag, compositionDigest: digest("e") },
  };
  if (disposition === "reused") {
    const safe = engineId.replaceAll(".", "-");
    const artifactDigest = digest(role === "runtime" ? "f" : "0");
    const repository = role === "runtime"
      ? `lunafox-engine-runtime-${engineId.slice("engine.lunafox.".length).replaceAll("_", "-")}`
      : `lunafox-engine-${engineId.slice("engine.lunafox.".length).replaceAll("_", "-")}`;
    const prefix = role === "runtime" ? "engine-runtime-evidence" : "engine-package-evidence";
    value.artifact = { ref: `docker.io/yyhuni/${repository}@${artifactDigest}`, digest: artifactDigest };
    value.evidence = {
      image: `${prefix}/${safe}.json`,
      provenance: `${prefix}/${safe}.json#provenance`,
      sbom: `${prefix}/${safe}.json#sbom`,
      signature: `${prefix}/${safe}.json#signature`,
    };
  }
  return value;
}

function plan(components) {
  const value = {
    schemaVersion: 1,
    kind: "lunafox.runtime-composition-plan",
    releaseTag,
    sourceRevisionDigest: digest("b"),
    publicMergeCommit: "a".repeat(40),
    capabilities: { dynamicFrontendUpstream: true },
    components: [...components].sort((left, right) => left.id.localeCompare(right.id)),
  };
  value.planDigest = sha256Digest(((item) => {
    const { planDigest: _ignored, ...payload } = item;
    return payload;
  })(value));
  return value;
}

const discovery = {
  schemaVersion: "lunafox.engine-release-discovery.v1",
  engineRoot: "/tmp/engines",
  engines: engineIDs.map((engineId) => {
    const directory = engineId.slice("engine.lunafox.".length);
    return { engineId, directory, dockerfile: `${directory}/Dockerfile`, buildContext: ".", repository: `lunafox-engine-runtime-${directory}` };
  }),
};

const mixedPlan = plan([
  component(engineIDs[0], "runtime", "built", "1"),
  component(engineIDs[0], "package", "built", "2"),
  component(engineIDs[1], "runtime", "reused", "3", "v1.2.2"),
  component(engineIDs[1], "package", "reused", "4", "v1.2.2"),
]);
const selection = resolveSelection(mixedPlan, discovery);
assert.deepEqual(selection.builtEngineIds, [engineIDs[0]]);
assert.deepEqual(selection.reusedEngineIds, [engineIDs[1]]);
assert.match(selection.engines[0].packageVersion, /^0\.0\.0\+[a-f0-9]{64}$/);
assert.equal(selection.engines[0].packageVersion, packageVersionForPair(engineIDs[0], {
  runtime: mixedPlan.components.find((entry) => entry.id === `${engineIDs[0]}.runtime`),
  package: mixedPlan.components.find((entry) => entry.id === `${engineIDs[0]}.package`),
}));

const split = structuredClone(mixedPlan);
split.components.find((entry) => entry.id === `${engineIDs[1]}.package`).disposition = "built";
split.components.find((entry) => entry.id === `${engineIDs[1]}.package`).sourceRelease = { tag: releaseTag, publicMergeCommit: "a".repeat(40) };
delete split.components.find((entry) => entry.id === `${engineIDs[1]}.package`).artifact;
delete split.components.find((entry) => entry.id === `${engineIDs[1]}.package`).evidence;
split.planDigest = sha256Digest(((item) => { const { planDigest: _ignored, ...payload } = item; return payload; })(split));
assert.throws(() => resolveSelection(split, discovery), /Runtime\/Package disposition must match/);

const incomplete = structuredClone(mixedPlan);
incomplete.components = incomplete.components.filter((entry) => entry.id !== `${engineIDs[1]}.package`);
incomplete.planDigest = sha256Digest(((item) => { const { planDigest: _ignored, ...payload } = item; return payload; })(incomplete));
assert.throws(() => resolveSelection(incomplete, discovery), /must contain Runtime and Package/);

const staleInventory = structuredClone(discovery);
staleInventory.engines.pop();
assert.throws(() => resolveSelection(mixedPlan, staleInventory), /inventory does not match source discovery/);

process.stdout.write("ok - Engine release selection binds source inventory, paired disposition, and content-addressed package versions\n");
