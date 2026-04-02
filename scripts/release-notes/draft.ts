#!/usr/bin/env bun
/**
 * Release Notes Drafter
 *
 * Parses git log between version tags, groups commits by conventional-commit
 * category, extracts Linear ticket IDs and PR numbers, and outputs Markdown.
 *
 * Usage:
 *   bun run scripts/release-notes/draft.ts [from-tag] [to-tag]
 *   bun run scripts/release-notes/draft.ts              # latest two tags
 *   bun run scripts/release-notes/draft.ts v0.3.3 v0.3.4
 */

import { execSync } from "child_process";

// ── Types ───────────────────────────────────────────────────────────────────

interface Commit {
  hash: string;
  subject: string;
  type: string;
  scope: string | null;
  description: string;
  prNumber: string | null;
  linearIds: string[];
}

type Category = {
  title: string;
  emoji: string;
  types: string[];
};

// ── Config ──────────────────────────────────────────────────────────────────

const CATEGORIES: Category[] = [
  { title: "Features", emoji: "🚀", types: ["feat"] },
  { title: "Bug Fixes", emoji: "🐛", types: ["fix"] },
  { title: "Documentation", emoji: "📚", types: ["docs"] },
  { title: "Maintenance", emoji: "🔧", types: ["chore", "refactor", "build", "ci", "style", "perf", "test"] },
];

const CONVENTIONAL_RE =
  /^(?<type>[a-z]+)(?:\((?<scope>[^)]+)\))?(?:!)?:\s*(?<desc>.+)$/i;

const PR_RE = /\(#(?<pr>\d+)\)\s*$/;
const LINEAR_RE = /\b([A-Z]{2,10}-\d+)\b/g;

// ── Helpers ─────────────────────────────────────────────────────────────────

function git(cmd: string): string {
  return execSync(`git ${cmd}`, { encoding: "utf-8" }).trim();
}

function getTags(): string[] {
  try {
    const raw = git("tag --sort=-version:refname");
    return raw.split("\n").filter(Boolean);
  } catch {
    return [];
  }
}

function getCommits(from: string, to: string): Commit[] {
  const SEP = "---COMMIT---";
  const raw = git(
    `log ${from}..${to} --pretty=format:"%H %s${SEP}" --no-merges`
  );
  if (!raw) return [];

  return raw
    .split(SEP)
    .map((entry) => entry.trim())
    .filter(Boolean)
    .map(parseCommit);
}

function parseCommit(line: string): Commit {
  const hash = line.slice(0, 40);
  const subject = line.slice(41);

  const prMatch = PR_RE.exec(subject);
  const prNumber = prMatch?.groups?.pr ?? null;

  // Strip PR suffix for cleaner display
  const cleanSubject = subject.replace(PR_RE, "").trim();

  const convMatch = CONVENTIONAL_RE.exec(cleanSubject);

  const linearIds: string[] = [];
  let m: RegExpExecArray | null;
  const linearRe = new RegExp(LINEAR_RE.source, LINEAR_RE.flags);
  while ((m = linearRe.exec(subject)) !== null) {
    linearIds.push(m[1]);
  }

  if (convMatch?.groups) {
    return {
      hash,
      subject,
      type: convMatch.groups.type.toLowerCase(),
      scope: convMatch.groups.scope ?? null,
      description: convMatch.groups.desc,
      prNumber,
      linearIds,
    };
  }

  return {
    hash,
    subject,
    type: "other",
    scope: null,
    description: cleanSubject,
    prNumber,
    linearIds,
  };
}

function formatCommit(c: Commit): string {
  let line = `- ${c.description}`;
  if (c.scope) line = `- **${c.scope}:** ${c.description}`;
  if (c.prNumber) line += ` ([#${c.prNumber}](../../pulls/${c.prNumber}))`;
  if (c.linearIds.length) line += ` [${c.linearIds.join(", ")}]`;
  line += ` (\`${c.hash.slice(0, 7)}\`)`;
  return line;
}

// ── Main ────────────────────────────────────────────────────────────────────

function main() {
  const args = process.argv.slice(2);
  let fromTag: string;
  let toTag: string;

  if (args.length >= 2) {
    fromTag = args[0];
    toTag = args[1];
  } else {
    const tags = getTags();
    if (tags.length < 2) {
      console.error("Error: need at least two version tags.");
      process.exit(1);
    }
    toTag = tags[0];
    fromTag = tags[1];
  }

  const commits = getCommits(fromTag, toTag);
  if (commits.length === 0) {
    console.error(`No commits found between ${fromTag} and ${toTag}.`);
    process.exit(1);
  }

  const tagDate = (() => {
    try {
      return git(`log -1 --format=%cs ${toTag}`);
    } catch {
      return new Date().toISOString().slice(0, 10);
    }
  })();

  // Group commits into categories
  const grouped = new Map<string, Commit[]>();
  const uncategorized: Commit[] = [];

  for (const commit of commits) {
    let placed = false;
    for (const cat of CATEGORIES) {
      if (cat.types.includes(commit.type)) {
        const existing = grouped.get(cat.title) ?? [];
        existing.push(commit);
        grouped.set(cat.title, existing);
        placed = true;
        break;
      }
    }
    if (!placed) {
      uncategorized.push(commit);
    }
  }

  // Build markdown
  const lines: string[] = [];
  lines.push(`# Release Notes — ${toTag}`);
  lines.push("");
  lines.push(`**Date:** ${tagDate}  `);
  lines.push(`**Tag range:** \`${fromTag}..${toTag}\`  `);
  lines.push(`**Commits:** ${commits.length}`);
  lines.push("");

  for (const cat of CATEGORIES) {
    const catCommits = grouped.get(cat.title);
    if (!catCommits?.length) continue;
    lines.push(`## ${cat.emoji} ${cat.title}`);
    lines.push("");
    for (const c of catCommits) {
      lines.push(formatCommit(c));
    }
    lines.push("");
  }

  if (uncategorized.length) {
    lines.push("## Other Changes");
    lines.push("");
    for (const c of uncategorized) {
      lines.push(formatCommit(c));
    }
    lines.push("");
  }

  // Stats
  const typeCount = new Map<string, number>();
  for (const c of commits) {
    typeCount.set(c.type, (typeCount.get(c.type) ?? 0) + 1);
  }
  lines.push("---");
  lines.push("");
  lines.push(
    `*Generated by release-notes drafter — ${commits.length} commits across ${typeCount.size} categories.*`
  );
  lines.push("");

  const output = lines.join("\n");
  console.log(output);
  return output;
}

const output = main();

// If --save flag or output file arg provided, write to file
const saveIdx = process.argv.indexOf("--save");
if (saveIdx !== -1) {
  const toTag = process.argv.length >= 4 ? process.argv[3] : getTags()[0];
  const outPath = process.argv[saveIdx + 1] ?? `docs/release-notes/${toTag}.md`;
  await Bun.write(outPath, output);
  console.error(`\nSaved to ${outPath}`);
}
