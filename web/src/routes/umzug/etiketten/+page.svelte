<script lang="ts">
	import { errorText, type Move } from '#lib/api.ts';
	import BoxLabel from '#lib/BoxLabel.svelte';
	import { loadMove } from '#lib/move.ts';

	// All labels of the active move; printing puts every label on a page of its own.

	let move = $state<Move | null>(null);
	let loaded = $state(false);
	let error = $state('');

	const boxes = $derived([...(move?.boxes ?? [])].sort((a, b) => a.number - b.number));

	loadMove()
		.then((m) => (move = m))
		.catch((e) => (error = errorText(e)))
		.finally(() => (loaded = true));
</script>

<main class="page">
	<header class="header no-print">
		<a class="button secondary" href="/umzug">← Umzug</a>
		<h1>Etiketten</h1>
	</header>

	{#if error}
		<p class="error no-print" role="alert">{error}</p>
	{/if}

	{#if boxes.length > 0}
		<div class="button-row no-print label-actions">
			<button onclick={() => window.print()}>Alle {boxes.length} drucken</button>
		</div>
		<div class="box-labels">
			{#each boxes as b (b.id)}<BoxLabel box={b} />{/each}
		</div>
	{:else if loaded && !error}
		<p class="muted">Noch keine Boxen.</p>
	{/if}
</main>
