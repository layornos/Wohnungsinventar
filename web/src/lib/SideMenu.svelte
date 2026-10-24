<script lang="ts">
	// Side menu in two forms with the same content, both rendered once by the layout for every logged-in page:
	// - "dialog" (narrow screens): opens via the ☰ button in the page header (MenuButton.svelte) or a swipe from the
	//   left edge, closes via a swipe to the left, a tap beside it or Escape. The modal <dialog> keeps focus in the
	//   menu and returns it on close.
	// - "sidebar" (wide screens): permanently visible sidebar on the left.
	// desktop.css decides which form is visible; the swipe only works while the sidebar is not visible.
	import { page } from '$app/state';
	import { MediaQuery } from 'svelte/reactivity';
	import { toLogin } from './api.ts';
	import { logout, session } from './session.svelte.ts';
	import { registerSideMenu, sideMenu } from './sideMenu.svelte.ts';

	let { kind = 'dialog' }: { kind?: 'dialog' | 'sidebar' } = $props();

	// Must match the breakpoint in desktop.css.
	const wide = new MediaQuery('min-width: 1024px');

	// A swipe only starts this close to the edge, so normal scrolling and swiping in the content isn't disturbed.
	const EDGE = 24;
	const MIN_DISTANCE = 60;

	let dialog = $state<HTMLDialogElement>();
	let start: { x: number; y: number } | null = null;
	let error = $state('');

	function open() {
		if (sideMenu.open || !dialog) return;
		dialog.showModal();
		sideMenu.open = true;
	}

	function close() {
		dialog?.close();
	}

	$effect(() => {
		if (kind === 'dialog') return registerSideMenu(open);
	});

	// When the window becomes wide, the sidebar takes over; an open dialog on top would show the menu twice.
	$effect(() => {
		if (wide.current && sideMenu.open) close();
	});

	function swipeStart(e: TouchEvent, onlyAtEdge: boolean) {
		const t = e.touches[0];
		start = !onlyAtEdge || t.clientX <= EDGE ? { x: t.clientX, y: t.clientY } : null;
	}

	// Returns the horizontal distance if the swipe was more horizontal than vertical, otherwise 0.
	function swipeDistance(e: TouchEvent) {
		if (!start) return 0;
		const t = e.changedTouches[0];
		const dx = t.clientX - start.x;
		const dy = t.clientY - start.y;
		start = null;
		return Math.abs(dx) > Math.abs(dy) ? dx : 0;
	}

	// Sub-pages count too, so that e.g. the packing mode is highlighted under „Umzug“.
	function isActive(href: string) {
		const p = page.url.pathname;
		return p === href || p.startsWith(href + '/');
	}

	async function logoutAndLeave() {
		error = '';
		try {
			await logout();
		} catch {
			// Without a response from the server the session stays there; don't pretend to be logged out.
			error = 'Abmelden nicht möglich. Ist der Server erreichbar?';
			return;
		}
		close();
		await toLogin();
	}
</script>

{#snippet icon(name: string)}
	<svg class="menu-icon" aria-hidden="true" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
		{#if name === 'inventory'}<rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" /><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" />
		{:else if name === 'unstored'}<path d="M12 21s-7-6.2-7-11.5a7 7 0 0 1 14 0C19 14.8 12 21 12 21z" /><path d="M9.5 9.5l5 5M14.5 9.5l-5 5" />
		{:else if name === 'apartments'}<path d="M3 10.5 12 3l9 7.5" /><path d="M5 9v12h14V9" /><path d="M10 21v-6h4v6" />
		{:else if name === 'move'}<path d="M3 7h11v10H3z" /><path d="M14 10h4l3 3v4h-7" /><circle cx="7" cy="18" r="2" /><circle cx="17" cy="18" r="2" />
		{:else if name === 'persons'}<circle cx="9" cy="8" r="3.5" /><path d="M2.5 20a6.5 6.5 0 0 1 13 0" /><path d="M16 4.5a3.5 3.5 0 0 1 0 7M18 14.2a6.5 6.5 0 0 1 3.5 5.8" />
		{:else if name === 'users'}<rect x="4" y="10" width="16" height="11" rx="2" /><path d="M8 10V7a4 4 0 0 1 8 0v3" />
		{:else if name === 'settings'}<circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" />
		{:else if name === 'logout'}<path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3" /><path d="M10 17l-5-5 5-5M5 12h11" />
		{/if}
	</svg>
{/snippet}

{#snippet entry(href: string, name: string, iconName: string)}
	<li>
		<a class="menu-link" {href} aria-current={isActive(href) ? 'page' : undefined} onclick={close}>
			{@render icon(iconName)}<span>{name}</span>
		</a>
	</li>
{/snippet}

{#snippet menuContent()}
	<div class="menu-header">
		<span class="menu-brand">Inventar</span>
		{#if kind === 'dialog'}
			<button type="button" class="menu-close" aria-label="Menü schließen" onclick={close}>
				<svg aria-hidden="true" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M6 6l12 12M18 6 6 18" /></svg>
			</button>
		{/if}
	</div>

	<ul class="menu-group">
		{@render entry('/', 'Alle Items', 'inventory')}
		{@render entry('/ohne-lagerort', 'Ohne Lagerort', 'unstored')}
	</ul>

	<h2 class="menu-title" id="menu-{kind}-places">Orte</h2>
	<ul class="menu-group" aria-labelledby="menu-{kind}-places">
		{@render entry('/wohnungen', 'Wohnungen', 'apartments')}
		{@render entry('/umzug', 'Umzug', 'move')}
	</ul>

	<h2 class="menu-title" id="menu-{kind}-household">Haushalt</h2>
	<ul class="menu-group" aria-labelledby="menu-{kind}-household">
		{@render entry('/personen', 'Personen', 'persons')}
		{@render entry('/nutzer', 'Nutzer', 'users')}
	</ul>

	<div class="menu-footer">
		{#if session.user}
			<p class="menu-user">
				<span class="menu-user-name">{session.user.name}</span>
				<span class="menu-user-email">{session.user.email}</span>
			</p>
		{/if}
		<ul class="menu-group">
			{@render entry('/einstellungen', 'Einstellungen', 'settings')}
			<li>
				<button type="button" class="menu-link" onclick={logoutAndLeave}>
					{@render icon('logout')}<span>Abmelden</span>
				</button>
			</li>
		</ul>
		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
	</div>
{/snippet}

<svelte:window
	ontouchstart={(e) => {
		if (kind === 'dialog' && !sideMenu.open && !wide.current) swipeStart(e, true);
	}}
	ontouchend={(e) => {
		if (kind === 'dialog' && !sideMenu.open && !wide.current && swipeDistance(e) >= MIN_DISTANCE) open();
	}}
/>

{#if kind === 'sidebar'}
	<nav class="menu-content sidebar" aria-label="Menü">
		{@render menuContent()}
	</nav>
{:else}
	<dialog
		id="side-menu"
		class="side-menu"
		aria-label="Menü"
		bind:this={dialog}
		onclose={() => (sideMenu.open = false)}
		onclick={(e) => {
			// Clicks on the dimmed backdrop land directly on the <dialog>, not on its content.
			if (e.target === dialog) close();
		}}
		ontouchstart={(e) => swipeStart(e, false)}
		ontouchend={(e) => {
			if (swipeDistance(e) <= -MIN_DISTANCE) close();
		}}
	>
		<nav class="menu-content">
			{@render menuContent()}
		</nav>
	</dialog>
{/if}
