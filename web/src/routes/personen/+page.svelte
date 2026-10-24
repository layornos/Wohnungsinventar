<script lang="ts">
	import { api, ApiError, errorText, type Person } from '#lib/api.ts';
	import MenuButton from '#lib/MenuButton.svelte';
	import { loadPersons } from '#lib/persons.ts';

	let persons = $state<Person[]>([]);
	let newName = $state('');
	let error = $state('');
	let busy = $state(false);
	let renaming = $state<{ id: number; name: string } | null>(null);

	async function reload() {
		try {
			persons = await loadPersons();
		} catch (e) {
			error = errorText(e);
		}
	}
	reload();

	async function run(action: () => Promise<unknown>, message: (e: unknown) => string = errorText) {
		error = '';
		busy = true;
		try {
			await action();
			await reload();
			return true;
		} catch (e) {
			error = message(e);
			return false;
		} finally {
			busy = false;
		}
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (await run(() => api('POST', '/api/persons', { name: newName }))) newName = '';
	}

	async function saveRename(event: SubmitEvent) {
		event.preventDefault();
		const r = renaming;
		if (r && (await run(() => api('PUT', `/api/persons/${r.id}`, { name: r.name })))) renaming = null;
	}

	function remove(p: Person) {
		if (!confirm(`${p.name} löschen?`)) return;
		run(
			() => api('DELETE', `/api/persons/${p.id}`),
			(e) =>
				e instanceof ApiError && e.code === 'last_owner'
					? `${p.name} ist bei mindestens einem Item einziger Besitzer. Gib diesen Items zuerst einen anderen Besitzer.`
					: errorText(e)
		);
	}
</script>

<main class="page">
	<header class="header">
		<MenuButton />
		<a class="button secondary" href="/">← Liste</a>
		<h1>Personen</h1>
	</header>

	<p class="muted">Personen können Items besitzen. Wer sich anmelden kann, hat einen Nutzerzugang.</p>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	<ul class="rows">
		{#each persons as p (p.id)}
			<li>
				{#if renaming?.id === p.id}
					<form class="inline" onsubmit={saveRename}>
						<input aria-label="Neuer Name" bind:value={renaming.name} required />
						<button type="submit" disabled={busy}>Speichern</button>
						<button type="button" class="secondary" onclick={() => (renaming = null)}>Abbrechen</button>
					</form>
				{:else}
					<span>
						{p.name}
						{#if p.isUser}<span class="badge">Nutzer</span>{/if}
					</span>
					<span class="button-row">
						<button class="secondary small" onclick={() => (renaming = { id: p.id, name: p.name })}>Umbenennen</button>
						{#if !p.isUser}
							<button class="secondary small" disabled={busy} onclick={() => remove(p)}>Löschen</button>
						{/if}
					</span>
				{/if}
			</li>
		{/each}
	</ul>

	<form class="inline" onsubmit={create}>
		<input aria-label="Name der neuen Person" placeholder="Name, z. B. Kim" bind:value={newName} required />
		<button type="submit" disabled={busy}>Anlegen</button>
	</form>
</main>
