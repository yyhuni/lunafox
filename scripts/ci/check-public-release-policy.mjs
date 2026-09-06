#!/usr/bin/env node

/**
 * Static proof for the private-to-public release boundary.
 *
 * This checker intentionally uses only the local files supplied by the
 * caller.  It is safe to run in the private release workflow and in the
 * generated public repository (where only the secretless workflow and the
 * machine-readable policy are available).
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const PUBLIC_REPOSITORY = "yyhuni/lunafox";
const PRIVATE_REPOSITORY = "yyhuni/lunafox-private";
const CANONICAL_NAMESPACE = "yyhuni";
const SIGNER_ISSUER = "https://token.actions.githubusercontent.com";
const PRIVATE_AGENT_SIGNER_IDENTITY = "https://github.com/yyhuni/lunafox-private/.github/workflows/release.yml@refs/tags/*";
const PUBLIC_RUNTIME_SIGNER_IDENTITY = "https://github.com/yyhuni/lunafox/.github/workflows/public-validate.yml@refs/heads/main";
const PUBLIC_APP_TOKEN_ACTION = "actions/create-github-app-token@v2";
const PUBLIC_APP_TOKEN_OUTPUT = "steps.public-repository-app-token.outputs.token";
const PUBLIC_APP_ID_VARIABLE_NAME = "LUNAFOX_PUBLIC_REPO_APP_ID";
const PUBLIC_APP_PRIVATE_KEY_SECRET_NAME = "LUNAFOX_PUBLIC_REPO_APP_PRIVATE_KEY";
const PUBLIC_APP_ID_VARIABLE = `vars.${PUBLIC_APP_ID_VARIABLE_NAME}`;
const PUBLIC_APP_PRIVATE_KEY_SECRET = `secrets.${PUBLIC_APP_PRIVATE_KEY_SECRET_NAME}`;
const PUBLIC_RELEASE_AUTOMATION_VARIABLE = "LUNAFOX_PUBLIC_RELEASE_AUTOMATION_ENABLED";
const PRIVATE_RELEASE_RUNNER_LABELS_VARIABLE = "LUNAFOX_PRIVATE_RELEASE_RUNNER_LABELS";
const PRIVATE_RELEASE_VERIFICATION_MODE_VARIABLE = "LUNAFOX_PRIVATE_RELEASE_VERIFICATION_MODE";
const DESTINATION_APP_JOBS = [
  "publish-public-export-pr",
  "request-public-export-auto-merge",
];
const PUBLIC_ENGINE_JOBS = [
  "public-engine-runtime-build",
  "public-engine-runtime-sign",
  "public-engine-package-build",
  "public-engine-package-publish",
  "public-engine-release-manifest",
];
const PUBLIC_RUNTIME_COMPONENTS = ["server", "frontend", "nginx", "bootstrap"];
const PUBLIC_RUNTIME_REPOSITORIES = PUBLIC_RUNTIME_COMPONENTS.map((component) => `lunafox-${component}`);
const FIRST_PUBLIC_TAG = "v0.0.1-alpha.57";
const SOURCE_TAG_PATTERN = "^refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+(?:-(?:alpha|beta|rc)\\.[0-9]+)?$";
const LEGACY_RE = /alpha\.46|SCHEMA_VERSION=2|IMAGE_REGISTRY=|IMAGE_NAMESPACE=|WORKER_IMAGE(?:_REF|_REFS)?=|lunafox-installer|checksums\.txt/;
const REQUIRED_PUBLIC_FRONTEND_PREFIXES = [
  "frontend/.storybook/",
  "frontend/__tests__/",
  "frontend/app/",
  "frontend/components/",
  "frontend/hooks/",
  "frontend/i18n/",
  "frontend/lib/",
  "frontend/messages/",
  "frontend/mock/",
  "frontend/public/",
  "frontend/scripts/",
  "frontend/services/",
  "frontend/styles/",
  "frontend/test/",
  "frontend/types/",
];
const REQUIRED_PUBLIC_FRONTEND_EXACT = [
  "frontend/Dockerfile",
  "frontend/.dockerignore",
  "frontend/eslint.config.mjs",
  "frontend/package.json",
  "frontend/pnpm-lock.yaml",
  "frontend/tsconfig.json",
  "frontend/vitest.config.ts",
  "frontend/vitest.setup.ts",
];
const REQUIRED_PUBLIC_FRONTEND_PATHS = [
  "frontend/package.json",
  "frontend/pnpm-lock.yaml",
  "frontend/app/page.tsx",
  "frontend/components/ui/button.tsx",
  "frontend/hooks/use-auth.ts",
  "frontend/lib/api-client.ts",
  "frontend/messages/en.json",
  "frontend/mock/server.ts",
  "frontend/scripts/route-inventory.mjs",
  "frontend/services/auth.service.ts",
  "frontend/styles/themes/index.css",
  "frontend/test/utils/render-with-providers.tsx",
  "frontend/types/auth.types.ts",
  "frontend/__tests__/vitest.config.contract.test.ts",
  "frontend/public/images/support/alipay-qr.jpg",
  "frontend/public/images/support/wechat-qr.jpg",
  "frontend/public/images/support/contact-qr.jpg",
];
const REQUIRED_PRIVATE_FRONTEND_TEST_DENIES = [
  "^frontend/__tests__/server-log-compose\\.contract\\.test\\.ts$",
  "^frontend/components/__tests__/ui-foundation-control-plane\\.contract\\.test\\.ts$",
  "^frontend/lib/__tests__/session-renewal-scope\\.contract\\.test\\.ts$",
];
const REQUIRED_PUBLIC_RUNTIME_PREFIXES = [
  "server/cmd/",
  "server/docs/",
  "server/internal/",
  "server/scripts/",
  "contracts/",
  "engine-go/",
  "proto/",
  "extensions/engines/",
  "extensions/workflows/",
  "docker/nginx/",
  "docker/bootstrap/",
  "tools/engine-release/",
  "tools/engine-oci-publish/",
];
const REQUIRED_PUBLIC_RUNTIME_EXACT = [
  "server/Dockerfile",
  "server/go.mod",
  "server/go.sum",
  "contracts/go.mod",
  "contracts/go.sum",
  "engine-go/go.mod",
  "engine-go/go.sum",
  "proto/buf.yaml",
  "proto/scripts/check-generated.sh",
  "extensions/engines/go.mod",
  "extensions/engines/go.sum",
  "extensions/workflows/default.scan-workflow.json",
  "docker/nginx/Dockerfile",
  "docker/nginx/nginx.conf",
  "docker/bootstrap/Dockerfile",
  "docker/bootstrap/bootstrap.sh",
  "tools/engine-release/go.mod",
  "tools/engine-release/go.sum",
  "tools/engine-oci-publish/go.mod",
  "tools/engine-oci-publish/go.sum",
  "scripts/ci/verify-public-runtime-contexts.mjs",
  "scripts/ci/verify-public-runtime-image-evidence.mjs",
  "scripts/ci/verify-public-runtime-image-evidence-selftest.mjs",
  "scripts/ci/verify-public-runtime-source.sh",
  "scripts/ci/check-migration-baseline-policy.mjs",
  "scripts/ci/check-engine-api-major-policy.mjs",
  "scripts/ci/publish-engine-runtime-images.sh",
  "scripts/ci/check-engine-image-tool-inventory.mjs",
  "scripts/ci/verify-distribution-registry-v2.mjs",
  "scripts/ci/verify-runtime-image-index.mjs",
];

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const args = {
    root: DEFAULT_ROOT,
    workflow: "",
    publicWorkflow: "",
    policy: "",
    exportPolicy: "",
    publicOnly: false,
    json: false,
  };
  const values = new Set(["--root-dir", "--workflow", "--public-workflow", "--policy", "--export-policy"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--public-only") { args.publicOnly = true; continue; }
    if (arg === "--json") { args.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/check-public-release-policy.mjs [--root-dir <dir>] [--public-only] [--json]\n");
      process.exit(0);
    }
    if (!values.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--root-dir") args.root = path.resolve(value);
    else if (arg === "--workflow") args.workflow = path.resolve(value);
    else if (arg === "--public-workflow") args.publicWorkflow = path.resolve(value);
    else if (arg === "--policy") args.policy = path.resolve(value);
    else if (arg === "--export-policy") args.exportPolicy = path.resolve(value);
  }
  return args;
}

function readText(file, label) {
  try { return fs.readFileSync(file, "utf8"); }
  catch (error) { fail(`cannot read ${label} ${file}: ${error.message}`); }
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label} ${file}: ${error.message}`); }
}

function jobBlock(workflow, name) {
  const start = workflow.indexOf(`  ${name}:`);
  if (start < 0) fail(`release workflow is missing ${name}`);
  const rest = workflow.slice(start + 1);
  const next = /^  [A-Za-z0-9_-]+:/m.exec(rest);
  return workflow.slice(start, next ? start + 1 + next.index : workflow.length);
}

function countOccurrences(text, value) {
  return text.split(value).length - 1;
}

function assertRuntimeComponentList(text, label) {
  // Keep the component inventory explicit in the workflow. A loose substring
  // check could accept a partial matrix and let one Runtime image disappear
  // from the release handoff.
  const inventory = text.match(/(?:components|PUBLIC_RUNTIME_COMPONENTS)\s*=\s*\(([^)]+)\)/)?.[1] ?? "";
  const words = new Set(inventory.match(/[a-z][a-z0-9-]*/g) ?? []);
  for (const component of PUBLIC_RUNTIME_COMPONENTS) {
    if (!words.has(component) && !new RegExp(`^\\s*(?:-\\s*)?component:\\s*${component}\\s*$`, "m").test(text)) {
      fail(`${label} must enumerate public Runtime component ${component}`);
    }
  }
}

