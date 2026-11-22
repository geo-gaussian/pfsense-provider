const { spawnSync } = require("node:child_process");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const mode = process.argv[2];
if (!["--check", "--write"].includes(mode) || process.argv.length !== 3) {
	console.error("Usage: node scripts/format-python.js --check|--write");
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
