<script lang="ts">
	// "Inserat" on the item page (spec 06): title and text for any selling platform, built by the server from the
	// household's template with the warranty disclaimer. Copy, share and the photos to upload; without an asking
	// price there is nothing to copy yet. Below it the actions for when the item is gone.
	import { api, ApiError, errorText, forSale, type Item } from './api.ts';
	import type { Outcome } from './HandoverDialog.svelte';

	let { item, handOver }: { item: Item; handOver: (outcome: Outcome) => void } = $props();

	let listing = $state<{ title: string; text: string } | null>(null);
	let missingPrice = $state(false);
	let error = $state('');
	let copied = $state('');
	const canShare = typeof navigator.share === 'function';

	// Reload whenever the item changes, e.g. after a new price.
	$effect(() => {
		void item.updatedAt;
		listing = null;
		missingPrice = false;
		error = '';
		api<{ title: string; text: string }>('GET', `/api/items/${item.id}/listing`)
			.then((l) => (listing = l))
			.catch((e) => {
				if (e instanceof ApiError && e.code === 'no_asking_price') missingPrice = true;
				else error = errorText(e);
			});
	});

	async function copy(what: 'Titel' | 'Text', value: string) {
		try {
			await navigator.clipboard.writeText(value);
			copied = `${what} kopiert.`;
		} catch {
			copied = '';
			error = 'Kopieren ging nicht; bitte den Text markieren und kopieren.';
		}
	}

	async function share() {
		if (!listing) return;
		try {
			await navigator.share({ title: listing.title, text: `${listing.title}\n\n${listing.text}` });
		} catch {
			// Closed by the user or not possible; nothing to do.
		}
	}
</script>

<section class="listing" aria-labelledby="listing-title">
	<h2 id="listing-title">Inserat</h2>
	{#if missingPrice}
		<p class="notice">Erst einen Preis eintragen: unter „Bearbeiten“ beim Angebotspreis.</p>
	{:else if listing}
		<label>
			Titel
			<input readonly value={listing.title} onfocus={(e) => e.currentTarget.select()} />
		</label>
		<label>
			Text
			<textarea readonly rows="8" value={listing.text} onfocus={(e) => e.currentTarget.select()}></textarea>
		</label>
		<div class="button-row">
			<button class="secondary" onclick={() => listing && copy('Titel', listing.title)}>Titel kopieren</button>
			<button class="secondary" onclick={() => listing && copy('Text', listing.text)}>Text kopieren</button>
			{#if canShare}<button class="secondary" onclick={share}>Teilen</button>{/if}
		</div>
		{#if copied}<p class="muted small-text" role="status">{copied}</p>{/if}
		<p class="small-text">
			Fotos zum Hochladen:
			{#each item.photos as p, i (p.id)}{i > 0 ? ' · ' : ' '}<a href={p.url} download="{item.name} {i + 1}">Foto {i + 1}</a>{/each}
		</p>
	{/if}
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	<div class="button-row">
		{#if forSale(item.disposition)}
			<button onclick={() => handOver('sold')}>Verkauft</button>
		{:else}
			<button onclick={() => handOver('givenAway')}>Verschenkt</button>
		{/if}
	</div>
</section>
