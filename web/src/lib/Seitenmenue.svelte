<script lang="ts">
	// Seitenmenü der Startseite. Öffnet per ☰-Knopf oder Wisch vom linken Rand, schließt per Wisch nach links,
	// Tipp daneben oder Escape. Der modale <dialog> hält den Fokus im Menü und gibt ihn beim Schließen zurück.
	import DesignWahl from './DesignWahl.svelte';

	let { abmelden }: { abmelden: () => void } = $props();

	// Wisch beginnt nur so nah am Rand, damit normales Scrollen und Wischen im Inhalt nicht stört.
	const RAND = 24;
	const MINDESTWEG = 60;

	let dialog: HTMLDialogElement;
	let offen = $state(false);
	let start: { x: number; y: number } | null = null;

	function oeffnen() {
		if (offen) return;
		dialog.showModal();
		offen = true;
	}

	function schliessen() {
		dialog.close();
	}

	function wischStart(e: TouchEvent, nurAmRand: boolean) {
		const t = e.touches[0];
		start = !nurAmRand || t.clientX <= RAND ? { x: t.clientX, y: t.clientY } : null;
	}

	// Liefert den waagrechten Weg, wenn der Wisch eher waagrecht als senkrecht war, sonst 0.
	function wischWeg(e: TouchEvent) {
		if (!start) return 0;
		const t = e.changedTouches[0];
		const dx = t.clientX - start.x;
		const dy = t.clientY - start.y;
		start = null;
		return Math.abs(dx) > Math.abs(dy) ? dx : 0;
	}
</script>

<svelte:window
	ontouchstart={(e) => {
		if (!offen) wischStart(e, true);
	}}
	ontouchend={(e) => {
		if (!offen && wischWeg(e) >= MINDESTWEG) oeffnen();
	}}
/>

<button type="button" class="sekundaer menueknopf" aria-label="Menü" aria-expanded={offen} aria-controls="seitenmenue" onclick={oeffnen}>☰</button>

<dialog
	id="seitenmenue"
	class="seitenmenue"
	aria-label="Menü"
	bind:this={dialog}
	onclose={() => (offen = false)}
	onclick={(e) => {
		// Klicks auf den abgedunkelten Hintergrund landen direkt beim <dialog>, nicht beim Inhalt.
		if (e.target === dialog) schliessen();
	}}
	ontouchstart={(e) => wischStart(e, false)}
	ontouchend={(e) => {
		if (wischWeg(e) <= -MINDESTWEG) schliessen();
	}}
>
	<nav class="menueinhalt">
		<a class="knopf sekundaer" href="/umzug" onclick={schliessen}>Umzug</a>
		<a class="knopf sekundaer" href="/wohnungen" onclick={schliessen}>Wohnungen</a>
		<a class="knopf sekundaer" href="/ohne-lagerort" onclick={schliessen}>Ohne Lagerort</a>
		<a class="knopf sekundaer" href="/personen" onclick={schliessen}>Personen</a>
		<a class="knopf sekundaer" href="/nutzer" onclick={schliessen}>Nutzer</a>
		<DesignWahl />
		<button
			type="button"
			class="sekundaer"
			onclick={() => {
				schliessen();
				abmelden();
			}}>Abmelden</button
		>
	</nav>
</dialog>
