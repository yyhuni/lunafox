#!/usr/bin/env node

/**
 * Build and validate the release-bound image closure consumed by the Compose
 * engine preheater.  The file is deliberately canonical JSON: the same
 * release inputs must always produce byte-identical output, and consumers
 * reject reformatted or partially substituted manifests before they touch the
 * Docker daemon.
 */

import crypto from "node:crypto";

import { canonicalJson, sha256Digest } from "./resolve-release-component-composition.mjs";
import { policyForProfile, validateThirdPartyPolicy } from "./third-party-image-policy.mjs";

export const PREHEAT_MANIFEST_SCHEMA_VERSION = 1;
export const PREHEAT_MANIFEST_KIND = "lunafox.preheat-manifest";
export const PREHEAT_MANIFEST_ALGORITHM = "sha256-canonical-json-v1";
export const PREHEAT_PROFILES = Object.freeze(["embedded", "external"]);
export const PREHEAT_PLATFORMS = Object.freeze(["linux/amd64", "linux/arm64"]);

const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const RELEASE_TAG_RE = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const REPOSITORY_RE = /^[a-z0-9][a-z0-9._-]*(?:\/[a-z0-9][a-z0-9._-]*)+$/;
const SOURCE_NAME_RE = /^[a-z][a-z0-9_.-]*$/;
const FIRST_PARTY_REPOSITORY_RE = /^yyhuni\/lunafox-[a-z0-9][a-z0-9._-]*$/;

// Byte-sha256 pins for already-published preheat manifests whose entry schema
// predates the cloudflareCandidates removal. A digest match is byte-level
// integrity, so strict schema validation is skipped for exactly these bytes —
// the same policy as the published-without-composition release-manifest
// exemption. Extending the list requires a reviewed code change; self-declared
// manifest fields never grant the exemption.
export const PUBLISHED_LEGACY_PREHEAT_MANIFEST_SHA256 = Object.freeze([
  // v0.0.1-alpha.198 deployment snapshot retained on public main.
  "c6d1308be2a3c1ca561938956079a42d00688504fffd98a784ecb1d228a39888",
]);

function fail(message) {
  throw new Error(message);
}

function assert(condition, message) {
  if (!condition) fail(message);
}

function isObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function assertExactKeys(value, keys, label) {
  assert(isObject(value), `${label} must be an object`);
  const actual = Object.keys(value).sort();
  const expected = [...keys].sort();
  assert(JSON.stringify(actual) === JSON.stringify(expected), `${label} has unknown or missing fields`);
}

function sha256Bytes(value) {
  return `sha256:${crypto.createHash("sha256").update(value).digest("hex")}`;
}

function requiredBytes(value, label) {
  assert((Buffer.isBuffer(value) || value instanceof Uint8Array) && value.length > 0, `${label} are required`);
  return Buffer.from(value);
}

function parseReference(raw, label) {
  assert(typeof raw === "string" && raw === raw.trim(), `${label} must be a canonical digest-qualified OCI reference`);
  const match = /^([a-z0-9][a-z0-9.-]*(?::\d+)?)\/([a-z0-9][a-z0-9._-]*(?:\/[a-z0-9][a-z0-9._-]*)+)@(sha256:[a-f0-9]{64})$/.exec(raw);
  assert(match, `${label} must be a canonical digest-qualified OCI reference`);
  return { raw, registry: match[1], repository: match[2], digest: match[3] };
}

function identity(entry) {
  return `${entry.repository}@${entry.digest}`;
}

function normalizePlatforms(platforms, label) {
  assert(Array.isArray(platforms) && platforms.length > 0, `${label} must contain at least one platform`);
  const normalized = [...new Set(platforms.map((platform) => String(platform)))].sort();
  assert(normalized.length === platforms.length, `${label} contains duplicate platforms`);
  assert(normalized.every((platform) => PREHEAT_PLATFORMS.includes(platform)), `${label} contains an unsupported platform`);
  assert(JSON.stringify(normalized) === JSON.stringify(platforms), `${label} must use canonical platform order`);
  return normalized;
}

function normalizeProfiles(profiles, label) {
  assert(Array.isArray(profiles) && profiles.length > 0, `${label} must contain at least one profile`);
  const normalized = [...new Set(profiles.map((profile) => String(profile)))].sort();
  assert(normalized.length === profiles.length, `${label} contains duplicate profiles`);
  assert(normalized.every((profile) => PREHEAT_PROFILES.includes(profile)), `${label} contains an unsupported profile`);
  assert(JSON.stringify(normalized) === JSON.stringify(profiles), `${label} must use canonical profile order`);
  return normalized;
}

