<script lang="ts">
	import { page } from '$app/state';
	import favicon from '#lib/assets/favicon.svg';
	import { ANMELDUNG, EINLADUNG, zurAnmeldung } from '#lib/api.ts';
	import { themenfarbeAnpassen } from '#lib/design.svelte.ts';
	import Seitenmenue from '#lib/Seitenmenue.svelte';
	import { ladeSitzung, sitzung } from '#lib/sitzung.svelte.ts';
	import '../app.css';
	import '../desktop.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const oeffentlich = (pfad: string) => pfad === ANMELDUNG || pfad === EINLADUNG;

	ladeSitzung();

	// Minimal folgt hell/dunkel des Systems; die Browser-Leiste soll mitwechseln.
	$effect(() => {
		themenfarbeAnpassen();
		const dunkel = matchMedia('(prefers-color-scheme: dark)');
		dunkel.addEventListener('change', themenfarbeAnpassen);
		return () => dunkel.removeEventListener('change', themenfarbeAnpassen);
	});

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
	<!-- Auf breiten Bildschirmen steht das Menü als Leiste links neben jeder Seite (desktop.css). -->
	<div class="rahmen">
		{#if sitzung.nutzer && !oeffentlich(page.url.pathname)}
			<Seitenmenue art="leiste" />
		{/if}
		<div class="rahmen-inhalt">
			{@render children()}
		</div>
	</div>
{/if}
