<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText, unpackPath, type Box } from '#lib/api.ts';
	import BoxLabel from '#lib/BoxLabel.svelte';

	// Minimal typing of Web NFC (Chrome on Android); TypeScript's DOM types don't include it.
	type NDEFWriter = { write(message: { records: { recordType: 'url'; data: string }[] }): Promise<void> };
	type NDEFReaderConstructor = new () => NDEFWriter;

	const id = $derived(Number(page.params.id));
	const link = $derived(`${location.origin}${unpackPath(id)}`);
	const nfcAvailable = 'NDEFReader' in window;

	let box = $state<Box | null>(null);
	let error = $state('');
	let copyState = $state<'' | 'copied' | 'failed'>('');
	let nfcState = $state<'' | 'waiting' | 'written' | 'failed'>('');
	let nfcError = $state('');

	$effect(() => {
		const boxID = id;
		box = null;
		error = '';
		api<{ box: Box }>('GET', `/api/boxes/${boxID}`)
			.then((r) => (box = r.box))
			.catch((e) => (error = errorText(e)));
	});

	async function copyLink() {
		try {
			await navigator.clipboard.writeText(link);
			copyState = 'copied';
		} catch {
			// Without clipboard access the link is shown below to copy by hand.
			copyState = 'failed';
		}
	}

	async function writeNfc() {
		nfcState = 'waiting';
		nfcError = '';
		try {
			const Reader = (window as unknown as { NDEFReader: NDEFReaderConstructor }).NDEFReader;
			await new Reader().write({ records: [{ recordType: 'url', data: link }] });
			nfcState = 'written';
		} catch (e) {
			nfcState = 'failed';
			nfcError = e instanceof DOMException && e.name === 'NotAllowedError' ? 'NFC ist nicht erlaubt oder ausgeschaltet.' : 'Der Tag ließ sich nicht beschreiben.';
		}
	}
</script>

<main class="page">
	<header class="header no-print">
		<a class="button secondary" href="/boxen/{id}">← Box</a>
		<h1>Etikett</h1>
	</header>

	{#if error}
		<p class="error no-print" role="alert">{error}</p>
	{/if}

	{#if box}
		<BoxLabel {box} />

		<div class="no-print label-actions">
			<div class="button-row">
				<button onclick={() => window.print()}>Drucken</button>
				<button class="secondary" onclick={copyLink}>Link kopieren</button>
				{#if nfcAvailable}
					<button class="secondary" disabled={nfcState === 'waiting'} onclick={writeNfc}>Auf NFC-Tag schreiben</button>
				{/if}
			</div>
			{#if copyState === 'copied'}
				<p role="status">Link kopiert.</p>
			{:else if copyState === 'failed'}
				<label>
					Kopieren ging nicht; Link zum Markieren:
					<input readonly value={link} onfocus={(e) => e.currentTarget.select()} />
				</label>
			{/if}
			{#if nfcState === 'waiting'}
				<p role="status">Halte den NFC-Tag jetzt an die Rückseite des Handys …</p>
			{:else if nfcState === 'written'}
				<p role="status">Auf den NFC-Tag geschrieben.</p>
			{:else if nfcState === 'failed'}
				<p class="error" role="alert">{nfcError}</p>
			{/if}
			<p class="muted small-text">
				iPhones und Android-Handys öffnen den Link vom QR-Code oder NFC-Tag ohne App. Einen Tag beschreiben kannst du
				{nfcAvailable ? 'hier direkt oder' : ''} auf dem iPhone mit einer App wie „NFC Tools“: Link kopieren und dort als URL schreiben.
				Auf dem iPhone öffnet sich der Link in Safari statt in der installierten App; dort meldest du dich einmal separat an.
			</p>
		</div>
	{:else if !error}
		<p class="muted">Lädt …</p>
	{/if}
</main>
