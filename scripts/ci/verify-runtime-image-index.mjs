#!/usr/bin/env node

import fs from "node:fs";
import crypto from "node:crypto";
import { pathToFileURL } from "node:url";

const OCI_INDEX_MEDIA_TYPE = "application/vnd.oci.image.index.v1+json";
const OCI_IMAGE_MANIFEST_MEDIA_TYPE = "application/vnd.oci.image.manifest.v1+json";
const BUILDKIT_ATTESTATION_REFERENCE_TYPE = "attestation-manifest";
const BUILDKIT_ATTESTATION_REFERENCE_DIGEST = "vnd.docker.reference.digest";
const BUILDKIT_ATTESTATION_REFERENCE_TYPE_ANNOTATION = "vnd.docker.reference.type";
const SHA256 = /^sha256:[a-f0-9]{64}$/;

function usage() {
  return `Usage: node scripts/ci/verify-runtime-image-index.mjs --raw-file FILE --expected-digest sha256:... [--expected-platforms linux/amd64,linux/arm64]\n\nOptions:\n  --raw-file FILE             Raw OCI image index JSON returned by the registry\n  --expected-digest DIGEST    Digest of the raw index bytes\n  --expected-platforms LIST   Exact comma-separated Linux platform set\n`;
}

function parseArgs(argv) {
  const args = { rawFile: "", expectedDigest: "", expectedPlatforms: "linux/amd64,linux/arm64" };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--raw-file") args.rawFile = argv[++i] ?? "";
    else if (arg === "--expected-digest") args.expectedDigest = argv[++i] ?? "";
    else if (arg === "--expected-platforms") args.expectedPlatforms = argv[++i] ?? "";
    else if (arg === "--help" || arg === "-h") {
      process.stdout.write(usage());
      process.exit(0);
    } else throw new Error(`unknown argument: ${arg}`);
  }
  if (!args.rawFile || !args.expectedDigest) throw new Error("--raw-file and --expected-digest are required");
  return args;
}

function parseExpectedPlatforms(raw) {
  const platforms = raw.split(",");
  if (platforms.length === 0 || platforms.some((platform) => platform !== "linux/amd64" && platform !== "linux/arm64")) {
    throw new Error("expected platforms must contain only linux/amd64 and linux/arm64");
  }
  if (new Set(platforms).size !== platforms.length) throw new Error("expected platforms must not repeat a platform");
  return platforms;
}