function assertSecretlessBlock(block, label) {
  // Do not match GITHUB_PATH or prose containing the substring "PAT". Only
  // actual credential references are forbidden in public validation jobs.
  const secretReference = /\bsecrets\.[A-Za-z_][A-Za-z0-9_]*\b|\b(?:GITHUB_TOKEN|GH_TOKEN|DOCKERHUB_TOKEN|COSIGN_PRIVATE_KEY|[A-Z][A-Z0-9_]*_PAT)\b|\bgithub\.token\b/;
  if (secretReference.test(block)) fail(`${label} must be secretless`);
}

function assertDestinationAppCredentialPolicy(policy) {
  const credentials = policy.destinationAppCredentials;
  if (credentials?.appIdVariable !== PUBLIC_APP_ID_VARIABLE_NAME ||
      credentials?.privateKeySecret !== PUBLIC_APP_PRIVATE_KEY_SECRET_NAME ||
      credentials?.preferredScope !== "release-environment" ||
      credentials?.githubFreePrivateFallbackScope !== "repository" ||
      credentials?.storedInstallationTokenAllowed !== false) {
    fail("public release policy credential-source contract drifted");
  }
  if (policy.publicReleaseAutomation?.authorizationVariable !== PUBLIC_RELEASE_AUTOMATION_VARIABLE ||
      policy.publicReleaseAutomation?.default !== "false") {
    fail("public release policy must default cross-repository automation to disabled");
  }
}

function assertPrivateTrustedRunnerPolicy(policy) {
  const runner = policy.privateReleaseExecution;
  if (runner?.runnerLabelsVariable !== PRIVATE_RELEASE_RUNNER_LABELS_VARIABLE ||
      JSON.stringify(runner?.defaultRunnerLabels) !== JSON.stringify(["ubuntu-22.04"]) ||
      JSON.stringify(runner?.requiredSelfHostedLabels) !== JSON.stringify(["self-hosted", "linux", "x64", "lunafox-private-x64"]) ||
      runner?.consumerJobs !== "all-private-release-jobs" ||
      runner?.arm64Strategy !== "go-cross-compile") {
    fail("private trusted-runner policy contract drifted");
  }
}

function assertFirstPublicReleaseIdentity(policy, exportPolicy) {
  const expectedFiles = [
    `channels/${FIRST_PUBLIC_TAG}.env`,
    "channels/canary.env",
    `manifests/${FIRST_PUBLIC_TAG}.yaml`,
  ];
  if (policy.firstRelease?.tag !== FIRST_PUBLIC_TAG ||
      policy.firstRelease?.channel !== "canary" ||
      policy.firstRelease?.stableAllowed !== false) {
    fail(`public release policy must pin ${FIRST_PUBLIC_TAG} as the first canary`);
  }
  if (exportPolicy.sourceRefs?.firstPublicTag !== FIRST_PUBLIC_TAG) {
    fail(`public export policy must pin ${FIRST_PUBLIC_TAG} as the first public tag`);
  }
  if (JSON.stringify(policy.channel?.firstFiles ?? []) !== JSON.stringify(expectedFiles)) {
    fail(`public release policy must declare the first ${FIRST_PUBLIC_TAG} channel files`);
  }
  if ((exportPolicy.allowlist?.firstReleaseRequired ?? []).length !== 0) {
    fail("public source export must not require final release channel files");
  }
}

