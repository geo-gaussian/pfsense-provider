import * as assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import { tmpdir } from "node:os";
import * as path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import {
	checkTypeScriptFiles,
	formatFinding,
	inspectTypeScript,
} from "./check-size.ts";

const source = fs.readFileSync(
	fileURLToPath(
		new URL("../tests/sizecheck/_fixtures/sample.ts", import.meta.url),
	),
	"utf8",
);

function fixture(name: string): string {
	return fs.readFileSync(
		fileURLToPath(
			new URL(`../tests/sizecheck/_fixtures/${name}`, import.meta.url),
		),
		"utf8",
	);
}

test("TypeScript function thresholds count physical lines", async () => {
	for (const [extra, expected] of [
		[36, undefined],
		[37, "warning"],
		[76, "warning"],
		[77, "error"],
	] as const) {
		const text = source.replace(
			"// padding",
			"// padding" + "\n".repeat(extra),
		);
		const findings = await inspectTypeScript("sample.ts", text);
		assert.equal(findings[0]?.level, expected);
	}
});

test("TypeScript file and test limits distinguish warnings and errors", async () => {
	for (const [name, lines, expected] of [
		["sample.ts", 350, undefined],
		["sample.ts", 351, "warning"],
		["sample.ts", 501, "error"],
		["sample.test.ts", 500, undefined],
		["sample.test.ts", 501, "warning"],
		["sample.test.ts", 601, "error"],
	] as const) {
		const text = source + "\n".repeat(lines - 4);
		const findings = await inspectTypeScript(name, text);
		assert.equal(findings[0]?.level, expected);
	}
});

test("TypeScript test functions use the test thresholds", async () => {
	for (const [extra, expected] of [
		[96, undefined],
		[97, "warning"],
		[146, "warning"],
		[147, "error"],
	] as const) {
		const text = source.replace(
			"// padding",
			"// padding" + "\n".repeat(extra),
		);
		const findings = await inspectTypeScript("sample.test.ts", text);
		assert.equal(findings[0]?.level, expected);
	}
});

test("nested TypeScript functions and methods retain their physical spans", async () => {
	const text = fixture("nested.ts").replace(
		"// padding",
		"// padding" + "\n".repeat(40),
	);
	const findings = await inspectTypeScript("nested.ts", text);
	assert.deepEqual(
		findings.map((item) => item.location),
		["nested.ts:2", "nested.ts:3"],
	);
});

test("ignores generated TypeScript and reports invalid source", async () => {
	const generated = fixture("generated.ts") + "\n".repeat(600);
	assert.deepEqual(await inspectTypeScript("generated.ts", generated), []);
	await assert.rejects(
		inspectTypeScript("invalid.ts", fixture("invalid-ts.txt")),
	);
});

test("formats findings with the correct GitHub annotation severity", () => {
	const file = {
		level: "warning" as const,
		location: "path/example.ts",
		subject: "file",
		lines: 351,
		limit: 350,
	};
	assert.equal(
		formatFinding(file, false),
		"path/example.ts: warning: file has 351 lines (limit 350)",
	);
	assert.equal(
		formatFinding(file, true),
		"::warning file=path/example.ts::file has 351 lines (limit 350)",
	);
	assert.equal(
		formatFinding(
			{
				level: "error",
				location: "path/a%,:\n.ts:12",
				subject: "function a%\n",
				lines: 81,
				limit: 80,
			},
			true,
		),
		"::error file=path/a%25%2C%3A%0A.ts,line=12::function a%25%0A has 81 lines (limit 80)",
	);
});

test("checks staged TypeScript rather than the working tree", async (t) => {
	const dir = fs.mkdtempSync(path.join(tmpdir(), "pfsense-size-"));
	t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
	const git = (...args: string[]) => {
		const result = spawnSync("git", args, { cwd: dir, encoding: "utf8" });
		assert.equal(result.status, 0, result.stderr);
	};
	git("init", "-q");
	const file = path.join(dir, "sample.ts");
	fs.writeFileSync(file, source + "\n".repeat(501 - 4));
	git("add", "sample.ts");
	fs.writeFileSync(file, source);
	assert.deepEqual(await checkTypeScriptFiles(dir, false), []);
	assert.equal((await checkTypeScriptFiles(dir, true))[0]?.level, "error");
});