function validateIndex(raw, expectedDigest, expectedPlatforms = "linux/amd64,linux/arm64") {
  if (!SHA256.test(expectedDigest)) throw new Error("expected digest must be a lowercase sha256 digest");
  const platforms = Array.isArray(expectedPlatforms) ? expectedPlatforms : parseExpectedPlatforms(expectedPlatforms);
  if (platforms.length === 0) throw new Error("expected platforms are required");
  const actualDigest = `sha256:${crypto.createHash("sha256").update(raw).digest("hex")}`;
  if (actualDigest !== expectedDigest) throw new Error(`index digest mismatch: got ${actualDigest}, want ${expectedDigest}`);
  let index;
  try {
    index = JSON.parse(raw.toString("utf8"));
  } catch (error) {
    throw new Error(`decode OCI image index: ${error.message}`);
  }
  if (index.schemaVersion !== 2) throw new Error(`OCI image index schemaVersion must be 2, got ${index.schemaVersion}`);
  if (index.mediaType !== OCI_INDEX_MEDIA_TYPE) throw new Error(`OCI image index mediaType must be ${OCI_INDEX_MEDIA_TYPE}`);
  if (index.artifactType !== undefined && index.artifactType !== "") throw new Error("Engine Runtime Image index must not declare artifactType");
  if (index.subject !== undefined && index.subject !== null) throw new Error("Engine Runtime Image index must not declare subject");
  if (!Array.isArray(index.manifests)) throw new Error("OCI image index manifests must be an array");
  const required = new Map(platforms.map((platform) => [platform, 0]));
  const digests = new Set();
  const platformDescriptors = [];
  const attestationDescriptors = [];
  for (const [position, descriptor] of index.manifests.entries()) {
    if (!descriptor || descriptor.mediaType !== OCI_IMAGE_MANIFEST_MEDIA_TYPE) {
      throw new Error(`manifests[${position}] must be an OCI image manifest descriptor`);
    }
    if (typeof descriptor.digest !== "string" || !SHA256.test(descriptor.digest)) {
      throw new Error(`manifests[${position}] digest must be a lowercase sha256 digest`);
    }
    if (!Number.isInteger(descriptor.size) || descriptor.size <= 0) throw new Error(`manifests[${position}] size must be positive`);
    if (!descriptor.platform || typeof descriptor.platform.os !== "string" || typeof descriptor.platform.architecture !== "string" || !descriptor.platform.os || !descriptor.platform.architecture) {
      throw new Error(`manifests[${position}] platform must be non-empty`);
    }
    if (digests.has(descriptor.digest)) throw new Error(`manifests[${position}] repeats child digest ${descriptor.digest}`);
    digests.add(descriptor.digest);
    const platform = `${descriptor.platform.os}/${descriptor.platform.architecture}`;
    if (platform === "unknown/unknown") {
      const annotations = descriptor.annotations;
      if (!annotations || typeof annotations !== "object" || Array.isArray(annotations) ||
          annotations[BUILDKIT_ATTESTATION_REFERENCE_TYPE_ANNOTATION] !== BUILDKIT_ATTESTATION_REFERENCE_TYPE) {
        throw new Error(`manifests[${position}] may use unknown/unknown only for a BuildKit attestation descriptor`);
      }
      const referencedDigest = annotations[BUILDKIT_ATTESTATION_REFERENCE_DIGEST];
      if (typeof referencedDigest !== "string" || !SHA256.test(referencedDigest)) {
        throw new Error(`manifests[${position}] BuildKit attestation must reference a lowercase platform manifest sha256 digest`);
      }
      attestationDescriptors.push({ position, referencedDigest });
      continue;
    }
    if (descriptor.platform.os === "unknown" || descriptor.platform.architecture === "unknown") {
      throw new Error(`manifests[${position}] must not mix an unknown platform dimension with a Runtime Image platform`);
    }
    platformDescriptors.push({ position, digest: descriptor.digest, platform });
  }
  if (platformDescriptors.length !== platforms.length) {
    throw new Error(`OCI image index must contain exactly ${platforms.length} platform manifests, got ${platformDescriptors.length}`);
  }
  const platformDigests = new Set();
  for (const descriptor of platformDescriptors) {
    platformDigests.add(descriptor.digest);
    if (required.has(descriptor.platform)) required.set(descriptor.platform, required.get(descriptor.platform) + 1);
  }
  for (const [platform, count] of required.entries()) if (count !== 1) throw new Error(`required platform ${platform} must appear exactly once, got ${count}`);
  const attestationReferences = new Map();
  for (const { position, referencedDigest } of attestationDescriptors) {
    if (!platformDigests.has(referencedDigest)) {
      throw new Error(`manifests[${position}] BuildKit attestation must reference a platform manifest in this index`);
    }
    const count = (attestationReferences.get(referencedDigest) ?? 0) + 1;
    if (count > 1) {
      throw new Error(`BuildKit attestations must have at most one descriptor per platform manifest, repeated ${referencedDigest}`);
    }
    attestationReferences.set(referencedDigest, count);
  }
  return { index, actualDigest, platforms: [...required.keys()], attestationCount: attestationDescriptors.length };
}

function main() {
  let args;
  try { args = parseArgs(process.argv.slice(2)); } catch (error) { process.stderr.write(`error: ${error.message}\n${usage()}`); process.exit(2); }
  try {
    const raw = fs.readFileSync(args.rawFile);
    const verified = validateIndex(raw, args.expectedDigest, args.expectedPlatforms);
    process.stdout.write(`${JSON.stringify({ digest: verified.actualDigest, mediaType: OCI_INDEX_MEDIA_TYPE, platforms: verified.platforms })}\n`);
  } catch (error) {
    process.stderr.write(`runtime image index verification failed: ${error.message}\n`);
    process.exit(1);
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main();

export { validateIndex };
