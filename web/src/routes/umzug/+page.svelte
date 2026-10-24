<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, ApiError, boxName, errorText, type ItemListEntry, type Move, type Apartment } from '#lib/api.ts';
	import { loadMove, loadApartments } from '#lib/move.ts';

	let move = $state<Move | null>(null);
	let loaded = $state(false);
	let completed = $state<Move[]>([]);
	let apartments = $state<Apartment[]>([]);
	let oldApartment = $state('');
	let newApartment = $state('');
	let number = $state('');
	let newBox = $state('');
	let withoutTargetRoom = $state<ItemListEntry[] | null>(null);
	let error = $state('');
	let busy = $state(false);

	async function load() {
		try {
			move = await loadMove();
			completed = (await api<{ moves: Move[] }>('GET', '/api/moves')).moves;
			if (!move) apartments = await loadApartments();
		} catch (e) {
			error = errorText(e);
		} finally {
			loaded = true;
		}
	}
	load();

	async function run(action: () => Promise<unknown>) {
		error = '';
		busy = true;
		try {
			await action();
			await load();
			return true;
		} catch (e) {
			error = errorText(e);
			return false;
		} finally {
			busy = false;
		}
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		await run(() => api('POST', '/api/move', { oldApartmentId: Number(oldApartment), newApartmentId: Number(newApartment) }));
	}

	async function createBox(event: SubmitEvent) {
		event.preventDefault();
		if (await run(() => api('POST', '/api/move/boxes', { label: newBox }))) newBox = '';
	}

	async function openBox(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		try {
			const r = await api<{ box: { id: number } }>('GET', `/api/move/boxes/${Number(number)}`);
			await goto(`/boxen/${r.box.id}`);
		} catch (e) {
			error = e instanceof ApiError && e.status === 404 ? `Box ${number} gibt es nicht.` : errorText(e);
		}
	}

	async function complete(confirmed: boolean) {
		if (!confirmed && !confirm('Umzug abschließen? Alle Items mit Zielraum stehen danach ohne Lagerort in ihrem Zielraum.')) return;
		error = '';
		busy = true;
		try {
			await api('POST', '/api/move/completion', { confirmed });
			withoutTargetRoom = null;
			await load();
		} catch (e) {
			if (e instanceof ApiError && e.code === 'items_without_target_room') {
				withoutTargetRoom = e.data.items as ItemListEntry[];
			} else {
				error = errorText(e);
			}
		} finally {
			busy = false;
		}
	}
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		<h1>Umzug</h1>
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if move}
		<p>{move.oldApartment.name} → <strong>{move.newApartment.name}</strong></p>

		<form class="inline large" onsubmit={openBox}>
			<input type="number" inputmode="numeric" min="1" placeholder="Box-Nr." aria-label="Boxnummer" bind:value={number} required />
			<button type="submit">Box öffnen</button>
		</form>

		<nav class="tiles">
			<a class="button" href="/umzug/packen">Packen</a>
			<a class="button secondary" href="/umzug/zielraeume">Nach Zielraum</a>
			<a class="button secondary" href="/umzug/ohne-zielraum">Ohne Zielraum</a>
		</nav>

		<section>
			<h2>Boxen</h2>
			<ul class="rows">
				{#each move.boxes as b (b.id)}
					<li>
						<a href="/boxen/{b.id}">{boxName(b)}</a>
						<span class="muted">{b.count} Items</span>
					</li>
				{:else}
					<li class="muted">Noch keine Boxen.</li>
				{/each}
			</ul>
			<form class="inline" onsubmit={createBox}>
				<input aria-label="Bezeichnung der neuen Box" placeholder="Bezeichnung (optional)" bind:value={newBox} />
				<button type="submit" class="secondary" disabled={busy}>Neue Box</button>
			</form>
		</section>

		<section>
			<h2>Abschluss</h2>
			{#if withoutTargetRoom}
				<p class="error" role="alert">Diese Items der alten Wohnung haben noch keinen Zielraum. Sie bleiben, wo sie sind:</p>
				<ul class="rows">
					{#each withoutTargetRoom as i (i.id)}
						<li><a class="with-image" href="/items/{i.id}"><img src={i.thumbnailUrl} alt="" />{i.name}</a></li>
					{/each}
				</ul>
				<div class="button-row">
					<button disabled={busy} onclick={() => complete(true)}>Trotzdem abschließen</button>
					<button class="secondary" onclick={() => (withoutTargetRoom = null)}>Abbrechen</button>
				</div>
			{:else}
				<button class="secondary" disabled={busy} onclick={() => complete(false)}>Umzug abschließen</button>
			{/if}
		</section>
	{:else if loaded}
		<section>
			<h2>Neuer Umzug</h2>
			{#if apartments.length < 2}
				<p class="muted">Lege zuerst die alte und die neue Wohnung unter <a href="/wohnungen">Wohnungen</a> an.</p>
			{:else}
				<form onsubmit={create}>
					<label>
						Von
						<select bind:value={oldApartment} required>
							<option value="">Alte Wohnung wählen …</option>
							{#each apartments as a (a.id)}<option value={String(a.id)}>{a.name}</option>{/each}
						</select>
					</label>
					<label>
						Nach
						<select bind:value={newApartment} required>
							<option value="">Neue Wohnung wählen …</option>
							{#each apartments as a (a.id)}<option value={String(a.id)} disabled={String(a.id) === oldApartment}>{a.name}</option>{/each}
						</select>
					</label>
					<button type="submit" disabled={busy}>Umzug anlegen</button>
				</form>
			{/if}
		</section>
	{/if}

	{#if completed.length > 0}
		<section>
			<h2>Abgeschlossene Umzüge</h2>
			<ul class="rows">
				{#each completed as m (m.id)}
					<li><a href="/umzuege/{m.id}">{m.oldApartment.name} → {m.newApartment.name}</a></li>
				{/each}
			</ul>
		</section>
	{/if}
</main>
