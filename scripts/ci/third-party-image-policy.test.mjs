#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import {
  canonicalThirdPartyPolicyBytes,
  policyForProfile,
  readThirdPartyPolicy,
  validatePolicyAgainstComposeTemplate,
  validateThirdPartyPolicy,
} from "./third-party-image-policy.mjs";

const root = path.resolve(import.meta.dirname, "../..");
const sourcePolicyPath = path.join(root, "scripts/ci/third-party-image-policy.json");
const composeTemplatePath = path.join(root, "deploy/compose.template.yaml");

function policyFixture() {
  return JSON.parse(fs.readFileSync(sourcePolicyPath, "utf8"));
}

function writePolicy(t, value) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-third-party-policy-"));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const file = path.join(directory, "third-party-image-policy.json");
  fs.writeFileSync(file, value);
  return file;
}

test("accepts the canonical complete policy and both profile closures", () => {
  const policy = readThirdPartyPolicy(sourcePolicyPath);
  assert.deepEqual(policyForProfile(policy, "embedded").map((entry) => entry.service), ["postgres", "redis", "loki", "alloy"]);
  assert.deepEqual(policyForProfile(policy, "external").map((entry) => entry.service), ["redis", "loki", "alloy"]);
  assert.deepEqual(canonicalThirdPartyPolicyBytes(policy), fs.readFileSync(sourcePolicyPath));
  assert.doesNotThrow(() => validatePolicyAgainstComposeTemplate(policy, fs.readFileSync(composeTemplatePath, "utf8")));
});

for (const [label, mutate, expected] of [
  ["digest drift", (policy) => { policy.entries[0].digest = `sha256:${"f".repeat(64)}`; }, /Compose template digest/],
  ["unknown service", (policy) => { policy.entries[3].service = "nginx"; }, /unsupported/],
  ["profile omission", (policy) => { policy.entries[1].profiles = ["embedded"]; }, /profiles/],
  ["extra entry", (policy) => { policy.entries.push(structuredClone(policy.entries[0])); }, /complete fixed service closure/],
  ["unbounded provenance", (policy) => { policy.entries[0].evidence = "release-frozen image digest reviewed by LunaFox and publisher verified"; }, /bounded LunaFox review/],
]) {
  test(`rejects ${label}`, () => {
    const policy = policyFixture();
    mutate(policy);
    if (label === "digest drift") {
      assert.throws(() => validatePolicyAgainstComposeTemplate(policy, fs.readFileSync(composeTemplatePath, "utf8")), expected);
    } else {
      assert.throws(() => validateThirdPartyPolicy(policy), expected);
    }
  });
}

test("rejects non-canonical JSON bytes", (t) => {
  const policy = policyFixture();
  const file = writePolicy(t, `${JSON.stringify(policy, null, 2)}\n\n`);
  assert.throws(() => readThirdPartyPolicy(file), /canonical JSON bytes/);
});
