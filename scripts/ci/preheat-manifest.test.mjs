import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";

import {
  buildPreheatManifest,
  canonicalPreheatManifestBytes,
  PREHEAT_MANIFEST_KIND,
  validatePreheatManifest,
} from "./preheat-manifest.mjs";
import { sha256Digest } from "./resolve-release-component-composition.mjs";
import { readThirdPartyPolicy } from "./third-party-image-policy.mjs";

const root = path.resolve(import.meta.dirname, "../..");
const policyPath = path.join(root, "scripts/ci/third-party-image-policy.json");
const digest = (letter) => `sha256:${letter.repeat(64)}`;
const ref = (registry, repository, letter) => `${registry}/${repository}@${digest(letter)}`;

function runtime(name, letter) {
  return {
    name,
    refs: [
      { raw: ref("docker.io", `yyhuni/lunafox-${name}`, letter) },
      { raw: ref("ghcr.io", `yyhuni/lunafox-${name}`, letter) },
    ],
  };
}

function component(id, repository, letter) {
  return {
    id,
    artifact: {
      ref: ref("ghcr.io", repository, letter),
      digest: digest(letter),
      platforms: ["linux/amd64", "linux/arm64"],
    },
  };
}

function fixture() {
  const runtimeImages = [
    runtime("server", "1"),
    runtime("frontend", "2"),
    runtime("nginx", "3"),
    runtime("agent", "4"),
    runtime("bootstrap", "5"),
  ];
  const enginePackages = [
    [
      { raw: ref("docker.io", "yyhuni/lunafox-engine-runtime-port-scan", "8") },
      { raw: ref("ghcr.io", "yyhuni/lunafox-engine-runtime-port-scan", "8") },
    ],
    [
      { raw: ref("docker.io", "yyhuni/lunafox-engine-runtime-nuclei-vulnerability", "9") },
      { raw: ref("ghcr.io", "yyhuni/lunafox-engine-runtime-nuclei-vulnerability", "9") },
    ],
  ];
  const composition = {
    releaseTag: "v1.2.3",
    compositionDigest: digest("a"),
    components: [
      ...runtimeImages.map((entry) => component(`runtime.${entry.name}`, `yyhuni/lunafox-${entry.name}`, entry.refs[0].raw.at(-1))),
      component("engine.lunafox.port_scan.runtime", "yyhuni/lunafox-engine-runtime-port-scan", "6"),
      component("engine.lunafox.nuclei_vulnerability.runtime", "yyhuni/lunafox-engine-runtime-nuclei-vulnerability", "7"),
      component("engine.lunafox.port_scan.package", "yyhuni/lunafox-engine-runtime-port-scan", "8"),
      component("engine.lunafox.nuclei_vulnerability.package", "yyhuni/lunafox-engine-runtime-nuclei-vulnerability", "9"),
    ],
  };
  const services = [
    ["server", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-server@sha256:" + "1".repeat(64)],
    ["frontend", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-frontend@sha256:" + "2".repeat(64)],
    ["nginx", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-nginx@sha256:" + "3".repeat(64)],
    ["agent", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-agent@sha256:" + "4".repeat(64)],
    ["agent-preflight", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-agent@sha256:" + "4".repeat(64)],
    ["engine-preheater", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-agent@sha256:" + "4".repeat(64)],
    ["bootstrap", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-bootstrap@sha256:" + "5".repeat(64)],
    ["config-init", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-bootstrap@sha256:" + "5".repeat(64)],
    ["migrate", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-bootstrap@sha256:" + "5".repeat(64)],
    ["cert-init", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-bootstrap@sha256:" + "5".repeat(64)],
    ["upgrader", "${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-bootstrap@sha256:" + "5".repeat(64)],
    ["postgres", "docker.io/library/postgres@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675", "embedded"],
    ["redis", "docker.io/library/redis@sha256:2afba59292f25f5d1af200496db41bea2c6c816b059f57ae74703a50a03a27d0"],
    ["loki", "docker.io/grafana/loki@sha256:3c8fd3570dd9219951a60d3f919c7f31923d10baee578b77bc26c4a0b32d092d"],
    ["alloy", "docker.io/grafana/alloy@sha256:b8ec653c44235fbe910879145dac3597d66b0aaecf60bcbbe82580767771a839"],
  ];
  const compose = `services:\n${services.map(([name, image, profile]) => `  ${name}:\n    image: ${image}\n${profile ? `    profiles:\n    - ${profile}\n` : ""}`).join("")}volumes:\n  data: {}\n`;
  return {
    releaseManifestBytes: Buffer.from("releaseVersion: 1.2.3\n"),
    releaseTag: "v1.2.3",
    composition,
    composeBytes: Buffer.from(compose),
    thirdPartyPolicyBytes: fs.readFileSync(policyPath),
    thirdPartyPolicy: readThirdPartyPolicy(policyPath),
    runtimeImages,
    enginePackages,
  };
}

test("builds a canonical release-bound closure including disabled Engine Runtimes", () => {
  const inputs = fixture();
  const first = buildPreheatManifest(inputs);
  const second = buildPreheatManifest(structuredClone(inputs));
  assert.equal(first.kind, PREHEAT_MANIFEST_KIND);
  assert.deepEqual(first, second);
  assert.deepEqual(canonicalPreheatManifestBytes(first), canonicalPreheatManifestBytes(second));
  assert.equal(first.manifestDigest, sha256Digest({
    schemaVersion: first.schemaVersion,
    kind: first.kind,
    release: first.release,
    entries: first.entries,
    profileClosures: first.profileClosures,
  }));
  const disabledNuclei = first.entries.find((entry) => entry.repository.endsWith("nuclei-vulnerability"));
  assert.ok(disabledNuclei, "the published but default-disabled nuclei Runtime remains in the closure");
  assert.deepEqual(disabledNuclei.profiles, ["embedded", "external"]);
  assert.equal(first.profileClosures.find((closure) => closure.profile === "embedded").entries.some((entry) => entry.includes("library/postgres@")), true);
  assert.equal(first.profileClosures.find((closure) => closure.profile === "external").entries.some((entry) => entry.includes("library/postgres@")), false);
  const agent = first.entries.find((entry) => entry.repository === "yyhuni/lunafox-agent");
  assert.deepEqual(agent.sources.map((source) => source.name), ["agent", "agent-preflight", "engine-preheater"]);
  assert.deepEqual(agent.candidates.map((candidate) => candidate.split("/")[0]), ["docker.io", "ghcr.io"]);
});

test("fails closed for bind drift, unbound runtime, duplicate identities, and unsupported platform", () => {
  const inputs = fixture();
  const manifest = buildPreheatManifest(inputs);
  const modifiedCompose = { ...inputs, composeBytes: Buffer.from(`${inputs.composeBytes}\n# drift\n`) };
  assert.notEqual(buildPreheatManifest(modifiedCompose).release.composeDigest, manifest.release.composeDigest);

  const missingBinding = structuredClone(inputs);
  missingBinding.composition.components = missingBinding.composition.components.filter((entry) => !entry.id.includes("nuclei_vulnerability"));
  assert.throws(() => buildPreheatManifest(missingBinding), /not bound by runtime composition/);

  const duplicate = structuredClone(manifest);
  duplicate.entries.push(structuredClone(duplicate.entries[0]));
  duplicate.manifestDigest = sha256Digest({
    schemaVersion: duplicate.schemaVersion,
    kind: duplicate.kind,
    release: duplicate.release,
    entries: duplicate.entries,
    profileClosures: duplicate.profileClosures,
  });
  assert.throws(() => validatePreheatManifest(duplicate), /duplicate image identities/);

  const unsupportedPlatform = structuredClone(manifest);
  unsupportedPlatform.entries[0].platforms = ["linux/s390x"];
  assert.throws(() => validatePreheatManifest(unsupportedPlatform), /unsupported platform/);
});

test("canonical bytes are a strict input contract", () => {
  const manifest = buildPreheatManifest(fixture());
  const bytes = canonicalPreheatManifestBytes(manifest);
  assert.equal(crypto.createHash("sha256").update(bytes).digest("hex").length, 64);
  assert.equal(bytes.toString("utf8"), JSON.stringify(JSON.parse(bytes.toString("utf8"))));
});
