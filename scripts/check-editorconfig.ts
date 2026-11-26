import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const ignored = new Set([
	".git",
	".terraform",
	"dist",
	"node_modules",
	"vendor",
	"provider_code_spec.json",
]);
const checkExtensions = new Set([
	".js",
	".ts",
	".json",
	".md",
	".py",
	".sh",
	".tf",
	".yaml",
	".yml",
]);
const yamlExtensionExceptions = new Set([
	".specify/workflows/speckit/workflow.yml",
	".specify/extensions.yml",
]);
const issues: string[] = [];

function walk(directory: string): void {
	for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
		if (ignored.has(entry.name)) {
			continue;
		}

		const absolute = path.join(directory, entry.name);
		if (entry.isDirectory()) {
			walk(absolute);
			continue;
		}

		checkFile(absolute, entry.name);
	}
}

function checkFile(absolute: string, name: string): void {
	const extension = path.extname(name);
	const relative = path.relative(root, absolute).split(path.sep).join("/");
	const isSpecKitYamlException =
		yamlExtensionExceptions.has(relative) ||
		/^\.specify\/extensions\/[^/]+\/local-config\.yml$/.test(relative) ||
		/^\.specify\/presets\/[^/]+\/preset\.yml$/.test(relative);
	if (extension === ".yml" && !isSpecKitYamlException) {
		issues.push(`${relative}: use the .yaml extension instead of .yml`);
	}
	if (
		!checkExtensions.has(extension) &&
		name !== ".editorconfig" &&
		name !== ".gitignore" &&
		name !== "Makefile"
	) {
		return;
	}

	const text = fs.readFileSync(absolute, "utf8");
	const tabsAllowed =
		[".js", ".ts", ".json", ".py", ".sh"].includes(extension) ||
		name === "Makefile";
	if (!text.endsWith("\n")) {
		issues.push(`${relative}: missing final newline`);
	}
	text.split("\n").forEach((line, index) => {
		if (line.startsWith("\t") && !tabsAllowed) {
			issues.push(`${relative}:${index + 1}: tab indentation`);
		}
		if (/[ \t]+$/.test(line)) {
			issues.push(`${relative}:${index + 1}: trailing whitespace`);
		}
	});
}

walk(root);

if (issues.length > 0) {
	console.error(issues.join("\n"));
	process.exit(1);
}

console.log("EditorConfig formatting checks passed.");
