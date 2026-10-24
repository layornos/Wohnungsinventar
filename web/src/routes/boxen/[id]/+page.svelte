<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, boxName, errorText, unpackPath, type Box, type Named, type PackedItem } from '#lib/api.ts';
	import PackingList from '#lib/PackingList.svelte';
	import { loadMove, loadMoveRooms, type BoxChangeBody } from '#lib/move.ts';

	const id = $derived(Number(page.params.id));

	let box = $state<Box | null>(null);
	let items = $state<PackedItem[]>([]);
	let active = $state(false);
	let newRooms = $state<Named[]>([]);
	// Label and target room while editing.
	let draft = $state<{ label: string; targetRoom: string } | null>(null);
	let error = $state('');
	let busy = $state(false);

	async function load(boxID: number) {
		error = '';
		try {
			const r = await api<{ box: Box; items: PackedItem[] }>('GET', `/api/boxes/${boxID}`);
			box = r.box;
			items = r.items;
			const move = await loadMove();
			active = !!move?.boxes.some((b) => b.id === boxID);
			if (move && active) newRooms = (await loadMoveRooms(move)).new;
		} catch (e) {
			error = errorText(e);
		}
	}

	$effect(() => {
		load(id);
	});

	function startEditing() {
		if (!box) return;
		draft = { label: box.label, targetRoom: box.targetRoom ? String(box.targetRoom.id) : '' };
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (!draft) return;
		error = '';
		busy = true;
		try {
			const body: BoxChangeBody = { label: draft.label.trim(), targetRoomId: draft.targetRoom ? Number(draft.targetRoom) : null };
			box = await api<Box>('PUT', `/api/boxes/${id}`, body);
			draft = null;
			await load(id);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	async function remove() {
		if (!box || !confirm(`${boxName(box)} löschen?`)) return;
		try {
			await api('DELETE', `/api/boxes/${id}`);
			await goto('/umzug', { replaceState: true });
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/umzug">← Umzug</a>
		{#if box}<a class="button secondary" href="/boxen/{box.id}/etikett">Etikett</a>{/if}
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if box}
		{#if draft}
			<form onsubmit={save}>
				<h1>Box {box.number}</h1>
				<label>
					Bezeichnung
					<input bind:value={draft.label} placeholder="z. B. Box A, Bücher" />
				</label>
				<label>
					Zielraum
					<select bind:value={draft.targetRoom}>
						<option value="">Noch offen</option>
						{#each newRooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
					</select>
				</label>
				<div class="button-row">
					<button type="submit" disabled={busy}>Speichern</button>
					<button type="button" class="secondary" onclick={() => (draft = null)}>Abbrechen</button>
				</div>
			</form>
		{:else}
			<h1 class="box-number">Box {box.number}</h1>
			{#if box.label}<p class="box-label">{box.label}</p>{/if}
			<p>→ {box.targetRoom?.name ?? 'Zielraum noch offen'}</p>
			{#if active}
				<a class="button unpack-button" href={unpackPath(box.id)}>Auspacken</a>
				<div class="button-row">
					<a class="button secondary" href="/umzug/packen?box={box.id}">In diese Box packen</a>
					<button class="secondary small" onclick={startEditing}>Bearbeiten</button>
					{#if items.length === 0}<button class="secondary small" onclick={remove}>Löschen</button>{/if}
				</div>
			{:else}
				<p class="muted">Gehört zu einem abgeschlossenen Umzug.</p>
			{/if}
		{/if}

		<section>
			<h2>Packliste ({items.length})</h2>
			<PackingList {items} />
		</section>
	{/if}
</main>