function normalizeSources(sources, label) {
  assert(Array.isArray(sources) && sources.length > 0, `${label} must contain logical sources`);
  const normalized = sources.map((source, index) => {
    assertExactKeys(source, ["kind", "name"], `${label}[${index}]`);
    assert(["compose-service", "engine-runtime"].includes(source.kind), `${label}[${index}].kind is unsupported`);
    assert(typeof source.name === "string" && SOURCE_NAME_RE.test(source.name), `${label}[${index}].name is invalid`);
    return { kind: source.kind, name: source.name };
  }).sort((left, right) => `${left.kind}:${left.name}`.localeCompare(`${right.kind}:${right.name}`));
  assert(new Set(normalized.map((source) => `${source.kind}:${source.name}`)).size === normalized.length, `${label} contains duplicate logical sources`);
  assert(canonicalJson(normalized) === canonicalJson(sources), `${label} must use canonical source order`);
  return normalized;
}

function normalizeReferenceList(rawReferences, label, repository, digest) {
  assert(Array.isArray(rawReferences) && rawReferences.length > 0, `${label} must contain candidates`);
  const references = rawReferences.map((reference, index) => parseReference(reference, `${label}[${index}]`));
  assert(references.every((reference) => reference.repository === repository && reference.digest === digest), `${label} does not preserve image identity`);
  assert(new Set(references.map((reference) => reference.raw)).size === references.length, `${label} contains duplicate candidates`);
  return references;
}

function normalizeEntry(entry, index) {
  const label = `entries[${index}]`;
  assertExactKeys(entry, [
    "candidates", "digest", "identityReference", "platforms",
    "profiles", "repository", "sources", "trust",
  ], label);
  assert(typeof entry.repository === "string" && REPOSITORY_RE.test(entry.repository), `${label}.repository is invalid`);
  assert(DIGEST_RE.test(entry.digest), `${label}.digest must be a sha256 digest`);
  assert(entry.trust === "first-party" || entry.trust === "third-party", `${label}.trust is unsupported`);
  const platforms = normalizePlatforms(entry.platforms, `${label}.platforms`);
  const profiles = normalizeProfiles(entry.profiles, `${label}.profiles`);
  const sources = normalizeSources(entry.sources, `${label}.sources`);
  const candidates = normalizeReferenceList(entry.candidates, `${label}.candidates`, entry.repository, entry.digest);
  const identityReference = parseReference(entry.identityReference, `${label}.identityReference`);
  assert(identityReference.repository === entry.repository && identityReference.digest === entry.digest, `${label}.identityReference does not preserve image identity`);

  if (entry.trust === "first-party") {
    assert(FIRST_PARTY_REPOSITORY_RE.test(entry.repository), `${label}.repository is not a first-party LunaFox repository`);
    assert(candidates.length === 2 && candidates[0].registry === "docker.io" && candidates[1].registry === "ghcr.io", `${label}.candidates must be Docker Hub then GHCR`);
    assert(identityReference.raw === candidates[1].raw, `${label}.identityReference must be the GHCR candidate`);
  } else {
    assert(candidates.length === 1, `${label}.candidates must contain one third-party origin`);
    assert(identityReference.raw === candidates[0].raw, `${label}.identityReference must be the third-party origin`);
  }
  return {
    repository: entry.repository,
    digest: entry.digest,
    trust: entry.trust,
    platforms,
    profiles,
    sources,
    candidates: candidates.map((reference) => reference.raw),
    identityReference: identityReference.raw,
  };
}

function closureDigest(profile, entries) {
  return sha256Digest({ profile, entries });
}

