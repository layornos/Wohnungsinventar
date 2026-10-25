<script lang="ts">
	// Reminders (spec 07): the open ones with their answers, or the history; "Meine" shows only those for items of
	// the logged-in user (or of nobody with access). The monthly mail links here.
	import { goto } from '$app/navigation';
	import { api, errorText, type Reaction, type Reminder } from '#lib/api.ts';
	import { euroInput, formatDay, formatEuro, parseEuro } from '#lib/money.ts';
	import { KIND_NAMES, leadsToSale, outcome, REACTIONS, refreshReminderCount } from '#lib/reminders.svelte.ts';

	let view = $state<'open' | 'done'>('open');
	let mine = $state(false);
	let reminders = $state<Reminder[]>([]);
	let more = $state(false);
	let loading = $state(false);
	let busy = $state<number | null>(null);
	let error = $state('');
	let pageNumber = 1;
	// New asking prices typed into price reminders, by reminder ID.
	let prices = $state<Record<number, string>>({});

	let request = 0;
	async function load(fresh: boolean) {
		const nr = ++request;
		const n = fresh ? 1 : pageNumber + 1;
		const p = new URLSearchParams({ state: view, page: String(n) });
		if (mine) p.set('mine', '1');
		loading = true;
		error = '';
		try {
			const r = await api<{ reminders: Reminder[]; more: boolean }>('GET', '/api/reminders?' + p);
			if (nr !== request) return;
			reminders = fresh ? r.reminders : [...reminders, ...r.reminders];
			more = r.more;
			pageNumber = n;
		} catch (e) {
			if (nr === request) error = errorText(e);
		} finally {
			if (nr === request) loading = false;
		}
	}

	$effect(() => {
		void [view, mine];
		load(true);
	});

	async function react(r: Reminder, reaction: Reaction) {
		error = '';
		const body: { reaction: Reaction; askingPrice?: number } = { reaction };
		if (reaction === 'changePrice') {
			const cents = parseEuro(prices[r.id] ?? '');
			if (cents === null || cents === undefined) {
				error = 'Bitte den neuen Preis in Euro angeben, z. B. 35 oder 34,50.';
				return;
			}
			body.askingPrice = cents;
		}
		busy = r.id;
		try {
			await api('POST', `/api/reminders/${r.id}/reaction`, body);
			reminders = reminders.filter((x) => x.id !== r.id);
			refreshReminderCount();
			// To sell, the item needs an asking price next; after a new price its listing text is to be updated.
			if (leadsToSale(reaction)) await goto(`/items/${r.item.id}?preis=1`);
			else if (reaction === 'changePrice') await goto(`/items/${r.item.id}`);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = null;
		}
	}
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		<h1>Erinnerungen</h1>
	</header>

	<div class="filter">
		<select aria-label="Offen oder erledigt" bind:value={view}>
			<option value="open">Offen</option>
			<option value="done">Verlauf</option>
		</select>
		<select aria-label="Meine oder alle" bind:value={mine}>
			<option value={false}>Alle</option>
			<option value={true}>Meine</option>
		</select>
	</div>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	<ul class="rows reminders">
		{#each reminders as r (r.id)}
			<li>
				<a class="with-image" href="/items/{r.item.id}">
					<img src={r.item.thumbnailUrl} alt="" loading="lazy" />
					<span>
						{KIND_NAMES[r.kind]}: {r.item.name}
						<span class="muted small-text">
							{#if r.kind === 'priceReminder' && r.item.askingPrice !== null}{formatEuro(r.item.askingPrice)} · {/if}
							{r.item.owners.map((o) => o.name).join(', ')} · fällig am {formatDay(r.dueOn)}
							{#if r.doneAt} · {outcome(r)}{/if}
						</span>
					</span>
				</a>
				{#if !r.doneAt && r.kind === 'priceReminder'}
					<div class="button-row">
						<input
							class="price-input"
							aria-label="Neuer Preis für {r.item.name} in Euro"
							inputmode="decimal"
							placeholder={r.item.askingPrice !== null ? euroInput(r.item.askingPrice) : 'Preis'}
							bind:value={prices[r.id]}
						/>
						<button class="small" disabled={busy === r.id} onclick={() => react(r, 'changePrice')}>Preis ändern</button>
					</div>
				{/if}
				{#if !r.doneAt}
					<div class="button-row">
						{#each REACTIONS[r.kind] as a, i (a.reaction)}
							<button class="small" class:secondary={i > 0 || r.kind === 'priceReminder'} disabled={busy === r.id} onclick={() => react(r, a.reaction)}>{a.name}</button>
						{/each}
						{#if r.kind === 'giveAwayReminder'}
							<a class="button small secondary" href="/items/{r.item.id}?verschenkt=1">Verschenkt</a>
						{/if}
					</div>
				{/if}
			</li>
		{:else}
			{#if !loading}
				<li class="muted">{view === 'open' ? 'Keine offenen Erinnerungen.' : 'Noch nichts erledigt.'}</li>
			{/if}
		{/each}
	</ul>
	{#if more}
		<button class="secondary full-width" onclick={() => load(false)} disabled={loading}>Mehr laden</button>
	{/if}
</main>
