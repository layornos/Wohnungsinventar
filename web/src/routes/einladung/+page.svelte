<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, ApiError } from '#lib/api.ts';
	import { acceptInvitation } from '#lib/session.svelte.ts';

	// The token is in the link's fragment so it is never sent to the server, except in the body.
	const token = page.url.hash.slice(1);

	const MESSAGES: Record<string, string> = {
		invitation_expired: 'Diese Einladung ist abgelaufen.',
		invitation_used: 'Diese Einladung wurde schon verwendet.',
		invitation_revoked: 'Diese Einladung wurde zurückgezogen.',
		invitation_invalid: 'Diesen Einladungslink gibt es nicht. Prüfe, ob er vollständig ist.',
		already_user: 'Du hast schon einen Zugang. Melde dich einfach an.'
	};

	/** Message if the invitation can't be accepted (any more); null for other errors. */
	function stateMessage(e: unknown): string | null {
		return e instanceof ApiError && e.code in MESSAGES ? MESSAGES[e.code] : null;
	}

	let invitation = $state<{ name: string; createdBy: string } | null>(null);
	let unusable = $state('');
	let error = $state('');
	let busy = $state(false);
	let email = $state('');
	let password = $state('');

	async function check() {
		try {
			invitation = await api('POST', '/api/invitations/check', { token }, { noRedirect: true });
		} catch (e) {
			unusable = stateMessage(e) ?? 'Die Einladung lässt sich gerade nicht prüfen. Ist der Server erreichbar?';
		}
	}
	check();

	async function accept(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		try {
			await acceptInvitation(token, email, password);
			await goto('/', { replaceState: true });
		} catch (e) {
			const state = stateMessage(e);
			if (state) {
				unusable = state;
			} else if (e instanceof ApiError) {
				error = e.code === 'email_taken' ? 'Diese E-Mail-Adresse ist schon vergeben. Bitte nimm eine andere.' : e.message;
			} else {
				error = 'Annehmen nicht möglich. Ist der Server erreichbar?';
			}
		} finally {
			busy = false;
		}
	}
</script>

<main class="page narrow">
	<h1>Einladung</h1>
	{#if unusable}
		<p class="error" role="alert">{unusable}</p>
		<p>Bitte die Person, die dich eingeladen hat, um einen neuen Link.</p>
		<p><a href="/anmeldung">Schon Nutzer? Zur Anmeldung</a></p>
	{:else if invitation}
		<p><strong>{invitation.createdBy}</strong> hat dich ({invitation.name}) zum Inventar eingeladen.</p>
		<p class="muted">Lege fest, mit welcher E-Mail-Adresse und welchem Passwort du dich anmeldest.</p>
		<form onsubmit={accept}>
			<label>
				E-Mail
				<input type="email" autocomplete="username" bind:value={email} required />
			</label>
			<label>
				Passwort (mindestens 8 Zeichen)
				<input type="password" autocomplete="new-password" minlength="8" bind:value={password} required />
			</label>
			{#if error}
				<p class="error" role="alert">{error}</p>
			{/if}
			<button type="submit" disabled={busy}>{busy ? 'Annehmen …' : 'Einladung annehmen'}</button>
		</form>
	{:else}
		<p class="muted">Einladung wird geprüft …</p>
	{/if}
</main>
