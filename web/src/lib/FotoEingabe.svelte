<script lang="ts">
	// Zwei Wege zum Foto: direkt die Kamera oder die Galerie (mehrere auf einmal).
	let { gewaehlt, deaktiviert = false }: { gewaehlt: (dateien: File[]) => void; deaktiviert?: boolean } = $props();

	let kamera: HTMLInputElement;
	let galerie: HTMLInputElement;

	function uebernehmen(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const dateien = [...(input.files ?? [])];
		input.value = '';
		if (dateien.length > 0) gewaehlt(dateien);
	}
</script>

<div class="knopfzeile">
	<button type="button" onclick={() => kamera.click()} disabled={deaktiviert}>Foto aufnehmen</button>
	<button type="button" class="sekundaer" onclick={() => galerie.click()} disabled={deaktiviert}>Aus Galerie</button>
</div>
<input bind:this={kamera} type="file" accept="image/*" capture="environment" hidden onchange={uebernehmen} />
<input bind:this={galerie} type="file" accept="image/*" multiple hidden onchange={uebernehmen} />
