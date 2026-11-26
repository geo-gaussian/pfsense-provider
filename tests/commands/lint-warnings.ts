const scenario = process.argv[2];

switch (scenario) {
	case "two":
		process.stdout.write("main.go:20: warning: function is too long\n");
		process.stderr.write("[warn] another lint finding\n");
		break;
	case "three":
		process.stdout.write(
			"::warning file=main.go,line=20::function is too long\n",
		);
		process.stderr.write("npm WARN dependency issue\n");
		process.stdout.write("\u001b[33mWarning: another lint finding\u001b[0m\n");
		break;
	case "split":
		process.stdout.write("warn");
		process.stdout.write("ing: unfinished output");
		break;
	case "split-three":
		process.stdout.write("Warning: first\n[warn] second\nwarn");
		process.stdout.write("ing: third");
		break;
	case "runtime-three":
		process.stdout.write("file.py:8: DeprecationWarning: old API\n");
		process.stdout.write("2:7  warning  invalid style\n");
		process.stderr.write("(node:123) [DEP0001] ExperimentalWarning: example\n");
		break;
	case "not-warnings":
		process.stdout.write("0 warnings\nNo warning was emitted\n");
		break;
	case "failed":
		process.stderr.write("lint failed\n");
		process.exitCode = 7;
		break;
	default:
		console.error(`Unknown fixture scenario: ${scenario}`);
		process.exitCode = 2;
}
