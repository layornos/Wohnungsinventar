// Caches the app shell, and fonts on first use; API responses are never cached.
// Caches only the app shell; API responses are never cached.
import { assets, immutable } from '$app/manifest';
import { version } from '$app/env';

const sw = self as unknown as ServiceWorkerGlobalScope;
const CACHE = `shell-${version}`;
// Paths from the manifest are relative to the base path; the app runs under "/".
const absolute = ({ path }: { path: string }) => '/' + path.replace(/^\//, '');
// Fonts (and their licences under /fonts/) are not precached: each theme needs only its own font, which is
// cached on first use below.
const isFont = (path: string) => path.endsWith('.woff2') || path.startsWith('/fonts/');
const SHELL = [...immutable.map(absolute), ...assets.map(absolute), '/'].filter((path) => !isFont(path));

sw.addEventListener('install', (event) => {
	event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(SHELL)));
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
	);
});

sw.addEventListener('fetch', (event) => {
	const url = new URL(event.request.url);
	if (event.request.method !== 'GET' || url.origin !== sw.location.origin || url.pathname.startsWith('/api/')) {
		return;
	}
	if (event.request.mode === 'navigate') {
		// Pages from the network first, offline the cached shell.
		event.respondWith(fetch(event.request).catch(async () => (await caches.match('/')) ?? Response.error()));
		return;
	}
	if (url.pathname.endsWith('.woff2')) {
		event.respondWith(cacheOnFirstUse(event.request));
		return;
	}
	event.respondWith(caches.match(event.request).then((hit) => hit ?? fetch(event.request)));
});

// Font files have hashed names, so a cached copy stays valid until the next version clears the cache.
async function cacheOnFirstUse(request: Request): Promise<Response> {
	const hit = await caches.match(request);
	if (hit) return hit;
	const response = await fetch(request);
	if (response.ok) {
		const copy = response.clone();
		caches.open(CACHE).then((cache) => cache.put(request, copy));
	}
	return response;
}
