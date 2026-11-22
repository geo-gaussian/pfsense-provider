import * as assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import * as path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const prettier = require.resolve("prettier/bin/prettier.cjs");
const root = fileURLToPath(new URL("..", import.meta.url));

function runPrettier(filename: string, input: string, check = true) {
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
	["fixture.ts", "if (true) {\n\tfoo();\n}\n", "if (true) {\n  foo();\n}\n"],
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

test("whitespace checker includes TypeScript files", (t) => {
	const dir = fs.mkdtempSync(path.join(tmpdir(), "pfsense-format-"));
	t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
	fs.mkdirSync(path.join(dir, "scripts"));
	const checker = path.join(dir, "scripts", "check-editorconfig.ts");
	fs.copyFileSync(
		fileURLToPath(new URL("./check-editorconfig.ts", import.meta.url)),
		checker,
	);
	fs.writeFileSync(path.join(dir, "package.json"), '{"type":"module"}\n');
	const fixture = path.join(dir, "fixture.ts");
	fs.writeFileSync(fixture, "\tconst value = 1;\n");
	const check = () =>
		spawnSync(process.execPath, [checker], { encoding: "utf8" });
	assert.equal(check().status, 0);
	fs.writeFileSync(fixture, "const value = 1; \n");
	const invalid = check();
	assert.equal(invalid.status, 1);
	assert.match(invalid.stderr, /fixture\.ts:1: trailing whitespace/);
});

for (const [filename, spaced] of [
	[".specify/integrations/fixture.json", '{\n  "value": 1\n}\n'],
	[".specify/scripts/bash/fixture.sh", "if true; then\n  echo ok\nfi\n"],
]) {
	test(`${filename} is not excluded from formatting`, () => {
		assert.equal(runPrettier(filename, spaced).status, 1);
	});
}
