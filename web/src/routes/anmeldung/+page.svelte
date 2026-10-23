<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError } from '#lib/api.ts';
	import { anmelden } from '#lib/sitzung.svelte.ts';

	let email = $state('');
	let passwort = $state('');
	let fehler = $state('');
	let laeuft = $state(false);

	async function absenden(event: SubmitEvent) {
		event.preventDefault();
		fehler = '';
		laeuft = true;
		try {
			await anmelden(email, passwort);
			await goto('/', { replaceState: true });
		} catch (e) {
			fehler = e instanceof ApiError ? e.message : 'Anmeldung nicht möglich. Ist der Server erreichbar?';
		} finally {
			laeuft = false;
		}
	}
</script>

<main class="seite schmal">
	<h1>Anmelden</h1>
	<form onsubmit={absenden}>
		<label>
			E-Mail
			<input type="email" autocomplete="username" bind:value={email} required />
		</label>
		<label>
			Passwort
			<input type="password" autocomplete="current-password" bind:value={passwort} required />
		</label>
		{#if fehler}
			<p class="fehler" role="alert">{fehler}</p>
		{/if}
		<button type="submit" disabled={laeuft}>{laeuft ? 'Anmelden …' : 'Anmelden'}</button>
	</form>
</main>
