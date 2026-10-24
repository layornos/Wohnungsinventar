<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText, type StorageLocation } from '#lib/api.ts';
	import LabelActions from '#lib/LabelActions.svelte';
	import StorageLocationLabel from '#lib/StorageLocationLabel.svelte';
	import { labelOf, storageLocationLink } from '#lib/storageLocationLabels.ts';

	const id = $derived(Number(page.params.id));

	let location = $state<StorageLocation | null>(null);
	let error = $state('');

	$effect(() => {
		const locationID = id;
		location = null;
		error = '';
		api<StorageLocation>('GET', `/api/storage-locations/${locationID}`)
			.then((l) => (location = l))
			.catch((e) => (error = errorText(e)));
	});
</script>

<main class="page">
	<header class="header no-print">
		<a class="button secondary" href="/lagerorte/{id}">← Lagerort</a>
		<h1>Etikett</h1>
	</header>

	{#if error}
		<p class="error no-print" role="alert">{error}</p>
	{/if}

	{#if location}
		<StorageLocationLabel label={labelOf(location.id, location.path)} />
		<LabelActions link={storageLocationLink(location.id)} />
	{:else if !error}
		<p class="muted">Lädt …</p>
	{/if}
</main>
