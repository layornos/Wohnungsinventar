<script lang="ts">
	// Seitenmenü in zwei Formen mit demselben Inhalt:
	// - "dialog" (Startseite, schmale Bildschirme): öffnet per ☰-Knopf oder Wisch vom linken Rand, schließt per Wisch
	//   nach links, Tipp daneben oder Escape. Der modale <dialog> hält den Fokus im Menü und gibt ihn beim Schließen zurück.
	// - "leiste" (Layout, breite Bildschirme): dauerhaft sichtbare Seitenleiste links auf allen Seiten.
	// Welche Form sichtbar ist, entscheidet desktop.css; der Wisch wirkt nur, solange die Leiste nicht zu sehen ist.
	import { page } from '$app/state';
	import { MediaQuery } from 'svelte/reactivity';
	import { zurAnmeldung } from './api.ts';
	import { abmelden, sitzung } from './sitzung.svelte.ts';

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

{#snippet symbol(name: string)}
	<svg class="menuesymbol" aria-hidden="true" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
		{#if name === 'inventar'}<rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" /><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" />
		{:else if name === 'ohne-lagerort'}<path d="M12 21s-7-6.2-7-11.5a7 7 0 0 1 14 0C19 14.8 12 21 12 21z" /><path d="M9.5 9.5l5 5M14.5 9.5l-5 5" />
		{:else if name === 'wohnungen'}<path d="M3 10.5 12 3l9 7.5" /><path d="M5 9v12h14V9" /><path d="M10 21v-6h4v6" />
		{:else if name === 'umzug'}<path d="M3 7h11v10H3z" /><path d="M14 10h4l3 3v4h-7" /><circle cx="7" cy="18" r="2" /><circle cx="17" cy="18" r="2" />
		{:else if name === 'personen'}<circle cx="9" cy="8" r="3.5" /><path d="M2.5 20a6.5 6.5 0 0 1 13 0" /><path d="M16 4.5a3.5 3.5 0 0 1 0 7M18 14.2a6.5 6.5 0 0 1 3.5 5.8" />
		{:else if name === 'nutzer'}<rect x="4" y="10" width="16" height="11" rx="2" /><path d="M8 10V7a4 4 0 0 1 8 0v3" />
		{:else if name === 'einstellungen'}<circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" />
		{:else if name === 'abmelden'}<path d="M15 4h3a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-3" /><path d="M10 17l-5-5 5-5M5 12h11" />
		{/if}
	</svg>
{/snippet}

{#snippet eintrag(href: string, name: string, icon: string)}
	<li>
		<a class="menuelink" {href} aria-current={aktiv(href) ? 'page' : undefined} onclick={schliessen}>
			{@render symbol(icon)}<span>{name}</span>
		</a>
	</li>
{/snippet}

{#snippet menueinhalt()}
	<div class="menuekopf">
		<span class="menuemarke">Inventar</span>
		{#if art === 'dialog'}
			<button type="button" class="menue-schliessen" aria-label="Menü schließen" onclick={schliessen}>
				<svg aria-hidden="true" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M6 6l12 12M18 6 6 18" /></svg>
			</button>
		{/if}
	</div>

	<ul class="menuegruppe">
		{@render eintrag('/', 'Alle Items', 'inventar')}
		{@render eintrag('/ohne-lagerort', 'Ohne Lagerort', 'ohne-lagerort')}
	</ul>

	<h2 class="menuetitel" id="menue-{art}-orte">Orte</h2>
	<ul class="menuegruppe" aria-labelledby="menue-{art}-orte">
		{@render eintrag('/wohnungen', 'Wohnungen', 'wohnungen')}
		{@render eintrag('/umzug', 'Umzug', 'umzug')}
	</ul>

	<h2 class="menuetitel" id="menue-{art}-haushalt">Haushalt</h2>
	<ul class="menuegruppe" aria-labelledby="menue-{art}-haushalt">
		{@render eintrag('/personen', 'Personen', 'personen')}
		{@render eintrag('/nutzer', 'Nutzer', 'nutzer')}
	</ul>

	<div class="menuefuss">
		{#if sitzung.nutzer}
			<p class="menuekonto">
				<span class="menuekonto-name">{sitzung.nutzer.name}</span>
				<span class="menuekonto-email">{sitzung.nutzer.email}</span>
			</p>
		{/if}
		<ul class="menuegruppe">
			{@render eintrag('/einstellungen', 'Einstellungen', 'einstellungen')}
			<li>
				<button type="button" class="menuelink" onclick={abmeldenUndWeiter}>
					{@render symbol('abmelden')}<span>Abmelden</span>
				</button>
			</li>
		</ul>
		{#if fehler}
			<p class="fehler" role="alert">{fehler}</p>
		{/if}
	</div>
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