function assertFixedIdentity(workflow, policy) {
  for (const required of [
    "CANONICAL_NAMESPACE: yyhuni",
    "PUBLIC_REPOSITORY: yyhuni/lunafox",
    "PRIVATE_REPOSITORY: yyhuni/lunafox-private",
    `SIGNER_ISSUER: ${SIGNER_ISSUER}`,
    `SIGNER_IDENTITY: ${PRIVATE_AGENT_SIGNER_IDENTITY}`,
  ]) {
    if (!workflow.includes(required)) fail(`release workflow is missing fixed identity: ${required}`);
  }
  if (workflow.includes("github.repository_owner") || workflow.includes("GITHUB_REPOSITORY_OWNER")) {
    fail("release workflow must not derive a public namespace from repository context");
  }
  if (/DOCKERHUB_NAMESPACE:\s*\$\{\{/.test(workflow) || /DOCKERHUB_NAMESPACE=\$\{\{/.test(workflow)) {
    fail("release workflow must not use a dynamic Docker Hub namespace expression");
  }
  if (policy.canonicalRepository !== PUBLIC_REPOSITORY || policy.privateBuilderRepository !== PRIVATE_REPOSITORY || policy.canonicalNamespace !== CANONICAL_NAMESPACE) {
    fail("public release policy repository or namespace identity drifted");
  }
  if (policy.signer?.issuer !== SIGNER_ISSUER || policy.signer?.identityPattern !== "^https://github\\.com/yyhuni/lunafox/\\.github/workflows/public-validate\\.yml@refs/heads/main$") {
    fail("public Runtime signer identity drifted");
  }
  if (policy.binarySigner?.issuer !== SIGNER_ISSUER || policy.binarySigner?.identityPattern !== "^https://github\\.com/yyhuni/lunafox-private/\\.github/workflows/release\\.yml@refs/tags/\\*$") {
    fail("private Agent binary signer identity drifted");
  }
  assertDestinationAppCredentialPolicy(policy);
  assertPrivateTrustedRunnerPolicy(policy);
}

function assertRuntimeDestinationAppToken(block, job, { actionsRead = false, administrationRead = false, readOnly = false, pullRequests = true } = {}) {
  const required = [
    "environment: release",
    PUBLIC_APP_TOKEN_ACTION,
    "id: public-repository-app-token",
    `app-id: \${{ ${PUBLIC_APP_ID_VARIABLE} }}`,
    `private-key: \${{ ${PUBLIC_APP_PRIVATE_KEY_SECRET} }}`,
    "owner: yyhuni",
    "repositories: lunafox",
    "permission-metadata: read",
    PUBLIC_APP_TOKEN_OUTPUT,
  ];
  required.push(readOnly ? "permission-contents: read" : "permission-contents: write");
  if (pullRequests) required.push(readOnly ? "permission-pull-requests: read" : "permission-pull-requests: write");
  for (const value of required) {
    if (!block.includes(value)) fail(`${job} must mint a destination-only GitHub App token at runtime: missing ${value}`);
  }
  if (actionsRead && !block.includes("permission-actions: read")) {
    fail(`${job} must request actions: read to retrieve the public workflow evidence artifact`);
  }
  if (!actionsRead && block.includes("permission-actions: read")) {
    fail(`${job} must not request actions: read outside the evidence consumer`);
  }
  if (administrationRead && !block.includes("permission-administration: read")) {
    fail(`${job} must request administration: read to inspect public branch protection`);
  }
  if (!administrationRead && block.includes("permission-administration: read")) {
    fail(`${job} must not request administration: read outside branch-protection consumers`);
  }
  if (readOnly && /permission-(?:contents|pull-requests): write/.test(block)) {
    fail(`${job} must use read-only destination App permissions`);
  }
  if (!readOnly && /permission-contents: read/.test(block)) {
    fail(`${job} must retain contents write for its destination mutation`);
  }
}

function assertPrivateWorkflowLegacy(workflow, policy, publisherSource = "", autoMergeSource = "") {
  assertFixedIdentity(workflow, policy);
  const configuredRunnerExpression = "runs-on: ${{ fromJSON(vars." + PRIVATE_RELEASE_RUNNER_LABELS_VARIABLE + " || '[\"ubuntu-22.04\"]') }}";
  const tagValidator = jobBlock(workflow, "validate-tag");
  const trustedRunnerBoundary = jobBlock(workflow, "private-trusted-runner-boundary");
  const trustedRunnerConsumer = jobBlock(workflow, "publish-images-amd64");
  const preflight = jobBlock(workflow, "public-export-preflight");
  const authorization = jobBlock(workflow, "public-release-authorization");
  const publisher = jobBlock(workflow, "publish-public-export-pr");
  const autoMerge = jobBlock(workflow, "request-public-export-auto-merge");
  const merge = jobBlock(workflow, "verify-public-main-merge");
  const publicImage = jobBlock(workflow, "verify-public-runtime-images");
  const finalManifest = jobBlock(workflow, "build-release-manifest");
  if (workflow.includes("secrets.LUNAFOX_PUBLIC_REPO_APP_TOKEN")) {
    fail("private release workflow must not store a short-lived GitHub App installation token");
  }
  if (countOccurrences(workflow, PUBLIC_APP_ID_VARIABLE) !== DESTINATION_APP_JOBS.length ||
      countOccurrences(workflow, PUBLIC_APP_PRIVATE_KEY_SECRET) !== DESTINATION_APP_JOBS.length) {
    fail("GitHub App credentials must be referenced only by the fixed protected release jobs");
  }
  for (const job of DESTINATION_APP_JOBS) {
    assertRuntimeDestinationAppToken(jobBlock(workflow, job), job, {
      actionsRead: job === "verify-public-runtime-images" || job === "verify-public-engine-release",
      administrationRead: job === "request-public-export-auto-merge" || job === "verify-public-main-merge",
      readOnly: job === "verify-public-main-merge" || job === "verify-public-runtime-images" || job === "verify-public-engine-release",
    });
  }
  if (!workflow.includes(`vars.${PUBLIC_RELEASE_AUTOMATION_VARIABLE}`) ||
      !authorization.includes("LUNAFOX_PUBLIC_RELEASE_AUTOMATION_ENABLED must be true or false") ||
      !authorization.includes("Public release automation is disabled") ||
      !authorization.includes("exit 1")) {
    fail("public release automation must fail closed until explicitly authorized");
  }
  if (!tagValidator.includes(configuredRunnerExpression)) {
    fail("validate-tag must use the configured private runner labels");
  }
  if (/^\s{4}runs-on:\s+ubuntu-22\.04\s*$/m.test(tagValidator) ||
      /^\s{4}runs-on:\s+ubuntu-22\.04\s*$/m.test(trustedRunnerBoundary)) {
    fail("private release must not reserve a separate GitHub-hosted bootstrap runner");
  }
  for (const required of [
    `PRIVATE_RELEASE_RUNNER_LABELS: \${{ vars.${PRIVATE_RELEASE_RUNNER_LABELS_VARIABLE} || '["ubuntu-22.04"]' }}`,
    `PRIVATE_RELEASE_VERIFICATION_MODE: \${{ vars.${PRIVATE_RELEASE_VERIFICATION_MODE_VARIABLE} || 'fast' }}`,
  ]) {
    if (!workflow.includes(required)) fail(`private release workflow is missing trusted runner configuration: ${required}`);
  }
  if (!trustedRunnerBoundary.includes("needs: validate-tag") ||
      !trustedRunnerBoundary.includes(configuredRunnerExpression) ||
      !trustedRunnerBoundary.includes('if [ "$GITHUB_REPOSITORY" != "$PRIVATE_REPOSITORY" ]; then') ||
      !trustedRunnerBoundary.includes('jq -cer') ||
      !trustedRunnerBoundary.includes('if . == ["ubuntu-22.04"] then .') ||
      !trustedRunnerBoundary.includes('echo "runner_labels=$runner_labels"') ||
      !/index\("self-hosted"\)\)\s*!=\s*null/.test(trustedRunnerBoundary) ||
      !/index\("linux"\)\)\s*!=\s*null/.test(trustedRunnerBoundary) ||
      !/index\("x64"\)\)\s*!=\s*null/.test(trustedRunnerBoundary) ||
      !/index\("lunafox-private-x64"\)\)\s*!=\s*null/.test(trustedRunnerBoundary) ||
      !trustedRunnerBoundary.includes('lunafox-private-x64') ||
      !trustedRunnerBoundary.includes('PRIVATE_RELEASE_VERIFICATION_MODE must be fast or full')) {
    fail("private runner and verification resolver must fail closed");
  }
  const trustedRunnerExpression = "runs-on: ${{ fromJSON(needs.private-trusted-runner-boundary.outputs.runner_labels) }}";
  for (const job of ["private-runner-preflight", "public-release-authorization", "migration-baseline-gate", "verify-public-engine-release", "prepare-github-release", "prepare-agent-public-context", "publish-images-amd64", "publish-images-arm64", "merge-staging-images", "sign-runtime-images", "build-release-manifest", "verify-engine-bootstrap-registries", "verify-ghcr-anonymous-distribution", "public-export-preflight", "publish-public-export-pr", "request-public-export-auto-merge", "verify-public-main-merge", "verify-public-runtime-images", "update-release-channel", "promote-images", "finalize-github-release"]) {
    const block = jobBlock(workflow, job);
    if (!block.includes(trustedRunnerExpression) || !/private-trusted-runner-boundary/.test(block)) {
      fail(`${job} must consume the validated private trusted-runner labels`);
    }
  }
  const runnerPreflight = jobBlock(workflow, "private-runner-preflight");
  if (!runnerPreflight.includes("docker/setup-qemu-action@v3") ||
      !runnerPreflight.includes("platforms: arm64") ||
      !runnerPreflight.includes("docker run --rm --platform linux/arm64 alpine:3.20 uname -m")) {
    fail("private runner preflight must install and verify arm64 QEMU before release jobs");
  }
  if (!publisher.includes("public-release-authorization") ||
      !publisher.includes("needs.public-release-authorization.outputs.enabled == 'true'")) {
    fail("public export PR creation must require explicit release authorization");
  }
  if (!autoMerge.includes("public-release-authorization") ||
      !autoMerge.includes("needs.public-release-authorization.outputs.enabled == 'true'")) {
    fail("public auto-merge must require explicit release authorization");
  }
  if (!preflight.includes("export-public-repository.mjs") ||
      !preflight.includes("check-public-export.mjs") ||
      !preflight.includes("verify-public-deployment.sh") ||
      !preflight.includes("audit-public-security-scope.mjs")) {
    fail("public-export-preflight must run exporter, exact export, deployment, and scope gates");
  }
  if (!/needs:\s*\n\s*- validate-tag\s*\n\s*- tests-gate\s*\n/.test(preflight) ||
      /build-bridge-release-manifest|release-bridge-manifest|bridge\.manifest|--overlay-dir|generate-public-channel\.mjs|verify-public-release\.mjs|--require-first-release/.test(preflight)) {
    fail("public-export-preflight must export source before channel publication without a Runtime bridge");
  }
  for (const required of [
    'SOURCE_REF="refs/tags/$TAG"',
    'git rev-parse --verify "${SOURCE_REF}^{commit}"',
    '[ "$CHECKED_OUT_REVISION" != "$SOURCE_REVISION" ]; then',
    '--revision "$SOURCE_REF"',
  ]) {
    if (!preflight.includes(required)) {
      fail(`public-export-preflight must bind the export to the checked-out release tag: missing ${required}`);
    }
  }
  if (!preflight.includes("git rev-parse --verify 'HEAD^{commit}'")) {
    fail("public-export-preflight must bind the export to the checked-out release tag: missing quoted HEAD commit resolution");
  }
  if (preflight.includes('--revision "$SOURCE_REVISION"')) {
    fail("public-export-preflight must not pass a detached revision SHA to the exporter");
  }
  if (!publisher.includes("environment: release") || !publisher.includes("LUNAFOX_PUBLIC_REPO_APP_TOKEN") ||
      publisher.includes("GITHUB_TOKEN") || publisher.includes("GITHUB_PAT") || publisher.includes("PAT")) {
    fail("public exporter publisher must use only the protected destination App token");
  }
  for (const required of ["verifyDestinationInstallation", "contents: \"write\"", "pull_requests: \"write\"", "metadata: \"read\"", "/installation/repositories"]) {
    if (!publisherSource.includes(required)) fail(`public exporter must preflight destination App permissions: ${required}`);
  }
  if (!merge.includes("verify-public-main-merge.mjs") || !merge.includes("LUNAFOX_PUBLIC_REPO_APP_TOKEN") ||
      merge.includes("GITHUB_TOKEN") || merge.includes("PAT")) {
    fail("public merge gate must use the destination App token and read-only verifier");
  }
  if (!autoMerge.includes("request-public-auto-merge.mjs")) {
    fail("public export auto-merge gate must request protected GitHub auto-merge without direct merge calls");
  }
  if (!autoMergeSource.includes("enablePullRequestAutoMerge") ||
      autoMergeSource.includes("/merges") ||
      autoMergeSource.includes("merge_method")) {
    fail("public auto-merge script must request GraphQL auto-merge without direct merge calls");
  }
  if (!/needs:\s*\[[^\]]*publish-public-export-pr[^\]]*\]/.test(autoMerge)) {
    fail("auto-merge gate must wait for the exact generated export PR");
  }
  if (!merge.includes("request-public-export-auto-merge") || !merge.includes("--pr-number")) {
    fail("public merge gate must wait for and bind to the auto-merge PR");
  }
  if (!publicImage.includes("verify-public-runtime-image-evidence.mjs") ||
      !publicImage.includes("permission-actions: read") ||
      !publicImage.includes("public-runtime-image-evidence-") ||
      !publicImage.includes("gh run download") ||
      !publicImage.includes("oras cp") ||
      !publicImage.includes("digestEqualityVerified:true")) {
    fail("private release must download and verify the exact public Runtime Image evidence set");
  }
  for (const required of [
    "mapfile -d '' evidence_files < <(find \"$evidence_dir\" -type f -print0)",
    '[ "${#evidence_files[@]}" -eq 1 ] || { echo "expected exactly one public Runtime evidence file for $component" >&2; exit 1; }',
    '[ "$(basename \"$evidence_file\")" = "public-runtime-image-evidence-${component}.json" ] || {',
  ]) {
    if (!publicImage.includes(required)) {
      fail(`private release must reject missing, duplicate, or unexpected public Runtime evidence files: ${required}`);
    }
  }
  assertRuntimeComponentList(publicImage, "private Runtime evidence consumer");
  const forbiddenRuntimeBridge = /LUNAFOX_PRIVATE_RUNTIME|PRIVATE_RUNTIME_(?:FALLBACK_MODE|SERVER_REFS|FRONTEND_REFS|NGINX_REFS|AGENT_REFS|BOOTSTRAP_REFS)|build-bridge-release-manifest|release-bridge-manifest|bridge\.manifest(?:\.yaml|\.metadata\.json)?|private-fallback-bridge/;
  if (forbiddenRuntimeBridge.test(workflow)) {
    fail("private release must not retain Runtime Image fallback or bridge-manifest inputs");
  }
  for (const required of ["source:\"public-evidence\"", "copiedWithoutRebuild:true", "digestEqualityVerified:true"]) {
    if (!publicImage.includes(required)) {
      fail(`private release must retain evidence-only Runtime input guard: ${required}`);
    }
  }
  if (!finalManifest.includes("verify-public-runtime-images") ||
      !finalManifest.includes("release-image-digests") ||
      /context:\s*(?:\.?\/)?(?:server|frontend|docker\/(?:nginx|bootstrap))\b|file:\s*\.?\/?(?:server|frontend|docker\/(?:nginx|bootstrap))\/Dockerfile\b/.test(finalManifest)) {
    fail("final release manifest must consume evidence-derived Runtime digests without rebuilding");
  }
  if (!/needs:\s*\[[^\]]*public-export-preflight[^\]]*\]/.test(publisher)) fail("publisher must wait for export preflight");
  if (!/needs:\s*\[[^\]]*publish-public-export-pr[^\]]*\]/.test(merge)) fail("merge gate must wait for the export PR");
  for (const job of ["prepare-github-release", "promote-images", "update-release-channel", "finalize-github-release"]) {
    const block = jobBlock(workflow, job);
    if (!block.includes("verify-public-main-merge") || !block.includes("verify-public-runtime-images")) {
      fail(`${job} must wait for the exact public-main merge and Runtime Image evidence gates`);
    }
  }
  const channel = jobBlock(workflow, "update-release-channel");
  if (!channel.includes("generate-public-channel.mjs") || !channel.includes("--publication-complete")) {
    fail("channel publication must use the guarded schema-v3 generator");
  }
  if (channel.includes("git push origin main") || channel.includes("refs/heads/main")) {
    fail("release workflow must not directly write public main");
  }
  if (workflow.includes("https://github.com/${GITHUB_REPOSITORY}") || workflow.includes("https://github.com/$GITHUB_REPOSITORY")) {
    fail("release URLs must use the canonical public repository");
  }
  if (/(?:gitee\.com|GITEE_|sync-gitee|finalize-gitee|prepare-gitee)/i.test(workflow)) {
    fail("private release workflow must not publish or synchronize Gitee refs/releases");
  }
  if (/(?:build-installers|release-installers|checksums\.txt|lunafox-installer-)/.test(workflow)) {
    fail("private release workflow must not build or publish standalone installer artifacts");
  }
  if (!workflow.includes("verify-agent-public-revision.sh") ||
      !workflow.includes("PUBLIC_MERGE_SHA") ||
      !workflow.includes("PUBLIC_EXPORT_MANIFEST_SHA256")) {
    fail("Agent release lane must bind its build to the evidence-bound public revision");
  }
  if (workflow.includes("run_frontend_scripts: true") || workflow.includes("run_server: true")) {
    fail("private release test gate must not repeat public Frontend/Server source suites");
  }
  for (const [job, architecture] of [["publish-images-amd64", "amd64"], ["publish-images-arm64", "arm64"]]) {
    const block = jobBlock(workflow, job);
    if (!block.includes("context: ./agent") || !block.includes("file: ./agent/Dockerfile") ||
        !block.includes("lunafox-agent") || !block.includes(`platforms: linux/${architecture}`) ||
        !block.includes(`lunafox-release-lunafox-agent-${architecture}`)) {
      fail(`${job} must build only the private Agent with its architecture-specific cache`);
    }
    for (const component of PUBLIC_RUNTIME_COMPONENTS) {
      if (new RegExp(`(?:context:|file:|tags:)[^\\n]*${component}`, "i").test(block)) {
        fail(`${job} must not build public Runtime component ${component}`);
      }
    }
    if (!block.includes("build-contexts:") || !block.includes("contracts=./dist/agent-public-context/contracts") ||
        !block.includes("engine-go=./dist/agent-public-context/engine-go") ||
        !block.includes("proto=./dist/agent-public-context/proto")) {
      fail(`${job} must consume evidence-bound public named contexts`);
    }
  }
}

