import * as assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import { tmpdir } from "node:os";
import * as path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));

test("Go format check reports unformatted files and formatter failures", (t) => {
	const dir = fs.mkdtempSync(path.join(tmpdir(), "pfsense-gofmt-"));
	t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
	const source = fs.readFileSync(
		path.join(root, "tests/format/_fixtures/fixture.go"),
		"utf8",
	);
	const fixture = path.join(dir, "fixture.go");
	fs.writeFileSync(fixture, source);
	const check = (env = process.env) =>
		spawnSync("sh", [path.join(root, "scripts/check-go-format.sh")], {
			cwd: dir,
			encoding: "utf8",
			env,
		});
	assert.equal(check().status, 0);
	fs.writeFileSync(fixture, source.replace("\tprintln", "  println"));
	const unformatted = check();
	assert.equal(unformatted.status, 1);
	assert.match(unformatted.stdout, /fixture\.go/);
	const mock = path.join(dir, "gofmt");
	fs.copyFileSync(path.join(root, "tests/commands/gofmt.sh"), mock);
	fs.chmodSync(mock, 0o755);
	const failure = check({
		...process.env,
		PATH: `${dir}${path.delimiter}${process.env.PATH}`,
	});
	assert.equal(failure.status, 42);
	assert.match(failure.stderr, /mock gofmt failure/);
});

test("Pages guard deploys only the current commit and propagates errors", (t) => {
	const dir = fs.mkdtempSync(path.join(tmpdir(), "pfsense-pages-"));
	t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
	const mock = path.join(dir, "gh");
	fs.copyFileSync(path.join(root, "tests/commands/gh.sh"), mock);
	fs.chmodSync(mock, 0o755);
	const response = path.join(dir, "response.json");
	const output = path.join(dir, "output");
	const argumentsFile = path.join(dir, "arguments");
	const env = {
		...process.env,
		PATH: `${dir}${path.delimiter}${process.env.PATH}`,
		GITHUB_REPOSITORY: "example/project",
		GITHUB_SHA: "current-commit",
		GITHUB_OUTPUT: output,
		MOCK_GH_RESPONSE: response,
		MOCK_GH_ARGUMENTS: argumentsFile,
	};
	const check = (extra: Record<string, string> = {}) =>
		spawnSync("sh", [path.join(root, "scripts/check-current-commit.sh")], {
			encoding: "utf8",
			env: { ...env, ...extra },
		});

	fs.writeFileSync(output, "");
	fs.writeFileSync(response, JSON.stringify({ sha: "current-commit" }));
	assert.equal(check().status, 0);
	assert.equal(fs.readFileSync(output, "utf8"), "deploy=true\n");
	assert.equal(
		fs.readFileSync(argumentsFile, "utf8"),
		"api\nrepos/example/project/commits/main\n",
	);
	fs.writeFileSync(output, "");
	fs.writeFileSync(response, JSON.stringify({ sha: "older-commit" }));
	const stale = check();
	assert.equal(stale.status, 0);
	assert.match(stale.stdout, /outdated/);
	assert.equal(fs.readFileSync(output, "utf8"), "");
	assert.notEqual(check({ MOCK_GH_STATUS: "1" }).status, 0);
	fs.writeFileSync(response, JSON.stringify({ sha: null }));
	assert.notEqual(check().status, 0);
	assert.equal(fs.readFileSync(output, "utf8"), "");
});

test("release tag resolution requires an existing tag at the validated commit", (t) => {
	const dir = fs.mkdtempSync(path.join(tmpdir(), "pfsense-release-"));
	t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
	const git = (...args: string[]) => {
		const result = spawnSync("git", args, { cwd: dir, encoding: "utf8" });
		assert.equal(result.status, 0, result.stderr);
		return result.stdout.trim();
	};
	git("init", "-q");
	git(
		"-c",
		"user.name=Test",
		"-c",
		"user.email=test@example.invalid",
		"commit",
		"--allow-empty",
		"-qm",
		"fixture",
	);
	const commit = git("rev-parse", "HEAD");
	git("tag", "v1.2.3");
	git(
		"-c",
		"user.name=Test",
		"-c",
		"user.email=test@example.invalid",
		"tag",
		"-a",
		"v1.2.4",
		"-m",
		"fixture",
	);
	git("branch", "v-branch");
	const output = path.join(dir, "output");
	const check = (tag: string, expected = "") => {
		fs.writeFileSync(output, "");
		return spawnSync(
			"sh",
			[path.join(root, "scripts/resolve-release-tag.sh")],
			{
				cwd: dir,
				encoding: "utf8",
				env: {
					...process.env,
					GITHUB_OUTPUT: output,
					RELEASE_TAG: tag,
					EXPECTED_COMMIT: expected,
				},
			},
		);
	};
	for (const tag of ["v1.2.3", "v1.2.4"]) {
		assert.equal(check(tag, commit).status, 0);
		assert.equal(fs.readFileSync(output, "utf8"), `commit=${commit}\n`);
	}
	for (const tag of ["1.2.3", "v-branch", "v-missing", "v-bad\ninjection"]) {
		assert.notEqual(check(tag).status, 0);
		assert.equal(fs.readFileSync(output, "utf8"), "");
	}
	const moved = check("v1.2.3", "different-commit");
	assert.notEqual(moved.status, 0);
	assert.match(moved.stderr, /no longer points/);
	assert.equal(fs.readFileSync(output, "utf8"), "");
});
