const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const ignored = new Set([".git", "node_modules", "vendor"]);
const checkExtensions = new Set([
  ".json",
  ".md",
  ".sh",
  ".tf",
  ".yaml",
  ".yml",
]);
const issues = [];

function walk(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    if (ignored.has(entry.name)) {
      continue;
    }

    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      walk(absolute);
      continue;
    }

    const extension = path.extname(entry.name);
    if (!checkExtensions.has(extension) && entry.name !== ".editorconfig" &&
      entry.name !== ".gitignore" && entry.name !== "Makefile") {
      continue;
    }

    const relative = path.relative(root, absolute).split(path.sep).join("/");
    const text = fs.readFileSync(absolute, "utf8");
    const lines = text.split("\n");

    if (!text.endsWith("\n")) {
      issues.push(`${relative}: missing final newline`);
    }
    lines.forEach((line, index) => {
      if (line.startsWith("\t") && entry.name !== "Makefile") {
        issues.push(`${relative}:${index + 1}: tab indentation`);
      }
      if (/[ \t]+$/.test(line)) {
        issues.push(`${relative}:${index + 1}: trailing whitespace`);
      }
    });
  }
}

walk(root);

if (issues.length > 0) {
  console.error(issues.join("\n"));
  process.exit(1);
}

console.log("EditorConfig formatting checks passed.");
