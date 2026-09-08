#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { randomUUID } from "node:crypto";
import { spawnSync } from "node:child_process";

const defaultRepoRoot = path.resolve(new URL("../..", import.meta.url).pathname);

function usage() {
  return `Usage: node scripts/ci/check-engine-image-tool-inventory.mjs [options]

Validate the builtin Engine Runtime Image tool inventory and product-image
exclusion boundary.

Options:
  --repo-root PATH               Repository root
  --inventory PATH               Inventory JSON path
  --docker-command PATH          Docker CLI path for runtime checks
  --product-image ID=REF         Verify a built product image has no engine tools
  --engine-image DIRECTORY=REF   Verify a built engine image has its inventory
  --skip-product-image-source-check
                              Verify an Engine image payload without private product Dockerfiles
  --require-runtime-images       Require refs for every product and engine image
  -h, --help                     Show this help
`;
}

function parseBinding(value, flag) {
  const separator = value.indexOf("=");
  if (separator <= 0 || separator === value.length - 1) {
    throw new Error(`${flag} requires ID=REF`);
  }
  return [value.slice(0, separator), value.slice(separator + 1)];
}

function parseArgs(argv) {
  const args = {
    repoRoot: defaultRepoRoot,
    inventoryPath: null,
    dockerCommand: "docker",
    productImages: new Map(),
    engineImages: new Map(),
    skipProductImageSourceCheck: false,
    requireRuntimeImages: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const next = () => {
      const value = argv[index + 1];
      if (!value) throw new Error(`${arg} requires a value`);
      index += 1;
      return value;
    };
    if (arg === "--repo-root") args.repoRoot = path.resolve(next());
    else if (arg === "--inventory") args.inventoryPath = path.resolve(next());
    else if (arg === "--docker-command") args.dockerCommand = next();
    else if (arg === "--product-image") {
      const [id, ref] = parseBinding(next(), arg);
      if (args.productImages.has(id)) throw new Error(`duplicate product image ref: ${id}`);
      args.productImages.set(id, ref);
    } else if (arg === "--engine-image") {
      const [id, ref] = parseBinding(next(), arg);
      if (args.engineImages.has(id)) throw new Error(`duplicate engine image ref: ${id}`);
      args.engineImages.set(id, ref);
    } else if (arg === "--skip-product-image-source-check") args.skipProductImageSourceCheck = true;
    else if (arg === "--require-runtime-images") args.requireRuntimeImages = true;
    else if (arg === "--help" || arg === "-h") {
      process.stdout.write(usage());
      process.exit(0);
    } else throw new Error(`unknown argument: ${arg}`);
  }

  if (!args.inventoryPath) {
    args.inventoryPath = path.join(args.repoRoot, "extensions/engines/container/tool-inventory.json");
  }
  if (args.skipProductImageSourceCheck) {
    if (args.engineImages.size === 0) {
      throw new Error("--skip-product-image-source-check requires at least one --engine-image");
    }
    if (args.productImages.size !== 0) {
      throw new Error("--skip-product-image-source-check cannot be combined with --product-image");
    }
    if (args.requireRuntimeImages) {
      throw new Error("--skip-product-image-source-check cannot be combined with --require-runtime-images");
    }
  }
  return args;
}

