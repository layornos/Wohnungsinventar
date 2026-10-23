<script lang="ts">
	import { page } from '$app/state';
	import { api, boxName, fehlerText, type Benannt, type ItemListenEintrag, type Umzug } from '#lib/api.ts';
	import { ladeUmzug, ladeUmzugsraeume, type AngabenAenderung } from '#lib/umzug.ts';

	let umzug = $state<Umzug | null>(null);
	let alteRaeume = $state<Benannt[]>([]);
	let neueRaeume = $state<Benannt[]>([]);
	let box = $state(page.url.searchParams.get('box') ?? '');
	let herkunft = $state('');
	let suche = $state('');
	let items = $state<ItemListenEintrag[]>([]);
	let auswahl = $state<number[]>([]);
	let zielraum = $state('');
	let fehler = $state('');
	let laeuft = $state(false);

	const aktiveBox = $derived(umzug?.boxen.find((b) => String(b.id) === box));

	async function umzugLaden() {
		try {
			umzug = await ladeUmzug();
			if (umzug) ({ alt: alteRaeume, neu: neueRaeume } = await ladeUmzugsraeume(umzug));
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
	umzugLaden();

	async function itemsLaden() {
		const p = new URLSearchParams({ proSeite: '100' });
		if (herkunft) p.set('herkunftsraum', herkunft);
		if (suche.trim()) p.set('q', suche.trim());
		try {
			items = (await api<{ items: ItemListenEintrag[] }>('GET', '/api/items?' + p)).items;
		} catch (e) {
			fehler = fehlerText(e);
		}
	}

	$effect(() => {
		void [herkunft, suche];
		const t = setTimeout(itemsLaden, 250);
		return () => clearTimeout(t);
	});

	async function ausfuehren(aktion: () => Promise<unknown>) {
		fehler = '';
		laeuft = true;
		try {
			await aktion();
			await Promise.all([itemsLaden(), umzugLaden()]);
			return true;
		} catch (e) {
			fehler = fehlerText(e);
			return false;
		} finally {
			laeuft = false;
		}
	}

	function einpacken(item: ItemListenEintrag) {
		if (!aktiveBox) {
			fehler = 'Bitte zuerst oben eine Box wählen.';
			return;
		}
		const raus = item.umzug?.box?.id === aktiveBox.id;
		const aenderung: AngabenAenderung = { boxId: raus ? null : aktiveBox.id };
		ausfuehren(() => api('PATCH', `/api/items/${item.id}/umzug`, aenderung));
	}

	function umschalten(id: number) {
		auswahl = auswahl.includes(id) ? auswahl.filter((a) => a !== id) : [...auswahl, id];
	}

	async function sammeln(aenderung: AngabenAenderung) {
		if (await ausfuehren(() => api('POST', '/api/umzug/sammeln', { itemIds: auswahl, ...aenderung }))) auswahl = [];
	}

	async function neueBoxAnlegen() {
		await ausfuehren(async () => {
			const b = await api<{ id: number }>('POST', '/api/umzug/boxen', { bezeichnung: '' });
			box = String(b.id);
		});
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/umzug">← Umzug</a>
		<h1>Packen</h1>
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	{#if umzug}
		<div class="packkopf">
			<label>
				Aktive Box
				<select bind:value={box}>
					<option value="">Keine</option>
					{#each umzug.boxen as b (b.id)}<option value={String(b.id)}>{boxName(b)} ({b.anzahl})</option>{/each}
				</select>
			</label>
			<button class="sekundaer" disabled={laeuft} onclick={neueBoxAnlegen}>Neue Box</button>
			{#if aktiveBox}
				<a class="knopf" href="/items/neu?box={aktiveBox.id}">+ Neu in {boxName(aktiveBox)}</a>
			{/if}
		</div>

		<div class="filter">
			<select aria-label="Herkunftsraum" bind:value={herkunft}>
				<option value="">Alle Räume</option>
				{#each alteRaeume as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
			</select>
			<input type="search" placeholder="Suchen …" aria-label="Suchen" bind:value={suche} />
		</div>

		<p class="leise">Antippen packt ein Item in die aktive Box oder nimmt es wieder heraus.</p>

		<ul class="zeilen">
			{#each items as item (item.id)}
				<li>
					<input type="checkbox" class="gross" aria-label="{item.name} auswählen" checked={auswahl.includes(item.id)} onchange={() => umschalten(item.id)} />
					<button class="packen" class:drin={!!aktiveBox && item.umzug?.box?.id === aktiveBox.id} disabled={laeuft} onclick={() => einpacken(item)}>
						<img src={item.vorschauUrl} alt="" loading="lazy" />
						<span>
							{item.name}
							<span class="leise klein-text">
								{item.umzug?.box ? boxName(item.umzug.box) : 'in keiner Box'} · {item.umzug?.zielraum?.name ?? 'kein Zielraum'}
							</span>
						</span>
					</button>
				</li>
			{:else}
				<li class="leise">Keine Items.</li>
			{/each}
		</ul>

		{#if auswahl.length > 0}
			<div class="leiste">
				<span>{auswahl.length} ausgewählt</span>
				<select aria-label="Zielraum" bind:value={zielraum}>
					<option value="">Zielraum …</option>
					{#each neueRaeume as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
				</select>
				<button disabled={laeuft || !zielraum} onclick={() => sammeln({ zielraumId: Number(zielraum) })}>Zielraum setzen</button>
				{#if aktiveBox}
					<button class="sekundaer" disabled={laeuft} onclick={() => sammeln({ boxId: aktiveBox.id })}>In {boxName(aktiveBox)}</button>
				{/if}
			</div>
		{/if}
	{:else}
		<p class="leise">Es läuft kein Umzug. <a href="/umzug">Umzug anlegen</a></p>
	{/if}
</main>
