<script lang="ts">
	import { goto } from '$app/navigation';
	import { abmelden, sitzung } from '#lib/sitzung.svelte.ts';

	async function abmeldenUndWeiter() {
		await abmelden();
		await goto('/anmeldung', { replaceState: true });
	}
</script>

<main class="seite">
	<header class="kopf">
		<h1>Inventar</h1>
		<button class="sekundaer" onclick={abmeldenUndWeiter}>Abmelden</button>
	</header>
	<p>Hallo {sitzung.nutzer?.name}!</p>
</main>
