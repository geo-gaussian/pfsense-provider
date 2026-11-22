import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const mode = process.argv[2];
if ((mode !== "--check" && mode !== "--write") || process.argv.length !== 3) {
	console.error("Usage: node scripts/format-python.ts --check|--write");
	process.exit(2);
}

const files = spawnSync(
	"git",
	["ls-files", "-co", "--exclude-standard", "-z", "--", "*.py"],
	{ cwd: root, encoding: "utf8" },
);
if (files.error) throw files.error;
if (files.status !== 0) {
	process.stderr.write(files.stderr);
	process.exit(files.status ?? 1);
}

const paths = files.stdout.split("\0").filter(Boolean);
if (paths.length > 0) {
	const result = spawnSync(
		"uvx",
		[
			"--from",
			"yapf==0.43.0",
			"yapf",
			mode === "--check" ? "--diff" : "--in-place",
			...paths,
		],
		{ cwd: root, stdio: "inherit" },
	);
	if (result.error) throw result.error;
	process.exit(result.status ?? 1);
}
