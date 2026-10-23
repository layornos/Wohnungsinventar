<script lang="ts">
	import { api, fehlerText, type Benannt, type ItemListenEintrag } from '#lib/api.ts';
	import { ladeUmzug, ladeUmzugsraeume } from '#lib/umzug.ts';

	let items = $state<ItemListenEintrag[]>([]);
	let raeume = $state<Benannt[]>([]);
	let zielraum = $state<Record<number, string>>({});
	let fehler = $state('');

	async function laden() {
		try {
			const u = await ladeUmzug();
			if (u) raeume = (await ladeUmzugsraeume(u)).neu;
			items = (await api<{ items: ItemListenEintrag[] }>('GET', '/api/items?ohneZielraum=1&proSeite=100')).items;
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
	laden();

	async function setzen(item: ItemListenEintrag) {
		fehler = '';
		try {
			await api('PATCH', `/api/items/${item.id}/umzug`, { zielraumId: Number(zielraum[item.id]) });
			items = items.filter((i) => i.id !== item.id);
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/umzug">← Umzug</a>
		<h1>Ohne Zielraum</h1>
	</header>
	<p class="leise">Items der alten Wohnung, die noch nicht wissen, wohin.</p>
	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}
	<ul class="zeilen">
		{#each items as i (i.id)}
			<li>
				<a class="mit-bild" href="/items/{i.id}"><img src={i.vorschauUrl} alt="" loading="lazy" />{i.name}</a>
				<span class="knopfzeile">
					<select aria-label="Zielraum für {i.name}" bind:value={zielraum[i.id]}>
						<option value="">Zielraum …</option>
						{#each raeume as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
					</select>
					<button class="klein" disabled={!zielraum[i.id]} onclick={() => setzen(i)}>Setzen</button>
				</span>
			</li>
		{:else}
			<li class="leise">Alle Items haben einen Zielraum.</li>
		{/each}
	</ul>
</main>
