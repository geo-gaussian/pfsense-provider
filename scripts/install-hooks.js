const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

const root = path.resolve(__dirname, "..");
execFileSync("git", ["config", "core.hooksPath", ".githooks"], {
  cwd: root,
  stdio: "inherit",
});

const hook = path.join(root, ".githooks", "pre-commit");
if (fs.existsSync(hook)) {
  fs.chmodSync(hook, 0o755);
}

console.log("Git hooks installed at .githooks.");
