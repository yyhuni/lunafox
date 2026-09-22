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
  "public-engine-runtime-discover",
  "public-engine-runtime-build",
  "public-engine-runtime-finalize",
  "public-engine-runtime-aggregate",
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
  "docker/bootstrap/config-init.sh",
  "tools/engine-release/go.mod",
  "tools/engine-release/go.sum",
  "tools/engine-oci-publish/go.mod",
  "tools/engine-oci-publish/go.sum",
  "scripts/ci/verify-public-runtime-contexts.mjs",
  "scripts/ci/verify-public-runtime-image-evidence.mjs",
  "scripts/ci/verify-public-runtime-image-evidence-selftest.mjs",
  "scripts/ci/resolve-release-component-composition.mjs",
  "scripts/ci/resolve-release-component-composition-selftest.mjs",
  "scripts/ci/render-release-component-build-contexts.mjs",
  "scripts/ci/render-release-component-build-contexts-selftest.mjs",
  "scripts/ci/resolve-public-runtime-composition.mjs",
  "scripts/ci/resolve-public-runtime-composition-selftest.mjs",
  "scripts/ci/build-component-evidence-bundle.mjs",
  "scripts/ci/verify-component-evidence-bundle.mjs",
  "scripts/ci/component-evidence-bundle-selftest.mjs",
  "scripts/ci/verify-release-component-composition.mjs",
  "scripts/ci/verify-release-component-composition-selftest.mjs",
  "scripts/ci/resolve-engine-release-disposition.mjs",
  "scripts/ci/resolve-engine-release-disposition-selftest.mjs",
  "scripts/ci/assemble-public-engine-release.mjs",
  "scripts/ci/assemble-public-engine-release-selftest.mjs",
  "scripts/ci/verify-public-runtime-source.sh",
  "scripts/ci/check-migration-baseline-policy.mjs",
  "scripts/ci/check-engine-api-major-policy.mjs",
  "scripts/ci/publish-engine-runtime-images.sh",
  "scripts/ci/aggregate-engine-runtime-image-shards.sh",
  "scripts/ci/check-engine-image-tool-inventory.mjs",
  "scripts/ci/verify-distribution-registry-v2.mjs",
  "scripts/ci/verify-runtime-image-index.mjs",
];
const REQUIRED_PUBLIC_UPGRADER_PATHS = [
  "server/cmd/lunafox-upgrader/main.go",
  "server/internal/modules/upgrade/upgrader/compose.go",
  "server/internal/modules/upgrade/upgrader/daemon.go",
  "server/internal/modules/upgrade/upgrader/journal.go",
];
const REQUIRED_PUBLIC_CHECKOUT_DEPLOYMENT_PATHS = [
  ".gitignore",
  "deploy/compose.template.yaml",
  "deploy/.env.example",
  "scripts/ci/publish-public-deployment.mjs",
  "scripts/ci/publish-public-deployment.test.mjs",
  "scripts/ci/prepare-legacy-public-deployment.mjs",
  "scripts/ci/prepare-legacy-public-deployment.test.mjs",
];
const REQUIRED_PUBLIC_DOCUMENTATION_PATHS = [
  "README.md",
  "README.zh-CN.md",
  "docs/public-deployment.md",
  "docs/public-deployment.zh-CN.md",
  "scripts/ci/validate-public-documentation.mjs",
  "scripts/ci/validate-public-documentation.test.mjs",
];
const DESTINATION_DEPLOYMENT_PATHS = [
  ".env",
  ".env.example",
  "compose.yaml",
  "engine-inventory.yaml",
  "release.manifest.yaml",
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

function workflowStepBlock(job, marker) {
  const start = job.indexOf(marker);
  if (start < 0) return "";
  const next = job.indexOf("\n      - ", start + marker.length);
  return job.slice(start, next < 0 ? job.length : next);
}

function hasRequiredJobNeeds(block, names) {
  const match = block.match(/^    needs:\s*\[([^\]]*)\]\s*$/m);
  if (!match) return false;
  const dependencies = new Set(match[1].split(",").map((name) => name.trim()).filter(Boolean));
  return names.every((name) => dependencies.has(name));
}

