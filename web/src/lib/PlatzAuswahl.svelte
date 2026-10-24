<script lang="ts">
	import { api, fehlerText, pfadText, type Lagerort, type Platz, type PlatzEintrag } from './api.ts';
	import { elternBody } from './plaetze.ts';

	// Durchsuchbare Liste aller Räume und Lagerorte als Pfade.
	// Mit `anlegen` lässt sich ein nicht gefundener Lagerort unter einem bestehenden Platz anlegen;
	// er kommt dann in `plaetze` dazu und ist gleich gewählt.
	let {
		plaetze = $bindable(),
		wert = $bindable(),
		titel = 'Platz',
		ohneErlaubt = true,
		anlegen = false,
		ausblenden = () => false
	}: {
		plaetze: PlatzEintrag[];
		wert: Platz | null;
		titel?: string;
		ohneErlaubt?: boolean;
		anlegen?: boolean;
		ausblenden?: (p: PlatzEintrag) => boolean;
	} = $props();

	let suche = $state('');
	let offen = $state(false);
	// Name des neuen Lagerorts, solange das übergeordnete Element gewählt wird.
	let neuerName = $state<string | null>(null);
	let fehler = $state('');
	let laeuft = $state(false);

	const gewaehlt = $derived(plaetze.find((p) => p.art === wert?.art && p.id === wert?.id));
	const treffer = $derived.by(() => {
		const s = suche.trim().toLowerCase();
		return plaetze.filter((p) => !ausblenden(p) && p.pfad.toLowerCase().includes(s)).slice(0, 50);
	});

	function waehlen(p: Platz | null) {
		wert = p ? { art: p.art, id: p.id } : null;
		offen = false;
		suche = '';
		neuerName = null;
		fehler = '';
	}

	function anlegenBeginnen() {
		neuerName = suche.trim();
		suche = '';
		fehler = '';
	}

	function anlegenAbbrechen() {
		suche = neuerName ?? '';
		neuerName = null;
		fehler = '';
	}

	async function anlegenIn(eltern: PlatzEintrag) {
		if (!neuerName) return;
		fehler = '';
		laeuft = true;
		try {
			const l = await api<Lagerort>('POST', '/api/lagerorte', { name: neuerName, ...elternBody(eltern) });
			const neu: PlatzEintrag = { art: 'lagerort', id: l.id, pfad: pfadText(l.pfad) };
			plaetze = [...plaetze, neu].sort((a, b) => (a.pfad < b.pfad ? -1 : a.pfad > b.pfad ? 1 : 0));
			waehlen(neu);
		} catch (e) {
			fehler = fehlerText(e);
		} finally {
			laeuft = false;
		}
	}

	function umschalten() {
		offen = !offen;
		neuerName = null;
		fehler = '';
	}
</script>

<div class="platzwahl">
	<span class="feldname">{titel}</span>
	<button type="button" class="sekundaer auswahl" onclick={umschalten} aria-expanded={offen}>
		{gewaehlt?.pfad ?? (ohneErlaubt ? 'Kein Platz' : 'Bitte wählen …')}
	</button>
	{#if offen && neuerName !== null}
		<p>„{neuerName}“ anlegen in …</p>
		<input type="search" placeholder="Raum oder Lagerort suchen …" aria-label="Anlegen in suchen" bind:value={suche} />
		{#if fehler}
			<p class="fehler" role="alert">{fehler}</p>
		{/if}
		<ul class="trefferliste">
			{#each treffer as p (p.art + p.id)}
				<li>
					<button type="button" class="sekundaer" disabled={laeuft} onclick={() => anlegenIn(p)}>{p.pfad}</button>
				</li>
			{:else}
				<li class="leise">Nichts gefunden. Räume legst du unter „Wohnungen“ an.</li>
			{/each}
		</ul>
		<button type="button" class="sekundaer" onclick={anlegenAbbrechen}>Abbrechen</button>
	{:else if offen}
		<input type="search" placeholder="Raum oder Lagerort suchen …" aria-label="{titel} suchen" bind:value={suche} />
		<ul class="trefferliste">
			{#if ohneErlaubt}
				<li><button type="button" class="sekundaer" onclick={() => waehlen(null)}>Kein Platz</button></li>
			{/if}
			{#each treffer as p (p.art + p.id)}
				<li>
					<button type="button" class="sekundaer" class:aktiv={p === gewaehlt} onclick={() => waehlen(p)}>
						{p.pfad}{p.art === 'raum' ? ' (direkt im Raum)' : ''}
					</button>
				</li>
			{:else}
				{#if anlegen && suche.trim()}
					<li><button type="button" class="sekundaer" onclick={anlegenBeginnen}>„{suche.trim()}“ anlegen in …</button></li>
				{:else}
					<li class="leise">Nichts gefunden. Räume und Lagerorte legst du unter „Wohnungen“ an.</li>
				{/if}
			{/each}
		</ul>
	{/if}
</div>
