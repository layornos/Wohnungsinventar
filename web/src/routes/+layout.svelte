<script lang="ts">
	import { page } from '$app/state';
	import favicon from '#lib/assets/favicon.svg';
	import { ANMELDUNG, EINLADUNG, zurAnmeldung } from '#lib/api.ts';
	import { ladeSitzung, sitzung } from '#lib/sitzung.svelte.ts';
	import '../app.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const oeffentlich = (pfad: string) => pfad === ANMELDUNG || pfad === EINLADUNG;

	ladeSitzung();

	$effect(() => {
		if (sitzung.geladen && !sitzung.nutzer && !oeffentlich(page.url.pathname)) {
			zurAnmeldung();
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if sitzung.geladen && (sitzung.nutzer || oeffentlich(page.url.pathname))}
	{@render children()}
{/if}
