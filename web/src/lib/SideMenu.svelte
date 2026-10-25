<script lang="ts">
	// Side menu in two forms with the same content, both rendered once by the layout for every logged-in page:
	// - "dialog" (narrow screens): opens via the floating ☰ button (MenuButton.svelte), closes via a swipe to the
	//   left, a tap beside it or Escape. The modal <dialog> keeps focus in the menu and returns it on close.
	//   There is deliberately no edge swipe to open it: in the installed app on iOS that is the system's back gesture.
	// - "sidebar" (wide screens): permanently visible sidebar on the left.
	// desktop.css decides which form is visible.
	import { page } from '$app/state';
	import { toLogin } from './api.ts';
	import MenuIcon, { type MenuIconName } from './MenuIcon.svelte';
	import { logout, session } from './session.svelte.ts';
	import { registerSideMenu, sideMenu } from './sideMenu.svelte.ts';

	let { kind = 'dialog' }: { kind?: 'dialog' | 'sidebar' } = $props();

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

	// When the window becomes wide, the sidebar takes over (desktop.css decides when); an open dialog on top would
	// show the menu twice.
	$effect(() => {
		if (kind !== 'dialog') return;
		const closeForSidebar = () => {
			const sidebar = document.querySelector('nav.sidebar');
			if (sideMenu.open && sidebar && getComputedStyle(sidebar).display !== 'none') close();
		};
		addEventListener('resize', closeForSidebar);
		return () => removeEventListener('resize', closeForSidebar);
	});

	function swipeStart(e: TouchEvent) {
		const t = e.touches[0];
		start = { x: t.clientX, y: t.clientY };
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

{#snippet entry(href: string, name: string, iconName: MenuIconName)}
	<li>
		<a class="menu-link" {href} aria-current={isActive(href) ? 'page' : undefined} onclick={close}>
			<MenuIcon name={iconName} /><span>{name}</span>
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
					<MenuIcon name="logout" /><span>Abmelden</span>
				</button>
			</li>
		</ul>
		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
	</div>
{/snippet}

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
		ontouchstart={swipeStart}
		ontouchend={(e) => {
			if (swipeDistance(e) <= -MIN_DISTANCE) close();
		}}
	>
		<nav class="menu-content">
			{@render menuContent()}
		</nav>
	</dialog>
{/if}