function normalizeProfileClosures(value, entries) {
  assert(Array.isArray(value) && value.length === PREHEAT_PROFILES.length, "profileClosures must contain embedded and external closures");
  const identitiesByProfile = new Map(PREHEAT_PROFILES.map((profile) => [profile, entries
    .filter((entry) => entry.profiles.includes(profile))
    .map(identity)
    .sort()]));
  const normalized = value.map((closure, index) => {
    const label = `profileClosures[${index}]`;
    assertExactKeys(closure, ["digest", "entries", "profile"], label);
    assert(PREHEAT_PROFILES.includes(closure.profile), `${label}.profile is unsupported`);
    assert(Array.isArray(closure.entries), `${label}.entries must be an array`);
    const entriesForProfile = closure.entries.map((entry) => String(entry));
    assert(new Set(entriesForProfile).size === entriesForProfile.length, `${label}.entries contains duplicate identities`);
    assert(JSON.stringify([...entriesForProfile].sort()) === JSON.stringify(entriesForProfile), `${label}.entries must use canonical order`);
    const expected = identitiesByProfile.get(closure.profile);
    assert(JSON.stringify(entriesForProfile) === JSON.stringify(expected), `${label}.entries does not match the profile closure`);
    assert(DIGEST_RE.test(closure.digest) && closure.digest === closureDigest(closure.profile, entriesForProfile), `${label}.digest does not match its profile closure`);
    return { profile: closure.profile, entries: entriesForProfile, digest: closure.digest };
  }).sort((left, right) => left.profile.localeCompare(right.profile));
  assert(new Set(normalized.map((closure) => closure.profile)).size === PREHEAT_PROFILES.length, "profileClosures contains duplicate profiles");
  assert(canonicalJson(normalized) === canonicalJson(value), "profileClosures must use canonical profile order");
  return normalized;
}

function normalizeRelease(release) {
  assertExactKeys(release, ["composeDigest", "compositionDigest", "manifestDigest", "tag", "thirdPartyPolicyDigest"], "release");
  assert(typeof release.tag === "string" && RELEASE_TAG_RE.test(release.tag), "release.tag is invalid");
  for (const key of ["manifestDigest", "compositionDigest", "composeDigest", "thirdPartyPolicyDigest"]) {
    assert(DIGEST_RE.test(release[key]), `release.${key} must be a sha256 digest`);
  }
  return {
    tag: release.tag,
    manifestDigest: release.manifestDigest,
    compositionDigest: release.compositionDigest,
    composeDigest: release.composeDigest,
    thirdPartyPolicyDigest: release.thirdPartyPolicyDigest,
  };
}

export function preheatManifestCore(manifest) {
  const { manifestDigest: _manifestDigest, ...core } = manifest;
  return core;
}

export function validatePreheatManifest(manifest, options = {}) {
  // A byte-exact match against the published-legacy pins is integrity on its
  // own: only the already-published bytes can hit the list, so the removed
  // entry schema they carry needs no re-validation here. Binding checks in
  // verify-public-deployment.sh still anchor the file to the deployment.
  const fileSha256 = typeof options.fileSha256 === "string" ? options.fileSha256 : "";
  if (fileSha256 && PUBLISHED_LEGACY_PREHEAT_MANIFEST_SHA256.includes(fileSha256)) {
    assert(isObject(manifest) && typeof manifest.manifestDigest === "string", "preheat manifest is malformed");
    return manifest;
  }
  assertExactKeys(manifest, ["entries", "kind", "manifestDigest", "profileClosures", "release", "schemaVersion"], "preheat manifest");
  assert(manifest.schemaVersion === PREHEAT_MANIFEST_SCHEMA_VERSION, "preheat manifest schemaVersion is unsupported");
  assert(manifest.kind === PREHEAT_MANIFEST_KIND, "preheat manifest kind is invalid");
  const release = normalizeRelease(manifest.release);
  assert(Array.isArray(manifest.entries) && manifest.entries.length > 0, "preheat manifest must contain entries");
  const entries = manifest.entries.map(normalizeEntry).sort((left, right) => identity(left).localeCompare(identity(right)));
  assert(new Set(entries.map(identity)).size === entries.length, "preheat manifest contains duplicate image identities");
  assert(canonicalJson(entries) === canonicalJson(manifest.entries), "preheat manifest entries must use canonical identity order");
  const profileClosures = normalizeProfileClosures(manifest.profileClosures, entries);
  assert(DIGEST_RE.test(manifest.manifestDigest), "preheat manifest manifestDigest must be a sha256 digest");
  const normalized = {
    schemaVersion: PREHEAT_MANIFEST_SCHEMA_VERSION,
    kind: PREHEAT_MANIFEST_KIND,
    release,
    entries,
    profileClosures,
    manifestDigest: manifest.manifestDigest,
  };
  assert(normalized.manifestDigest === sha256Digest(preheatManifestCore(normalized)), "preheat manifest manifestDigest does not match canonical payload");
  return normalized;
}