function assertObject(value, label) {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${label} must be an object`);
  }
}

function assertExactKeys(value, keys, label) {
  const actual = Object.keys(value).sort();
  const expected = [...keys].sort();
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`${label} fields are ${actual.join(", ")}; expected ${expected.join(", ")}`);
  }
}

function assertCanonicalText(value, label, pattern) {
  if (typeof value !== "string" || value.trim() !== value || !pattern.test(value)) {
    throw new Error(`${label} is invalid: ${JSON.stringify(value)}`);
  }
}

function loadInventory(inventoryPath) {
  let inventory;
  try {
    inventory = JSON.parse(fs.readFileSync(inventoryPath, "utf8"));
  } catch (error) {
    throw new Error(`read inventory ${inventoryPath}: ${error.message}`);
  }
  assertObject(inventory, "inventory");
  assertExactKeys(inventory, ["schemaVersion", "engines", "productImages"], "inventory");
  if (inventory.schemaVersion !== "lunafox-engine-image-tool-inventory/v2") {
    throw new Error(`unsupported inventory schema: ${inventory.schemaVersion}`);
  }
  if (!Array.isArray(inventory.engines) || inventory.engines.length === 0) {
    throw new Error("inventory.engines must be non-empty");
  }
  if (!Array.isArray(inventory.productImages) || inventory.productImages.length === 0) {
    throw new Error("inventory.productImages must be non-empty");
  }
  const engineDirectories = new Set();
  const engineIds = new Set();
  for (const [engineIndex, engine] of inventory.engines.entries()) {
    const label = `engines[${engineIndex}]`;
    assertObject(engine, label);
    assertExactKeys(engine, ["engineId", "directory", "engineBinary", "tools"], label);
    assertCanonicalText(engine.engineId, `${label}.engineId`, /^engine\.lunafox\.[a-z0-9_]+$/);
    assertCanonicalText(engine.directory, `${label}.directory`, /^[a-z0-9_]+$/);
    assertCanonicalText(engine.engineBinary, `${label}.engineBinary`, /^[a-z0-9-]+$/);
    if (engineDirectories.has(engine.directory)) throw new Error(`duplicate engine directory: ${engine.directory}`);
    if (engineIds.has(engine.engineId)) throw new Error(`duplicate engine ID: ${engine.engineId}`);
    engineDirectories.add(engine.directory);
    engineIds.add(engine.engineId);
    if (!Array.isArray(engine.tools) || engine.tools.length === 0) {
      throw new Error(`${label}.tools must be non-empty`);
    }
    const engineToolNames = new Set();
    const engineVersionArgs = new Set();
    for (const [toolIndex, tool] of engine.tools.entries()) {
      const toolLabel = `${label}.tools[${toolIndex}]`;
      assertObject(tool, toolLabel);
      assertExactKeys(tool, ["name", "versionArg", "version"], toolLabel);
      assertCanonicalText(tool.name, `${toolLabel}.name`, /^[a-z0-9][a-z0-9-]*$/);
      assertCanonicalText(tool.versionArg, `${toolLabel}.versionArg`, /^[A-Z][A-Z0-9_]*_VERSION$/);
      assertCanonicalText(tool.version, `${toolLabel}.version`, /^v?[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][0-9A-Za-z.-]+)?$/);
      if (engineToolNames.has(tool.name)) throw new Error(`${label} declares tool ${tool.name} more than once`);
      if (engineVersionArgs.has(tool.versionArg)) throw new Error(`${label} declares version arg ${tool.versionArg} more than once`);
      engineToolNames.add(tool.name);
      engineVersionArgs.add(tool.versionArg);
    }
  }

  const productIds = new Set();
  const productPaths = new Set();
  for (const [imageIndex, image] of inventory.productImages.entries()) {
    const label = `productImages[${imageIndex}]`;
    assertObject(image, label);
    assertExactKeys(image, ["id", "dockerfile"], label);
    assertCanonicalText(image.id, `${label}.id`, /^[a-z][a-z0-9-]*$/);
    assertCanonicalText(image.dockerfile, `${label}.dockerfile`, /^[A-Za-z0-9_./-]+Dockerfile$/);
    if (productIds.has(image.id)) throw new Error(`duplicate product image ID: ${image.id}`);
    if (productPaths.has(image.dockerfile)) throw new Error(`duplicate product Dockerfile: ${image.dockerfile}`);
    productIds.add(image.id);
    productPaths.add(image.dockerfile);
  }
  return inventory;
}

function parseDockerfile(source) {
  const instructions = [];
  let logical = "";
  for (const rawLine of source.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!logical && (!line || line.startsWith("#"))) continue;
    const continued = line.endsWith("\\");
    const fragment = (continued ? line.slice(0, -1) : line).trim();
    logical = `${logical} ${fragment}`.trim();
    if (!continued) {
      instructions.push(logical.replace(/\s+/g, " "));
      logical = "";
    }
  }
  if (logical) throw new Error("Dockerfile ends with an unterminated continuation");
  return instructions;
}

function readInstructions(repoRoot, relativePath) {
  const absolutePath = path.join(repoRoot, relativePath);
  if (!fs.existsSync(absolutePath)) throw new Error(`missing Dockerfile: ${relativePath}`);
  return parseDockerfile(fs.readFileSync(absolutePath, "utf8"));
}

function instructionContains(instructions, marker) {
  return instructions.some((instruction) => instruction.includes(marker));
}

function toolSignals(tool) {
  return [
    `/opt/lunafox-tools/bin/${tool.name}`,
    `${tool.versionArg}=`,
    new RegExp(`(^|[^A-Za-z0-9_-])${tool.name.replaceAll("-", "\\-")}([^A-Za-z0-9_-]|$)`, "i"),
  ];
}

function instructionHasTool(instruction, tool) {
  return toolSignals(tool).some((signal) => signal instanceof RegExp ? signal.test(instruction) : instruction.includes(signal));
}

function discoverEngineDirectories(repoRoot) {
  const enginesRoot = path.join(repoRoot, "extensions/engines");
  const directories = [];
  for (const entry of fs.readdirSync(enginesRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const root = path.join(enginesRoot, entry.name);
    const manifestPath = path.join(root, "engine.json");
    const dockerfilePath = path.join(root, "Dockerfile");
    const hasManifest = fs.existsSync(manifestPath);
    const hasDockerfile = fs.existsSync(dockerfilePath);
    if (hasDockerfile && !hasManifest) {
      throw new Error(`Engine directory ${entry.name} has a Dockerfile but no engine.json`);
    }
    if (hasManifest && !hasDockerfile) {
      throw new Error(`Engine directory ${entry.name} has engine.json but no Dockerfile`);
    }
    if (hasManifest && hasDockerfile) directories.push(entry.name);
  }
  return directories.sort();
}

function requireRegularNonSymlink(filePath, label) {
  let info;
  try {
    info = fs.lstatSync(filePath);
  } catch (error) {
    if (error.code === "ENOENT") throw new Error(`missing ${label}: ${filePath}`);
    throw new Error(`inspect ${label} ${filePath}: ${error.message}`);
  }
  if (!info.isFile() || info.isSymbolicLink()) {
    throw new Error(`${label} must be a regular non-symlink file: ${filePath}`);
  }
}

function validateURLCollectionPythonLock(repoRoot, dockerfile, instructions, engine) {
  if (engine.directory !== "url_collection") return;
  const lockPath = path.join(repoRoot, "extensions", "engines", engine.directory, "requirements.lock");
  requireRegularNonSymlink(lockPath, "URL Collection Python dependency lock");
  const entries = fs.readFileSync(lockPath, "utf8")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#"));
  if (entries.length === 0 || entries.some((entry) => !/^[A-Za-z0-9_.-]+==[^\s=]+$/.test(entry))) {
    throw new Error(`${path.relative(repoRoot, lockPath)} must contain only exact Python package pins`);
  }
  if (!instructions.some((instruction) => instruction.includes("COPY url_collection/requirements.lock /tmp/requirements.lock")) || !instructions.some((instruction) => instruction.includes("--requirement /tmp/requirements.lock"))) {
    throw new Error(`${dockerfile} must install URL Collection Python dependencies from requirements.lock`);
  }
}

function conformanceSourcePaths(repoRoot, engine) {
  const root = path.join(repoRoot, "extensions", "engines", engine.directory, "tests", "container");
  let rootInfo;
  try {
    rootInfo = fs.lstatSync(root);
  } catch (error) {
    if (error.code === "ENOENT") throw new Error(`missing container-test directory: ${path.relative(repoRoot, root)}`);
    throw new Error(`inspect container-test directory ${path.relative(repoRoot, root)}: ${error.message}`);
  }
  if (!rootInfo.isDirectory() || rootInfo.isSymbolicLink()) {
    throw new Error(`container-test directory must be a non-symlink directory: ${path.relative(repoRoot, root)}`);
  }

  const scriptPath = path.join(root, "container-conformance.sh");
  const profilePath = path.join(root, "image-conformance.json");
  requireRegularNonSymlink(scriptPath, "image conformance script");
  requireRegularNonSymlink(profilePath, "image conformance profile");
  return { root, scriptPath, profilePath };
}

function assertDockerfileUsesSourceOwnedConformance(instructions, dockerfile, engine) {
  const runtimeBase = instructions.findIndex((instruction) => /^FROM\b.*\sAS runtime-base$/i.test(instruction));
  const verify = instructions.findIndex((instruction) => /^FROM runtime-base AS verify$/i.test(instruction));
  const runtime = instructions.findIndex((instruction) => /^FROM runtime-base AS runtime$/i.test(instruction));
  if (runtimeBase < 0 || verify < 0 || runtime < 0 || !(runtimeBase < verify && verify < runtime)) {
    throw new Error(`${dockerfile} must use ordered runtime-base, verify, and runtime stages`);
  }

  const sourceMount = `--mount=type=bind,source=${engine.directory}/tests/container,target=/run/lunafox-conformance`;
  const markerMount = "--mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed";
  const enginePath = `/opt/lunafox-engine/bin/${engine.engineBinary}`;
  if (!instructions.some((instruction) => instruction.includes(sourceMount) && instruction.includes("/run/lunafox-conformance/container-conformance.sh") && instruction.includes(enginePath) && /:\s*>\s*\/conformance-passed/.test(instruction))) {
    throw new Error(`${dockerfile} must run its tests/container conformance script through the verify-stage bind mount`);
  }
  if (!instructions.some((instruction) => instruction.includes(markerMount) && instruction.includes("test -f /run/.conformance-passed"))) {
    throw new Error(`${dockerfile} must gate the final runtime stage on the verify marker`);
  }
  if (instructions.some((instruction) => instruction.includes("/usr/local/bin/") && /conformance/i.test(instruction)) || instructions.some((instruction) => /^(COPY|ADD)\b/i.test(instruction) && (instruction.includes(`${engine.directory}/tests/container`) || instruction.includes("container-conformance.sh")))) {
    throw new Error(`${dockerfile} must not retain an image-resident conformance command`);
  }
  if (instructions.slice(runtime + 1).some((instruction) => /^COPY\b.*--from=verify\b/i.test(instruction))) {
    throw new Error(`${dockerfile} final runtime stage must not copy persistent data from verify`);
  }
}

function validateNucleiReleasePin(instructions, dockerfile, engine) {
  if (engine.engineId !== "engine.lunafox.nuclei_vulnerability") return;
  const required = [
    'ARG NUCLEI_VERSION=v3.4.10',
    'nuclei_3.4.10_linux_amd64.zip',
    '234c12cc5288af071abdcd6f854245b6067345556e1235cf96b76725c1004357',
    'nuclei_3.4.10_linux_arm64.zip',
    'd1ed1a5c0df49d8fcd64cab4ff5840b793d6bf133b082bb6a7f67d5fc0f9c327',
    'sha256sum -c -',
  ];
  for (const marker of required) {
    if (!instructionContains(instructions, marker)) {
      throw new Error(`${dockerfile} is missing the exact Nuclei release pin ${marker}`);
    }
  }
  if (instructions.some((instruction) => instruction.includes('go install "github.com/projectdiscovery/nuclei/'))) {
    throw new Error(`${dockerfile} must not install Nuclei from an unverified Go module`);
  }
}

function validateEngineSources(repoRoot, inventory) {
  const inventoryDirectories = inventory.engines.map((engine) => engine.directory).sort();
  const discoveredDirectories = discoverEngineDirectories(repoRoot);
  if (JSON.stringify(inventoryDirectories) !== JSON.stringify(discoveredDirectories)) {
    throw new Error(`engine Dockerfile inventory mismatch: inventory=${inventoryDirectories.join(",")} discovered=${discoveredDirectories.join(",")}`);
  }
  const allTools = inventory.engines.flatMap((engine) => engine.tools);
  const allToolNames = [...new Set(allTools.map((tool) => tool.name))];

  for (const engine of inventory.engines) {
    const dockerfile = `extensions/engines/${engine.directory}/Dockerfile`;
    const instructions = readInstructions(repoRoot, dockerfile);
    const expectedEntrypoint = `ENTRYPOINT [\"/opt/lunafox-engine/bin/${engine.engineBinary}\"]`;
    if (!instructions.includes(expectedEntrypoint)) {
      throw new Error(`${dockerfile} must define ${expectedEntrypoint}`);
    }
    assertDockerfileUsesSourceOwnedConformance(instructions, dockerfile, engine);
    validateNucleiReleasePin(instructions, dockerfile, engine);
    validateURLCollectionPythonLock(repoRoot, dockerfile, instructions, engine);

    const assigned = new Map(engine.tools.map((tool) => [tool.name, tool]));
    for (const name of allToolNames) {
      const tool = assigned.get(name) ?? allTools.find((candidate) => candidate.name === name);
      const matching = instructions.filter((instruction) => instructionHasTool(instruction, tool));
      if (assigned.has(name)) {
        const versionArg = `ARG ${tool.versionArg}=${tool.version}`;
        if (!instructions.includes(versionArg)) {
          throw new Error(`${dockerfile} must pin ${versionArg}`);
        }
        if (!instructionContains(instructions, `/opt/lunafox-tools/bin/${tool.name}`)) {
          throw new Error(`${dockerfile} must install ${tool.name} under /opt/lunafox-tools/bin`);
        }
        if (matching.length === 0) throw new Error(`${dockerfile} has no build instruction for ${tool.name}`);
      } else if (matching.length > 0) {
        throw new Error(`${dockerfile} contains unassigned engine tool ${tool.name}`);
      }
    }

    const { scriptPath } = conformanceSourcePaths(repoRoot, engine);

    const script = fs.readFileSync(scriptPath, "utf8");
    for (const tool of engine.tools) {
      if (!script.includes(tool.name) || !script.includes(tool.version)) {
        throw new Error(`${path.relative(repoRoot, scriptPath)} must verify ${tool.name} ${tool.version}`);
      }
    }
  }
}

