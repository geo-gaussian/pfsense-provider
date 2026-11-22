const assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const path = require("node:path");
const test = require("node:test");

const prettier = require.resolve("prettier/bin/prettier.cjs");
const root = path.resolve(__dirname, "..");

function runPrettier(filename, input, check = true) {
	const args = [prettier, "--stdin-filepath", filename];
	if (check) args.push("--check");
	const result = spawnSync(process.execPath, args, {
		cwd: root,
		encoding: "utf8",
		input,
	});
	if (result.error) throw result.error;
	return result;
}

for (const [filename, tabbed, spaced] of [
	["fixture.js", "if (true) {\n\tfoo();\n}\n", "if (true) {\n  foo();\n}\n"],
	["fixture.json", '{\n\t"value": 1\n}\n', '{\n  "value": 1\n}\n'],
	[
		"fixture.sh",
		"if true; then\n\techo ok\nfi\n",
		"if true; then\n  echo ok\nfi\n",
	],
]) {
	test(`${filename} requires tabs for indentation`, () => {
		assert.equal(runPrettier(filename, tabbed).status, 0);
		assert.equal(runPrettier(filename, spaced).status, 1);
	});
}

test("shell formatting preserves here-document content", () => {
	const result = runPrettier(
		"fixture.sh",
		"if true; then\n\tcat <<EOF\n  keep spaces\nEOF\nfi\n",
		false,
	);
	assert.equal(result.status, 0);
	assert.match(result.stdout, /\n  keep spaces\n/);
});
