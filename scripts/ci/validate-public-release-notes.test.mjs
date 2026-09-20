#!/usr/bin/env node

import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import test from "node:test";

import {
  FORBIDDEN_EVIDENCE_FIELDS,
  MAX_NOTES_BYTES,
  expectedNotesPath,
  validateCandidateEvidence,
  validateReleaseNotes,
} from "./validate-public-release-notes.mjs";

function fixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-release-notes-"));
  const notesDir = path.join(root, "release-notes");
  fs.mkdirSync(notesDir, { recursive: true });
  return { root, notesDir };
}

function sha256(value) {
  return crypto.createHash("sha256").update(value).digest("hex");
}

function candidateEvidence({
  tag,
  baselineCommit = "a".repeat(40),
  targetRevision = "b".repeat(40),
  notes,
  sourceId = "pr:1",
  category = "Fixed",
} = {}) {
  const factsDigest = "c".repeat(64);
  const auditDigest = "d".repeat(64);
  return {
    schemaVersion: 2,
    status: "validated",
    identity: { tag, baselineCommit, targetRevision },
    baselineRelease: "v1.2.2",
    factsDigest,
    auditDigest,
    agentAudit: {
      schemaVersion: 1,
      mode: "release-executor",
      factsDigest,
      digest: auditDigest,
    },
    sourceIds: [sourceId],
    commitSourceIds: [`commit:${targetRevision}`],
    commitInventory: [{ sourceId: `commit:${targetRevision}`, sha: targetRevision, recordSourceIds: [sourceId] }],
    coverage: {
      included: [{ sourceId, commitShas: [targetRevision], category }],
      excluded: [],
      unmapped: [],
      failed: [],
    },
    entries: [{ sourceIds: [sourceId], category, claims: { english: [], chinese: [] } }],
    notesDigest: sha256(notes),
  };
}

function writeCandidate(root, tag, notes, evidence) {
  fs.mkdirSync(path.join(root, "release-notes"), { recursive: true });
  fs.mkdirSync(path.join(root, "release-note-evidence"), { recursive: true });
  fs.writeFileSync(path.join(root, "release-notes", `${tag}.md`), notes);
  fs.writeFileSync(path.join(root, "release-note-evidence", `${tag}.json`), `${JSON.stringify(evidence)}\n`);
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
      sha256: sha256(body),
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

test("validates agent-audit evidence, provenance, digest bindings, and source coverage", () => {
  const { root } = fixture();
  try {
    const tag = "v1.2.3";
    const baselineCommit = "a".repeat(40);
    const targetRevision = "b".repeat(40);
    const notes = "# v1.2.3\n\n## English\n\n- A fix.\n\n## 简体中文\n\n- 一个修复。\n";
    const evidence = candidateEvidence({ tag, baselineCommit, targetRevision, notes });
    writeCandidate(root, tag, notes, evidence);

    const result = validateCandidateEvidence({
      rootDir: root,
      tag,
      notesFile: `release-notes/${tag}.md`,
      evidenceFile: `release-note-evidence/${tag}.json`,
      expectedTargetRevision: targetRevision,
      expectedBaselineCommit: baselineCommit,
    });
    assert.equal(result.passed, true);
    assert.equal(result.auditDigest, evidence.auditDigest);
    assert.throws(() => validateCandidateEvidence({
      rootDir: root,
      tag,
      expectedTargetRevision: "d".repeat(40),
    }), /target revision/);

    evidence.coverage.excluded.push({ sourceId: "pr:1", reason: "duplicate" });
    writeCandidate(root, tag, notes, evidence);
    assert.throws(() => validateCandidateEvidence({ rootDir: root, tag }), /duplicate source IDs/);

    evidence.coverage.excluded = [];
    evidence.provider = { mode: "legacy" };
    writeCandidate(root, tag, notes, evidence);
    assert.throws(() => validateCandidateEvidence({ rootDir: root, tag }), /must not persist evidence\.provider/);

    delete evidence.provider;
    evidence.entries[0].claims.chinese = ["2"];
    writeCandidate(root, tag, notes, evidence);
    assert.throws(() => validateCandidateEvidence({ rootDir: root, tag }), /claim tokens differ/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("rejects unsupported agent-audit fields", () => {
  const { root } = fixture();
  try {
    const tag = "v1.2.5";
    const notes = "# v1.2.5\n\n## English\n\n- A fix.\n\n## 简体中文\n\n- 一个修复。\n";
    const evidence = candidateEvidence({ tag, notes });
    evidence.agentAudit.model = "not-permitted";
    writeCandidate(root, tag, notes, evidence);
    assert.throws(() => validateCandidateEvidence({ rootDir: root, tag }), /unsupported field/);
    assert.deepEqual(FORBIDDEN_EVIDENCE_FIELDS.includes("provider"), true);

    delete evidence.agentAudit.model;
    evidence.coverage.included[0].auditInput = { prompt: "do not persist" };
    writeCandidate(root, tag, notes, evidence);
    assert.throws(() => validateCandidateEvidence({ rootDir: root, tag }), /must not persist evidence\.coverage\.included\[0\]\.auditInput/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("requires target ancestry for both candidate PR heads and Tag trees", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "lunafox-release-notes-ancestry-"));
  const git = (args) => execFileSync("git", ["-C", root, ...args], { encoding: "utf8" }).trim();
  try {
    git(["init", "-q", "--initial-branch=main"]);
    git(["config", "user.name", "Fixture"]);
    git(["config", "user.email", "fixture@example.com"]);
    fs.writeFileSync(path.join(root, "README.md"), "source\n");
    git(["add", "README.md"]);
    git(["commit", "-q", "-m", "source"]);
    const targetRevision = git(["rev-parse", "HEAD"]);
    const tag = "v1.2.4";
    const notes = `# ${tag}\n\n## English\n\n- A fix.\n\n## 简体中文\n\n- 一个修复。\n`;
    const evidence = candidateEvidence({ tag, targetRevision, notes });
    writeCandidate(root, tag, notes, evidence);
    git(["add", "release-notes", "release-note-evidence"]);
    git(["commit", "-q", "-m", "candidate"]);
    const candidateCommit = git(["rev-parse", "HEAD"]);

    assert.equal(validateCandidateEvidence({
      rootDir: root,
      tag,
      expectedCandidateCommit: candidateCommit,
      expectedTagCommit: candidateCommit,
    }).passed, true);

    git(["checkout", "-q", "--orphan", "unrelated"]);
    git(["add", "-A"]);
    git(["commit", "-q", "-m", "unrelated"]);
    const unrelatedCommit = git(["rev-parse", "HEAD"]);
    assert.throws(() => validateCandidateEvidence({ rootDir: root, tag, expectedCandidateCommit: unrelatedCommit }), /ancestor/);
    assert.throws(() => validateCandidateEvidence({ rootDir: root, tag, expectedTagCommit: unrelatedCommit }), /ancestor/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