function assertPrivateWorkflow(workflow, policy, publisherSource = "", autoMergeSource = "") {
  assertFixedIdentity(workflow, policy);
  const configuredRunnerExpression = "runs-on: ${{ fromJSON(vars." + PRIVATE_RELEASE_RUNNER_LABELS_VARIABLE + " || '[\"ubuntu-22.04\"]') }}";
  const trustedRunnerExpression = "runs-on: ${{ fromJSON(needs.private-trusted-runner-boundary.outputs.runner_labels) }}";
  const tagValidator = jobBlock(workflow, "validate-tag");
  const trustedRunnerBoundary = jobBlock(workflow, "private-trusted-runner-boundary");
  const authorization = jobBlock(workflow, "public-release-authorization");
  const preparation = jobBlock(workflow, "prepare-release-inputs");
  const publisher = jobBlock(workflow, "publish-public-export-pr");
  const autoMerge = jobBlock(workflow, "request-public-export-auto-merge");

  if (workflow.includes("secrets.LUNAFOX_PUBLIC_REPO_APP_TOKEN")) fail("private release workflow must not store a short-lived GitHub App installation token");
  if (countOccurrences(workflow, PUBLIC_APP_ID_VARIABLE) !== DESTINATION_APP_JOBS.length || countOccurrences(workflow, PUBLIC_APP_PRIVATE_KEY_SECRET) !== DESTINATION_APP_JOBS.length) {
    fail("GitHub App credentials must be referenced only by the fixed protected release jobs");
  }
  assertRuntimeDestinationAppToken(publisher, "publish-public-export-pr");
  assertRuntimeDestinationAppToken(autoMerge, "request-public-export-auto-merge", { administrationRead: true });

  // The private workflow is terminal at the handoff request. Keep the
  // boundary explicit so a future synchronous verifier cannot return quietly.
  const removedJobs = [
    "private-runner-preflight", "migration-baseline-gate", "tests-gate", "public-export-preflight",
    "build-agent-binaries", "publish-agent-bundle", "verify-public-main-merge", "verify-public-publication",
    "verify-public-runtime-images", "publish-images-amd64", "publish-images-arm64", "merge-staging-images",
    "sign-runtime-images", "build-release-manifest", "update-release-channel", "promote-images", "finalize-github-release",
  ];
  for (const job of removedJobs) {
    if (workflow.includes(`  ${job}:`)) fail(`private release must not retain synchronous verifier job ${job}`);
  }
  const forbiddenSynchronousMarkers = [
    "gh run list",
    "gh run download",
    "sleep 20",
    "public-final-release-",
    "verify-public-release.mjs",
    "verify-public-main-merge.mjs",
  ];
  for (const marker of forbiddenSynchronousMarkers) {
    if (workflow.includes(marker)) fail(`private release must not retain synchronous public verification marker ${marker}`);
  }
  const jobsStart = workflow.indexOf("\njobs:");
  const jobNames = [...(jobsStart >= 0 ? workflow.slice(jobsStart).matchAll(/^  ([A-Za-z0-9_-]+):/gm) : [])].map((match) => match[1]);
  if (jobNames.at(-1) !== "request-public-export-auto-merge") fail("request-public-export-auto-merge must be the last private release job");
  if (!autoMerge.includes("name: Request Public Export Auto-Merge (Handoff Requested)") ||
      !autoMerge.includes("### Public release handoff requested") ||
      !autoMerge.includes("public repository Actions workflow owns merge, publication, channel, and GitHub Release status")) {
    fail("private release handoff job must report handoff requested without claiming final publication");
  }

  if (!workflow.includes(`vars.${PUBLIC_RELEASE_AUTOMATION_VARIABLE}`) || !authorization.includes("LUNAFOX_PUBLIC_RELEASE_AUTOMATION_ENABLED must be true or false") || !authorization.includes("Public release automation is disabled") || !authorization.includes("exit 1")) {
    fail("public release automation must fail closed until explicitly authorized");
  }
  if (!tagValidator.includes(configuredRunnerExpression)) fail("validate-tag must use the configured private runner labels");
  if (/^\s{4}runs-on:\s+ubuntu-22\.04\s*$/m.test(tagValidator) || /^\s{4}runs-on:\s+ubuntu-22\.04\s*$/m.test(trustedRunnerBoundary)) fail("private release must not reserve a separate GitHub-hosted bootstrap runner");
  for (const required of [`PRIVATE_RELEASE_RUNNER_LABELS: \${{ vars.${PRIVATE_RELEASE_RUNNER_LABELS_VARIABLE} || '[\"ubuntu-22.04\"]' }}`]) {
    if (!workflow.includes(required)) fail(`private release workflow is missing trusted runner configuration: ${required}`);
  }
  if (!trustedRunnerBoundary.includes("needs: validate-tag") || !trustedRunnerBoundary.includes(configuredRunnerExpression) || !trustedRunnerBoundary.includes('if [ "$GITHUB_REPOSITORY" != "$PRIVATE_REPOSITORY" ]; then') || !trustedRunnerBoundary.includes("jq -cer") || !trustedRunnerBoundary.includes('if . == ["ubuntu-22.04"] then .') || !trustedRunnerBoundary.includes('echo "runner_labels=$runner_labels"') || !/index\("self-hosted"\)\)\s*!=\s*null/.test(trustedRunnerBoundary) || !/index\("linux"\)\)\s*!=\s*null/.test(trustedRunnerBoundary) || !/index\("x64"\)\)\s*!=\s*null/.test(trustedRunnerBoundary) || !/index\("lunafox-private-x64"\)\)\s*!=\s*null/.test(trustedRunnerBoundary)) {
    fail("private runner and verification resolver must fail closed");
  }
  for (const job of ["public-release-authorization", "prepare-release-inputs", "publish-public-export-pr", "request-public-export-auto-merge"]) {
    const block = jobBlock(workflow, job);
    if (!block.includes(trustedRunnerExpression) || !block.includes("private-trusted-runner-boundary")) fail(`${job} must consume the validated private trusted-runner labels`);
  }
  for (const required of [
    "needs: [validate-tag, public-release-authorization, private-trusted-runner-boundary]",
    "actions/setup-node@v4", "actions/setup-go@v5", "sigstore/cosign-installer@v3",
    "go -C agent test ./... -count=1", "scripts/ci/export-public-repository.mjs", "scripts/ci/build-agent-binaries.sh",
    "verify-agent-binary-bundle.mjs", "cosign verify-blob", "agent-bundle.sigstore.json", "id-token: write",
    "agent/bin/$RELEASE_TAG", "public-export.tar.gz", "actions/upload-artifact@v4",
  ]) if (!preparation.includes(required)) fail(`prepare-release-inputs is missing ${required}`);
  if (/publish-agent-bundle-asset|agent-input-|staging|private-runner-preflight|tests-gate|docker buildx build/.test(preparation)) fail("prepare-release-inputs must not retain staging Asset or duplicate image/preflight work");
  if (!publisher.includes("needs: [prepare-release-inputs, validate-tag, public-release-authorization, private-trusted-runner-boundary]") || !publisher.includes("actions/download-artifact@v4") || !publisher.includes("publish-public-export.mjs")) fail("public export PR must consume the complete prepared tree artifact");
  if (!publisher.includes("public-release-authorization") || !publisher.includes("needs.public-release-authorization.outputs.enabled == 'true'")) fail("public export PR creation must require explicit release authorization");
  if (!autoMerge.includes("public-release-authorization") || !autoMerge.includes("needs.public-release-authorization.outputs.enabled == 'true'")) fail("public auto-merge must require explicit release authorization");
  if (!publisher.includes("environment: release") || !publisher.includes("LUNAFOX_PUBLIC_REPO_APP_TOKEN") || publisher.includes("GITHUB_TOKEN") || publisher.includes("GITHUB_PAT") || /\bPAT\b/.test(publisher)) fail("public exporter publisher must use only the protected destination App token");
  for (const required of ["verifyDestinationInstallation", "contents: \"write\"", "pull_requests: \"write\"", "metadata: \"read\"", "/installation/repositories"]) if (!publisherSource.includes(required)) fail(`public exporter must preflight destination App permissions: ${required}`);
  if (!autoMerge.includes("request-public-auto-merge.mjs")) fail("public export auto-merge gate must request protected GitHub auto-merge without direct merge calls");
  if (!autoMergeSource.includes("enablePullRequestAutoMerge") || autoMergeSource.includes("/merges") || autoMergeSource.includes("merge_method")) fail("public auto-merge script must request GraphQL auto-merge without direct merge calls");
  if (!/needs:\s*\[[^\]]*publish-public-export-pr[^\]]*\]/.test(autoMerge)) fail("auto-merge gate must wait for the exact generated export PR");
  if (!/needs:\s*\[[^\]]*prepare-release-inputs[^\]]*\]/.test(publisher)) fail("publisher must wait for prepared release inputs");
  if (!/needs:\s*\[[^\]]*publish-public-export-pr[^\]]*\]/.test(autoMerge)) fail("auto-merge gate must wait for the export PR");

  const forbiddenJobs = ["publish-images-amd64", "publish-images-arm64", "merge-staging-images", "sign-runtime-images", "build-release-manifest", "update-release-channel", "promote-images", "finalize-github-release", "prepare-github-release", "verify-public-runtime-images", "verify-public-engine-release", "prepare-agent-public-context", ...removedJobs];
  for (const job of forbiddenJobs) if (workflow.includes(`  ${job}:`)) fail(`private release must not retain retired publication job ${job}`);
  const forbiddenWorkflowMarkers = ["generate-release-manifest", "generate-public-channel", "gh release create", "gh release edit", "gh release upload", "verify-agent-public-revision.sh", "PUBLIC_MERGE_SHA", "LUNAFOX_PRIVATE_RUNTIME", "PRIVATE_RUNTIME_FALLBACK", "build-bridge-release-manifest", "release-bridge-manifest", "private-fallback-bridge", "publish-agent-bundle-asset.mjs", "agent-input-", "staging Release Asset", ...forbiddenSynchronousMarkers];
  for (const marker of forbiddenWorkflowMarkers) if (workflow.includes(marker)) fail(`private release must not retain final publication or fallback marker ${marker}`);
  if (workflow.includes("https://github.com/${GITHUB_REPOSITORY}") || workflow.includes("https://github.com/$GITHUB_REPOSITORY")) fail("release URLs must use the canonical public repository");
  if (/(?:gitee\.com|GITEE_|sync-gitee|finalize-gitee|prepare-gitee)/i.test(workflow)) fail("private release workflow must not publish or synchronize Gitee refs/releases");
  if (/(?:build-installers|release-installers|checksums\.txt|lunafox-installer-)/.test(workflow)) fail("private release workflow must not build or publish standalone installer artifacts");
}