// GitHub Actions permits a folded multiline `if`; normalize it before
// checking dependency result guards so formatting cannot bypass the policy.
function jobCondition(block) {
  const lines = block.split("\n");
  const index = lines.findIndex((line) => /^    if:\s*/.test(line));
  if (index < 0) return "";
  const first = lines[index].replace(/^    if:\s*/, "").trim();
  if (!/^(?:>-?|\|-?|\|)$/.test(first)) return first;
  const continuation = [];
  for (const line of lines.slice(index + 1)) {
    if (/^    \S/.test(line) || /^      - /.test(line)) break;
    if (/^\s{6,}\S/.test(line)) continuation.push(line.trim());
  }
  return continuation.join(" ");
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

function assertLegacyDeploymentBootstrapPolicy(policy) {
  const bootstrap = policy.legacyDeploymentBootstrap;
  if (bootstrap?.releaseTag !== "v0.0.1-alpha.114" ||
      bootstrap?.packageName !== "lunafox-v0.0.1-alpha.114-dockerhub.zip" ||
      bootstrap?.sha256 !== "d02fe6f9da2576e3a89b52d38a03b5af6afd91670de26e67fe1f1efcf809c55c" ||
      bootstrap?.manifestSha256 !== "e0e742054888daf0fb6482be721c82bd8e162d795143badbf2089cbd978c5e8b" ||
      JSON.stringify(bootstrap?.paths) !== JSON.stringify(DESTINATION_DEPLOYMENT_PATHS)) {
    fail("legacy public deployment bootstrap must remain pinned to the verified alpha.114 Docker Hub package");
  }
  if (policy.publicMain?.deploymentBranchPattern !== "^deployment/v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[A-Za-z0-9.-]+)?(?:-retry-[0-9]+)?$") {
    fail("public main policy must constrain deployment snapshot branches");
  }
  if (policy.publicMain?.workflowBranchPattern !== "^workflow/v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[A-Za-z0-9.-]+)?(?:-retry-[0-9]+)?$") {
    fail("public main policy must constrain validation workflow branches");
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
    `manifests/${FIRST_PUBLIC_TAG}/runtime-composition.json`,
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
      !preflight.includes("validate-public-documentation.mjs") ||
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
  assertRuntimeDestinationAppToken(autoMerge, "request-public-export-auto-merge");

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
    "validate-public-release-notes.mjs", "release-notes-validation.json", "final-release-notes-validation.json",
    "validate-public-documentation.mjs", "documentation-validation.json", "final-documentation-validation.json",
    "go -C agent test ./... -count=1", "scripts/ci/export-public-repository.mjs", "scripts/ci/build-agent-binaries.sh",
    "verify-agent-binary-bundle.mjs", "cosign verify-blob", "agent-bundle.sigstore.json", "id-token: write",
    "PUBLIC_TREE_PATH: agent/bin/${{ steps.agent_input.outputs.agent_artifact_id }}", "tar --exclude='./.git/hooks'", "public-export.tar.gz", "actions/upload-artifact@v4",
  ]) if (!preparation.includes(required)) fail(`prepare-release-inputs is missing ${required}`);
  const earlyWorkflowCheck = preparation.indexOf("Verify public workflow before Agent preparation");
  if (earlyWorkflowCheck < 0 || earlyWorkflowCheck > preparation.indexOf("actions/setup-go@v5") ||
      !preparation.includes('cmp -s .github/workflows/public-validate.yml "$public_workflow"') ||
      !preparation.includes("merge the protected workflow maintenance PR before tagging a release")) {
    fail("public workflow compatibility must fail before expensive Agent preparation");
  }
  if (/publish-agent-bundle-asset|agent-input-|staging|private-runner-preflight|tests-gate|docker buildx build/.test(preparation)) fail("prepare-release-inputs must not retain staging Asset or duplicate image/preflight work");
  if (!publisher.includes("needs: [prepare-release-inputs, validate-tag, public-release-authorization, private-trusted-runner-boundary]") || !publisher.includes("actions/download-artifact@v4") || !publisher.includes("publish-public-export.mjs")) fail("public export PR must consume the complete prepared tree artifact");
  if (!publisher.includes("public-release-authorization") || !publisher.includes("needs.public-release-authorization.outputs.enabled == 'true'")) fail("public export PR creation must require explicit release authorization");
  if (!autoMerge.includes("public-release-authorization") || !autoMerge.includes("needs.public-release-authorization.outputs.enabled == 'true'")) fail("public auto-merge must require explicit release authorization");
  if (!publisher.includes("environment: release") || !publisher.includes("LUNAFOX_PUBLIC_REPO_APP_TOKEN") || publisher.includes("GITHUB_TOKEN") || publisher.includes("GITHUB_PAT") || /\bPAT\b/.test(publisher)) fail("public exporter publisher must use only the protected destination App token");
  if (/^\s+GH_TOKEN:\s/m.test(publisher)) fail("public exporter step must not expose GH_TOKEN to the cross-repository publisher process");
  if (countOccurrences(publisher, 'GH_TOKEN="$LUNAFOX_PUBLIC_REPO_APP_TOKEN" gh api') !== 2) fail("public exporter must scope GH_TOKEN to the two destination API probes only");
  for (const required of ["verifyDestinationInstallation", "contents: \"write\"", "pull_requests: \"write\"", "metadata: \"read\"", "/installation/repositories"]) if (!publisherSource.includes(required)) fail(`public exporter must preflight destination App permissions: ${required}`);
  if (!autoMerge.includes("request-public-auto-merge.mjs")) fail("public export auto-merge gate must request protected GitHub auto-merge without direct merge calls");
  if (!autoMergeSource.includes("enablePullRequestAutoMerge") || autoMergeSource.includes("/merges") || autoMergeSource.includes("merge_method") || autoMergeSource.includes("/branches/main/protection")) fail("public auto-merge script must request GraphQL auto-merge without direct merge calls or branch administration access");
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
  if (!workflow.includes("github.event_name == 'pull_request' && 'pr' || github.run_id")) {
    fail("public concurrency must cancel superseded PR runs without cancelling protected-main releases");
  }
  const frontendValidation = jobBlock(workflow, "validate-frontend");
  const scopeValidation = jobBlock(workflow, "validation-scope");
  const goValidation = jobBlock(workflow, "validate-public-go");
  const protoValidation = jobBlock(workflow, "validate-public-proto");
  const contextValidation = jobBlock(workflow, "validate-runtime-contexts");
  const aggregate = jobBlock(workflow, "public-validation");
  const publicationIntent = jobBlock(workflow, "publication-intent");
  const runtimeCompositionResolver = jobBlock(workflow, "resolve-public-runtime-composition");
  const publication = jobBlock(workflow, "publish-runtime-images");
  const agentPublication = jobBlock(workflow, "publish-agent-image");
  const validationBlocks = [validation, frontendValidation, scopeValidation, goValidation, protoValidation, contextValidation, aggregate, publicationIntent];
  validationBlocks.forEach((block, index) => {
    const label = `public validation job ${index + 1}`;
    // Scope lookup uses only the ephemeral, read-only repository token.
    const secretless = block === scopeValidation
      ? block.replace("          GITHUB_TOKEN: ${{ github.token }}", "")
      : block;
    assertSecretlessBlock(secretless, label);
    if (block === scopeValidation && /(?:contents|actions|pull-requests|packages|id-token):\s*write/.test(block)) {
      fail("validation scope must have read-only API access");
    }
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
  if (!validation.includes("check-public-export.mjs") || !validation.includes("validate-public-documentation.mjs") || !validation.includes("verify-public-deployment.sh") ||
      !validation.includes("audit-public-security-scope.mjs") || !validation.includes("verify-public-release.mjs") ||
      !validation.includes("verify-public-runtime-source.sh") || !validation.includes("verify-public-runtime-contexts.mjs")) {
    fail("public validation workflow is missing the secretless boundary gates");
  }
  if (!validation.includes("channel_dir=channels") ||
      !validation.includes("channels and manifests must be both present directories or both absent") ||
      !validation.includes('if [ -e "$channel_dir" ] || [ -e "$manifest_dir" ]; then')) {
    fail("public validation must allow source exports before release-channel records exist");
  }
  if (!validation.includes("Install public deployment guard dependencies") ||
      !validation.includes("sudo apt-get install -y --no-install-recommends ripgrep")) {
    fail("public source validation must install ripgrep before running deployment guards");
  }
  for (const required of [
    'if [[ "$agent_directory" =~ ^sha256-[a-f0-9]{64}$ ]]; then',
    '--bundle-dir "agent/bin/$agent_directory" --artifact-id "$agent_directory"',
    '[ "$agent_directory" = "$release_tag" ] || {',
    '--bundle-dir "agent/bin/$release_tag" --public-export-root "$GITHUB_WORKSPACE"',
    '--tag "$release_tag" --source-revision-digest "$source_revision_digest"',
    '--public-provenance-sha256 "$public_provenance_sha256"',
  ]) {
    if (!validation.includes(required)) {
      fail(`public validation must retain the strict immutable and versioned Agent bundle transition: ${required}`);
    }
  }
  if (!validation.includes('tee "$RUNNER_TEMP/public-documentation-validation.json"') ||
      validation.includes("tee dist/public-documentation-validation.json")) {
    fail("public documentation validation must keep evidence outside the exported workspace");
  }
  for (const required of [
    "pnpm/action-setup@v4",
    "cache: pnpm",
    "cache-dependency-path: frontend/pnpm-lock.yaml",
    "working-directory: frontend",
    "pnpm install --frozen-lockfile",
    "pnpm exec playwright install --with-deps chromium",
    "pnpm run typecheck",
    "pnpm run lint",
    "pnpm run test",
    "pnpm run check:ui-foundation",
    "pnpm run check:service-mock-mode",
    "pnpm run check:foundation-ledger",
  ]) {
    if (!frontendValidation.includes(required)) fail(`public validation workflow is missing frontend gate: ${required}`);
  }
  if (!scopeValidation.includes("select-public-validation-scope.mjs") ||
      !scopeValidation.includes("fetch-depth: 0") ||
      !scopeValidation.includes('echo "scope=full"') ||
      !frontendValidation.includes("lane: [types, lint, test-1, test-2, test-3, test-4, test-5, test-6, test-7, test-8]") ||
      !frontendValidation.includes('pnpm run test --shard="${LANE#test-}/8"') ||
      !scopeValidation.includes("actions: read") ||
      !scopeValidation.includes("pull-requests: read") ||
      !aggregate.includes("validated-source") ||
      !aggregate.includes("needs.validate-frontend.result") ||
      !aggregate.includes('needs.validation-scope.result') ||
      !aggregate.includes('[ "$scope" = deployment ] || [ "$scope" = validated-source ]') ||
      workflow.includes("      - export/**")) {
    fail("public validation must use complete deployment scope, exhaustive frontend shards and an explicit aggregate gate without duplicate export pushes");
  }
  for (const block of [frontendValidation, goValidation, protoValidation, contextValidation]) {
    if (!block.includes("needs.validation-scope.outputs.scope == 'full'")) fail("application checks must consume full validation scope");
  }
  if (!goValidation.includes("cache-dependency-path: ${{ matrix.directory }}/go.sum")) fail("public Go cache must use module dependency paths");
  assertRuntimeComponentList(publication, "public Runtime publication");
  if (!/^permissions:\s*\n\s+contents:\s+read\s*$/m.test(workflow)) fail("public validation workflow must grant contents read only");
  for (const required of ["server", "contracts", "engine-go", "extensions", "go test ./...", "proto/scripts/check-generated.sh"]) {
    if (!goValidation.includes(required) && !protoValidation.includes(required)) fail(`public workflow is missing public source gate: ${required}`);
  }
  if (!goValidation.includes("Install server guard dependencies") ||
      !goValidation.includes("if: matrix.name == 'server'") ||
      !goValidation.includes("sudo apt-get install -y --no-install-recommends ripgrep")) {
    fail("public Server validation must install ripgrep before running the architecture guards");
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
  for (const required of [
    "if: github.repository == 'yyhuni/lunafox'",
    "name: Resolve Public Release Intent",
    "fetch-depth: 2",
    "publish: ${{ steps.intent.outputs.publish }}",
    'echo "publish=false" >> "$GITHUB_OUTPUT"',
    '[ "$GITHUB_EVENT_NAME" = push ] || exit 0',
    '[ "$GITHUB_REF" = refs/heads/main ] || exit 0',
    "git rev-parse --verify HEAD^",
    'release_tag="$(jq -er \'.releaseTag\' PUBLIC_PROVENANCE.json)"',
    "merge_subject_pattern=",
    "squash_subject_pattern=",
    '[[ "$head_subject" =~ $merge_subject_pattern || "$head_subject" =~ $squash_subject_pattern ]] || exit 0',
    "git diff-tree --no-commit-id --name-only -r HEAD^ HEAD -- PUBLIC_EXPORT_MANIFEST.json",
    "grep -Fxq PUBLIC_EXPORT_MANIFEST.json || exit 0",
    'echo "publish=true" >> "$GITHUB_OUTPUT"',
  ]) {
    if (!publicationIntent.includes(required)) fail(`public publication-intent gate is missing: ${required}`);
  }
  const executableWorkflow = workflow
    .split("\n")
    .filter((line) => !line.trimStart().startsWith("#"))
    .join("\n");
  if (/github\.event\.head_commit\.(?:message|modified)/.test(executableWorkflow)) {
    fail("public publication intent must not depend on incomplete push payload commit metadata");
  }
  if (runtimeCompositionResolver.includes(".prerelease == false")) {
    fail("runtime composition reuse lookup must consider prior prerelease releases");
  }
  const previousCompositionLookup = workflowStepBlock(
    runtimeCompositionResolver,
    "      - id: previous\n        name: Retrieve the previous verified composition asset",
  );
  for (const required of [
    "map(select(.draft == false and .tag_name != $current))",
    'any(.assets[]?; .name == "runtime-composition.json")',
    'any(.assets[]?; .name == "release.manifest.yaml")',
    'any(.assets[]?; .name == "component-evidence.json")',
    'any(.assets[]?; .name == "public-release-provenance.json")',
    "download_verified_asset()",
    "if length == 1 then .[0] else error(\"release asset must exist exactly once\") end",
    "expected=\"$(jq -er '.digest' <<<\"$asset_json\")\"",
    "actual=\"sha256:$(sha256sum \"$output_file\"",
    "download_verified_asset runtime-composition.json dist/public-composition/previous.json",
    "download_verified_asset release.manifest.yaml dist/public-composition/previous.manifest.yaml",
    "download_verified_asset component-evidence.json dist/public-composition/previous.component-evidence.json",
    "download_verified_asset public-release-provenance.json dist/public-composition/previous.public-release-provenance.json",
    "dist/public-composition/previous.manifest.yaml",
    "dist/public-composition/previous.component-evidence.json",
    "dist/public-composition/previous.public-release-provenance.json",
    "node scripts/ci/verify-public-release.mjs",
    "--manifest dist/public-composition/previous.manifest.yaml",
    "--tag \"$previous_release_tag\"",
    "node scripts/ci/verify-public-release-evidence.mjs",
    "--composition dist/public-composition/previous.json",
    "--bundle dist/public-composition/previous.component-evidence.json",
    "--provenance dist/public-composition/previous.public-release-provenance.json",
    "node scripts/ci/verify-release-component-composition.mjs",
    "--composition dist/public-composition/previous.json",
    "--manifest dist/public-composition/previous.manifest.yaml",
    "node scripts/ci/verify-component-evidence-bundle.mjs",
    "--bundle dist/public-composition/previous.component-evidence.json",
    "--manifest dist/public-composition/previous.manifest.yaml",
  ]) {
    if (!previousCompositionLookup.includes(required)) {
      fail(`runtime composition reuse lookup must fail closed on an unbound prior release: ${required}`);
    }
  }
  // Both validators must consume the exact downloaded manifest. Checking only
  // for one occurrence would let a maintenance edit silently disconnect the
  // component-evidence verifier while the composition verifier still passes.
  if (countOccurrences(previousCompositionLookup, "--manifest dist/public-composition/previous.manifest.yaml") < 4) {
    fail("runtime composition reuse lookup must fail closed on an unbound prior release: every prior evidence validator must bind the downloaded manifest");
  }
  for (const [name, block] of [["public Runtime publication", publication], ["public Agent publication", agentPublication]]) {
    if (!block.includes("needs.publication-intent.outputs.publish == 'true'") ||
        !hasRequiredJobNeeds(block, ["publication-intent", "public-validation", "resolve-public-runtime-composition"])) {
      fail(`${name} must consume the checked-out publication intent, completed public validation, and runtime composition plan`);
    }
  }
  if (!publication.includes("squash_subject_pattern") ||
      !publication.includes("chore\\\\(export\\\\):") ||
      !publication.includes("head_subject\" =~ $merge_subject_pattern || \"$head_subject\" =~ $squash_subject_pattern")) {
    fail("public Runtime publication must accept the configured squash-merge commit identity");
  }
  for (const required of [
    'commit_ref="${image}:public-${GITHUB_SHA}"',
    "if: steps.composition.outputs.disposition == 'built' && steps.preflight.outputs.commit_digest == ''",
    "EXISTING_DIGEST",
    "BUILT_DIGEST",
    "REUSED_DIGEST",
    'image_digest="$BUILT_DIGEST"',
    '[ -n "$image_digest" ] || image_digest="$EXISTING_DIGEST"',
    '[ -n "$image_digest" ] || image_digest="$REUSED_DIGEST"',
  ]) {
    if (!publication.includes(required)) fail(`public Runtime publication is missing immutable retry guard: ${required}`);
  }
  if (/tags:\s*\n\s+[^\n]*latest\b/.test(publication) || /imagetools\s+create/.test(publication)) {
    fail("public Runtime publication must not overwrite mutable tags or recreate an existing digest");
  }
  for (const required of [
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
  if (/DOCKERHUB_TOKEN|DOCKERHUB_USERNAME|DOCKERHUB_PASSWORD|context:\s*agent\b|dist\/agent-public-context/.test(publication)) {
    fail("public Runtime publication must not receive Docker Hub credentials or private build contexts");
  }
  for (const required of [
    "id: base-contexts",
    "render-release-component-build-contexts.mjs",
    "--plan dist/public-composition/runtime-composition-plan.json",
    "--base-images dist/public-composition/base-images.json",
    "build-contexts: ${{ steps.base-contexts.outputs.build_contexts }}",
  ]) {
    if (!publication.includes(required)) fail(`public Runtime publication must pin BuildKit base contexts through the composition renderer: ${required}`);
  }
  const runtimeBuildStep = workflowStepBlock(
    publication,
    "      - id: build\n        name: Build and publish immutable public Runtime image",
  );
  const runtimeBaseContextStep = workflowStepBlock(
    publication,
    "      - id: base-contexts\n        name: Pin Runtime BuildKit base-image contexts to the composition plan",
  );
  const runtimeBuildCondition = "if: steps.composition.outputs.disposition == 'built' && steps.preflight.outputs.commit_digest == ''";
  if (!runtimeBaseContextStep.includes(runtimeBuildCondition) || countOccurrences(publication, runtimeBuildCondition) !== 2) {
    fail("public Runtime base-context rendering and image build must share the immutable retry guard");
  }
  if (!runtimeBuildStep.includes("uses: docker/build-push-action@v6") ||
      countOccurrences(runtimeBuildStep, "build-contexts: ${{ steps.base-contexts.outputs.build_contexts }}") !== 1 ||
      !runtimeBuildStep.includes(runtimeBuildCondition)) {
    fail("public Runtime build must consume exactly one renderer-produced BuildKit context mapping");
  }
  for (const required of [
    "contents: read",
    "packages: write",
    "id-token: write",
    "attestations: write",
    "verify-agent-binary-bundle.mjs",
    '--bundle-dir "$BUNDLE_DIR"',
    '--artifact-id "$artifact_id"',
    '--input-fingerprint "$input_fingerprint"',
    '--source-release-tag "$bundle_source_release_tag"',
    "cosign verify-blob",
    "agent-bundle.sigstore.json",
    "--certificate-oidc-issuer https://token.actions.githubusercontent.com",
    "--certificate-identity-regexp",
    "refs/tags/${bundle_source_release_tag}$",
    "docker/agent/Dockerfile",
    "push: true",
    "provenance: mode=max",
    "sbom: true",
    "actions/attest-build-provenance@v2",
    "docker/login-action@v3",
    "registry: ghcr.io",
    "username: ${{ github.actor }}",
    "password: ${{ github.token }}",
    "GH_TOKEN: ${{ github.token }}",
    "/user/packages/container/lunafox-agent",
    "GHCR Agent package must be public",
  ]) {
    if (!agentPublication.includes(required)) fail(`public Agent publication is missing: ${required}`);
  }
  for (const required of [
    'bundle_source_release_tag="${{ steps.composition.outputs.source_release_tag }}"',
    '[ -n "$bundle_source_release_tag" ] || bundle_source_release_tag="$RELEASE_TAG"',
    'refs/tags/${bundle_source_release_tag}$',
  ]) {
    if (!agentPublication.includes(required)) fail(`public Agent publication must verify the immutable bundle against its planned source release: ${required}`);
  }
  if (agentPublication.includes("--public-export-root")) {
    fail("public Agent publication must bind verification to the immutable artifact identity");
  }
  if (!agentPublication.includes('--json | tee "$RUNNER_TEMP/agent-bundle-verification.json"') ||
      !agentPublication.includes('cp "$RUNNER_TEMP/agent-bundle-verification.json" dist/agent-bundle-verification.json') ||
      agentPublication.includes("--json | tee dist/agent-bundle-verification.json")) {
    fail("public Agent bundle verification must keep temporary evidence outside the exported workspace");
  }
  if (!agentPublication.includes("context: .") ||
      !agentPublication.includes("AGENT_ARTIFACT_ID=${{ steps.verify.outputs.artifact_id }}") ||
      agentPublication.includes('context: ${{ steps.verify.outputs.bundle_dir }}') ||
      agentPublication.includes("stagingIdentity") || agentPublication.includes("agent-input-") ||
      /DOCKERHUB_TOKEN|DOCKERHUB_USERNAME|DOCKERHUB_PASSWORD/.test(agentPublication)) {
    fail("public Agent publication must build from the repository root with the verified immutable artifact context and without Docker Hub credentials");
  }
  for (const required of [
    "id: base-contexts",
    "render-release-component-build-contexts.mjs",
    "--plan dist/public-composition/runtime-composition-plan.json",
    "--component-id runtime.agent",
    "--base-images dist/public-composition/base-images.json",
    "build-contexts: ${{ steps.base-contexts.outputs.build_contexts }}",
  ]) {
    if (!agentPublication.includes(required)) fail(`public Agent publication must pin BuildKit base contexts through the composition renderer: ${required}`);
  }
  const agentBuildStep = workflowStepBlock(
    agentPublication,
    "      - id: build\n        name: Build and publish the multi-architecture Agent image",
  );
  if (!agentBuildStep.includes("uses: docker/build-push-action@v6") ||
      countOccurrences(agentBuildStep, "build-contexts: ${{ steps.base-contexts.outputs.build_contexts }}") !== 1) {
    fail("public Agent build must consume exactly one renderer-produced BuildKit context mapping");
  }
  const agentGhcrLogin = agentPublication.indexOf("name: Log in to GHCR");
  const agentBuild = agentPublication.indexOf("name: Build and publish the multi-architecture Agent image");
  const agentVisibilityCheck = agentPublication.indexOf("name: Ensure Agent GHCR package is public");
  const agentSign = agentPublication.indexOf("name: Keylessly sign and verify the public Agent image");
  if (agentGhcrLogin < 0 || agentBuild < 0 || agentVisibilityCheck < 0 || agentSign < 0 ||
      !(agentGhcrLogin < agentBuild && agentBuild < agentVisibilityCheck && agentVisibilityCheck < agentSign)) {
    fail("public Agent publication must log in before pushing, then require public GHCR visibility before signing");
  }
  // A public workflow may be exported without the private release workflow;
  // its machine-readable policy still has to pin the same identity.
  if (policy.publicMain?.allowDirectPush !== false || policy.publicMain?.allowForcePush !== false) {
    fail("public main policy must disable direct and force pushes");
  }

  const engineDiscovery = jobBlock(workflow, "public-engine-runtime-discover");
  const engineBuild = jobBlock(workflow, "public-engine-runtime-build");
  const engineAggregate = jobBlock(workflow, "public-engine-runtime-aggregate");
  const engineFinalize = jobBlock(workflow, "public-engine-runtime-finalize");
  const engineSign = jobBlock(workflow, "public-engine-runtime-sign");
  const packageBuild = jobBlock(workflow, "public-engine-package-build");
  const packagePublish = jobBlock(workflow, "public-engine-package-publish");
  const engineManifest = jobBlock(workflow, "public-engine-release-manifest");
  const finalRelease = jobBlock(workflow, "publish-final-release");
  const engineDiscoveryArtifactName = "public-engine-runtime-discovery-${{ github.sha }}";
  const engineDiscoveryArtifactDirectory = "dist/public-engine-runtime-discovery";
  const engineDiscoveryArtifactUpload = [
    "      - uses: actions/upload-artifact@v4",
    "        with:",
    `          name: ${engineDiscoveryArtifactName}`,
    `          path: ${engineDiscoveryArtifactDirectory}/**`,
  ].join("\n");
  const engineDiscoveryArtifactDownload = [
    "      - uses: actions/download-artifact@v4",
    "        with:",
    `          name: ${engineDiscoveryArtifactName}`,
    `          path: ${engineDiscoveryArtifactDirectory}`,
  ].join("\n");
  for (const [name, block] of PUBLIC_ENGINE_JOBS.map((name) => [name, jobBlock(workflow, name)])) {
    if (!block.includes("needs.publication-intent.outputs.publish == 'true'") ||
        !/needs:\s*\[[^\]]*publication-intent[^\]]*\]/.test(block)) {
      fail(`${name} must consume the checked-out publication intent`);
    }
    if (name === "public-engine-runtime-build") {
      if (!block.includes("runs-on: ${{ matrix.runner }}") ||
          !engineDiscovery.includes('runner:"ubuntu-24.04"') ||
          !engineDiscovery.includes('runner:"ubuntu-24.04-arm"')) {
        fail("native Engine matrix must select fixed GitHub-hosted architectures");
      }
    } else {
      assertGitHubHostedRunner(block, name);
    }
  }
  for (const [name, block] of [["public Engine Runtime discovery", engineDiscovery], ["public Engine Runtime build", engineBuild], ["public Engine Runtime finalize", engineFinalize], ["public Engine Runtime aggregate", engineAggregate], ["public Engine Runtime sign", engineSign], ["public Engine Package build", packageBuild], ["public Engine Package publish", packagePublish], ["public Engine release manifest", engineManifest]]) {
    if (/agent\/|lunafox-private|PRIVATE_TRUSTED_RUNNER|GITHUB_PAT|COSIGN_PRIVATE_KEY/.test(block)) {
      fail(`${name} must not contain private Agent/release authority`);
    }
  }
  if (!hasRequiredJobNeeds(engineDiscovery, ["publication-intent", "public-validation", "resolve-public-runtime-composition"]) ||
      !engineDiscovery.includes("outputs:") ||
      !engineDiscovery.includes("steps.matrix.outputs.matrix") ||
      !engineDiscovery.includes("-command discover") ||
      !engineDiscovery.includes("PUBLIC_PROVENANCE.json") ||
      !engineDiscovery.includes("PUBLIC_EXPORT_MANIFEST.json") ||
      !engineDiscovery.includes("public-engine-runtime-discovery-") ||
      !engineDiscovery.includes(engineDiscoveryArtifactUpload) ||
      engineDiscovery.includes("packages: write") ||
      engineDiscovery.includes("environment: public-engine-release")) {
    fail("public Engine Runtime discovery must source-bind and emit the dynamic matrix with its canonical artifact layout and without publication authority");
  }
  if (!hasRequiredJobNeeds(engineBuild, ["publication-intent", "public-engine-runtime-discover"]) ||
      !engineBuild.includes("strategy:") ||
      !engineBuild.includes("fail-fast: true") ||
      !engineBuild.includes("max-parallel: 16") ||
      !engineBuild.includes("fromJSON(needs.public-engine-runtime-discover.outputs.platform_matrix)") ||
      !engineBuild.includes("runs-on: ${{ matrix.runner }}") ||
      !engineBuild.includes("matrix.platform") ||
      !engineBuild.includes("ENGINE_RELEASE_ENGINE_ID") ||
      !engineBuild.includes("matrix.engineId") ||
      !engineBuild.includes("public-engine-runtime-platform-") ||
      !engineBuild.includes("build-engine-runtime-platform.sh") ||
      !engineBuild.includes("packages: write")) {
    fail("public Engine Runtime build must use a bounded source-derived matrix and the validated production publisher");
  }
  if (!engineBuild.includes(engineDiscoveryArtifactDownload)) {
    fail("public Engine Runtime build must restore source-bound discovery at its canonical artifact path");
  }
  for (const required of [
    "id: base-contexts",
    "render-release-component-build-contexts.mjs",
    "--plan dist/public-composition/runtime-composition-plan.json",
    "--base-images dist/public-composition/base-images.json",
    "ENGINE_BASE_IMAGE_CONTEXTS_FILE: ${{ steps.base-contexts.outputs.path }}",
  ]) {
    if (!engineBuild.includes(required)) fail(`public Engine Runtime build must pin BuildKit base contexts through the composition renderer: ${required}`);
  }
  if (/continue-on-error\s*:/.test(engineBuild) || /fail-fast\s*:\s*false/.test(engineBuild)) {
    fail("public Engine Runtime matrix must fail fast without optional children");
  }
  if (!hasRequiredJobNeeds(engineFinalize, ["publication-intent", "public-engine-runtime-discover", "public-engine-runtime-build"]) ||
      !engineFinalize.includes("max-parallel: 8") ||
      !engineFinalize.includes("finalize-engine-runtime-platforms.sh") ||
      !engineFinalize.includes("public-engine-runtime-shard-") ||
      !engineFinalize.includes("packages: write")) {
    fail("public Engine Runtime finalize must assemble each exact native platform pair with publication authority");
  }
  if (!hasRequiredJobNeeds(engineAggregate, ["publication-intent", "public-engine-runtime-discover", "public-engine-runtime-finalize"]) ||
      !engineAggregate.includes("aggregate-engine-runtime-image-shards.sh") ||
      !engineAggregate.includes("ENGINE_RUNTIME_IMAGE_DISCOVERY") ||
      !engineAggregate.includes("ENGINE_RUNTIME_IMAGE_SHARDS_ROOT") ||
      !engineAggregate.includes("ENGINE_RUNTIME_IMAGE_AGGREGATE_OUTPUT_ROOT") ||
      !engineAggregate.includes("pattern: public-engine-runtime-shard-") ||
      !engineAggregate.includes("merge-multiple: false") ||
      !engineAggregate.includes("name: public-engine-runtime-build-")) {
    fail("public Engine Runtime aggregate must strictly restore the complete receipt from every shard");
  }
  if (!engineAggregate.includes(engineDiscoveryArtifactDownload)) {
    fail("public Engine Runtime aggregate must restore source-bound discovery at its canonical artifact path");
  }
  if (/packages:\s*write|id-token:\s*write|attestations:\s*write|environment:\s*public-engine-release|docker\/login-action|cosign /.test(engineAggregate)) {
    fail("public Engine Runtime aggregate must not receive publishing or signing authority");
  }
  for (const lane of [engineSign, packagePublish, finalRelease]) {
    if (!lane.includes("sigstore/cosign-installer@v4.1.2") ||
        !lane.includes("cosign-release: v3.1.3") ||
        !lane.includes('./cmd/verify-signature') ||
        !lane.includes('"$RUNNER_TEMP/verify-oci-signature" "$ref"')) {
      fail("public signatures must use pinned Sigstore bundle publication and the anonymous installation verifier");
    }
  }
  for (const name of ["publish-runtime-images", "publish-agent-image", ...PUBLIC_ENGINE_JOBS]) {
    const block = jobBlock(executableWorkflow, name);
    const condition = jobCondition(block);
    const dependencies = block.match(/^    needs: \[([^\]]+)\]/m)?.[1].split(", ") ?? [];
    const hasAllDirectSuccess = dependencies.every((dependency) => condition.includes(`needs.${dependency}.result == 'success'`));
    const hasConditionalFinalizeSkip = name === "public-engine-runtime-aggregate" &&
      condition.includes("needs.public-engine-runtime-finalize.result == 'success'") &&
      condition.includes("needs.public-engine-runtime-finalize.result == 'skipped'") &&
      condition.includes("fromJSON(needs.public-engine-runtime-discover.outputs.built_count) == 0");
    if (!condition.includes("always()") || condition.includes("needs.*.result") || dependencies.length === 0 ||
        (!hasAllDirectSuccess && !hasConditionalFinalizeSkip)) {
      fail("public publication must explicitly accept successful direct dependencies after validation reuse");
    }
  }

  const publicCosignIdentityRegexp = "'^https://github\\.com/yyhuni/lunafox/\\.github/workflows/public-validate\\.yml@refs/heads/main$'";
  const overescapedPublicCosignIdentityRegexp = "'^https://github\\\\.com/yyhuni/lunafox/\\\\.github/workflows/public-validate\\\\.yml@refs/heads/main$'";
  if (workflow.includes(overescapedPublicCosignIdentityRegexp) || countOccurrences(workflow, publicCosignIdentityRegexp) !== 4) {
    fail("public workflow must pass the exact single-escaped main workflow identity to cosign");
  }
  if (!hasRequiredJobNeeds(engineSign, ["publication-intent", "public-engine-runtime-discover", "public-engine-runtime-aggregate"]) ||
      !engineSign.includes("cosign sign --yes") ||
      !engineSign.includes("cosign verify") ||
      !engineSign.includes("public-validate\\.yml@refs/heads/main") ||
      !engineSign.includes("public-engine-runtime-evidence-") ||
      !engineSign.includes("registryAuth:\"empty\"")) {
    fail("public Engine Runtime lane must sign, anonymously verify, and retain evidence");
  }
  const runtimeReceiptStart = engineSign.indexOf("      - name: Attest public Engine Runtime release receipt");
  const runtimeReceiptEnd = engineSign.indexOf("\n      - uses: actions/upload-artifact@v4", runtimeReceiptStart);
  const runtimeReceipt = engineSign.slice(runtimeReceiptStart, runtimeReceiptEnd);
  if (runtimeReceiptStart < 0 || runtimeReceiptEnd < 0 ||
      !runtimeReceipt.includes("subject-path: dist/public-engine-runtime/build-results.json") ||
      runtimeReceipt.includes("push-to-registry:")) {
    fail("public Engine Runtime release receipt must use its file subject without registry delivery");
  }
  if (!hasRequiredJobNeeds(packageBuild, ["publication-intent", "public-engine-runtime-discover", "public-engine-runtime-sign"]) ||
      !packageBuild.includes("-command build-packages") ||
      !packageBuild.includes("-build-results") ||
      !packageBuild.includes("-mode production") ||
      !packageBuild.includes("-command validate-package-artifacts") ||
      !packageBuild.includes("-package-build-results") ||
      !packageBuild.includes("-out-root") ||
      packageBuild.includes("check-engine-release-contract.mjs")) {
    fail("public Engine Package build must derive and validate packages with the public Go release tool");
  }
  if (!packageBuild.includes("package-version-map.json") ||
      countOccurrences(packageBuild, "-package-version-map") < 2 ||
      !packageBuild.includes("schemaVersion: \"lunafox.engine-package-version-map.v1\"") ||
      !packageBuild.includes(".planDigest") ||
      !packageBuild.includes("dist/public-composition/runtime-composition-plan.json") ||
      packageBuild.includes('-engine-version "${release_tag#v}"')) {
    fail("public Engine Package build must bind per-Engine content-addressed versions from the resolved composition plan");
  }
  if (!hasRequiredJobNeeds(packagePublish, ["publication-intent", "public-engine-runtime-discover", "public-engine-package-build"]) ||
      !packagePublish.includes("oras cp") ||
      !packagePublish.includes("cosign sign --yes") ||
      !packagePublish.includes("cosign verify") ||
      !packagePublish.includes("public-engine-package-evidence-") ||
      !packagePublish.includes("public-engine-package-digests")) {
    fail("public Engine Package lane must publish, sign, verify, and retain digest evidence");
  }
  if (!packagePublish.includes('package_tag="package-v2-${package_digest#sha256:}"') ||
      !packagePublish.includes('--tag "$package_tag"') ||
      !packagePublish.includes('oras cp "$docker_ref" "$ghcr_location:$package_tag"') ||
      packagePublish.includes("package-v2-${release_tag}")) {
    fail("public Engine Package lane must derive its publication handle from immutable archive bytes, not the product release tag");
  }
  const packageEvidenceValidation = 'all(.packages[]; .registryAuth == "empty" and .digestEqualityVerified and .envelopeVerified and (.artifactManifestDigest | test("^sha256:[a-f0-9]{64}$")))';
  if (!packagePublish.includes(packageEvidenceValidation) || packagePublish.includes('all(.[]; .registryAuth == "empty"')) {
    fail("public Engine Package lane must validate each package evidence record");
  }
  const packageEvidenceRecordProjection = 'map({engineId,packageDigest,artifactManifestDigest,candidates,registryAuth,digestEqualityVerified,envelopeVerified})';
  if (!packagePublish.includes(packageEvidenceRecordProjection)) {
    fail("public Engine Package lane must retain registry verification evidence in aggregate records");
  }
  if (!hasRequiredJobNeeds(engineManifest, ["publication-intent", "public-engine-runtime-discover", "public-engine-runtime-aggregate", "public-engine-package-publish", "resolve-public-runtime-composition"]) ||
      !engineManifest.includes("assemble-public-engine-release.mjs") ||
      !engineManifest.includes("--output-dir dist/public-engine-release") ||
      !engineManifest.includes("assembly-verification.json") ||
      !engineManifest.includes("name: Publish Public Engine Release Manifest")) {
    fail("public Engine lane must publish an independently verified manifest handoff");
  }
  if (!hasRequiredJobNeeds(finalRelease, ["publication-intent", "resolve-public-runtime-composition", "publish-runtime-images", "publish-agent-image", "public-engine-release-manifest"])) {
    fail("public final release must wait for the resolved Runtime composition and every component publication");
  }
  const finalCompositionPlanDownload = workflowStepBlock(
    finalRelease,
    "      - uses: actions/download-artifact@v4\n        with:\n          name: public-runtime-composition-plan-${{ github.sha }}",
  );
  if (!finalCompositionPlanDownload.includes("path: dist/final/composition-plan")) {
    fail("public final release must restore the resolved runtime composition plan at its canonical final path");
  }
  const finalCompositionFinalization = workflowStepBlock(
    finalRelease,
    "      - id: composition\n        name: Finalize the canonical runtime composition core",
  );
  for (const required of [
    "node scripts/ci/resolve-public-runtime-composition.mjs",
    "--mode final",
    '--plan "$plan"',
    "--output dist/final/runtime-composition.core.json",
    "composition_digest=\"$(jq -er '.compositionDigest' dist/final/runtime-composition.core.json)\"",
    'echo "composition_digest=$composition_digest" >> "$GITHUB_OUTPUT"',
  ]) {
    if (!finalCompositionFinalization.includes(required)) {
      fail(`public final release must finalize the canonical runtime composition before manifest generation: ${required}`);
    }
  }
  const finalManifestGeneration = workflowStepBlock(
    finalRelease,
    "      - name: Generate and verify the complete release manifest",
  );
  if (!finalManifestGeneration.includes("RUNTIME_COMPOSITION_SHA256: ${{ steps.composition.outputs.composition_digest }}") ||
      !finalManifestGeneration.includes("generate-release-manifest.sh")) {
    fail("public final release must pass the canonical runtime composition digest into manifest generation");
  }
  for (const required of [
    "--mode bind",
    "--plan dist/final/runtime-composition.core.json",
    '--manifest-digest "$manifest_digest"',
    "--output dist/final/runtime-composition.json",
  ]) {
    if (!finalManifestGeneration.includes(required)) {
      fail(`public final release must bind the runtime composition to the generated manifest: ${required}`);
    }
  }
  for (const required of [
    "node scripts/ci/verify-release-component-composition.mjs",
    "--composition dist/final/runtime-composition.json",
    "--manifest dist/final/release.manifest.yaml",
  ]) {
    if (!finalManifestGeneration.includes(required)) {
      fail(`public final release must verify the manifest-bound runtime composition: ${required}`);
    }
  }
  const finalComponentEvidence = workflowStepBlock(
    finalRelease,
    "      - name: Assemble and verify durable component evidence",
  );
  for (const required of [
    "node scripts/ci/build-component-evidence-bundle.mjs",
    "--composition dist/final/runtime-composition.json",
    "--runtime-evidence-dir dist/final/runtime-evidence",
    "--output dist/final/component-evidence.json",
    "node scripts/ci/verify-component-evidence-bundle.mjs",
    "--bundle dist/final/component-evidence.json",
    "--manifest dist/final/release.manifest.yaml",
  ]) {
    if (!finalComponentEvidence.includes(required)) {
      fail("public final release must assemble and verify original component evidence: " + required);
    }
  }
  const finalComposePublication = workflowStepBlock(
    finalRelease,
    "      - name: Build immutable Docker Compose deployment packages",
  );
  if (!finalComposePublication.includes("generate-compose-deployment.mjs") ||
      !finalComposePublication.includes("--runtime-composition dist/final/runtime-composition.json")) {
    fail("public final release must pass the bound runtime composition to Compose generation");
  }
  const finalChannelPublication = workflowStepBlock(
    finalRelease,
    "      - name: Generate and publish the release-channel branch",
  );
  if (!finalChannelPublication.includes("generate-public-channel.mjs") ||
      !finalChannelPublication.includes("--runtime-composition dist/final/runtime-composition.json")) {
    fail("public final release must pass the bound runtime composition to channel generation");
  }
  const finalGitHubRelease = workflowStepBlock(
    finalRelease,
    "      - name: Publish the final GitHub Release metadata",
  );
  for (const required of [
    "publish_asset dist/final/runtime-composition.json",
    "publish_asset dist/final/component-evidence.json",
  ]) {
    if (!finalGitHubRelease.includes(required)) {
      fail("public final release must publish durable release evidence as a GitHub Release asset: " + required);
    }
  }
  const finalEvidenceArtifact = workflowStepBlock(
    finalRelease,
    "      - name: Upload the complete final publication evidence",
  );
  if (!finalEvidenceArtifact.includes("dist/final/component-evidence.json")) {
    fail("public final release artifact must retain the durable component evidence bundle");
  }
  if (!finalRelease.includes('node scripts/ci/check-public-channel.mjs --root-dir "$channel_root"') ||
      finalRelease.includes("check-public-channel.mjs --root-dir \"$channel_root\" --require-first-release")) {
    fail("public final release must validate the actual append-only channel without requiring an absent historical first record");
  }
  if (!finalRelease.includes("validate-public-release-notes.mjs") ||
      !finalRelease.includes("validate-public-documentation.mjs") ||
      !finalRelease.includes('NOTES_FILE="release-notes/${RELEASE_TAG}.md"') ||
      !finalRelease.includes('release_args=("$RELEASE_TAG" --repo "$PUBLIC_REPOSITORY" --verify-tag --title "$RELEASE_TAG" --notes-file "$NOTES_FILE")') ||
      !finalRelease.includes('gh release edit "$RELEASE_TAG" --repo "$PUBLIC_REPOSITORY" --draft=false --prerelease="$prerelease" --notes-file "$NOTES_FILE"') ||
      finalRelease.includes("--generate-notes")) {
    fail("public final release must validate and publish the exact Tag release notes without generated fallbacks");
  }
  for (const required of [
    "publish-public-deployment.mjs",
    "--snapshot dist/final/deployment-snapshot",
    "--timeout-seconds 10800",
    "steps.deployment.outputs.merge_sha",
    "lunafox-${RELEASE_TAG}.zip",
    "DEPLOYMENT_SNAPSHOT_COMMIT",
    "gh release create",
    "--verify-tag",
    "needs.publication-intent.outputs.publish == 'true'",
  ]) {
    if (!finalRelease.includes(required)) fail(`public final release is missing deployment finalization: ${required}`);
  }
  if (!finalRelease.includes("[ \"$(jq 'length' dist/final/deployment-packages.json)\" -eq 1 ]") ||
      finalRelease.includes("-dockerhub.zip") || finalRelease.includes("-ghcr.zip")) {
    fail("public final release must publish exactly one unified deployment ZIP");
  }
  if (!finalRelease.includes('if gh api "/repos/${PUBLIC_REPOSITORY}/git/ref/tags/${RELEASE_TAG}" >"$tag_ref_response" 2>/dev/null; then') ||
      !finalRelease.includes("tag_ref_status=\"$(jq -r '.status // empty' \"$tag_ref_response\" 2>/dev/null || true)\"") ||
      !finalRelease.includes('[ "$tag_ref_status" = "404" ]') ||
      finalRelease.includes("--jq '.object.sha' 2>/dev/null || true")) {
    fail("public final release must distinguish an absent immutable tag from GitHub API failures");
  }
  if (!finalRelease.includes("final tag cross-registry drift") ||
      !finalRelease.includes("reused_final_tag=true") ||
      !finalRelease.includes("REUSED_FINAL_TAG") ||
      !finalRelease.includes('DOCKER_CONFIG="$anonymous_config" cosign verify') ||
      !finalRelease.includes('"^https://github\\\\.com/yyhuni/lunafox/\\\\.github/workflows/public-validate\\\\.yml@refs/heads/main$"')) {
    fail("public final release must anonymously verify and reuse a consistent immutable final tag on a maintenance retry");
  }
}

function assertExportPolicy(exportPolicy) {
  if (exportPolicy.sourceRepository !== PRIVATE_REPOSITORY || exportPolicy.destinationRepository !== PUBLIC_REPOSITORY) {
    fail("export policy repository identity drifted");
  }
  const rawExact = [
    ...(exportPolicy.allowlist?.exact ?? []),
    ...(exportPolicy.allowlist?.generatedExact ?? []),
  ];
  if (new Set(rawExact).size !== rawExact.length) {
    fail("public export policy must not contain duplicate exact allowlist paths");
  }
  const rawRequiredPaths = exportPolicy.requiredPaths ?? [];
  if (new Set(rawRequiredPaths).size !== rawRequiredPaths.length) {
    fail("public export policy must not contain duplicate required paths");
  }
  const exact = new Set(rawExact);
  const gitPolicy = exportPolicy.git ?? {};
  if (gitPolicy.workflowAuthorName !== "LunaFox Workflow Publisher" ||
      gitPolicy.workflowAuthorEmail !== "workflow-publisher@users.noreply.github.com" ||
      gitPolicy.workflowCommitMessagePattern !== "^chore\\(workflow\\): update public validation for v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[A-Za-z0-9.-]+)?$" ||
      gitPolicy.workflowSquashCommitMessagePattern !== "^chore\\(workflow\\): update public validation for v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[A-Za-z0-9.-]+)? \\(#[0-9]+\\)$" ||
      gitPolicy.workflowMergeCommitMessagePattern !== "^Merge pull request #[0-9]+ from [A-Za-z0-9_.-]+/workflow/v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[A-Za-z0-9.-]+)?(?:-retry-[0-9]+)?$") {
    fail("public export policy must constrain protected validation workflow maintenance history");
  }
  const destinationOwnedExact = [...new Set(exportPolicy.destinationOwnedExact ?? [])].sort();
  const expectedDestinationOwned = [".github/workflows/public-validate.yml", ...DESTINATION_DEPLOYMENT_PATHS].sort();
  if (JSON.stringify(destinationOwnedExact) !== JSON.stringify(expectedDestinationOwned)) {
    fail("public export policy must declare the validation workflow and deployment snapshot as destination-owned");
  }
  for (const destinationPath of destinationOwnedExact) {
    if (!exact.has(destinationPath)) fail(`destination-owned path is not allowlisted: ${destinationPath}`);
  }
  for (const required of ["scripts/ci/check-public-release-policy.mjs", "scripts/ci/verify-public-main-merge.mjs"]) {
    if (!exact.has(required)) fail(`export policy does not allow ${required}`);
  }
  for (const required of [
    "release-notes/README.md",
    "scripts/ci/validate-public-release-notes.mjs",
    "scripts/ci/validate-public-release-notes.test.mjs",
  ]) {
    if (!exact.has(required)) fail(`export policy does not allow release notes input: ${required}`);
  }
  for (const required of REQUIRED_PUBLIC_DOCUMENTATION_PATHS) {
    if (!exact.has(required)) fail(`export policy does not allow bilingual public documentation: ${required}`);
  }
  if (!(exportPolicy.allowlist?.prefixes ?? []).includes("release-notes/")) {
    fail("export policy must allow the release-notes/ prefix");
  }
  if (!(exportPolicy.denylist ?? []).some((pattern) => String(pattern) === "^release-note-evidence(?:/|$)")) {
    fail("export policy must keep private release-note evidence outside the public projection");
  }
  for (const required of REQUIRED_PUBLIC_CHECKOUT_DEPLOYMENT_PATHS) {
    if (!exact.has(required)) fail(`export policy does not allow checkout deployment input: ${required}`);
  }
  const groups = exportPolicy.destinationOwnedGroups ?? [];
  if (groups.length !== 1 || groups[0]?.sentinel !== ".env" ||
      JSON.stringify(groups[0]?.paths) !== JSON.stringify(DESTINATION_DEPLOYMENT_PATHS)) {
    fail("public export policy must define one complete .env-sentinel deployment snapshot group");
  }
  for (const required of DESTINATION_DEPLOYMENT_PATHS) {
    if (!exact.has(required)) fail(`export policy does not allow destination deployment path: ${required}`);
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
  for (const required of REQUIRED_PUBLIC_DOCUMENTATION_PATHS.slice(0, 4)) {
    if (!requiredPaths.has(required)) fail(`export policy must require bilingual public documentation: ${required}`);
  }
  const generatedMarkers = new Set(exportPolicy.generatedMarkers ?? []);
  for (const required of REQUIRED_PUBLIC_DOCUMENTATION_PATHS.slice(0, 4)) {
    if (!generatedMarkers.has(required)) fail(`export policy must mark generated public documentation: ${required}`);
  }
  if (!requiredPaths.has("release-notes/README.md")) {
    fail("export policy must require release-notes/README.md");
  }
  for (const required of REQUIRED_PUBLIC_CHECKOUT_DEPLOYMENT_PATHS) {
    if (!requiredPaths.has(required)) {
      fail(`export policy is missing required checkout deployment path: ${required}`);
    }
  }
  for (const required of REQUIRED_PUBLIC_UPGRADER_PATHS) {
    if (!requiredPaths.has(required)) {
      fail(`export policy is missing required public upgrader path: ${required}`);
    }
  }
  const bootstrapDescriptor = exportPolicy.publicRuntimeImages.find((item) => item.component === "bootstrap");
  for (const required of REQUIRED_PUBLIC_UPGRADER_PATHS) {
    if (!(bootstrapDescriptor?.requiredPaths ?? []).includes(required)) {
      fail(`Bootstrap Runtime descriptor is missing upgrader input: ${required}`);
    }
  }
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
  for (const required of ["server", "server/scripts", "contracts", "engine-go", "proto", "extensions", "docker/bootstrap", "docker/nginx", "tools/engine-release", "tools/engine-oci-publish"]) {
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
  assertLegacyDeploymentBootstrapPolicy(policy);
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