function composeServices(compose) {
  assert(typeof compose === "string" && compose.length > 0, "rendered Compose content is required");
  const services = [];
  const section = /^services:\s*\n([\s\S]*?)(?=^volumes:|^networks:|^secrets:|(?![\s\S]))/m.exec(compose)?.[1] ?? "";
  for (const match of section.matchAll(/^  ([a-z][a-z0-9-]*):\n([\s\S]*?)(?=^  [a-z][a-z0-9-]*:|(?![\s\S]))/gm)) {
    const name = match[1];
    const block = match[2];
    const image = /^    image:\s*(\S+)\s*$/m.exec(block)?.[1] ?? "";
    assert(image, `Compose service ${name} is missing an image`);
    const profileBlock = /^    profiles:\s*\n((?:    - [a-z]+\s*\n?)+)/m.exec(block)?.[1] ?? "";
    const profiles = profileBlock
      ? [...profileBlock.matchAll(/^    - ([a-z]+)\s*$/gm)].map((entry) => entry[1])
      : [...PREHEAT_PROFILES];
    services.push({ name, image, profiles: normalizeProfiles([...profiles].sort(), `Compose service ${name} profiles`) });
  }
  assert(services.length > 0, "rendered Compose has no services");
  return services;
}

function componentRuntimeIdentity(composition, componentID, label) {
  const component = (composition.components ?? []).find((candidate) => candidate.id === componentID);
  assert(component, `${label} is missing from runtime composition`);
  assert(isObject(component.artifact), `${label} artifact is missing from runtime composition`);
  const reference = parseReference(component.artifact.ref, `${label}.artifact.ref`);
  assert(component.artifact.digest === reference.digest, `${label} artifact digest is inconsistent`);
  const platforms = normalizePlatforms(component.artifact.platforms, `${label}.artifact.platforms`);
  return { reference, platforms };
}

function engineRuntimeComponents(composition) {
  return (composition.components ?? [])
    .filter((component) => /^engine\.[a-z][a-z0-9_.-]*\.runtime$/.test(component.id))
    .map((component) => {
      const runtimeName = component.id.slice("engine.".length, -".runtime".length);
      const identity = componentRuntimeIdentity(composition, component.id, component.id);
      return {
        component,
        runtimeName,
        repository: identity.reference.repository,
        reference: identity.reference,
        platforms: identity.platforms,
      };
    })
    .sort((left, right) => left.component.id.localeCompare(right.component.id));
}

function firstPartyTransport(reference) {
  assert(reference.registry === "docker.io" || reference.registry === "ghcr.io", "first-party release candidates must use Docker Hub or GHCR");
  const docker = `docker.io/${reference.repository}@${reference.digest}`;
  const ghcr = `ghcr.io/${reference.repository}@${reference.digest}`;
  return {
    trust: "first-party",
    identityReference: ghcr,
    candidates: [docker, ghcr],
  };
}

function thirdPartyTransport(reference) {
  return {
    trust: "third-party",
    identityReference: reference.raw,
    candidates: [reference.raw],
  };
}

function addEntry(entries, draft) {
  const key = `${draft.repository}@${draft.digest}`;
  const existing = entries.get(key);
  if (!existing) {
    entries.set(key, {
      ...draft,
      profiles: new Set(draft.profiles),
      sources: new Map([[`${draft.source.kind}:${draft.source.name}`, draft.source]]),
    });
    return;
  }
  assert(existing.trust === draft.trust && existing.identityReference === draft.identityReference && JSON.stringify(existing.candidates) === JSON.stringify(draft.candidates), `preheat image identity has conflicting transport policy: ${key}`);
  assert(JSON.stringify(existing.platforms) === JSON.stringify(draft.platforms), `preheat image identity has conflicting platform declarations: ${key}`);
  for (const profile of draft.profiles) existing.profiles.add(profile);
  existing.sources.set(`${draft.source.kind}:${draft.source.name}`, draft.source);
}

/**
 * Build one preheat manifest from already validated release inputs.  Compose
 * parsing is intentionally limited to its image/profile surface: this keeps
 * the generated closure tied to what Compose will actually create without
 * adding a second deployment parser or a host-side pull implementation.
 */
