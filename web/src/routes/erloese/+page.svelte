<script lang="ts">
	// Proceeds summary (spec 06): sum and counts of the archive in a period (days in the household's time zone),
	// optionally for one owner, whose items then count with their equal share.
	import { api, errorText, type ItemListEntry, type Person } from '#lib/api.ts';
	import { formatDay, formatEuro, today } from '#lib/money.ts';
	import { loadPersons } from '#lib/persons.ts';

	type Entry = ItemListEntry & { share: number };
	type Summary = { total: number; soldCount: number; givenAwayCount: number; items: Entry[] };
	type Range = 'month' | 'year' | 'twelveMonths' | 'all' | 'custom';

	const RANGES: { value: Range; name: string }[] = [
		{ value: 'month', name: 'Dieser Monat' },
		{ value: 'year', name: 'Dieses Jahr' },
		{ value: 'twelveMonths', name: 'Letzte 12 Monate' },
		{ value: 'all', name: 'Gesamt' },
		{ value: 'custom', name: 'Zeitraum' }
	];

	let range = $state<Range>('month');
	let customFrom = $state('');
	let customTo = $state('');
	let owner = $state('');
	let persons = $state<Person[]>([]);
	let summary = $state<Summary | null>(null);
	let error = $state('');

	loadPersons()
		.then((p) => (persons = p))
		.catch((e) => (error = errorText(e)));

	/** The day after the same date one year earlier, so that "last 12 months" ends today. */
	function yearBefore(day: string): string {
		const d = new Date(`${day}T00:00:00Z`);
		d.setUTCFullYear(d.getUTCFullYear() - 1);
		d.setUTCDate(d.getUTCDate() + 1);
		return d.toISOString().slice(0, 10);
	}

	const period = $derived.by((): { from: string; to: string } => {
		const t = today();
		switch (range) {
			case 'month':
				return { from: `${t.slice(0, 8)}01`, to: t };
			case 'year':
				return { from: `${t.slice(0, 5)}01-01`, to: t };
			case 'twelveMonths':
				return { from: yearBefore(t), to: t };
			case 'all':
				return { from: '', to: '' };
			case 'custom':
				return { from: customFrom, to: customTo };
		}
	});

	// Only the response to the latest request counts.
	let request = 0;
	$effect(() => {
		const p = new URLSearchParams();
		if (period.from) p.set('from', period.from);
		if (period.to) p.set('to', period.to);
		if (owner) p.set('owner', owner);
		const nr = ++request;
		error = '';
		api<Summary>('GET', '/api/proceeds?' + p)
			.then((s) => {
				if (nr === request) summary = s;
			})
			.catch((e) => {
				if (nr === request) error = errorText(e);
			});
	});
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/archiv">← Archiv</a>
		<h1>Erlöse</h1>
	</header>

	<div class="filter">
		<select aria-label="Zeitraum" bind:value={range}>
			{#each RANGES as r (r.value)}<option value={r.value}>{r.name}</option>{/each}
		</select>
		<select aria-label="Besitzer" bind:value={owner}>
			<option value="">Alle Besitzer</option>
			{#each persons as p (p.id)}<option value={String(p.id)}>{p.name}</option>{/each}
		</select>
		{#if range === 'custom'}
			<label>
				Von
				<input type="date" bind:value={customFrom} />
			</label>
			<label>
				Bis
				<input type="date" bind:value={customTo} />
			</label>
		{/if}
	</div>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if summary}
		<section class="notice proceeds-total">
			<p class="proceeds-sum">{formatEuro(summary.total)}</p>
			<p class="muted">{summary.soldCount} verkauft · {summary.givenAwayCount} verschenkt</p>
		</section>

		<ul class="rows">
			{#each summary.items as i (i.id)}
				<li>
					<a class="with-image" href="/items/{i.id}">
						<img src={i.thumbnailUrl} alt="" loading="lazy" />
						<span>
							{i.name}
							<span class="muted small-text">{i.handedOverOn ? formatDay(i.handedOverOn) : ''}</span>
						</span>
					</a>
					<span>
						{#if i.disposition === 'give_away'}
							verschenkt
						{:else if owner && i.share !== i.saleProceeds}
							{formatEuro(i.saleProceeds ?? 0)} <span class="muted">(Anteil {formatEuro(i.share)})</span>
						{:else}
							{formatEuro(i.saleProceeds ?? 0)}
						{/if}
					</span>
				</li>
			{:else}
				<li class="muted">In diesem Zeitraum wurde nichts abgegeben.</li>
			{/each}
		</ul>
	{/if}
</main>
