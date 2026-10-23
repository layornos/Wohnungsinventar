<script lang="ts">
	import { api, fehlerText, type Wohnung } from '#lib/api.ts';

	let wohnungen = $state<Wohnung[]>([]);
	let neueWohnung = $state('');
	let neuerRaum = $state<Record<number, string>>({});
	let umbenennen = $state<{ id: number; name: string } | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);

	async function neuLaden() {
		try {
			wohnungen = (await api<{ wohnungen: Wohnung[] }>('GET', '/api/wohnungen')).wohnungen;
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
	neuLaden();

	async function ausfuehren(aktion: () => Promise<unknown>) {
		fehler = '';
		laeuft = true;
		try {
			await aktion();
			await neuLaden();
			return true;
		} catch (e) {
			fehler = fehlerText(e);
			return false;
		} finally {
			laeuft = false;
		}
	}

	async function wohnungAnlegen(event: SubmitEvent) {
		event.preventDefault();
		if (await ausfuehren(() => api('POST', '/api/wohnungen', { name: neueWohnung }))) neueWohnung = '';
	}

	async function raumAnlegen(event: SubmitEvent, wohnung: number) {
		event.preventDefault();
		if (await ausfuehren(() => api('POST', `/api/wohnungen/${wohnung}/raeume`, { name: neuerRaum[wohnung] ?? '' })))
			neuerRaum[wohnung] = '';
	}

	async function umbenennenSpeichern(event: SubmitEvent) {
		event.preventDefault();
		const u = umbenennen;
		if (u && (await ausfuehren(() => api('PUT', `/api/wohnungen/${u.id}`, { name: u.name })))) umbenennen = null;
	}

	function wohnungLoeschen(w: Wohnung) {
		if (!confirm(`Wohnung „${w.name}“ löschen?`)) return;
		ausfuehren(() => api('DELETE', `/api/wohnungen/${w.id}`));
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/">← Liste</a>
		<h1>Wohnungen</h1>
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	{#each wohnungen as w (w.id)}
		<section>
			{#if umbenennen?.id === w.id}
				<form class="inline" onsubmit={umbenennenSpeichern}>
					<input aria-label="Neuer Name" bind:value={umbenennen.name} required />
					<button type="submit" disabled={laeuft}>Speichern</button>
					<button type="button" class="sekundaer" onclick={() => (umbenennen = null)}>Abbrechen</button>
				</form>
			{:else}
				<div class="kopf">
					<h2>{w.name}</h2>
					<span class="knopfzeile">
						<button class="sekundaer klein" onclick={() => (umbenennen = { id: w.id, name: w.name })}>Umbenennen</button>
						<button class="sekundaer klein" disabled={laeuft} onclick={() => wohnungLoeschen(w)}>Löschen</button>
					</span>
				</div>
			{/if}
			<ul class="zeilen">
				{#each w.raeume as r (r.id)}
					<li><a href="/raeume/{r.id}">{r.name}</a></li>
				{:else}
					<li class="leise">Noch keine Räume.</li>
				{/each}
			</ul>
			<form class="inline" onsubmit={(e) => raumAnlegen(e, w.id)}>
				<input aria-label="Name des neuen Raums in {w.name}" placeholder="Neuer Raum, z. B. Küche" bind:value={neuerRaum[w.id]} required />
				<button type="submit" class="sekundaer" disabled={laeuft}>Raum anlegen</button>
			</form>
		</section>
	{/each}

	<section>
		<h2>Neue Wohnung</h2>
		<form class="inline" onsubmit={wohnungAnlegen}>
			<input aria-label="Name der neuen Wohnung" placeholder="z. B. Altbau Weststadt" bind:value={neueWohnung} required />
			<button type="submit" disabled={laeuft}>Anlegen</button>
		</form>
	</section>
</main>
