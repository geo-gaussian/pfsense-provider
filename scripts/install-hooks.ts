import * as fs from "node:fs";
import * as path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
execFileSync("git", ["config", "core.hooksPath", ".githooks"], {
	cwd: root,
	stdio: "inherit",
});

const hook = path.join(root, ".githooks", "pre-commit");
if (fs.existsSync(hook)) {
	fs.chmodSync(hook, 0o755);
}

console.log("Git hooks installed at .githooks.");
