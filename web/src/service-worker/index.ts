
// Caches only the app shell; API responses are never cached.
import { assets, immutable } from '$app/manifest';
import { version } from '$app/env';

const sw = self as unknown as ServiceWorkerGlobalScope;
const CACHE = `shell-${version}`;
// Paths from the manifest are relative to the base path; the app runs under "/".
const absolute = ({ path }: { path: string }) => '/' + path.replace(/^\//, '');
const SHELL = [...immutable.map(absolute), ...assets.map(absolute), '/'];

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
	event.respondWith(caches.match(event.request).then((hit) => hit ?? fetch(event.request)));
});
