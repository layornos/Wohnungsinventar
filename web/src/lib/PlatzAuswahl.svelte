<script lang="ts">
	import type { Platz, PlatzEintrag } from './api.ts';

	// Durchsuchbare Liste aller Räume und Lagerorte als Pfade.
	let {
		plaetze,
		wert = $bindable(),
		titel = 'Platz',
		ohneErlaubt = true,
		ausblenden = () => false
	}: {
		plaetze: PlatzEintrag[];
		wert: Platz | null;
		titel?: string;
		ohneErlaubt?: boolean;
		ausblenden?: (p: PlatzEintrag) => boolean;
	} = $props();

	let suche = $state('');
	let offen = $state(false);

	const gewaehlt = $derived(plaetze.find((p) => p.art === wert?.art && p.id === wert?.id));
	const treffer = $derived.by(() => {
		const s = suche.trim().toLowerCase();
		return plaetze.filter((p) => !ausblenden(p) && p.pfad.toLowerCase().includes(s)).slice(0, 50);
	});

	function waehlen(p: Platz | null) {
		wert = p ? { art: p.art, id: p.id } : null;
		offen = false;
		suche = '';
	}
</script>

<div class="platzwahl">
	<span class="feldname">{titel}</span>
	<button type="button" class="sekundaer auswahl" onclick={() => (offen = !offen)} aria-expanded={offen}>
		{gewaehlt?.pfad ?? (ohneErlaubt ? 'Kein Platz' : 'Bitte wählen …')}
	</button>
	{#if offen}
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
				<li class="leise">Nichts gefunden. Räume und Lagerorte legst du unter „Wohnungen“ an.</li>
			{/each}
		</ul>
	{/if}
</div>
