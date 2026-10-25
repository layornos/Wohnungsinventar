<script lang="ts">
	import { api, errorText } from '#lib/api.ts';
	import { appearance } from '#lib/theme.svelte.ts';
	import { THEMES, type Theme } from '#lib/themes.ts';
	import { saveTheme } from '#lib/session.svelte.ts';

	let error = $state('');

	// Listing template and warranty disclaimer of the household (spec 06).
	type ListingSettings = { template: string; warrantyDisclaimer: string };
	let listing = $state<ListingSettings | null>(null);
	let listingError = $state('');
	let listingNotice = $state('');
	let listingBusy = $state(false);

	api<ListingSettings>('GET', '/api/settings/listing')
		.then((l) => (listing = l))
		.catch((e) => (listingError = errorText(e)));

	// The preview fills the template like the server does: empty paragraphs drop out, the disclaimer comes last.
	const preview = $derived.by(() => {
		if (!listing) return '';
		const filled = listing.template
			.replaceAll('{name}', 'Wasserkocher')
			.replaceAll('{beschreibung}', 'Edelstahl, 1,7 l, funktioniert einwandfrei.')
			.replaceAll('{preis}', '15 €');
		return [...filled.split('\n\n'), listing.warrantyDisclaimer]
			.map((p) => p.trim())
			.filter(Boolean)
			.join('\n\n');
	});

	async function saveListing(event: SubmitEvent) {
		event.preventDefault();
		if (!listing) return;
		listingError = '';
		listingNotice = '';
		listingBusy = true;
		try {
			listing = await api<ListingSettings>('PUT', '/api/settings/listing', listing);
			listingNotice = 'Gespeichert.';
		} catch (e) {
			listingError = errorText(e);
		} finally {
			listingBusy = false;
		}
	}

	// The theme takes effect immediately; if saving it for the user fails, the previous one applies again.
	async function choose(t: Theme) {
		error = '';
		try {
			await saveTheme(t);
		} catch (e) {
			error = `Darstellung nicht gespeichert: ${errorText(e)}`;
		}
	}
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		<h1>Einstellungen</h1>
	</header>

	<section>
		<!-- Real radio buttons in cards; arrow keys switch as usual. -->
		<fieldset class="theme-picker" aria-describedby="theme-note">
			<legend><h2>Darstellung</h2></legend>
			<p id="theme-note" class="muted">Gilt für dich auf allen Geräten.</p>
			<div class="theme-cards">
				{#each THEMES as t (t.value)}
					<label class="theme-card">
						<span class="theme-preview" data-preview={t.value} aria-hidden="true">
							<span></span><span></span><span></span>
						</span>
						<span class="theme-card-row">
							<input
								type="radio"
								name="theme"
								value={t.value}
								checked={appearance.theme === t.value}
								onchange={() => choose(t.value)}
							/>
							<span class="theme-card-text">
								<span class="theme-card-name">{t.name}</span>
								<span class="muted small-text">{t.description}</span>
							</span>
						</span>
					</label>
				{/each}
			</div>
		</fieldset>

		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
	</section>

	<section>
		<h2>Inserate</h2>
		<p class="muted">Gilt für den ganzen Haushalt. Platzhalter: {'{name}'}, {'{beschreibung}'}, {'{preis}'}. Der Gewährleistungsausschluss steht immer am Ende.</p>
		{#if listing}
			<form onsubmit={saveListing}>
				<label>
					Vorlage
					<textarea bind:value={listing.template} rows="6"></textarea>
				</label>
				<label>
					Gewährleistungsausschluss
					<textarea bind:value={listing.warrantyDisclaimer} rows="3" required></textarea>
				</label>
				<div class="field-name">Vorschau</div>
				<p class="notice description">{preview}</p>
				{#if listingError}<p class="error" role="alert">{listingError}</p>{/if}
				{#if listingNotice}<p class="muted" role="status">{listingNotice}</p>{/if}
				<div class="button-row">
					<button type="submit" disabled={listingBusy}>Speichern</button>
				</div>
			</form>
		{:else if listingError}
			<p class="error" role="alert">{listingError}</p>
		{/if}
	</section>
</main>
