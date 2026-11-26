import * as assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import { tmpdir } from "node:os";
import * as path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const budget = fileURLToPath(new URL("./lint-budget.ts", import.meta.url));
const fixture = fileURLToPath(
	new URL("../tests/commands/lint-warnings.ts", import.meta.url),
);
const root = fileURLToPath(new URL("..", import.meta.url));
const sizeChecks = path.join(root, "tests", "commands", "lint-size-checks.ts");

function runScenario(scenario: string, githubActions = false) {
	return spawnSync(
		process.execPath,
		[budget, process.execPath, fixture, scenario],
		{
			encoding: "utf8",
			env: { ...process.env, GITHUB_ACTIONS: githubActions ? "true" : "false" },
		},
	);
}

test("allows fewer than three warnings from multiple lint output streams", () => {
	for (const scenario of ["not-warnings", "two", "split"]) {
		const result = runScenario(scenario);
		assert.equal(result.status, 0, result.stderr);
	}
	const result = runScenario("two");
	assert.match(result.stdout, /main\.go:20: warning:/);
	assert.match(result.stderr, /\[warn\] another lint finding/);
});

test("fails at three warnings across formats and emits a GitHub error", () => {
	for (const githubActions of [false, true]) {
		const result = runScenario("three", githubActions);
		assert.equal(result.status, 1);
		assert.match(result.stdout, /::warning file=main\.go,line=20::/);
		assert.match(result.stderr, /npm WARN dependency issue/);
		assert.match(
			result.stderr,
			/Lint warning budget exceeded: 3 warnings \(maximum 2\)\./,
		);
		assert.equal(result.stderr.includes("::error::"), githubActions);
	}
	const split = runScenario("split-three");
	assert.equal(split.status, 1);
	assert.match(split.stdout, /warning: third$/);
	assert.match(split.stderr, /3 warnings/);
	const runtime = runScenario("runtime-three");
	assert.equal(runtime.status, 1);
	assert.match(runtime.stderr, /3 warnings/);
});

test("propagates lint failures and reports unavailable commands", () => {
	const failed = runScenario("failed");
	assert.equal(failed.status, 7);
	assert.match(failed.stderr, /lint failed/);
	const missing = spawnSync(
		process.execPath,
		[budget, "pfsense-nonexistent-lint-command"],
		{
			encoding: "utf8",
		},
	);
	assert.equal(missing.status, 1);
	assert.match(missing.stderr, /Unable to run lint command/);
});

test("aggregates real Go and TypeScript findings for staged and working files", (t) => {
	const dir = fs.mkdtempSync(path.join(tmpdir(), "pfsense-lint-budget-"));
	t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
	const goChecker = path.join(dir, "sizecheck");
	const build = spawnSync(
		"go",
		["build", "-o", goChecker, "./tools/sizecheck"],
		{
			cwd: root,
			encoding: "utf8",
		},
	);
	assert.equal(build.status, 0, build.stderr);
	for (const args of [
		["init", "-q"],
		["add", "first.go", "first.ts"],
	]) {
		if (args[0] === "add") {
			writeSizeFixture(dir, "first.go", "sample.go", 346);
			writeSizeFixture(dir, "first.ts", "sample.ts", 347);
		}
		const git = spawnSync("git", args, { cwd: dir, encoding: "utf8" });
		assert.equal(git.status, 0, git.stderr);
	}
	for (const mode of ["working", "staged"]) {
		const two = runSizeChecks(dir, goChecker, mode);
		assert.equal(two.status, 0, two.stderr);
		assert.match(two.stdout, /first\.go: warning:/);
		assert.match(two.stdout, /first\.ts: warning:/);
	}
	writeSizeFixture(dir, "second.go", "sample.go", 346);
	const three = runSizeChecks(dir, goChecker, "working");
	assert.equal(three.status, 1);
	assert.match(three.stderr, /3 warnings/);
	assert.equal(runSizeChecks(dir, goChecker, "staged").status, 0);
});

function writeSizeFixture(
	dir: string,
	name: string,
	source: string,
	padding: number,
) {
	const fixturePath = path.join(
		root,
		"tests",
		"sizecheck",
		"_fixtures",
		source,
	);
	fs.writeFileSync(
		path.join(dir, name),
		fs.readFileSync(fixturePath, "utf8") + "\n".repeat(padding),
	);
}

function runSizeChecks(dir: string, goChecker: string, mode: string) {
	return spawnSync(
		process.execPath,
		[budget, process.execPath, sizeChecks, goChecker, mode],
		{
			cwd: dir,
			encoding: "utf8",
			env: { ...process.env, GITHUB_ACTIONS: "false" },
		},
	);
}
