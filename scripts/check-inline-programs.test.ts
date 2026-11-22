import * as assert from "node:assert/strict";
import * as fs from "node:fs";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { findInlinePrograms } from "./check-inline-programs.ts";

const terraformFixture = fs.readFileSync(
	fileURLToPath(
		new URL("../testdata/terraform/lifecycle/main.tf", import.meta.url),
	),
	"utf8",
);
const scriptFixture = fs.readFileSync(
	fileURLToPath(new URL("../testdata/format/fixture.ts", import.meta.url)),
	"utf8",
);
const goFixture = fs.readFileSync(
	fileURLToPath(new URL("../testdata/format/config.go", import.meta.url)),
	"utf8",
);
const workflowFixture = fs.readFileSync(
	fileURLToPath(new URL("../testdata/format/workflow.yaml", import.meta.url)),
	"utf8",
);
const queryFixture = fs.readFileSync(
	fileURLToPath(new URL("./current-commit-sha.jq", import.meta.url)),
	"utf8",
);

test("detects Terraform configuration in Go raw strings", () => {
	const go = goFixture.replace('"PLACEHOLDER"', "`" + terraformFixture + "`");
	assert.match(
		findInlinePrograms("fixture_test.go", go)[0],
		/Terraform configuration/,
	);
});

test("detects inline interpreter and filter programs in scripts and manifests", () => {
	const inline = ["node", "-e", JSON.stringify(scriptFixture)].join(" ");
	const filter = ["awk", JSON.stringify(scriptFixture)].join(" ");
	assert.match(
		findInlinePrograms("script.sh", inline)[0],
		/interpreter program/,
	);
	assert.match(
		findInlinePrograms(
			"script.sh",
			["python3", "-c", JSON.stringify(scriptFixture)].join(" "),
		)[0],
		/interpreter program/,
	);
	assert.match(findInlinePrograms("Makefile", filter)[0], /filter program/);
	for (const program of ["sed -e", "jq -r"]) {
		assert.match(
			findInlinePrograms(
				"script.sh",
				program + " " + JSON.stringify(queryFixture),
			)[0],
			/filter program/,
		);
	}
	const manifest = JSON.stringify({ scripts: { check: inline } });
	assert.match(
		findInlinePrograms("package.json", manifest)[0],
		/check: interpreter program/,
	);
	assert.match(
		findInlinePrograms(
			"script.sh",
			"gh api path --jq " + JSON.stringify(queryFixture),
		)[0],
		/query/,
	);
});

test("detects multiline workflow programs and inline commands", () => {
	const command = ["git", "--version"].join(" ");
	const block = workflowFixture.replace(
		command,
		command + "\n          " + command,
	);
	assert.match(
		findInlinePrograms(".github/workflows/ci.yml", block)[0],
		/multiline workflow/,
	);
	const inline = ["node", "-e", JSON.stringify(scriptFixture)].join(" ");
	assert.match(
		findInlinePrograms(
			".github/workflows/ci.yml",
			workflowFixture.replace(command, inline),
		)[0],
		/interpreter program/,
	);
});

test("allows ordinary commands, standalone filter files and managed Spec Kit scripts", () => {
	const ordinary = [
		"node scripts/check-format.test.ts",
		"awk -f scripts/filter.awk input.txt",
		"jq -r -f scripts/query.jq input.json",
		"sed -f scripts/edit.sed input.txt",
	].join("\n");
	assert.deepEqual(findInlinePrograms("script.sh", ordinary), []);
	assert.deepEqual(
		findInlinePrograms(
			".specify/scripts/bash/common.sh",
			"node -e " + JSON.stringify(scriptFixture),
		),
		[],
	);
	assert.match(
		findInlinePrograms(
			".specify/scripts/bash/custom.sh",
			"node -e " + JSON.stringify(scriptFixture),
		)[0],
		/interpreter program/,
	);
	assert.deepEqual(
		findInlinePrograms("README.md", "node -e " + JSON.stringify(scriptFixture)),
		[],
	);
});
