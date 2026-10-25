import { spawn, spawnSync, type ChildProcess } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test as base, expect, type APIRequestContext, type Page } from '@playwright/test';

export { expect };

export const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
export const BINARY = join(REPO_ROOT, 'bin', 'inventar');
export const PHOTO = join(dirname(fileURLToPath(import.meta.url)), 'fixtures', 'photo.jpg');

export const USER = { name: 'Testnutzer', email: 'test@example.org', password: 'test-passwort-123' };

type Server = { url: string };

/** An API response the test provokes on purpose, e.g. GET /api/me before logging in. */
type ExpectedError = { status: number; path: string | RegExp };

type Guard = {
	/** Allows API responses with this status whose path contains (or matches) path. */
	allow(status: number, path: string | RegExp): void;
	readonly expected: readonly ExpectedError[];
};

type TestFixtures = {
	guard: Guard;
	/** Logs the test user in via the API; the page then shares the session cookie. */
	loggedIn: void;
	api: Api;
};

type WorkerFixtures = { server: Server };

function freePort(): Promise<number> {
	return new Promise((resolve, reject) => {
		const s = createServer();
		s.once('error', reject);
		s.listen(0, '127.0.0.1', () => {
			const address = s.address();
			s.close(() => (typeof address === 'object' && address ? resolve(address.port) : reject(new Error('no port'))));
		});
	});
}

/** Environment without INVENTAR_* settings of the developer's shell, e.g. DuckDNS or SMTP. */
function serverEnv(dataDir: string, addr: string): NodeJS.ProcessEnv {
	const env: NodeJS.ProcessEnv = {};
	for (const [k, v] of Object.entries(process.env)) if (!k.startsWith('INVENTAR_')) env[k] = v;
	return { ...env, INVENTAR_DATA_DIR: dataDir, INVENTAR_ADDR: addr };
}

async function waitUntilUp(url: string, child: ChildProcess, log: () => string) {
	const deadline = Date.now() + 15_000;
	while (Date.now() < deadline) {
		if (child.exitCode !== null) throw new Error(`server exited with ${child.exitCode}:\n${log()}`);
		try {
			const r = await fetch(url + '/api/me');
			if (r.status === 401) return;
		} catch {
			// Not listening yet.
		}
		await new Promise((r) => setTimeout(r, 100));
	}
	throw new Error(`server did not start:\n${log()}`);
}

/** Small helper for setting up data through the API with the page's session. */
export class Api {
	constructor(private readonly request: APIRequestContext) {}

	async call<T = any>(method: string, path: string, data?: unknown): Promise<T> {
		const r = await this.request.fetch(path, { method, data: data as any });
		if (!r.ok()) throw new Error(`${method} ${path}: ${r.status()} ${await r.text()}`);
		return (r.status() === 204 ? undefined : await r.json()) as T;
	}

	apartment(name: string) {
		return this.call<{ id: number }>('POST', '/api/apartments', { name });
	}

	room(apartmentId: number, name: string) {
		return this.call<{ id: number }>('POST', `/api/apartments/${apartmentId}/rooms`, { name });
	}

	storageLocation(name: string, parent: { roomId: number } | { parentId: number }) {
		return this.call<{ id: number }>('POST', '/api/storage-locations', { name, ...parent });
	}

	/** Creates an item with the fixture photo, owned by the test user, placed in the given room. */
	async item(name: string, roomId: number) {
		const me = await this.call<{ personId: number }>('GET', '/api/me');
		const r = await this.request.post('/api/items', {
			multipart: {
				name,
				owner: String(me.personId),
				room: String(roomId),
				photo: { name: 'photo.jpg', mimeType: 'image/jpeg', buffer: readFileSync(PHOTO) }
			}
		});
		if (!r.ok()) throw new Error(`POST /api/items: ${r.status()} ${await r.text()}`);
		return (await r.json()) as { id: number };
	}
}

/** Unique suffix for names, so tests sharing a worker's server don't see each other's data. */
export function unique() {
	return Math.random().toString(36).slice(2, 7);
}

/** True for the phone project: the menu is a dialog behind ☰ in the bottom bar instead of a sidebar. */
export function isNarrow(page: Page) {
	return (page.viewportSize()?.width ?? 0) < 1024;
}

export const test = base.extend<TestFixtures, WorkerFixtures>({
	// Every worker gets its own server with an empty data directory and the test user.
	server: [
		async ({}, use) => {
			const dataDir = mkdtempSync(join(tmpdir(), 'inventar-e2e-'));
			const port = await freePort();
			const addr = `127.0.0.1:${port}`;
			const env = serverEnv(dataDir, addr);

			const created = spawnSync(BINARY, ['create-user', '--name', USER.name, '--email', USER.email], {
				env,
				input: `${USER.password}\n${USER.password}\n`,
				encoding: 'utf8'
			});
			if (created.status !== 0) throw new Error(`create-user failed: ${created.stderr}${created.stdout}`);

			let output = '';
			const child = spawn(BINARY, ['serve'], { env, stdio: ['ignore', 'pipe', 'pipe'] });
			child.stdout?.on('data', (d) => (output += d));
			child.stderr?.on('data', (d) => (output += d));
			const url = `http://${addr}`;
			let stopping = false;
			// A server that dies during the tests would only show up as refused connections; say why.
			child.once('exit', (code, signal) => {
				if (!stopping) console.error(`inventar serve on ${addr} exited early (code ${code}, signal ${signal}):\n${output}`);
			});
			try {
				await waitUntilUp(url, child, () => output);
				await use({ url });
			} finally {
				stopping = true;
				child.kill('SIGTERM');
				await new Promise((r) => (child.exitCode !== null ? r(null) : child.once('exit', r)));
				rmSync(dataDir, { recursive: true, force: true });
			}
		},
		{ scope: 'worker' }
	],

	baseURL: async ({ server }, use) => {
		await use(server.url);
	},

	guard: async ({}, use) => {
		const expected: ExpectedError[] = [];
		await use({ allow: (status, path) => expected.push({ status, path }), expected });
	},

	// Fails the test on console errors, uncaught exceptions and error responses of the server it didn't expect.
	page: async ({ page, baseURL, guard }, use) => {
		const problems: string[] = [];
		const origin = new URL(baseURL!).origin;

		page.on('console', (m) => {
			// Failed requests also show up as console errors; the response check below decides about those.
			if (m.type() === 'error' && !m.text().startsWith('Failed to load resource')) problems.push(`console: ${m.text()}`);
		});
		page.on('pageerror', (e) => problems.push(`exception: ${e.message}`));
		page.on('response', (r) => {
			const url = new URL(r.url());
			if (url.origin !== origin || r.status() < 400) return;
			const ok = url.pathname.startsWith('/api/') && guard.expected.some((e) => e.status === r.status() && (typeof e.path === 'string' ? url.pathname.includes(e.path) : e.path.test(url.pathname)));
			if (!ok) problems.push(`response: ${r.request().method()} ${url.pathname} → ${r.status()}`);
		});

		await use(page);

		expect(problems, 'console errors or unexpected error responses').toEqual([]);
	},

	loggedIn: async ({ page }, use) => {
		const r = await page.request.post('/api/login', { data: { email: USER.email, password: USER.password } });
		expect(r.ok(), 'login via API').toBe(true);
		await use();
	},

	api: async ({ page }, use) => {
		await use(new Api(page.request));
	}
});
