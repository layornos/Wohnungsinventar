<script lang="ts">
	import { page } from '$app/state';
	import { api, boxName, formatDate, errorText, type Move } from '#lib/api.ts';
	import MenuButton from '#lib/MenuButton.svelte';

	let move = $state<Move | null>(null);
	let error = $state('');

	$effect(() => {
		api<Move>('GET', `/api/moves/${page.params.id}`)
			.then((m) => (move = m))
			.catch((e) => (error = errorText(e)));
	});
</script>

<main class="page">
	<header class="header">
		<MenuButton />
		<a class="button secondary" href="/umzug">← Umzug</a>
	</header>
	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}
	{#if move}
		<h1>{move.oldApartment.name} → {move.newApartment.name}</h1>
		<p class="muted">
			Angelegt am {formatDate(move.createdAt)}{#if move.completedAt}, abgeschlossen am {formatDate(move.completedAt)}{/if}
		</p>
		<ul class="rows">
			{#each move.boxes as b (b.id)}
				<li><a href="/boxen/{b.id}">{boxName(b)}</a> <span class="muted">{b.count} Items</span></li>
			{:else}
				<li class="muted">Keine Boxen.</li>
			{/each}
		</ul>
	{/if}
</main>
