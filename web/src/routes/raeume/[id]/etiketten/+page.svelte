<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText, type Room } from '#lib/api.ts';
	import StorageLocationLabel from '#lib/StorageLocationLabel.svelte';
	import { labelsOfRoom } from '#lib/storageLocationLabels.ts';

	// Labels of all storage locations of a room; printing puts every label on a page of its own.

	const id = $derived(Number(page.params.id));

	let room = $state<Room | null>(null);
	let error = $state('');

	const labels = $derived(room ? labelsOfRoom(room) : []);

	$effect(() => {
		const roomID = id;
		room = null;
		error = '';
		api<Room>('GET', `/api/rooms/${roomID}`)
			.then((r) => (room = r))
			.catch((e) => (error = errorText(e)));
	});
</script>

<main class="page">
	<header class="header no-print">
		<a class="button secondary" href="/raeume/{id}">← {room?.name ?? 'Raum'}</a>
		<h1>Etiketten</h1>
	</header>

	{#if error}
		<p class="error no-print" role="alert">{error}</p>
	{/if}

	{#if labels.length > 0}
		<div class="button-row no-print label-actions">
			<button onclick={() => window.print()}>Alle {labels.length} drucken</button>
		</div>
		<div class="box-labels">
			{#each labels as l (l.id)}<StorageLocationLabel label={l} />{/each}
		</div>
	{:else if room}
		<p class="muted">Dieser Raum hat noch keine Lagerorte.</p>
	{:else if !error}
		<p class="muted">Lädt …</p>
	{/if}
</main>
