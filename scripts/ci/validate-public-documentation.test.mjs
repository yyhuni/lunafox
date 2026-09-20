#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import { validatePublicDocumentation } from "./validate-public-documentation.mjs";

const GENERATED_MARKER = "> **GENERATED / READ-ONLY**\n\n";
const ENGLISH = `# Public\n\n${GENERATED_MARKER}[Chinese](README.zh-CN.md)\n\n## Install\n\n\`\`\`console\ngit clone https://github.com/yyhuni/lunafox.git\n\`\`\`\n\nUse \`PUBLIC_HOST\`.\n`;
const CHINESE = `# 公开部署\n\n${GENERATED_MARKER}[English](README.md)\n\n## 安装\n\n\`\`\`console\ngit clone https://github.com/yyhuni/lunafox.git\n\`\`\`\n\n使用 \`PUBLIC_HOST\`。\n`;

function fixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-public-docs-"));
  fs.mkdirSync(path.join(root, "docs"), { recursive: true });
  return root;
}

function writePair(root, english = ENGLISH, chinese = CHINESE) {
  fs.writeFileSync(path.join(root, "README.md"), english);
  fs.writeFileSync(path.join(root, "README.zh-CN.md"), chinese);
  fs.writeFileSync(path.join(root, "docs/public-deployment.md"), english.replaceAll("README.zh-CN.md", "public-deployment.zh-CN.md").replaceAll("README.md", "public-deployment.md"));
  fs.writeFileSync(path.join(root, "docs/public-deployment.zh-CN.md"), chinese.replaceAll("README.md", "public-deployment.md").replaceAll("README.zh-CN.md", "public-deployment.zh-CN.md"));
}

test("accepts paired documents and returns structural evidence", () => {
  const root = fixture();
  try {
    writePair(root);
    const result = validatePublicDocumentation({ rootDir: root });
    assert.equal(result.passed, true);
    assert.equal(result.documents.length, 2);
    assert.equal(result.documents[0].codeBlockCount, 1);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("rejects missing and empty paired documents", () => {
  const missingRoot = fixture();
  try {
    writePair(missingRoot);
    fs.rmSync(path.join(missingRoot, "README.zh-CN.md"));
    assert.throws(() => validatePublicDocumentation({ rootDir: missingRoot }), /missing required document: README\.zh-CN\.md/);
  } finally {
    fs.rmSync(missingRoot, { recursive: true, force: true });
  }

  const emptyRoot = fixture();
  try {
    writePair(emptyRoot);
    fs.writeFileSync(path.join(emptyRoot, "docs/public-deployment.zh-CN.md"), "\n");
    assert.throws(() => validatePublicDocumentation({ rootDir: emptyRoot }), /docs\/public-deployment\.zh-CN\.md must not be empty/);
  } finally {
    fs.rmSync(emptyRoot, { recursive: true, force: true });
  }
});

test("rejects a missing generated/read-only projection marker", () => {
  const cases = [
    ["README.md", ENGLISH.replace(GENERATED_MARKER, "")],
    ["README.zh-CN.md", CHINESE.replace(GENERATED_MARKER, "")],
    ["docs/public-deployment.md", ENGLISH.replace(GENERATED_MARKER, "").replaceAll("README.zh-CN.md", "public-deployment.zh-CN.md").replaceAll("README.md", "public-deployment.md")],
    ["docs/public-deployment.zh-CN.md", CHINESE.replace(GENERATED_MARKER, "").replaceAll("README.md", "public-deployment.md").replaceAll("README.zh-CN.md", "public-deployment.zh-CN.md")],
  ];
  for (const [relativePath, content] of cases) {
    const root = fixture();
    try {
      writePair(root);
      fs.writeFileSync(path.join(root, relativePath), content);
      assert.throws(() => validatePublicDocumentation({ rootDir: root }), /must include a GENERATED \/ READ-ONLY marker/);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }
});

test("rejects missing language links and Chinese text", () => {
  const root = fixture();
  try {
    writePair(root, ENGLISH, ENGLISH);
    assert.throws(() => validatePublicDocumentation({ rootDir: root }), /Simplified Chinese text|must link to each other/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("rejects code, token, and link drift", () => {
  const cases = [
    ["code", CHINESE.replace("git clone", "docker compose")],
    ["token", CHINESE.replace("PUBLIC_HOST", "PUBLIC_PORT")],
    ["link", CHINESE.replace("使用 `PUBLIC_HOST`。", "访问 https://example.com/lunafox.git，并使用 `PUBLIC_HOST`。")],
  ];
  for (const [label, chinese] of cases) {
    const root = fixture();
    try {
      writePair(root, ENGLISH, chinese);
      assert.throws(() => validatePublicDocumentation({ rootDir: root }), new RegExp(label === "link" ? "links" : label === "token" ? "technical inline tokens" : "code blocks"));
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }
});
