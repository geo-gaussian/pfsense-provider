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

		const extension = path.extname(entry.name);
		if (
			!checkExtensions.has(extension) &&
			entry.name !== ".editorconfig" &&
			entry.name !== ".gitignore" &&
			entry.name !== "Makefile"
		) {
			continue;
		}

		const relative = path.relative(root, absolute).split(path.sep).join("/");
		const text = fs.readFileSync(absolute, "utf8");
		const lines = text.split("\n");
		const tabsAllowed =
			[".js", ".ts", ".json", ".py", ".sh"].includes(extension) ||
			entry.name === "Makefile";

		if (!text.endsWith("\n")) {
			issues.push(`${relative}: missing final newline`);
		}
		lines.forEach((line, index) => {
			if (line.startsWith("\t") && !tabsAllowed) {
				issues.push(`${relative}:${index + 1}: tab indentation`);
			}
			if (/[ \t]+$/.test(line)) {
				issues.push(`${relative}:${index + 1}: trailing whitespace`);
			}
		});
	}
}

walk(root);

if (issues.length > 0) {
	console.error(issues.join("\n"));
	process.exit(1);
}

console.log("EditorConfig formatting checks passed.");
