<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, ApiError, boxName, fehlerText, type ItemListenEintrag, type Umzug, type Wohnung } from '#lib/api.ts';
	import { ladeUmzug, ladeWohnungen } from '#lib/umzug.ts';

	let umzug = $state<Umzug | null>(null);
	let geladen = $state(false);
	let abgeschlossene = $state<Umzug[]>([]);
	let wohnungen = $state<Wohnung[]>([]);
	let alt = $state('');
	let neu = $state('');
	let nummer = $state('');
	let neueBox = $state('');
	let ohneZielraum = $state<ItemListenEintrag[] | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);

	async function laden() {
		try {
			umzug = await ladeUmzug();
			abgeschlossene = (await api<{ umzuege: Umzug[] }>('GET', '/api/umzuege')).umzuege;
			if (!umzug) wohnungen = await ladeWohnungen();
		} catch (e) {
			fehler = fehlerText(e);
		} finally {
			geladen = true;
		}
	}
	laden();

	async function ausfuehren(aktion: () => Promise<unknown>) {
		fehler = '';
		laeuft = true;
		try {
			await aktion();
			await laden();
			return true;
		} catch (e) {
			fehler = fehlerText(e);
			return false;
		} finally {
			laeuft = false;
		}
	}

	async function anlegen(event: SubmitEvent) {
		event.preventDefault();
		await ausfuehren(() => api('POST', '/api/umzug', { alteWohnungId: Number(alt), neueWohnungId: Number(neu) }));
	}

	async function boxAnlegen(event: SubmitEvent) {
		event.preventDefault();
		if (await ausfuehren(() => api('POST', '/api/umzug/boxen', { bezeichnung: neueBox }))) neueBox = '';
	}

	async function boxOeffnen(event: SubmitEvent) {
		event.preventDefault();
		fehler = '';
		try {
			const r = await api<{ box: { id: number } }>('GET', `/api/umzug/boxen/${Number(nummer)}`);
			await goto(`/boxen/${r.box.id}`);
		} catch (e) {
			fehler = e instanceof ApiError && e.status === 404 ? `Box ${nummer} gibt es nicht.` : fehlerText(e);
		}
	}

	async function abschliessen(bestaetigt: boolean) {
		if (!bestaetigt && !confirm('Umzug abschließen? Alle Items mit Zielraum stehen danach ohne Lagerort in ihrem Zielraum.')) return;
		fehler = '';
		laeuft = true;
		try {
			await api('POST', '/api/umzug/abschluss', { bestaetigt });
			ohneZielraum = null;
			await laden();
		} catch (e) {
			if (e instanceof ApiError && e.code === 'items_ohne_zielraum') {
				ohneZielraum = e.daten.items as ItemListenEintrag[];
			} else {
				fehler = fehlerText(e);
			}
		} finally {
			laeuft = false;
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/">← Liste</a>
		<h1>Umzug</h1>
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	{#if umzug}
		<p>{umzug.alteWohnung.name} → <strong>{umzug.neueWohnung.name}</strong></p>

		<form class="inline gross" onsubmit={boxOeffnen}>
			<input type="number" inputmode="numeric" min="1" placeholder="Box-Nr." aria-label="Boxnummer" bind:value={nummer} required />
			<button type="submit">Box öffnen</button>
		</form>

		<nav class="kacheln">
			<a class="knopf" href="/umzug/packen">Packen</a>
			<a class="knopf sekundaer" href="/umzug/zielraeume">Nach Zielraum</a>
			<a class="knopf sekundaer" href="/umzug/ohne-zielraum">Ohne Zielraum</a>
		</nav>

		<section>
			<h2>Boxen</h2>
			<ul class="zeilen">
				{#each umzug.boxen as b (b.id)}
					<li>
						<a href="/boxen/{b.id}">{boxName(b)}</a>
						<span class="leise">{b.anzahl} Items</span>
					</li>
				{:else}
					<li class="leise">Noch keine Boxen.</li>
				{/each}
			</ul>
			<form class="inline" onsubmit={boxAnlegen}>
				<input aria-label="Bezeichnung der neuen Box" placeholder="Bezeichnung (optional)" bind:value={neueBox} />
				<button type="submit" class="sekundaer" disabled={laeuft}>Neue Box</button>
			</form>
		</section>

		<section>
			<h2>Abschluss</h2>
			{#if ohneZielraum}
				<p class="fehler" role="alert">Diese Items der alten Wohnung haben noch keinen Zielraum. Sie bleiben, wo sie sind:</p>
				<ul class="zeilen">
					{#each ohneZielraum as i (i.id)}
						<li><a class="mit-bild" href="/items/{i.id}"><img src={i.vorschauUrl} alt="" />{i.name}</a></li>
					{/each}
				</ul>
				<div class="knopfzeile">
					<button disabled={laeuft} onclick={() => abschliessen(true)}>Trotzdem abschließen</button>
					<button class="sekundaer" onclick={() => (ohneZielraum = null)}>Abbrechen</button>
				</div>
			{:else}
				<button class="sekundaer" disabled={laeuft} onclick={() => abschliessen(false)}>Umzug abschließen</button>
			{/if}
		</section>
	{:else if geladen}
		<section>
			<h2>Neuer Umzug</h2>
			{#if wohnungen.length < 2}
				<p class="leise">Lege zuerst die alte und die neue Wohnung unter <a href="/wohnungen">Wohnungen</a> an.</p>
			{:else}
				<form onsubmit={anlegen}>
					<label>
						Von
						<select bind:value={alt} required>
							<option value="">Alte Wohnung wählen …</option>
							{#each wohnungen as w (w.id)}<option value={String(w.id)}>{w.name}</option>{/each}
						</select>
					</label>
					<label>
						Nach
						<select bind:value={neu} required>
							<option value="">Neue Wohnung wählen …</option>
							{#each wohnungen as w (w.id)}<option value={String(w.id)} disabled={String(w.id) === alt}>{w.name}</option>{/each}
						</select>
					</label>
					<button type="submit" disabled={laeuft}>Umzug anlegen</button>
				</form>
			{/if}
		</section>
	{/if}

	{#if abgeschlossene.length > 0}
		<section>
			<h2>Abgeschlossene Umzüge</h2>
			<ul class="zeilen">
				{#each abgeschlossene as u (u.id)}
					<li><a href="/umzuege/{u.id}">{u.alteWohnung.name} → {u.neueWohnung.name}</a></li>
				{/each}
			</ul>
		</section>
	{/if}
</main>
