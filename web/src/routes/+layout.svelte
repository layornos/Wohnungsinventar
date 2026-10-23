<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import favicon from '#lib/assets/favicon.svg';
	import { ladeSitzung, sitzung } from '#lib/sitzung.svelte.ts';
	import '../app.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	const oeffentlich = (pfad: string) => pfad === '/anmeldung';

	ladeSitzung();

	$effect(() => {
		if (sitzung.geladen && !sitzung.nutzer && !oeffentlich(page.url.pathname)) {
			goto('/anmeldung', { replaceState: true });
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if sitzung.geladen && (sitzung.nutzer || oeffentlich(page.url.pathname))}
	{@render children()}
{/if}
