<script lang="ts" module>
	export type Outcome = 'sold' | 'givenAway';
</script>

<script lang="ts">
	// Marks an item as sold or given away (spec 06), or corrects that in the archive: proceeds (sold only, preset
	// with the asking price) and date (today by default, not in the future). For a piece of furniture it names the
	// storage locations that go away and the items that are then left without storage location.
	import { onMount, untrack } from 'svelte';
	import { api, errorText, type Item, type ItemListEntry } from './api.ts';
	import { euroInput, parseEuro, today } from './money.ts';

	let { item, outcome, done }: { item: Item; outcome: Outcome; done: (item: Item | null) => void } = $props();

	// The dialog starts from the item as it is when opened.
	const start = untrack(() => item);
	const correcting = start.handedOverOn !== null;
	let dialog: HTMLDialogElement;
	let proceeds = $state(euroInput(correcting ? start.saleProceeds : start.askingPrice));
	let date = $state(start.handedOverOn ?? today());
	let unstored = $state<ItemListEntry[]>([]);
	let error = $state('');
	let busy = $state(false);

	onMount(() => {
		dialog.showModal();
		if (!correcting && item.furnitureStorageLocations.length > 0) {
			api<{ items: ItemListEntry[] }>('GET', `/api/items/${item.id}/removal`)
				.then((r) => (unstored = r.items))
				.catch((e) => (error = errorText(e)));
		}
	});

	async function save(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		const body: { outcome: Outcome; date: string; saleProceeds?: number } = { outcome, date };
		if (outcome === 'sold') {
			const cents = parseEuro(proceeds);
			if (cents === null || cents === undefined) {
				error = 'Bitte den Verkaufserlös in Euro angeben, z. B. 45 oder 45,50.';
				return;
			}
			body.saleProceeds = cents;
		}
		busy = true;
		try {
			const saved = await api<Item>('PUT', `/api/items/${item.id}/handover`, body);
			dialog.close();
			done(saved);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	function cancel() {
		dialog.close();
		done(null);
	}
</script>

<dialog class="box-dialog" bind:this={dialog} aria-labelledby="handover-title" oncancel={(e) => (e.preventDefault(), cancel())}>
	<form onsubmit={save}>
		<h2 id="handover-title">
			{correcting ? 'Abgabe korrigieren' : outcome === 'sold' ? `„${item.name}“ verkauft` : `„${item.name}“ verschenkt`}
		</h2>
		{#if outcome === 'sold'}
			<label>
				Verkaufserlös in Euro
				<input bind:value={proceeds} inputmode="decimal" autocomplete="off" required />
			</label>
		{/if}
		<label>
			{outcome === 'sold' ? 'Verkauft am' : 'Verschenkt am'}
			<input type="date" bind:value={date} max={today()} required />
		</label>
		{#if !correcting}
			<p class="muted small-text">
				Das Item kommt ins Archiv und verliert seinen Platz.
				{#if item.furnitureStorageLocations.length > 0}
					Seine Lagerorte ({item.furnitureStorageLocations.map((s) => s.name).join(', ')}) fallen weg.
				{/if}
			</p>
			{#if unstored.length > 0}
				<p class="small-text">
					Danach ohne Lagerort im Raum: {unstored.map((i) => i.name).join(', ')}
				</p>
			{/if}
		{/if}
		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
		<div class="button-row">
			<button type="submit" disabled={busy}>{correcting ? 'Speichern' : 'Ins Archiv'}</button>
			<button type="button" class="secondary" onclick={cancel}>Abbrechen</button>
		</div>
	</form>
</dialog>
