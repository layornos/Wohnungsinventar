<script lang="ts">
	import { goto } from '$app/navigation';
	import { ApiError } from '#lib/api.ts';
	import { login } from '#lib/session.svelte.ts';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		try {
			await login(email, password);
			await goto('/', { replaceState: true });
		} catch (e) {
			error = e instanceof ApiError ? e.message : 'Anmeldung nicht möglich. Ist der Server erreichbar?';
		} finally {
			busy = false;
		}
	}
</script>

<main class="page narrow">
	<h1>Anmelden</h1>
	<form onsubmit={submit}>
		<label>
			E-Mail
			<input type="email" autocomplete="username" bind:value={email} required />
		</label>
		<label>
			Passwort
			<input type="password" autocomplete="current-password" bind:value={password} required />
		</label>
		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
		<button type="submit" disabled={busy}>{busy ? 'Anmelden …' : 'Anmelden'}</button>
	</form>
</main>