function validateProductImageSources(repoRoot, inventory) {
  const allTools = inventory.engines.flatMap((engine) => engine.tools);
  for (const image of inventory.productImages) {
    const instructions = readInstructions(repoRoot, image.dockerfile);
    for (const tool of allTools) {
      const offender = instructions.find((instruction) => instructionHasTool(instruction, tool));
      if (offender) {
        throw new Error(`${image.dockerfile} contains forbidden builtin engine tool ${tool.name}: ${offender}`);
      }
    }
  }
}

function validateRetiredWorkerExclusion(repoRoot) {
  if (fs.existsSync(path.join(repoRoot, "worker"))) {
    throw new Error("retired worker/ module must be absent");
  }

  const workflowRoot = path.join(repoRoot, ".github/workflows");
  const forbiddenWorkflowMarkers = [
    "worker/Dockerfile",
    "working-directory: worker",
    "go-version-file: worker/go.mod",
    "image: lunafox-worker",
    "run_worker",
    "worker-tests",
  ];
  if (fs.existsSync(workflowRoot)) {
    for (const entry of fs.readdirSync(workflowRoot, { withFileTypes: true })) {
      if (!entry.isFile() || !/\.ya?ml$/.test(entry.name)) continue;
      const workflowPath = path.join(workflowRoot, entry.name);
      const source = fs.readFileSync(workflowPath, "utf8");
      for (const marker of forbiddenWorkflowMarkers) {
        if (source.includes(marker)) {
          throw new Error(`${path.relative(repoRoot, workflowPath)} contains retired Worker build/test marker ${marker}`);
        }
      }
    }
  }
}