function assertGitHubHostedRunner(block, label) {
  const runner = block.match(/^\s{4}runs-on:\s*([^\s#]+)\s*$/m)?.[1];
  if (!runner || !/^ubuntu-(?:latest|\d+\.\d+(?:-arm)?)$/.test(runner)) {
    fail(`${label} must use a GitHub-hosted runner`);
  }
}

function assertPrivateTestWorkflow(workflow) {
  const workflowCall = workflow.slice(workflow.indexOf("  workflow_call:"), workflow.indexOf("\njobs:"));
  if (!/^\s{6}runner_labels:\s*$/m.test(workflowCall) ||
      !workflowCall.includes("type: string") ||
      !workflowCall.includes("default: '[\"ubuntu-22.04\"]'")) {
    fail("private reusable test workflow must expose the validated runner_labels input");
  }
  for (const job of [
    "codex-platform-contracts",
    "frontend-and-scripts",
    "server-tests",
    "target-cleanup-release-gates",
    "scan-operations-release-gates",
    "global-asset-search-release-gates",
    "agent-tests",
    "contracts-tests",
    "engine-boundary-fast",
    "shell-static",
    "test",
  ]) {
    const block = jobBlock(workflow, job);
    if (!block.includes("runs-on: ${{ fromJSON(inputs.runner_labels) }}")) {
      fail(`${job} must consume the reusable runner_labels input`);
    }
  }
}

function assertPublicWorkflow(workflow, policy) {
  if (!workflow.includes(`if: github.repository == '${PUBLIC_REPOSITORY}'`)) {
    fail("public validation workflow must run only in the canonical public repository");
  }
  const validation = jobBlock(workflow, "validate-export");
  const goValidation = jobBlock(workflow, "validate-public-go");
  const protoValidation = jobBlock(workflow, "validate-public-proto");
  const contextValidation = jobBlock(workflow, "validate-runtime-contexts");
  const aggregate = jobBlock(workflow, "public-validation");
  const publication = jobBlock(workflow, "publish-runtime-images");
  const agentPublication = jobBlock(workflow, "publish-agent-image");
  const validationBlocks = [validation, goValidation, protoValidation, contextValidation, aggregate];
  validationBlocks.forEach((block, index) => {
    const label = `public validation job ${index + 1}`;
    assertSecretlessBlock(block, label);
    assertGitHubHostedRunner(block, label);
  });
  if (!validation.includes("ref: ${{ github.event_name == 'pull_request' && github.event.pull_request.head.sha || github.sha }}")) {
    fail("public export validation must check the real PR head commit instead of GitHub's synthetic merge commit");
  }
  assertGitHubHostedRunner(publication, "public Runtime publication job");
  assertGitHubHostedRunner(agentPublication, "public Agent publication job");
  if (validationBlocks.some((block) => /docker\/login-action|pull:.*private|self-hosted/i.test(block))) {
    fail("public validation jobs must not use registry credentials or self-hosted runners");
  }
  if (/PRIVATE_TRUSTED_RUNNER|\bself-hosted\b/i.test(workflow)) {
    fail("public workflow must not contain private trusted-runner configuration");
  }
  if (!validation.includes("check-public-export.mjs") || !validation.includes("verify-public-deployment.sh") ||
      !validation.includes("audit-public-security-scope.mjs") || !validation.includes("verify-public-release.mjs") ||
      !validation.includes("verify-public-runtime-source.sh") || !validation.includes("verify-public-runtime-contexts.mjs")) {
    fail("public validation workflow is missing the secretless boundary gates");
  }
  if (!validation.includes("channel_dir=channels") ||
      !validation.includes("channels and manifests must be both present directories or both absent") ||
      !validation.includes('if [ -e "$channel_dir" ] || [ -e "$manifest_dir" ]; then')) {
    fail("public validation must allow source exports before release-channel records exist");
  }
  for (const required of [
    "pnpm/action-setup@v4",
    "cache: pnpm",
    "cache-dependency-path: frontend/pnpm-lock.yaml",
    "working-directory: frontend",
    "pnpm install --frozen-lockfile",
    "pnpm run typecheck",
    "pnpm run lint",
    "pnpm run test",
    "pnpm run check:ui-foundation",
    "pnpm run check:service-mock-mode",
    "pnpm run check:foundation-ledger",
  ]) {
    if (!validation.includes(required)) fail(`public validation workflow is missing frontend gate: ${required}`);
  }
  assertRuntimeComponentList(publication, "public Runtime publication");
  if (!/^permissions:\s*\n\s+contents:\s+read\s*$/m.test(workflow)) fail("public validation workflow must grant contents read only");
  for (const required of ["server", "contracts", "engine-go", "extensions", "go test ./...", "proto/scripts/check-generated.sh"]) {
    if (!goValidation.includes(required) && !protoValidation.includes(required)) fail(`public workflow is missing public source gate: ${required}`);
  }
  if (!contextValidation.includes("docker/build-push-action@v6") || !contextValidation.includes("push: false")) {
    fail("public workflow must run non-publishing Docker context builds");
  }
  for (const required of ["server/Dockerfile", "frontend/Dockerfile", "docker/nginx/Dockerfile", "docker/bootstrap/Dockerfile"]) {
    if (!contextValidation.includes(required)) fail(`public workflow is missing push:false context: ${required}`);
  }
  if (!aggregate.includes("Public Projection Validation") ||
      !aggregate.includes("always()") ||
      !aggregate.includes("if [ \"$result\" != success ]") ||
      !aggregate.includes("exit 1")) {
    fail("public workflow must expose an explicit failing aggregate Public Projection Validation check");
  }
  if (!publication.includes("github.event_name == 'push'") || !publication.includes("github.ref == 'refs/heads/main'")) {
    fail("public Runtime publication must run only for canonical main pushes");
  }
  if (!agentPublication.includes("github.event_name == 'push'") || !agentPublication.includes("github.ref == 'refs/heads/main'")) {
    fail("public Agent publication must run only for canonical main pushes");
  }
  if (!publication.includes("squash_subject_pattern") ||
      !publication.includes("chore\\\\(export\\\\):") ||
      !publication.includes("head_subject\" =~ $merge_subject_pattern || \"$head_subject\" =~ $squash_subject_pattern")) {
    fail("public Runtime publication must accept the configured squash-merge commit identity");
  }
  for (const required of [
    'commit_ref="${image}:public-${GITHUB_SHA}"',
    "if: steps.preflight.outputs.commit_digest == ''",
    "EXISTING_DIGEST",
    "BUILT_DIGEST",
    'image_digest="${BUILT_DIGEST:-$EXISTING_DIGEST}"',
  ]) {
    if (!publication.includes(required)) fail(`public Runtime publication is missing immutable retry guard: ${required}`);
  }
  if (/tags:\s*\n\s+[^\n]*latest\b/.test(publication) || /imagetools\s+create/.test(publication)) {
    fail("public Runtime publication must not overwrite mutable tags or recreate an existing digest");
  }
  for (const required of [
    "needs: public-validation",
    "contents: read",
    "packages: write",
    "id-token: write",
    "attestations: write",
    "push: true",
    "provenance: mode=max",
    "sbom: true",
    "actions/attest-build-provenance@v2",
    "commit_ref=\"${image}:public-${GITHUB_SHA}\"",
    "publicExportManifestSha256",
    "public-runtime-image-evidence-",
  ]) {
    if (!publication.includes(required)) fail(`public Runtime publication is missing: ${required}`);
  }
  for (const required of ["server/Dockerfile", "frontend/Dockerfile", "docker/nginx/Dockerfile", "docker/bootstrap/Dockerfile"]) {
    if (!publication.includes(required)) fail(`public Runtime publication is missing component context: ${required}`);
  }
  if (/DOCKERHUB_TOKEN|DOCKERHUB_USERNAME|DOCKERHUB_PASSWORD|build-contexts:|context:\s*agent\b/.test(publication)) {
    fail("public Runtime publication must not receive Docker Hub credentials or private build contexts");
  }
  for (const required of [
    "needs: public-validation",
    "contents: read",
    "packages: write",
    "id-token: write",
    "attestations: write",
    "verify-agent-binary-bundle.mjs",
    "--public-export-root",
    "cosign verify-blob",
    "agent-bundle.sigstore.json",
    "--certificate-oidc-issuer https://token.actions.githubusercontent.com",
    "--certificate-identity-regexp",
    "refs/tags/${RELEASE_TAG}$",
    "docker/agent/Dockerfile",
    "push: true",
    "provenance: mode=max",
    "sbom: true",
    "actions/attest-build-provenance@v2",
  ]) {
    if (!agentPublication.includes(required)) fail(`public Agent publication is missing: ${required}`);
  }
  if (!agentPublication.includes('context: ${{ steps.verify.outputs.bundle_dir }}') ||
      agentPublication.includes("stagingIdentity") || agentPublication.includes("agent-input-") ||
      /DOCKERHUB_TOKEN|DOCKERHUB_USERNAME|DOCKERHUB_PASSWORD/.test(agentPublication)) {
    fail("public Agent publication must build only from the verified bundle context without Docker Hub credentials");
  }
  // A public workflow may be exported without the private release workflow;
  // its machine-readable policy still has to pin the same identity.
  if (policy.publicMain?.allowDirectPush !== false || policy.publicMain?.allowForcePush !== false) {
    fail("public main policy must disable direct and force pushes");
  }

  const engineBuild = jobBlock(workflow, "public-engine-runtime-build");
  const engineSign = jobBlock(workflow, "public-engine-runtime-sign");
  const packageBuild = jobBlock(workflow, "public-engine-package-build");
  const packagePublish = jobBlock(workflow, "public-engine-package-publish");
  const engineManifest = jobBlock(workflow, "public-engine-release-manifest");
  for (const [name, block] of PUBLIC_ENGINE_JOBS.map((name) => [name, jobBlock(workflow, name)])) {
    if (!block.includes("github.repository == 'yyhuni/lunafox'") ||
        !block.includes("github.event_name == 'push'") ||
        !block.includes("github.ref == 'refs/heads/main'")) {
      fail(`${name} must run only on protected public main pushes`);
    }
    assertGitHubHostedRunner(block, name);
  }
  for (const [name, block] of [["public Engine Runtime build", engineBuild], ["public Engine Runtime sign", engineSign], ["public Engine Package build", packageBuild], ["public Engine Package publish", packagePublish], ["public Engine release manifest", engineManifest]]) {
    if (/agent\/|lunafox-private|PRIVATE_TRUSTED_RUNNER|GITHUB_PAT|COSIGN_PRIVATE_KEY/.test(block)) {
      fail(`${name} must not contain private Agent/release authority`);
    }
  }
  if (!engineBuild.includes("needs: public-validation") ||
      !engineBuild.includes("public-engine-runtime-build") ||
      !engineBuild.includes("publish-engine-runtime-images.sh") ||
      !engineBuild.includes("ENGINE_RELEASE_MODE: production") ||
      !engineBuild.includes("ENGINE_REUSE_EXISTING: true") ||
      !engineBuild.includes("LUNAFOX_CANONICAL_NAMESPACE: yyhuni") ||
      !engineBuild.includes("packages: write")) {
    fail("public Engine Runtime build must use the validated production publisher and public registry permissions");
  }
  if (!engineSign.includes("needs: public-engine-runtime-build") ||
      !engineSign.includes("cosign sign --yes") ||
      !engineSign.includes("cosign verify") ||
      !engineSign.includes("public-validate\\\\.yml@refs/heads/main") ||
      !engineSign.includes("public-engine-runtime-evidence-") ||
      !engineSign.includes("registryAuth:\"empty\"")) {
    fail("public Engine Runtime lane must sign, anonymously verify, and retain evidence");
  }
  if (!packageBuild.includes("needs: public-engine-runtime-sign") ||
      !packageBuild.includes("-command build-packages") ||
      !packageBuild.includes("-build-results") ||
      !packageBuild.includes("-mode production") ||
      !packageBuild.includes("-command validate-package-artifacts") ||
      !packageBuild.includes("-package-build-results") ||
      !packageBuild.includes("-out-root") ||
      packageBuild.includes("check-engine-release-contract.mjs")) {
    fail("public Engine Package build must derive and validate packages with the public Go release tool");
  }
  if (!packagePublish.includes("needs: public-engine-package-build") ||
      !packagePublish.includes("oras cp") ||
      !packagePublish.includes("cosign sign --yes") ||
      !packagePublish.includes("cosign verify") ||
      !packagePublish.includes("public-engine-package-evidence-") ||
      !packagePublish.includes("public-engine-package-digests")) {
    fail("public Engine Package lane must publish, sign, verify, and retain digest evidence");
  }
  if (!engineManifest.includes("needs: public-engine-package-publish") ||
      !engineManifest.includes("engine-release-manifest.json") ||
      !engineManifest.includes("verify-public-engine-release.mjs") ||
      !engineManifest.includes("name: public-engine-release")) {
    fail("public Engine lane must publish an independently verified manifest handoff");
  }
}

function assertExportPolicy(exportPolicy) {
  if (exportPolicy.sourceRepository !== PRIVATE_REPOSITORY || exportPolicy.destinationRepository !== PUBLIC_REPOSITORY) {
    fail("export policy repository identity drifted");
  }
  const exact = new Set([...(exportPolicy.allowlist?.exact ?? []), ...(exportPolicy.allowlist?.generatedExact ?? [])]);
  const destinationOwnedExact = [...new Set(exportPolicy.destinationOwnedExact ?? [])].sort();
  if (JSON.stringify(destinationOwnedExact) !== JSON.stringify([".github/workflows/public-validate.yml"])) {
    fail("public export policy must declare the public validation workflow as destination-owned");
  }
  for (const destinationPath of destinationOwnedExact) {
    if (!exact.has(destinationPath)) fail(`destination-owned path is not allowlisted: ${destinationPath}`);
  }
  for (const required of ["scripts/ci/check-public-release-policy.mjs", "scripts/ci/verify-public-main-merge.mjs"]) {
    if (!exact.has(required)) fail(`export policy does not allow ${required}`);
  }
  const denyPatterns = (exportPolicy.denylist ?? []).map((pattern) => new RegExp(String(pattern)));
  // Every exact allowlisted path must survive the denylist as well.  Keeping
  // this invariant here prevents a newly exported verifier or workflow from
  // passing static policy checks but failing only inside the tag-triggered
  // exporter.
  for (const allowedPath of exact) {
    if (denyPatterns.some((pattern) => pattern.test(allowedPath))) {
      fail(`export policy allowlist/denylist conflict: ${allowedPath}`);
    }
  }
  for (const required of REQUIRED_PUBLIC_RUNTIME_EXACT) {
    if (!exact.has(required)) fail(`export policy is missing public Runtime input: ${required}`);
  }
  const prefixes = new Set(exportPolicy.allowlist?.prefixes ?? []);
  for (const required of REQUIRED_PUBLIC_RUNTIME_PREFIXES) {
    if (!prefixes.has(required)) fail(`export policy is missing public Runtime prefix: ${required}`);
  }
  if ((exportPolicy.publicRuntimeImages ?? []).length !== PUBLIC_RUNTIME_COMPONENTS.length) {
    fail("export policy must declare exactly four public Runtime Image descriptors");
  }
  if (JSON.stringify(exportPolicy.allowlist?.generatedPrefixes ?? []) !== JSON.stringify(["channels/", "manifests/", "agent/bin/"])) {
    fail("public export policy must retain the generated channel and manifest prefix boundary");
  }
  if ((exportPolicy.allowlist?.firstReleaseRequired ?? []).length !== 0) {
    fail("public source export must not require final release channel files");
  }
  for (const component of PUBLIC_RUNTIME_COMPONENTS) {
    const descriptor = exportPolicy.publicRuntimeImages.find((item) => item.component === component);
    if (!descriptor || descriptor.repository !== `lunafox-${component}` || !descriptor.dockerfile || !descriptor.context) {
      fail(`export policy has an invalid public Runtime descriptor: ${component}`);
    }
  }
  if (exportPolicy.symlinks?.allow !== false) fail("public export must reject symlinks");
  const sourceRefs = exportPolicy.sourceRefs;
  if (JSON.stringify(sourceRefs?.allowed) !== JSON.stringify(["refs/heads/main"])) {
    fail("public export policy must allow only refs/heads/main as a branch source");
  }
  if (sourceRefs?.tagPattern !== SOURCE_TAG_PATTERN) {
    fail("public export policy must support stable and alpha/beta/rc source tags");
  }
  const allowedTag = new RegExp(sourceRefs.tagPattern);
  for (const ref of [
    "refs/tags/v1.2.3",
    "refs/tags/v1.2.3-alpha.1",
    "refs/tags/v1.2.3-beta.1",
    "refs/tags/v1.2.3-rc.1",
  ]) {
    if (!allowedTag.test(ref)) fail(`public export policy must allow source tag ${ref}`);
  }
  for (const ref of ["refs/tags/v1.2.3-dev.1", "refs/tags/v1.2.3-alpha"]) {
    if (allowedTag.test(ref)) fail(`public export policy must reject source tag ${ref}`);
  }
  if (!(exportPolicy.denylist ?? []).some((pattern) => String(pattern).includes("tools/installer"))) {
    fail("export policy must deny installer source");
  }
  if (prefixes.has("frontend/")) {
    fail("export policy must not use a broad frontend/ prefix");
  }
  for (const required of REQUIRED_PUBLIC_FRONTEND_PREFIXES) {
    if (!prefixes.has(required)) {
      fail(`export policy is missing approved frontend source prefix: ${required}`);
    }
  }
  for (const required of REQUIRED_PUBLIC_FRONTEND_EXACT) {
    if (!exact.has(required)) {
      fail(`export policy is missing approved frontend configuration: ${required}`);
    }
  }
  const requiredPaths = new Set(exportPolicy.requiredPaths ?? []);
  for (const required of REQUIRED_PUBLIC_FRONTEND_PATHS) {
    if (!requiredPaths.has(required)) {
      fail(`export policy is missing required public frontend path: ${required}`);
    }
  }
  const denyPatternText = (exportPolicy.denylist ?? []).map(String);
  for (const required of [
    "test-results",
    "test-plan",
    "node_modules",
    "\\.next",
  ]) {
    if (!denyPatternText.some((pattern) => pattern.includes(required))) {
      fail(`export policy is missing frontend exclusion: ${required}`);
    }
  }
  if (denyPatternText.some((pattern) => /\(\?:server\|frontend\|agent/.test(pattern))) {
    fail("export policy must not deny the approved frontend root");
  }
  for (const required of REQUIRED_PRIVATE_FRONTEND_TEST_DENIES) {
    if (!denyPatternText.includes(required)) {
      fail(`export policy must deny private frontend contract test: ${required}`);
    }
  }
  for (const required of ["^agent(?:/|$)", "docker/nginx/ssl", "server/bin", "server/\\.air\\.toml"]) {
    if (!denyPatternText.some((pattern) => pattern.includes(required.replace("(?:/|$)", "")))) {
      fail(`export policy must keep private Runtime material denied: ${required}`);
    }
  }
  for (const required of ["server", "contracts", "engine-go", "proto", "extensions", "docker/bootstrap", "docker/nginx", "tools/engine-release", "tools/engine-oci-publish"]) {
    if (!prefixes.has(`${required}/`) && ![...prefixes].some((prefix) => prefix.startsWith(`${required}/`))) {
      fail(`export policy does not expose the approved Runtime closure: ${required}`);
    }
  }
}

function assertPublicTree(root) {
  // Documentation and validator source intentionally mention retired names
  // to explain the fail-closed boundary.  Only machine-consumed channel,
  // manifest, and deployment configuration files are identity-bearing input.
  const isMachineIdentityFile = (relative) =>
    relative.startsWith("channels/") ||
    relative.startsWith("manifests/") ||
    relative === ".env.example" ||
    relative === "compose.yaml" ||
    relative === "PUBLIC_PROVENANCE.json" ||
    relative === "PUBLIC_EXPORT_MANIFEST.json";
  const stack = [root];
  while (stack.length) {
    const current = stack.pop();
    for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
      if (entry.name === ".git" || entry.name === "node_modules") continue;
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) stack.push(full);
      else if (entry.isFile()) {
        const relative = path.relative(root, full).split(path.sep).join("/");
        if (!isMachineIdentityFile(relative)) continue;
        const text = readText(full, "public policy input");
        if (LEGACY_RE.test(text)) fail(`public tree contains retired identity: ${relative}`);
      }
    }
  }
}

function check(options) {
  const policyPath = options.policy || path.join(options.root, "scripts/ci/public-release-policy.json");
  const exportPolicyPath = options.exportPolicy || path.join(options.root, "scripts/ci/public-export-policy.json");
  const policy = readJson(policyPath, "public release policy");
  const exportPolicy = readJson(exportPolicyPath, "public export policy");
  assertPrivateTrustedRunnerPolicy(policy);
  assertFirstPublicReleaseIdentity(policy, exportPolicy);
  assertExportPolicy(exportPolicy);
  if (options.publicOnly) {
    const workflowPath = options.publicWorkflow || path.join(options.root, ".github/workflows/public-validate.yml");
    assertPublicWorkflow(readText(workflowPath, "public validation workflow"), policy);
    assertPublicTree(options.root);
  } else {
    const workflowPath = options.workflow || path.join(options.root, ".github/workflows/release.yml");
    const publicWorkflowPath = options.publicWorkflow || path.join(options.root, ".github/workflows/public-validate.yml");
    const workflow = readText(workflowPath, "private release workflow");
    assertPrivateTestWorkflow(readText(path.join(options.root, ".github/workflows/test.yml"), "private reusable test workflow"));
    const publisherSource = readText(path.join(options.root, "scripts/ci/publish-public-export.mjs"), "public exporter");
    const autoMergeSource = readText(path.join(options.root, "scripts/ci/request-public-auto-merge.mjs"), "public auto-merge requester");
    assertPrivateWorkflow(workflow, policy, publisherSource, autoMergeSource);
    assertPublicWorkflow(readText(publicWorkflowPath, "public validation workflow"), policy);
  }
  return {
    schemaVersion: 1,
    passed: true,
    canonicalRepository: PUBLIC_REPOSITORY,
    privateBuilderRepository: PRIVATE_REPOSITORY,
    canonicalNamespace: CANONICAL_NAMESPACE,
    signerIssuer: SIGNER_ISSUER,
    signerIdentity: PUBLIC_RUNTIME_SIGNER_IDENTITY,
    publicOnly: options.publicOnly,
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = check(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : "public release policy verified"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); }
  catch (error) { process.stderr.write(`public release policy check failed: ${error.message}\n`); process.exitCode = 1; }
}

export { check, parseArgs, assertPrivateWorkflow, assertPrivateTestWorkflow, assertPublicWorkflow, assertExportPolicy, assertFirstPublicReleaseIdentity };
