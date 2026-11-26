import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const specKitScripts = new Set([
	".specify/scripts/bash/check-prerequisites.sh",
	".specify/scripts/bash/common.sh",
	".specify/scripts/bash/create-new-feature.sh",
	".specify/scripts/bash/resolve-template.sh",
	".specify/scripts/bash/setup-plan.sh",
	".specify/scripts/bash/setup-tasks.sh",
]);

function commandIssues(line: string): string[] {
	const issues: string[] = [];
	if (/^\s*#/.test(line)) return issues;

	if (
		new RegExp(
			String.raw`\b(?:python(?:3(?:\.\d+)?)?|node|ruby|perl|bash|sh)\s+` +
				String.raw`(?:--[\w-]+(?:=\S+)?\s+)*-(?:c|e|p|-eval|-print)(?:\s|=|["']|$)`,
		).test(line) ||
		/\b(?:python(?:3)?|node|ruby|perl|bash|sh)\s+[^#\n]*<<-?\s*['"]?[A-Za-z_]/.test(
			line,
		)
	) {
		issues.push("interpreter program supplied inline");
	}

	for (const match of line.matchAll(
		/(?<![\w-])(?:awk|sed|jq)\s+((?:-[A-Za-z]+\s+)*)(?:['"]|[.({])/g,
	)) {
		const options = match[1].trim().split(/\s+/);
		if (!options.some((option) => /^-[A-Za-z]*f/.test(option))) {
			issues.push("filter program supplied inline");
		}
	}
	if (/(?:^|\s)--jq(?:\s|=)/.test(line)) {
		issues.push("GitHub CLI query supplied inline");
	}
	return issues;
}

export function findInlinePrograms(filename: string, text: string): string[] {
	const name = filename.split(path.sep).join("/");
	if (specKitScripts.has(name)) return [];
	const issues: string[] = [];
	const record = (line: number, message: string) =>
		issues.push(`${name}:${line}: ${message}`);

	if (path.posix.basename(name) === "package.json") {
		return findManifestPrograms(name, text);
	}

	if (name.endsWith(".go") && !name.endsWith("_gen.go")) {
		for (const match of text.matchAll(/`([^`]*)`/gs)) {
			if (
				/(?:^|\n)[ \t]*(?:terraform\s*\{|(?:provider|resource|data|variable|output)\s+"[^"]+"\s*(?:"[^"]+"\s*)?\{)/.test(
					match[1],
				)
			) {
				record(
					text.slice(0, match.index).split("\n").length,
					"Terraform configuration embedded in a Go raw string",
				);
			}
		}
		return issues;
	}

	return findShellPrograms(name, text);
}

function findManifestPrograms(name: string, text: string): string[] {
	const issues: string[] = [];
	const manifest: unknown = JSON.parse(text);
	if (
		typeof manifest === "object" &&
		manifest !== null &&
		"scripts" in manifest &&
		typeof manifest.scripts === "object" &&
		manifest.scripts !== null
	) {
		for (const [script, command] of Object.entries(manifest.scripts)) {
			if (typeof command === "string") {
				for (const issue of commandIssues(command)) {
					issues.push(`${name}:1: ${script}: ${issue}`);
				}
			}
		}
	}
	return issues;
}

function findShellPrograms(name: string, text: string): string[] {
	const issues: string[] = [];
	const record = (line: number, message: string) =>
		issues.push(`${name}:${line}: ${message}`);
	const workflow =
		name.startsWith(".github/workflows/") && /\.ya?ml$/.test(name);
	const shell =
		name.endsWith(".sh") ||
		name.startsWith(".githooks/") ||
		name === "Makefile";
	if (!workflow && !shell) return issues;

	const lines = text.split("\n");
	for (let index = 0; index < lines.length; index++) {
		const line = lines[index];
		if (workflow) {
			const run = line.match(/^(\s*)(?:-\s+)?run:\s*(.*)$/);
			if (!run) continue;
			const content = run[2];
			if (/^[|>][-+]?\s*(?:#.*)?$/.test(content)) {
				const { body, next } = workflowBody(lines, index + 1, run[1].length);
				if (body.length > 1) record(index + 1, "multiline workflow program");
				for (const command of body) {
					for (const issue of commandIssues(command)) record(index + 1, issue);
				}
				index = next - 1;
			} else {
				for (const issue of commandIssues(content)) record(index + 1, issue);
			}
		} else {
			for (const issue of commandIssues(line)) record(index + 1, issue);
		}
	}
	return issues;
}

function workflowBody(
	lines: string[],
	start: number,
	indent: number,
): { body: string[]; next: number } {
	const body: string[] = [];
	let next = start;
	while (next < lines.length) {
		const candidate = lines[next];
		if (
			candidate.trim() &&
			candidate.length - candidate.trimStart().length <= indent
		)
			break;
		if (candidate.trim() && !candidate.trimStart().startsWith("#")) {
			body.push(candidate.trim());
		}
		next++;
	}
	return { body, next };
}

function main(): void {
	const files = spawnSync(
		"git",
		["ls-files", "-co", "--exclude-standard", "-z"],
		{
			cwd: root,
			encoding: "utf8",
		},
	);
	if (files.error) throw files.error;
	if (files.status !== 0) {
		process.stderr.write(files.stderr);
		process.exit(files.status ?? 1);
	}
	const issues = files.stdout
		.split("\0")
		.filter(Boolean)
		.filter(
			(name) =>
				!specKitScripts.has(name) &&
				(name.endsWith(".go") ||
					name.endsWith(".sh") ||
					name.startsWith(".githooks/") ||
					name === "Makefile" ||
					path.posix.basename(name) === "package.json" ||
					(name.startsWith(".github/workflows/") && /\.ya?ml$/.test(name))) &&
				fs.existsSync(path.join(root, name)),
		)
		.flatMap((name) =>
			findInlinePrograms(name, fs.readFileSync(path.join(root, name), "utf8")),
		);
	if (issues.length) {
		console.error(issues.join("\n"));
		process.exit(1);
	}
	console.log("Inline program checks passed.");
}

if (
	process.argv[1] &&
	path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
	main();
}
