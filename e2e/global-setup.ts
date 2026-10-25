import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { BINARY, REPO_ROOT } from './fixtures.ts';

// Builds bin/inventar with the client embedded, once per test run.
// E2E_SKIP_BUILD=1 reuses an existing binary, e.g. while writing tests.
export default function globalSetup() {
	if (process.env.E2E_SKIP_BUILD === '1' && existsSync(BINARY)) return;
	execFileSync('make', ['build'], { cwd: REPO_ROOT, stdio: 'inherit' });
}
