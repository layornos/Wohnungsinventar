// Notices when a new version of the app has been deployed and switches to it on request (UpdateNotice.svelte).
//
// The installed app on iOS stays in memory for days and only ever navigates client-side, so the browser itself
// rarely looks for a new service worker; and a new one waits until every page of the old one is closed. Hence:
// look for updates on start and whenever the app comes back to the foreground, offer a waiting worker as
// "Neue Version verfügbar", and on "Neu laden" let it take over and reload once it controls the page.

/** Whether a new version is installed and waits for "Neu laden". */
export const appUpdate = $state<{ available: boolean }>({ available: false });

/** At most one update check per minute, however often the app is brought back to the foreground. */
const CHECK_INTERVAL_MS = 60_000;

let registration: ServiceWorkerRegistration | null = null;
let waiting: ServiceWorker | null = null;
let lastCheck = 0;
// Set only by applyUpdate, so a controller change caused elsewhere (another tab) never reloads by itself.
let reloading = false;

function offer(worker: ServiceWorker) {
	// Without a controller this is the first install; there is no old version to replace.
	if (!navigator.serviceWorker.controller) return;
	waiting = worker;
	appUpdate.available = true;
}

function track(worker: ServiceWorker | null) {
	if (!worker) return;
	if (worker.state === 'installed') {
		offer(worker);
		return;
	}
	worker.addEventListener('statechange', () => {
		if (worker.state === 'installed') offer(worker);
	});
}

function check() {
	if (!registration || Date.now() - lastCheck < CHECK_INTERVAL_MS) return;
	lastCheck = Date.now();
	// Offline or server unreachable: try again on the next return to the app.
	registration.update().catch(() => {});
}

/** Starts watching for new versions; call once from the layout inside $effect. Returns the cleanup. */
export function watchForUpdates(): () => void {
	if (!('serviceWorker' in navigator)) return () => {};
	const container = navigator.serviceWorker;
	let stopped = false;

	const onUpdateFound = () => track(registration?.installing ?? null);
	const onVisibilityChange = () => {
		if (document.visibilityState === 'visible') check();
	};
	const onControllerChange = () => {
		if (reloading) location.reload();
	};

	// SvelteKit registers the worker on "load"; ready resolves once a registration has an active worker.
	container.ready.then((reg) => {
		if (stopped) return;
		registration = reg;
		reg.addEventListener('updatefound', onUpdateFound);
		// An update may already be waiting (found before this page loaded) or installing right now.
		track(reg.waiting);
		track(reg.installing);
		check();
	});
	document.addEventListener('visibilitychange', onVisibilityChange);
	container.addEventListener('controllerchange', onControllerChange);

	return () => {
		stopped = true;
		registration?.removeEventListener('updatefound', onUpdateFound);
		document.removeEventListener('visibilitychange', onVisibilityChange);
		container.removeEventListener('controllerchange', onControllerChange);
	};
}

/** Lets the waiting version take over; the page reloads as soon as it controls it ("controllerchange"). */
export function applyUpdate() {
	if (reloading) return;
	reloading = true;
	if (waiting && waiting.state === 'installed') {
		waiting.postMessage({ type: 'skip-waiting' });
	} else {
		// Already activated, e.g. from another tab: the page only needs to load the new version.
		location.reload();
	}
}
