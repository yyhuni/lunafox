import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { execFileSync } from "node:child_process";

import { prepareLegacyDeployment } from "./prepare-legacy-public-deployment.mjs";

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-legacy-bootstrap-test-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const content = path.join(root, "content");
  fs.mkdirSync(path.join(root, "scripts", "ci"), { recursive: true });
  fs.mkdirSync(content);
  const files = {
    ".env": "PUBLIC_HOST=localhost\n",
    ".env.example": "PUBLIC_HOST=localhost\n",
    "compose.yaml": "services: {}\n",
    "engine-inventory.yaml": "enginePackages: []\n",
    "release.manifest.yaml": 'releaseVersion: "1.2.3-alpha.4"\n',
  };
  for (const [name, value] of Object.entries(files)) fs.writeFileSync(path.join(content, name), value);
  const packageName = "lunafox-v1.2.3-alpha.4-dockerhub.zip";
  const packagePath = path.join(root, packageName);
  execFileSync("python3", ["-c", "import pathlib,sys,zipfile\nr=pathlib.Path(sys.argv[1])\nwith zipfile.ZipFile(sys.argv[2],'w') as z:\n [z.write(p,p.relative_to(r).as_posix()) for p in sorted(r.iterdir())]", content, packagePath]);
  const digest = execFileSync("shasum", ["-a", "256", packagePath], { encoding: "utf8" }).split(/\s+/)[0];
  fs.writeFileSync(path.join(root, "scripts", "ci", "public-release-policy.json"), JSON.stringify({
    legacyDeploymentBootstrap: { releaseTag: "v1.2.3-alpha.4", packageName, sha256: digest, paths: Object.keys(files) },
  }));
  return { root, packagePath, output: path.join(root, "output") };
}

test("extracts only the pinned complete deployment group", (t) => {
  const input = fixture(t);
  let generation;
  const result = prepareLegacyDeployment({
    root: input.root,
    package: input.packagePath,
    output: input.output,
    generateDeployment(options) {
      generation = options;
      fs.mkdirSync(options.output, { recursive: true });
      fs.mkdirSync(options.snapshot, { recursive: true });
      const generated = {
        ".env": "RELEASE_REGISTRY=docker.io\nPUBLIC_HOST=localhost\n",
        ".env.example": "RELEASE_REGISTRY=docker.io\nPUBLIC_HOST=localhost\n",
        "compose.yaml": "services:\n  server:\n    image: ${RELEASE_REGISTRY:-docker.io}/yyhuni/lunafox-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n",
        "engine-inventory.yaml": "enginePackages:\n  - refs:\n      - docker.io/yyhuni/lunafox-engine-runtime-test@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n      - ghcr.io/yyhuni/lunafox-engine-runtime-test@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n",
        "release.manifest.yaml": fs.readFileSync(options.manifest),
      };
      for (const [name, value] of Object.entries(generated)) fs.writeFileSync(path.join(options.snapshot, name), value);
      return [{ name: `lunafox-${options.tag}.zip`, sha256: "a".repeat(64) }];
    },
  });
  assert.equal(result.releaseTag, "v1.2.3-alpha.4");
  assert.equal(generation.tag, "v1.2.3-alpha.4");
  assert.deepEqual(fs.readdirSync(input.output).sort(), [".env", ".env.example", "compose.yaml", "engine-inventory.yaml", "release.manifest.yaml"].sort());
  assert.match(fs.readFileSync(path.join(input.output, ".env"), "utf8"), /^RELEASE_REGISTRY=docker\.io$/m);
  assert.match(fs.readFileSync(path.join(input.output, "compose.yaml"), "utf8"), /\$\{RELEASE_REGISTRY:-docker\.io\}/);
  assert.notEqual(fs.readFileSync(path.join(input.output, "compose.yaml"), "utf8"), "services: {}\n");
});

test("rejects immutable digest drift", (t) => {
  const input = fixture(t);
  fs.appendFileSync(input.packagePath, "drift");
  assert.throws(() => prepareLegacyDeployment({ root: input.root, package: input.packagePath, output: input.output }), /digest/);
});
