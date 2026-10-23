<script lang="ts">
	import { zurAnmeldung } from '#lib/api.ts';
	import { abmelden, sitzung } from '#lib/sitzung.svelte.ts';

	let fehler = $state('');

	async function abmeldenUndWeiter() {
		fehler = '';
		try {
			await abmelden();
		} catch {
			// Ohne Antwort des Servers bleibt die Sitzung dort bestehen; nicht so tun, als wäre man abgemeldet.
			fehler = 'Abmelden nicht möglich. Ist der Server erreichbar?';
			return;
		}
		await zurAnmeldung();
	}
</script>

<main class="seite">
	<header class="kopf">
		<h1>Inventar</h1>
		<button class="sekundaer" onclick={abmeldenUndWeiter}>Abmelden</button>
	</header>
	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}
	<p>Hallo {sitzung.nutzer?.name}!</p>
</main>
