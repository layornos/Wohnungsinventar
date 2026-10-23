
// Speichert nur die App-Hülle zwischen; API-Antworten werden nie zwischengespeichert.
import { assets, immutable } from '$app/manifest';
import { version } from '$app/env';

const sw = self as unknown as ServiceWorkerGlobalScope;
const CACHE = `huelle-${version}`;
// Pfade aus dem Manifest sind relativ zum Basispfad; die App läuft unter "/".
const absolut = ({ path }: { path: string }) => '/' + path.replace(/^\//, '');
const HUELLE = [...immutable.map(absolut), ...assets.map(absolut), '/'];

sw.addEventListener('install', (event) => {
	event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(HUELLE)));
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
		// Seiten zuerst vom Netz, offline die zwischengespeicherte Hülle.
		event.respondWith(fetch(event.request).catch(async () => (await caches.match('/')) ?? Response.error()));
		return;
	}
	event.respondWith(caches.match(event.request).then((hit) => hit ?? fetch(event.request)));
});
