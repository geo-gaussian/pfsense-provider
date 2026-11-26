import { spawnSync } from "node:child_process";
import {
	checkTypeScriptFiles,
	formatFinding,
} from "../../scripts/check-size.ts";

const [goChecker, mode] = process.argv.slice(2);
const staged = mode === "staged";
const go = spawnSync(goChecker, staged ? ["--staged"] : [], {
	stdio: "inherit",
});
if (go.error) throw go.error;
if (go.status !== 0) process.exit(go.status ?? 1);
const findings = await checkTypeScriptFiles(process.cwd(), staged);
for (const finding of findings) {
	console.log(formatFinding(finding, process.env.GITHUB_ACTIONS === "true"));
}
if (findings.some((finding) => finding.level === "error")) process.exitCode = 1;
