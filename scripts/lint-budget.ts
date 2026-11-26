import { spawn } from "node:child_process";
import { createInterface } from "node:readline";
import { stripVTControlCharacters } from "node:util";

const warningThreshold = 3;
const severity = String.raw`(?:\[warn(?:ing)?\]|(?:[a-z]*warning|warn)(?=[:\s-]|$))`;
const prefix =
	String.raw`(?:.*?:\s*|(?:npm|pnpm|yarn)\s+|` +
	String.raw`\(node:\d+\)\s+(?:\[[^\]]+\]\s+)?|\d+:\d+\s+)?`;
const warningLine = new RegExp(`^${prefix}${severity}`, "i");

function isWarning(line: string): boolean {
	const text = stripVTControlCharacters(line).trimStart();
	return /^::warning(?:\s[^:]*)?::/i.test(text) || warningLine.test(text);
}

function countWarnings(
	stream: NodeJS.ReadableStream,
	destination: NodeJS.WritableStream,
	record: () => void,
): void {
	stream.pipe(destination, { end: false });
	const lines = createInterface({ input: stream, crlfDelay: Infinity });
	lines.on("line", (line) => {
		if (isWarning(line)) record();
	});
}

function main(): void {
	const [command, ...args] = process.argv.slice(2);
	if (!command) {
		console.error("Usage: node scripts/lint-budget.ts COMMAND [ARGUMENTS...]");
		process.exitCode = 1;
		return;
	}
	const child = spawn(command, args, { stdio: ["inherit", "pipe", "pipe"] });
	let warnings = 0;
	let spawnError: Error | undefined;
	const record = () => warnings++;
	countWarnings(child.stdout, process.stdout, record);
	countWarnings(child.stderr, process.stderr, record);
	child.on("error", (error) => {
		spawnError = error;
	});
	child.on("close", (code, signal) => {
		if (spawnError) {
			console.error(`Unable to run lint command: ${spawnError.message}`);
		} else if (signal) {
			console.error(`Lint command terminated by ${signal}.`);
		}
		if (warnings >= warningThreshold) {
			const message = `Lint warning budget exceeded: ${warnings} warnings (maximum 2).`;
			console.error(
				process.env.GITHUB_ACTIONS === "true" ? `::error::${message}` : message,
			);
		}
		if (
			spawnError ||
			signal ||
			code === null ||
			code !== 0 ||
			warnings >= warningThreshold
		) {
			process.exitCode = code && code > 0 ? code : 1;
		}
	});
}

main();
