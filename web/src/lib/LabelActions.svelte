<script lang="ts">
	// Actions under a printable label: print, copy the link of the QR code and, where Web NFC exists
	// (Chrome on Android), write the link to an NFC tag.

	// Minimal typing of Web NFC; TypeScript's DOM types don't include it.
	type NDEFWriter = { write(message: { records: { recordType: 'url'; data: string }[] }): Promise<void> };
	type NDEFReaderConstructor = new () => NDEFWriter;

	let { link }: { link: string } = $props();

	const nfcAvailable = 'NDEFReader' in window;

	let copyState = $state<'' | 'copied' | 'failed'>('');
	let nfcState = $state<'' | 'waiting' | 'written' | 'failed'>('');
	let nfcError = $state('');

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
