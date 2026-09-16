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
  const result = prepareLegacyDeployment({ root: input.root, package: input.packagePath, output: input.output });
  assert.equal(result.releaseTag, "v1.2.3-alpha.4");
  assert.deepEqual(fs.readdirSync(input.output).sort(), [".env", ".env.example", "compose.yaml", "engine-inventory.yaml", "release.manifest.yaml"].sort());
});

test("rejects immutable digest drift", (t) => {
  const input = fixture(t);
  fs.appendFileSync(input.packagePath, "drift");
  assert.throws(() => prepareLegacyDeployment({ root: input.root, package: input.packagePath, output: input.output }), /digest/);
});