function shellQuote(value) {
  return `'${value.replaceAll("'", `'"'"'`)}'`;
}

function runContainerCheck(dockerCommand, imageRef, script, label) {
  const result = spawnSync(dockerCommand, ["run", "--rm", "--entrypoint", "/bin/sh", imageRef, "-ec", script], {
    encoding: "utf8",
  });
  if (result.error) throw new Error(`${label}: run Docker: ${result.error.message}`);
  if (result.status !== 0) {
    const detail = [result.stdout, result.stderr].filter(Boolean).join("\n").trim();
    throw new Error(`${label}: runtime image check failed${detail ? `: ${detail}` : ""}`);
  }
}

function dockerFailure(label, action, result) {
  if (result.error) return new Error(`${label}: ${action} Docker: ${result.error.message}`);
  const detail = [result.stdout, result.stderr].filter(Boolean).join("\n").trim();
  return new Error(`${label}: ${action} failed${detail ? `: ${detail}` : ""}`);
}

function runCopiedConformanceCheck(dockerCommand, imageRef, sourceScriptPath, engineBinary, label) {
  const containerName = `lunafox-engine-image-conformance-${process.pid}-${randomUUID()}`;
  const containerScriptPath = "/tmp/container-conformance.sh";
  let created = false;
  let executionError;

  try {
    let result = spawnSync(dockerCommand, [
      "create",
      "--name",
      containerName,
      "--network",
      "none",
      "--entrypoint",
      "/bin/sh",
      imageRef,
      containerScriptPath,
      `/opt/lunafox-engine/bin/${engineBinary}`,
    ], { encoding: "utf8" });
    if (result.error || result.status !== 0) throw dockerFailure(label, "create conformance container", result);
    created = true;

    result = spawnSync(dockerCommand, ["cp", sourceScriptPath, `${containerName}:${containerScriptPath}`], {
      encoding: "utf8",
    });
    if (result.error || result.status !== 0) throw dockerFailure(label, "copy conformance script", result);

    result = spawnSync(dockerCommand, ["start", "-a", containerName], { encoding: "utf8" });
    if (result.error || result.status !== 0) throw dockerFailure(label, "runtime image check", result);
  } catch (error) {
    executionError = error;
  }

  let cleanupError;
  if (created) {
    const result = spawnSync(dockerCommand, ["rm", "--force", containerName], { encoding: "utf8" });
    if (result.error || result.status !== 0) cleanupError = dockerFailure(label, "remove conformance container", result);
  }
  if (executionError && cleanupError) {
    throw new Error(`${executionError.message}; additionally ${cleanupError.message}`);
  }
  if (executionError) throw executionError;
  if (cleanupError) throw cleanupError;
}