export function buildPreheatManifest({
  releaseManifestBytes,
  releaseTag,
  composition,
  composeBytes,
  thirdPartyPolicyBytes,
  thirdPartyPolicy,
  runtimeImages,
  enginePackages,
}) {
  releaseManifestBytes = requiredBytes(releaseManifestBytes, "release manifest bytes");
  assert(typeof releaseTag === "string" && RELEASE_TAG_RE.test(releaseTag), "release tag is invalid");
  assert(isObject(composition), "runtime composition is required");
  composeBytes = requiredBytes(composeBytes, "rendered Compose bytes");
  thirdPartyPolicyBytes = requiredBytes(thirdPartyPolicyBytes, "third-party image policy bytes");
  const policy = validateThirdPartyPolicy(thirdPartyPolicy);
  assert(Array.isArray(runtimeImages) && runtimeImages.length > 0, "release runtime images are required");
  assert(Array.isArray(enginePackages) && enginePackages.length > 0, "release Engine Runtime images are required");
  assert(composition.releaseTag === releaseTag && DIGEST_RE.test(composition.compositionDigest), "runtime composition release binding is invalid");

  const entries = new Map();
  const runtimeByIdentity = new Map();
  for (const runtime of runtimeImages) {
    assert(typeof runtime.name === "string" && SOURCE_NAME_RE.test(runtime.name), "release Runtime image name is invalid");
    assert(Array.isArray(runtime.refs) && runtime.refs.length === 2, `release Runtime image ${runtime.name} must have Docker Hub and GHCR candidates`);
    const references = runtime.refs.map((reference, index) => parseReference(reference.raw ?? reference, `release Runtime image ${runtime.name}.refs[${index}]`));
    assert(references[0].registry === "docker.io" && references[1].registry === "ghcr.io" && references[0].repository === references[1].repository && references[0].digest === references[1].digest, `release Runtime image ${runtime.name} candidates are not equivalent`);
    const component = componentRuntimeIdentity(composition, `runtime.${runtime.name}`, `runtime.${runtime.name}`);
    assert(component.reference.repository === references[0].repository && component.reference.digest === references[0].digest, `runtime composition identity does not match release Runtime image ${runtime.name}`);
    runtimeByIdentity.set(`${references[0].repository}@${references[0].digest}`, { name: runtime.name, reference: references[0], platforms: component.platforms });
  }

  const thirdPartyByIdentity = new Map(policy.entries.map((entry) => {
    const reference = parseReference(`${entry.registry}/${entry.repository}@${entry.digest}`, `third-party policy ${entry.service}`);
    return [`${reference.repository}@${reference.digest}`, { entry, reference }];
  }));

  for (const service of composeServices(composeBytes.toString("utf8"))) {
    const firstParty = /(?:^|\/)yyhuni\/(lunafox-[a-z0-9][a-z0-9._-]*)@(sha256:[a-f0-9]{64})$/.exec(service.image);
    if (firstParty) {
      const key = `yyhuni/${firstParty[1]}@${firstParty[2]}`;
      const runtime = runtimeByIdentity.get(key);
      assert(runtime, `Compose service ${service.name} uses an unbound first-party image`);
      addEntry(entries, {
        repository: runtime.reference.repository,
        digest: runtime.reference.digest,
        platforms: runtime.platforms,
        profiles: service.profiles,
        source: { kind: "compose-service", name: service.name },
        ...firstPartyTransport(runtime.reference),
      });
      continue;
    }
    const thirdParty = parseReference(service.image, `Compose service ${service.name} image`);
    const policyEntry = thirdPartyByIdentity.get(`${thirdParty.repository}@${thirdParty.digest}`);
    assert(policyEntry && policyEntry.reference.registry === thirdParty.registry, `Compose service ${service.name} uses an unbound third-party image`);
    assert(JSON.stringify(service.profiles) === JSON.stringify(policyEntry.entry.profiles), `Compose service ${service.name} profile does not match third-party image policy`);
    addEntry(entries, {
      repository: thirdParty.repository,
      digest: thirdParty.digest,
      // The deployment contract supports both node platforms. Third-party
      // policy is the reviewed source for this fixed Compose service closure.
      platforms: [...PREHEAT_PLATFORMS],
      profiles: service.profiles,
      source: { kind: "compose-service", name: service.name },
      ...thirdPartyTransport(thirdParty),
    });
  }

  // The release YAML calls this field `enginePackages`, but its refs are the
  // OCI package artifacts.  Runtime image identity is deliberately sourced
  // from the composition's `engine.*.runtime` components: package and runtime
  // artifacts normally have different manifest digests even though they share
  // a repository.  Keeping this distinction prevents a package digest from
  // being pulled as if it were runnable.
  const packageByRepository = new Map();
  for (const [index, packageRefs] of enginePackages.entries()) {
    assert(Array.isArray(packageRefs) && packageRefs.length === 2, `release Engine Runtime ${index} must have Docker Hub and GHCR candidates`);
    const references = packageRefs.map((reference, candidateIndex) => parseReference(reference.raw ?? reference, `release Engine Runtime ${index}.refs[${candidateIndex}]`));
    assert(references[0].registry === "docker.io" && references[1].registry === "ghcr.io" && references[0].repository === references[1].repository && references[0].digest === references[1].digest, `release Engine Runtime ${index} candidates are not equivalent`);
    assert(!packageByRepository.has(references[0].repository), `release Engine Runtime ${index} duplicates repository ${references[0].repository}`);
    packageByRepository.set(references[0].repository, references);
  }
  const engineRuntimes = engineRuntimeComponents(composition);
  assert(engineRuntimes.length > 0, "runtime composition does not contain Engine Runtime components");
  const seenRuntimeRepositories = new Set();
  for (const runtime of engineRuntimes) {
    const packageRefs = packageByRepository.get(runtime.repository);
    assert(packageRefs, `Engine Runtime ${runtime.component.id} is missing from the release enginePackages inventory`);
    const packageComponent = (composition.components ?? []).find(
      (component) => component.id === `engine.${runtime.runtimeName}.package`,
    );
    assert(packageComponent, `${runtime.component.id} is missing its Engine Package composition component`);
    const packageReference = parseReference(packageComponent.artifact?.ref, `${packageComponent.id}.artifact.ref`);
    assert(
      packageReference.repository === packageRefs[0].repository && packageReference.digest === packageRefs[0].digest,
      `${runtime.component.id} Engine Package identity does not match the release enginePackages inventory`,
    );
    seenRuntimeRepositories.add(runtime.repository);
    addEntry(entries, {
      repository: runtime.repository,
      digest: runtime.reference.digest,
      platforms: runtime.platforms,
      profiles: [...PREHEAT_PROFILES],
      source: { kind: "engine-runtime", name: runtime.runtimeName },
      ...firstPartyTransport(runtime.reference),
    });
  }
  assert(seenRuntimeRepositories.size === packageByRepository.size, "release enginePackages inventory contains an Engine not bound by runtime composition");

  for (const profile of PREHEAT_PROFILES) {
    const expectedPolicyServices = new Set(policyForProfile(policy, profile).map((entry) => entry.service));
    const actualPolicyServices = new Set([...entries.values()]
      .filter((entry) => entry.trust === "third-party" && entry.profiles.has(profile))
      .flatMap((entry) => [...entry.sources.values()])
      .filter((source) => source.kind === "compose-service")
      .map((source) => source.name));
    assert(JSON.stringify([...actualPolicyServices].sort()) === JSON.stringify([...expectedPolicyServices].sort()), `Compose ${profile} closure does not match the third-party image policy`);
  }

  const normalizedEntries = [...entries.values()].map((entry) => ({
    repository: entry.repository,
    digest: entry.digest,
    trust: entry.trust,
    platforms: [...entry.platforms],
    profiles: [...entry.profiles].sort(),
    sources: [...entry.sources.values()].sort((left, right) => `${left.kind}:${left.name}`.localeCompare(`${right.kind}:${right.name}`)),
    candidates: [...entry.candidates],
    identityReference: entry.identityReference,
  })).sort((left, right) => identity(left).localeCompare(identity(right)));
  const manifest = {
    schemaVersion: PREHEAT_MANIFEST_SCHEMA_VERSION,
    kind: PREHEAT_MANIFEST_KIND,
    release: {
      tag: releaseTag,
      manifestDigest: sha256Bytes(releaseManifestBytes),
      compositionDigest: composition.compositionDigest,
      composeDigest: sha256Bytes(composeBytes),
      thirdPartyPolicyDigest: sha256Bytes(thirdPartyPolicyBytes),
    },
    entries: normalizedEntries,
    profileClosures: PREHEAT_PROFILES.map((profile) => {
      const closureEntries = normalizedEntries.filter((entry) => entry.profiles.includes(profile)).map(identity).sort();
      return { profile, entries: closureEntries, digest: closureDigest(profile, closureEntries) };
    }),
  };
  manifest.manifestDigest = sha256Digest(preheatManifestCore(manifest));
  return validatePreheatManifest(manifest);
}

export function canonicalPreheatManifestBytes(manifest) {
  return Buffer.from(canonicalJson(validatePreheatManifest(manifest)));
}
