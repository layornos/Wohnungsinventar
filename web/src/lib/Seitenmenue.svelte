<script lang="ts">
	// Seitenmenü in zwei Formen mit demselben Inhalt:
	// - "dialog" (Startseite, schmale Bildschirme): öffnet per ☰-Knopf oder Wisch vom linken Rand, schließt per Wisch
	//   nach links, Tipp daneben oder Escape. Der modale <dialog> hält den Fokus im Menü und gibt ihn beim Schließen zurück.
	// - "leiste" (Layout, breite Bildschirme): dauerhaft sichtbare Seitenleiste links auf allen Seiten.
	// Welche Form sichtbar ist, entscheidet desktop.css; der Wisch wirkt nur, solange die Leiste nicht zu sehen ist.
	import { page } from '$app/state';
	import { MediaQuery } from 'svelte/reactivity';
	import { zurAnmeldung } from './api.ts';
	import { abmelden } from './sitzung.svelte.ts';

	let { art = 'dialog' }: { art?: 'dialog' | 'leiste' } = $props();

	// Muss zum Breakpoint in desktop.css passen.
	const breit = new MediaQuery('min-width: 1024px');

	// Wisch beginnt nur so nah am Rand, damit normales Scrollen und Wischen im Inhalt nicht stört.
	const RAND = 24;
	const MINDESTWEG = 60;

	let dialog = $state<HTMLDialogElement>();
	let offen = $state(false);
	let start: { x: number; y: number } | null = null;
	let fehler = $state('');

	function oeffnen() {
		if (offen || !dialog) return;
		dialog.showModal();
		offen = true;
	}

	function schliessen() {
		dialog?.close();
	}

	// Wird das Fenster breit, übernimmt die Leiste; ein offener Dialog darüber wäre doppelt.
	$effect(() => {
		if (breit.current && offen) schliessen();
	});

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

	// Unterseiten zählen mit, damit z. B. der Pack-Modus unter „Umzug“ hervorgehoben ist.
	function aktiv(href: string) {
		const p = page.url.pathname;
		return p === href || p.startsWith(href + '/');
	}

	async function abmeldenUndWeiter() {
		fehler = '';
		try {
			await abmelden();
		} catch {
			// Ohne Antwort des Servers bleibt die Sitzung dort bestehen; nicht so tun, als wäre man abgemeldet.
			fehler = 'Abmelden nicht möglich. Ist der Server erreichbar?';
			return;
		}
		schliessen();
		await zurAnmeldung();
	}
</script>

{#snippet eintrag(href: string, name: string)}
	<a class={['knopf', !aktiv(href) && 'sekundaer']} {href} aria-current={aktiv(href) ? 'page' : undefined} onclick={schliessen}>{name}</a>
{/snippet}

{#snippet menueinhalt()}
	{#if art === 'leiste'}
		{@render eintrag('/', 'Inventar')}
	{/if}
	{@render eintrag('/umzug', 'Umzug')}
	{@render eintrag('/wohnungen', 'Wohnungen')}
	{@render eintrag('/ohne-lagerort', 'Ohne Lagerort')}
	{@render eintrag('/personen', 'Personen')}
	{@render eintrag('/nutzer', 'Nutzer')}
	<button type="button" class="sekundaer" onclick={abmeldenUndWeiter}>Abmelden</button>
	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}
{/snippet}

<svelte:window
	ontouchstart={(e) => {
		if (art === 'dialog' && !offen && !breit.current) wischStart(e, true);
	}}
	ontouchend={(e) => {
		if (art === 'dialog' && !offen && !breit.current && wischWeg(e) >= MINDESTWEG) oeffnen();
	}}
/>

{#if art === 'leiste'}
	<nav class="menueinhalt seitenleiste" aria-label="Menü">
		{@render menueinhalt()}
	</nav>
{:else}
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
			{@render menueinhalt()}
		</nav>
	</dialog>
{/if}