function assertNoRetainedConformanceScript(dockerCommand, imageRef, label) {
  const script = "set -eu; if find / -xdev -name container-conformance.sh -print -quit | grep -q .; then echo 'retained container conformance script found in Runtime Image' >&2; exit 1; fi";
  runContainerCheck(dockerCommand, imageRef, script, label);
}

function validateRuntimeImages(args, inventory) {
  const productIds = new Set(inventory.productImages.map((image) => image.id));
  const engineDirectories = new Set(inventory.engines.map((engine) => engine.directory));
  for (const id of args.productImages.keys()) {
    if (!productIds.has(id)) throw new Error(`unknown product image ID: ${id}`);
  }
  for (const directory of args.engineImages.keys()) {
    if (!engineDirectories.has(directory)) throw new Error(`unknown engine image directory: ${directory}`);
  }
  if (args.requireRuntimeImages) {
    for (const id of productIds) if (!args.productImages.has(id)) throw new Error(`missing runtime product image ref: ${id}`);
    for (const directory of engineDirectories) if (!args.engineImages.has(directory)) throw new Error(`missing runtime engine image ref: ${directory}`);
  }

  const allTools = inventory.engines.flatMap((engine) => engine.tools);
  for (const image of inventory.productImages) {
    const ref = args.productImages.get(image.id);
    if (!ref) continue;
    const script = [
      "set -eu",
      ...allTools.map((tool) => `if command -v ${shellQuote(tool.name)} >/dev/null 2>&1; then echo ${shellQuote(`forbidden engine tool found: ${tool.name}`)} >&2; exit 1; fi`),
    ].join("\n");
    runContainerCheck(args.dockerCommand, ref, script, `${image.id} image`);
  }

  for (const engine of inventory.engines) {
    const ref = args.engineImages.get(engine.directory);
    if (!ref) continue;
    const { scriptPath: sourceScriptPath } = conformanceSourcePaths(args.repoRoot, engine);
    runCopiedConformanceCheck(args.dockerCommand, ref, sourceScriptPath, engine.engineBinary, `${engine.directory} image`);
    assertNoRetainedConformanceScript(args.dockerCommand, ref, `${engine.directory} image conformance payload absence`);
  }
}

function main() {
  try {
    const args = parseArgs(process.argv.slice(2));
    const inventory = loadInventory(args.inventoryPath);
    validateEngineSources(args.repoRoot, inventory);
    // Public Engine release projections intentionally exclude private Agent source.
    // Engine-only payload checks still enforce Engine source and retired-Worker boundaries.
    if (!args.skipProductImageSourceCheck) validateProductImageSources(args.repoRoot, inventory);
    validateRetiredWorkerExclusion(args.repoRoot);
    validateRuntimeImages(args, inventory);
    const sourceScope = args.skipProductImageSourceCheck
      ? "Engine-only runtime payload"
      : `${inventory.productImages.length} product image exclusions`;
    process.stdout.write(`OK: ${inventory.engines.length} engine image inventories and ${sourceScope} verified\n`);
  } catch (error) {
    process.stderr.write(`engine image tool inventory verification failed: ${error.message}\n`);
    process.exit(1);
  }
}

main();
