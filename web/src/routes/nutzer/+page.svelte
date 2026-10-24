<script lang="ts">
	import { api, ApiError, formatDateTime, errorText, type Invitation, type User, type Person } from '#lib/api.ts';
	import { loadPersons } from '#lib/persons.ts';

	let users = $state<User[]>([]);
	let invitations = $state<Invitation[]>([]);
	let persons = $state<Person[]>([]);
	let error = $state('');
	let busy = $state(false);

	// "" means: new person, otherwise the ID of an existing person without user access.
	let personChoice = $state('');
	let name = $state('');
	let link = $state<{ name: string; url: string } | null>(null);
	let copied = $state(false);

	const withoutAccess = $derived(persons.filter((p) => !p.isUser));
	const canShare = typeof navigator !== 'undefined' && 'share' in navigator;

	async function reload() {
		try {
			const [u, i, p] = await Promise.all([
				api<{ users: User[] }>('GET', '/api/users'),
				api<{ invitations: Invitation[] }>('GET', '/api/invitations'),
				loadPersons()
			]);
			users = u.users;
			invitations = i.invitations;
			persons = p;
		} catch (e) {
			error = errorText(e);
		}
	}
	reload();

	async function invite(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		try {
			const body = personChoice ? { personId: Number(personChoice) } : { name };
			const i = await api<Invitation & { link: string }>('POST', '/api/invitations', body);
			link = { name: i.name, url: i.link };
			copied = false;
			personChoice = '';
			name = '';
			await reload();
		} catch (e) {
			error =
				e instanceof ApiError && e.code === 'already_user'
					? 'Diese Person hat schon einen Nutzerzugang.'
					: errorText(e);
		} finally {
			busy = false;
		}
	}

	async function copy() {
		if (!link) return;
		try {
			await navigator.clipboard.writeText(link.url);
			copied = true;
		} catch {
			error = 'Kopieren nicht möglich. Bitte den Link markieren und selbst kopieren.';
		}
	}

	async function share() {
		if (!link) return;
		try {
			await navigator.share({ title: 'Einladung zum Inventar', text: `Einladung für ${link.name}`, url: link.url });
		} catch {
			// Cancelling the share sheet is not an error.
		}
	}

	async function revoke(i: Invitation) {
		if (!confirm(`Einladung für ${i.name} zurückziehen? Der Link funktioniert dann nicht mehr.`)) return;
		error = '';
		busy = true;
		try {
			await api('DELETE', `/api/invitations/${i.id}`);
			await reload();
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		<h1>Nutzer</h1>
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	<ul class="rows">
		{#each users as u (u.personId)}
			<li>
				<span>{u.name}</span>
				<span class="muted">{u.email}</span>
			</li>
		{/each}
	</ul>

	<section>
		<h2>Einladen</h2>
		<p class="muted">
			Du bekommst einen Link, den du selbst weiterschickst. Er funktioniert einmal und läuft nach 7 Tagen ab.
		</p>
		<form onsubmit={invite}>
			<label>
				Person
				<select bind:value={personChoice}>
					<option value="">Neue Person</option>
					{#each withoutAccess as p (p.id)}
						<option value={String(p.id)}>{p.name}</option>
					{/each}
				</select>
			</label>
			{#if !personChoice}
				<label>
					Name
					<input placeholder="z. B. Kim" bind:value={name} required />
				</label>
			{/if}
			<button type="submit" disabled={busy}>Einladung erstellen</button>
		</form>

		{#if link}
			<div class="notice">
				<p>Einladungslink für {link.name}. Er wird nur jetzt angezeigt.</p>
				<input readonly aria-label="Einladungslink" value={link.url} onfocus={(e) => e.currentTarget.select()} />
				<div class="button-row">
					<button type="button" onclick={copy}>{copied ? 'Kopiert' : 'Kopieren'}</button>
					{#if canShare}
						<button type="button" class="secondary" onclick={share}>Teilen …</button>
					{/if}
				</div>
			</div>
		{/if}
	</section>

	<section>
		<h2>Offene Einladungen</h2>
		{#if invitations.length === 0}
			<p class="muted">Keine offenen Einladungen.</p>
		{:else}
			<ul class="rows">
				{#each invitations as i (i.id)}
					<li>
						<span>
							{i.name}
							<span class="muted">· von {i.createdBy} · läuft ab {formatDateTime(i.expiresAt)}</span>
						</span>
						<button class="secondary small" disabled={busy} onclick={() => revoke(i)}>Zurückziehen</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</main>
