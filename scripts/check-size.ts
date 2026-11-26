import { spawnSync } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import type { ParserOptions } from "prettier";
import { parsers } from "prettier/plugins/typescript";

const root = fileURLToPath(new URL("..", import.meta.url));

type Finding = {
	level: "warning" | "error";
	location: string;
	subject: string;
	lines: number;
	limit: number;
};

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function report(
	findings: Finding[],
	location: string,
	subject: string,
	lines: number,
	warning: number,
	error: number,
): void {
	if (lines > error) {
		findings.push({ level: "error", location, subject, lines, limit: error });
	} else if (lines > warning) {
		findings.push({
			level: "warning",
			location,
			subject,
			lines,
			limit: warning,
		});
	}
}

function location(node: Record<string, unknown>): [number, number] {
	const loc = node.loc;
	if (
		!isRecord(loc) ||
		!isRecord(loc.start) ||
		!isRecord(loc.end) ||
		typeof loc.start.line !== "number" ||
		typeof loc.end.line !== "number"
	) {
		throw new Error("TypeScript parser omitted source location");
	}
	return [loc.start.line, loc.end.line];
}

export async function inspectTypeScript(
	name: string,
	text: string,
): Promise<Finding[]> {
	if (/^\s*\/\/ Code generated .* DO NOT EDIT\.\s*(?:\r?\n|$)/.test(text))
		return [];
	const test = /\.(?:test|spec)\.[cm]?[jt]sx?$/.test(name);
	const findings: Finding[] = [];
	const lines =
		text.length === 0
			? 0
			: text.split("\n").length - Number(text.endsWith("\n"));
	report(findings, name, "file", lines, test ? 500 : 350, test ? 600 : 500);
	const options = { filepath: name } as ParserOptions;
	const tree: unknown = await parsers.typescript.parse(text, options);
	const seen = new Set<object>();
	function visit(value: unknown, parent?: Record<string, unknown>): void {
		if (typeof value !== "object" || value === null || seen.has(value)) return;
		seen.add(value);
		if (Array.isArray(value)) {
			for (const item of value) visit(item, parent);
			return;
		}
		if (!isRecord(value)) return;
		if (
			[
				"FunctionDeclaration",
				"FunctionExpression",
				"ArrowFunctionExpression",
			].includes(String(value.type)) &&
			value.body
		) {
			const owner =
				(parent?.type === "MethodDefinition" ||
					(parent?.type === "Property" && parent.method === true)) &&
				parent.value === value
					? parent
					: value;
			const [first, last] = location(owner);
			report(
				findings,
				`${name}:${first}`,
				"function",
				last - first + 1,
				test ? 100 : 40,
				test ? 150 : 80,
			);
		}
		for (const child of Object.values(value)) visit(child, value);
	}
	visit(tree);
	return findings;
}

export async function checkTypeScriptFiles(
	directory: string,
	staged: boolean,
): Promise<Finding[]> {
	const args = staged
		? ["ls-files", "-z"]
		: ["ls-files", "-co", "--exclude-standard", "-z"];
	const files = spawnSync("git", args, {
		cwd: directory,
		encoding: "utf8",
	});
	if (files.error) throw files.error;
	if (files.status !== 0) throw new Error(files.stderr);
	const findings: Finding[] = [];
	for (const name of files.stdout.split("\0").filter(Boolean)) {
		if (
			!/\.[cm]?[jt]sx?$/.test(name) ||
			[
				"tests/commands/",
				"tests/format/",
				"tests/terraform/",
				"tests/sizecheck/_fixtures/",
				".specify/scripts/",
			].some((prefix) => name.startsWith(prefix)) ||
			(!staged && !fs.existsSync(path.join(directory, name)))
		) {
			continue;
		}
		let source: string;
		if (staged) {
			const result = spawnSync("git", ["show", `:${name}`], {
				cwd: directory,
				encoding: "utf8",
			});
			if (result.error) throw result.error;
			if (result.status !== 0) throw new Error(result.stderr);
			source = result.stdout;
		} else {
			source = fs.readFileSync(path.join(directory, name), "utf8");
		}
		findings.push(...(await inspectTypeScript(name, source)));
	}
	return findings;
}

export function formatFinding(item: Finding, githubActions: boolean): string {
	const message = `${item.subject} has ${item.lines} lines (limit ${item.limit})`;
	if (!githubActions) return `${item.location}: ${item.level}: ${message}`;
	const index = item.subject === "file" ? -1 : item.location.lastIndexOf(":");
	const name = index === -1 ? item.location : item.location.slice(0, index);
	const line = index === -1 ? "" : `,line=${item.location.slice(index + 1)}`;
	const escapeData = (value: string) =>
		value
			.replaceAll("%", "%25")
			.replaceAll("\r", "%0D")
			.replaceAll("\n", "%0A");
	const escapedName = escapeData(name)
		.replaceAll(":", "%3A")
		.replaceAll(",", "%2C");
	return `::${item.level} file=${escapedName}${line}::${escapeData(message)}`;
}

async function main(): Promise<void> {
	const staged = process.argv.slice(2).includes("--staged");
	if (process.argv.slice(2).some((arg) => arg !== "--staged")) {
		throw new Error("Usage: check-size.ts [--staged]");
	}
	const findings = await checkTypeScriptFiles(root, staged);
	for (const item of findings) {
		console.log(formatFinding(item, process.env.GITHUB_ACTIONS === "true"));
	}
	if (findings.some((item) => item.level === "error")) process.exitCode = 1;
	else console.log("TypeScript and JavaScript size checks passed.");
}

if (
	process.argv[1] &&
	path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
	await main();
}
