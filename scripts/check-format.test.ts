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
const fixture = (name: string) =>
	fs.readFileSync(path.join(root, "tests", "format", name), "utf8");

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

for (const [filename, fixtureName] of [
	["fixture.js", "fixture.ts"],
	["fixture.ts", "fixture.ts"],
	["fixture.json", "fixture.json"],
	["fixture.sh", "fixture.sh"],
]) {
	test(`${filename} requires tabs for indentation`, () => {
		const tabbed = fixture(fixtureName);
		const spaced = tabbed.replaceAll("\t", "  ");
		assert.equal(runPrettier(filename, tabbed).status, 0);
		assert.equal(runPrettier(filename, spaced).status, 1);
	});
}

test("shell formatting preserves here-document content", () => {
	const result = runPrettier("fixture.sh", fixture("here-document.sh"), false);
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
	const fixturePath = path.join(dir, "fixture.ts");
	const source = fixture("fixture.ts");
	fs.writeFileSync(fixturePath, source);
	const check = () =>
		spawnSync(process.execPath, [checker], { encoding: "utf8" });
	assert.equal(check().status, 0);
	const legacyYamlPath = path.join(dir, "legacy.yml");
	fs.writeFileSync(legacyYamlPath, "value: true\n");
	const legacyYaml = check();
	assert.equal(legacyYaml.status, 1);
	assert.match(
		legacyYaml.stderr,
		/legacy\.yml: use the \.yaml extension instead of \.yml/,
	);
	fs.rmSync(legacyYamlPath);
	fs.writeFileSync(fixturePath, source.replace("\n", " \n"));
	const invalid = check();
	assert.equal(invalid.status, 1);
	assert.match(invalid.stderr, /fixture\.ts:1: trailing whitespace/);
});

for (const [filename, fixtureName] of [
	[".specify/integrations/fixture.json", "fixture.json"],
	[".specify/scripts/bash/fixture.sh", "fixture.sh"],
]) {
	test(`${filename} is not excluded from formatting`, () => {
		assert.equal(
			runPrettier(filename, fixture(fixtureName).replaceAll("\t", "  ")).status,
			1,
		);
	});
}
