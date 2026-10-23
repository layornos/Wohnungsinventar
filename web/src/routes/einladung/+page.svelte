<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, ApiError } from '#lib/api.ts';
	import { einladungAnnehmen } from '#lib/sitzung.svelte.ts';

	// Der Token steht im Fragment des Links, damit er nie an den Server geschickt wird, außer im Body.
	const token = page.url.hash.slice(1);

	const MELDUNGEN: Record<string, string> = {
		einladung_abgelaufen: 'Diese Einladung ist abgelaufen.',
		einladung_verwendet: 'Diese Einladung wurde schon verwendet.',
		einladung_zurueckgezogen: 'Diese Einladung wurde zurückgezogen.',
		einladung_ungueltig: 'Diesen Einladungslink gibt es nicht. Prüfe, ob er vollständig ist.'
	};

	/** Meldung, wenn die Einladung nicht (mehr) annehmbar ist; null bei anderen Fehlern. */
	function zustandsMeldung(e: unknown): string | null {
		return e instanceof ApiError && e.code in MELDUNGEN ? MELDUNGEN[e.code] : null;
	}

	let einladung = $state<{ name: string; ersteller: string } | null>(null);
	let unbrauchbar = $state('');
	let fehler = $state('');
	let laeuft = $state(false);
	let email = $state('');
	let passwort = $state('');

	async function pruefen() {
		try {
			einladung = await api('POST', '/api/einladungen/pruefen', { token }, { ohneUmleitung: true });
		} catch (e) {
			unbrauchbar = zustandsMeldung(e) ?? 'Die Einladung lässt sich gerade nicht prüfen. Ist der Server erreichbar?';
		}
	}
	pruefen();

	async function annehmen(event: SubmitEvent) {
		event.preventDefault();
		fehler = '';
		laeuft = true;
		try {
			await einladungAnnehmen(token, email, passwort);
			await goto('/', { replaceState: true });
		} catch (e) {
			const zustand = zustandsMeldung(e);
			if (zustand) {
				unbrauchbar = zustand;
			} else if (e instanceof ApiError) {
				fehler = e.code === 'email_vergeben' ? 'Diese E-Mail-Adresse ist schon vergeben. Bitte nimm eine andere.' : e.message;
			} else {
				fehler = 'Annehmen nicht möglich. Ist der Server erreichbar?';
			}
		} finally {
			laeuft = false;
		}
	}
</script>

<main class="seite schmal">
	<h1>Einladung</h1>
	{#if unbrauchbar}
		<p class="fehler" role="alert">{unbrauchbar}</p>
		<p>Bitte die Person, die dich eingeladen hat, um einen neuen Link.</p>
		<p><a href="/anmeldung">Schon Nutzer? Zur Anmeldung</a></p>
	{:else if einladung}
		<p><strong>{einladung.ersteller}</strong> hat dich ({einladung.name}) zum Inventar eingeladen.</p>
		<p class="leise">Lege fest, mit welcher E-Mail-Adresse und welchem Passwort du dich anmeldest.</p>
		<form onsubmit={annehmen}>
			<label>
				E-Mail
				<input type="email" autocomplete="username" bind:value={email} required />
			</label>
			<label>
				Passwort (mindestens 8 Zeichen)
				<input type="password" autocomplete="new-password" minlength="8" bind:value={passwort} required />
			</label>
			{#if fehler}
				<p class="fehler" role="alert">{fehler}</p>
			{/if}
			<button type="submit" disabled={laeuft}>{laeuft ? 'Annehmen …' : 'Einladung annehmen'}</button>
		</form>
	{:else}
		<p class="leise">Einladung wird geprüft …</p>
	{/if}
</main>
