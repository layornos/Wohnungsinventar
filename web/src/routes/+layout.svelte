<script lang="ts">
	import { page } from '$app/state';
	import favicon from '#lib/assets/favicon.svg';
	import { LOGIN, INVITATION, toLogin } from '#lib/api.ts';
	import { updateThemeColor } from '#lib/theme.svelte.ts';
	import BottomBar from '#lib/BottomBar.svelte';
	import SideMenu from '#lib/SideMenu.svelte';
	import { loadSession, session } from '#lib/session.svelte.ts';
	import '../fonts.css';
	import '../app.css';
	import '../desktop.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const isPublic = (path: string) => path === LOGIN || path === INVITATION;

	loadSession();

	// Minimal follows the system's light/dark mode; the browser bar should switch along.
	$effect(() => {
		updateThemeColor();
		const dark = matchMedia('(prefers-color-scheme: dark)');
		dark.addEventListener('change', updateThemeColor);
		return () => dark.removeEventListener('change', updateThemeColor);
	});

	$effect(() => {
		if (session.loaded && !session.user && !isPublic(page.url.pathname)) {
			toLogin();
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if session.loaded && (session.user || isPublic(page.url.pathname))}
	<!--
		On wide screens the menu is a sidebar to the left of every page (desktop.css); on narrow screens it is a
		dialog that the ☰ button in the bottom bar opens.
	-->
	<div class="frame">
		{#if session.user && !isPublic(page.url.pathname)}
			<SideMenu kind="sidebar" />
		{/if}
		<div class="frame-content">
			{@render children()}
		</div>
	</div>
	{#if session.user && !isPublic(page.url.pathname)}
		<BottomBar />
		<SideMenu kind="dialog" />
	{/if}
{/if}
