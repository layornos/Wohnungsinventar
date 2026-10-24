<script lang="ts">
	import { errorText, api, type Apartment } from '#lib/api.ts';
	import { loadApartments } from '#lib/move.ts';

	let apartments = $state<Apartment[]>([]);
	let newApartment = $state('');
	let newRoom = $state<Record<number, string>>({});
	let renaming = $state<{ id: number; name: string } | null>(null);
	let error = $state('');
	let busy = $state(false);

	async function reload() {
		try {
			apartments = await loadApartments();
		} catch (e) {
			error = errorText(e);
		}
	}
	reload();

	async function run(action: () => Promise<unknown>) {
		error = '';
		busy = true;
		try {
			await action();
			await reload();
			return true;
		} catch (e) {
			error = errorText(e);
			return false;
		} finally {
			busy = false;
		}
	}

	async function createApartment(event: SubmitEvent) {
		event.preventDefault();
		if (await run(() => api('POST', '/api/apartments', { name: newApartment }))) newApartment = '';
	}

	async function createRoom(event: SubmitEvent, apartment: number) {
		event.preventDefault();
		if (await run(() => api('POST', `/api/apartments/${apartment}/rooms`, { name: newRoom[apartment] ?? '' })))
			newRoom[apartment] = '';
	}

	async function saveRename(event: SubmitEvent) {
		event.preventDefault();
		const r = renaming;
		if (r && (await run(() => api('PUT', `/api/apartments/${r.id}`, { name: r.name })))) renaming = null;
	}

	function removeApartment(a: Apartment) {
		if (!confirm(`Wohnung „${a.name}“ löschen?`)) return;
		run(() => api('DELETE', `/api/apartments/${a.id}`));
	}
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		<h1>Wohnungen</h1>
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#each apartments as a (a.id)}
		<section>
			{#if renaming?.id === a.id}
				<form class="inline" onsubmit={saveRename}>
					<input aria-label="Neuer Name" bind:value={renaming.name} required />
					<button type="submit" disabled={busy}>Speichern</button>
					<button type="button" class="secondary" onclick={() => (renaming = null)}>Abbrechen</button>
				</form>
			{:else}
				<div class="header">
					<h2>{a.name}</h2>
					<span class="button-row">
						<button class="secondary small" onclick={() => (renaming = { id: a.id, name: a.name })}>Umbenennen</button>
						<button class="secondary small" disabled={busy} onclick={() => removeApartment(a)}>Löschen</button>
					</span>
				</div>
			{/if}
			<ul class="rows">
				{#each a.rooms as r (r.id)}
					<li><a href="/raeume/{r.id}">{r.name}</a></li>
				{:else}
					<li class="muted">Noch keine Räume.</li>
				{/each}
			</ul>
			<form class="inline" onsubmit={(e) => createRoom(e, a.id)}>
				<input aria-label="Name des neuen Raums in {a.name}" placeholder="Neuer Raum, z. B. Küche" bind:value={newRoom[a.id]} required />
				<button type="submit" class="secondary" disabled={busy}>Raum anlegen</button>
			</form>
		</section>
	{/each}

	<section>
		<h2>Neue Wohnung</h2>
		<form class="inline" onsubmit={createApartment}>
			<input aria-label="Name der neuen Wohnung" placeholder="z. B. Altbau Weststadt" bind:value={newApartment} required />
			<button type="submit" disabled={busy}>Anlegen</button>
		</form>
	</section>
</main>
