<script lang="ts">
	import { api, ApiError, fehlerText, type Person } from '#lib/api.ts';
	import { ladePersonen } from '#lib/personen.ts';

	let personen = $state<Person[]>([]);
	let neu = $state('');
	let fehler = $state('');
	let laeuft = $state(false);
	let umbenennen = $state<{ id: number; name: string } | null>(null);

	async function neuLaden() {
		try {
			personen = await ladePersonen();
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
	neuLaden();

	async function ausfuehren(aktion: () => Promise<unknown>, meldung: (e: unknown) => string = fehlerText) {
		fehler = '';
		laeuft = true;
		try {
			await aktion();
			await neuLaden();
			return true;
		} catch (e) {
			fehler = meldung(e);
			return false;
		} finally {
			laeuft = false;
		}
	}

	async function anlegen(event: SubmitEvent) {
		event.preventDefault();
		if (await ausfuehren(() => api('POST', '/api/personen', { name: neu }))) neu = '';
	}

	async function umbenennenSpeichern(event: SubmitEvent) {
		event.preventDefault();
		const u = umbenennen;
		if (u && (await ausfuehren(() => api('PUT', `/api/personen/${u.id}`, { name: u.name })))) umbenennen = null;
	}

	function loeschen(p: Person) {
		if (!confirm(`${p.name} löschen?`)) return;
		ausfuehren(
			() => api('DELETE', `/api/personen/${p.id}`),
			(e) =>
				e instanceof ApiError && e.code === 'letzter_besitzer'
					? `${p.name} ist bei mindestens einem Item einziger Besitzer. Gib diesen Items zuerst einen anderen Besitzer.`
					: fehlerText(e)
		);
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/">← Liste</a>
		<h1>Personen</h1>
	</header>

	<p class="leise">Personen können Items besitzen. Wer sich anmelden kann, hat einen Nutzerzugang.</p>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	<ul class="zeilen">
		{#each personen as p (p.id)}
			<li>
				{#if umbenennen?.id === p.id}
					<form class="inline" onsubmit={umbenennenSpeichern}>
						<input aria-label="Neuer Name" bind:value={umbenennen.name} required />
						<button type="submit" disabled={laeuft}>Speichern</button>
						<button type="button" class="sekundaer" onclick={() => (umbenennen = null)}>Abbrechen</button>
					</form>
				{:else}
					<span>
						{p.name}
						{#if p.nutzer}<span class="marke">Nutzer</span>{/if}
					</span>
					<span class="knopfzeile">
						<button class="sekundaer klein" onclick={() => (umbenennen = { id: p.id, name: p.name })}>Umbenennen</button>
						{#if !p.nutzer}
							<button class="sekundaer klein" disabled={laeuft} onclick={() => loeschen(p)}>Löschen</button>
						{/if}
					</span>
				{/if}
			</li>
		{/each}
	</ul>

	<form class="inline" onsubmit={anlegen}>
		<input aria-label="Name der neuen Person" placeholder="Name, z. B. Kim" bind:value={neu} required />
		<button type="submit" disabled={laeuft}>Anlegen</button>
	</form>
</main>
