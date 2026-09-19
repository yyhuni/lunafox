#!/usr/bin/env node

import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import {
  MAX_NOTES_BYTES,
  expectedNotesPath,
  validateReleaseNotes,
} from "./validate-public-release-notes.mjs";

function fixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-release-notes-") );
  const notesDir = path.join(root, "release-notes");
  fs.mkdirSync(notesDir, { recursive: true });
  return { root, notesDir };
}

test("accepts a public Markdown file and returns byte evidence", () => {
  const { root, notesDir } = fixture();
  try {
    const tag = "v1.2.3-alpha.1";
    const body = "# v1.2.3\n\n## English\n\n- Faster scans.\n\n## 简体中文\n\n- 扫描速度更快。\n\nFull Changelog: https://github.com/yyhuni/lunafox/compare/v1.2.2...v1.2.3\n";
    fs.writeFileSync(path.join(notesDir, `${tag}.md`), body);
    const result = validateReleaseNotes({ rootDir: root, tag });
    assert.deepEqual(result, {
      schemaVersion: 1,
      passed: true,
      releaseTag: tag,
      notesPath: expectedNotesPath(tag),
      bytes: Buffer.byteLength(body),
      sha256: crypto.createHash("sha256").update(body).digest("hex"),
      bilingualSections: {
        English: { present: true, nonEmpty: true },
        "简体中文": { present: true, nonEmpty: true },
      },
    });
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("rejects missing and mismatched files", () => {
  const { root, notesDir } = fixture();
  try {
    assert.throws(() => validateReleaseNotes({ rootDir: root, tag: "v1.2.3" }), /missing release notes/);
    fs.writeFileSync(path.join(notesDir, "v1.2.2.md"), "# old\n");
    assert.throws(() => validateReleaseNotes({ rootDir: root, tag: "v1.2.3", notesFile: "release-notes/v1.2.2.md" }), /path must be/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("rejects empty, oversized, invalid UTF-8, and internal markers", () => {
  const cases = [
    ["empty", Buffer.from("   \n")],
    ["oversized", Buffer.alloc(MAX_NOTES_BYTES + 1, 0x61)],
    ["invalid UTF-8", Buffer.from([0xc3, 0x28])],
    ["private marker", Buffer.from("See yyhuni/lunafox-private for details\n")],
    ["export marker", Buffer.from("chore(export): generated deployment projection\n")],
  ];
  for (const [label, content] of cases) {
    const { root, notesDir } = fixture();
    try {
      fs.writeFileSync(path.join(notesDir, "v1.2.3.md"), content);
      assert.throws(() => validateReleaseNotes({ rootDir: root, tag: "v1.2.3" }), new RegExp(label === "invalid UTF-8" ? "UTF-8" : label === "private marker" ? "blocked internal marker" : label === "export marker" ? "blocked internal marker" : label === "oversized" ? "exceeds" : "must not be empty", "i"));
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }
});

test("rejects missing, empty, and duplicate language sections", () => {
  const cases = [
    ["missing English", "# v1.2.3\n\n## 简体中文\n\n- 修复。\n", /missing required ## English/],
    ["missing Chinese", "# v1.2.3\n\n## English\n\n- Fix.\n", /missing required ## 简体中文/],
    ["empty Chinese", "# v1.2.3\n\n## English\n\n- Fix.\n\n## 简体中文\n\n", /empty ## 简体中文/],
    ["duplicate English", "# v1.2.3\n\n## English\n\n- Fix.\n\n## English\n\n- Again.\n\n## 简体中文\n\n- 修复。\n", /duplicate ## English/],
  ];
  for (const [label, body, expected] of cases) {
    const { root, notesDir } = fixture();
    try {
      fs.writeFileSync(path.join(notesDir, "v1.2.3.md"), body);
      assert.throws(() => validateReleaseNotes({ rootDir: root, tag: "v1.2.3" }), expected, label);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  }
});

test("rejects symlinks and invalid tags", () => {
  const { root, notesDir } = fixture();
  try {
    fs.writeFileSync(path.join(root, "safe.md"), "# safe\n");
    fs.symlinkSync("../safe.md", path.join(notesDir, "v1.2.3.md"));
    assert.throws(() => validateReleaseNotes({ rootDir: root, tag: "v1.2.3" }), /regular file/);
    assert.throws(() => validateReleaseNotes({ rootDir: root, tag: "release/latest" }), /invalid release tag/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
