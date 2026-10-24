<script lang="ts">
	// Dreht ein Foto in 90°-Schritten und schneidet es frei zu, als modaler <dialog> über der Seite.
	// Der Zuschnitt lässt sich mit Finger oder Maus (Pointer Events) und per Pfeiltasten verschieben.
	// fertig bekommt das bearbeitete Foto als JPEG, oder null bei Abbruch oder ohne Änderung.
	import { onMount } from 'svelte';
	import {
		bearbeitetesFoto,
		GANZES_BILD,
		gedrehteGroesse,
		istGanzesBild,
		kantenDrehen,
		verschieben,
		vorschauZeichnen,
		type Drehung,
		type Griff,
		type Kanten
	} from '#lib/zuschnitt.ts';

	let { quelle, fertig }: { quelle: Blob; fertig: (datei: File | null) => void } = $props();

	// Längere Kante der Vorschau im Dialog; das Ergebnis entsteht aus dem vollen Bild.
	const VORSCHAU_KANTE = 1600;
	// Ein Pfeiltastendruck verschiebt um diesen Anteil der Bildbreite bzw. -höhe.
	const SCHRITT = 0.02;

	const ECKEN: { griff: Exclude<Griff, 'ganz'>; name: string }[] = [
		{ griff: { x: 'l', y: 'o' }, name: 'Linke obere Ecke' },
		{ griff: { x: 'r', y: 'o' }, name: 'Rechte obere Ecke' },
		{ griff: { x: 'l', y: 'u' }, name: 'Linke untere Ecke' },
		{ griff: { x: 'r', y: 'u' }, name: 'Rechte untere Ecke' }
	];

	let dialog: HTMLDialogElement;
	let buehne = $state<HTMLDivElement>();
	let leinwand = $state<HTMLCanvasElement>();
	let bild = $state<HTMLImageElement | null>(null);
	let drehung = $state<Drehung>(0);
	let kanten = $state<Kanten>(GANZES_BILD);
	let fehler = $state('');
	let laeuft = $state(false);

	const seitenverhaeltnis = $derived.by(() => {
		if (!bild) return 1;
		const [w, h] = gedrehteGroesse(bild, drehung);
		return w / h;
	});
	const geaendert = $derived(drehung !== 0 || !istGanzesBild(kanten));

	onMount(() => {
		dialog.showModal();
		const adresse = URL.createObjectURL(quelle);
		const img = new Image();
		img.src = adresse;
		img.decode()
			.then(() => (bild = img))
			.catch(() => (fehler = 'Das Foto lässt sich nicht öffnen.'));
		return () => URL.revokeObjectURL(adresse);
	});

	$effect(() => {
		if (bild && leinwand) vorschauZeichnen(leinwand, bild, drehung, VORSCHAU_KANTE);
	});

	function drehen(imUhrzeigersinn: boolean) {
		drehung = ((drehung + (imUhrzeigersinn ? 1 : 3)) % 4) as Drehung;
		kanten = kantenDrehen(kanten, imUhrzeigersinn);
	}

	let zug: { griff: Griff; pointerId: number; x: number; y: number; kanten: Kanten; breite: number; hoehe: number } | null = null;

	function zugBeginnen(e: PointerEvent, griff: Griff) {
		if (!buehne) return;
		e.preventDefault();
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		const r = buehne.getBoundingClientRect();
		zug = { griff, pointerId: e.pointerId, x: e.clientX, y: e.clientY, kanten, breite: r.width, hoehe: r.height };
	}

	function zugBewegen(e: PointerEvent) {
		if (!zug || e.pointerId !== zug.pointerId) return;
		kanten = verschieben(zug.kanten, zug.griff, (e.clientX - zug.x) / zug.breite, (e.clientY - zug.y) / zug.hoehe);
	}

	function zugBeenden(e: PointerEvent) {
		if (zug?.pointerId === e.pointerId) zug = null;
	}

	function taste(e: KeyboardEvent, griff: Griff) {
		const richtung: Record<string, [number, number]> = {
			ArrowLeft: [-1, 0],
			ArrowRight: [1, 0],
			ArrowUp: [0, -1],
			ArrowDown: [0, 1]
		};
		const r = richtung[e.key];
		if (!r) return;
		e.preventDefault();
		kanten = verschieben(kanten, griff, r[0] * SCHRITT, r[1] * SCHRITT);
	}

	function abbrechen() {
		dialog.close();
		fertig(null);
	}

	async function uebernehmen() {
		if (!bild || !geaendert) return abbrechen();
		laeuft = true;
		try {
			const datei = await bearbeitetesFoto(bild, drehung, kanten);
			dialog.close();
			fertig(datei);
		} catch {
			fehler = 'Das Foto ließ sich nicht bearbeiten.';
		} finally {
			laeuft = false;
		}
	}

	const prozent = (wert: number) => `${wert * 100}%`;
</script>

<dialog
	class="foto-bearbeiten"
	aria-label="Foto drehen und zuschneiden"
	bind:this={dialog}
	oncancel={(e) => {
		e.preventDefault();
		abbrechen();
	}}
>
	<div class="bearbeiten-inhalt">
		<h2>Drehen und zuschneiden</h2>
		{#if bild}
			<div
				class="buehne"
				bind:this={buehne}
				style:aspect-ratio={seitenverhaeltnis}
				style:width="min(100%, calc(62dvh * {seitenverhaeltnis}))"
			>
				<canvas bind:this={leinwand} aria-hidden="true"></canvas>
				<div class="abdunklung" aria-hidden="true">
					<div
						class="ausschnitt-schatten"
						style:left={prozent(kanten.l)}
						style:top={prozent(kanten.o)}
						style:width={prozent(kanten.r - kanten.l)}
						style:height={prozent(kanten.u - kanten.o)}
					></div>
				</div>
				<button
					type="button"
					class="ausschnitt"
					aria-label="Zuschnitt verschieben (Pfeiltasten)"
					style:left={prozent(kanten.l)}
					style:top={prozent(kanten.o)}
					style:width={prozent(kanten.r - kanten.l)}
					style:height={prozent(kanten.u - kanten.o)}
					onpointerdown={(e) => zugBeginnen(e, 'ganz')}
					onpointermove={zugBewegen}
					onpointerup={zugBeenden}
					onpointercancel={zugBeenden}
					onkeydown={(e) => taste(e, 'ganz')}
				></button>
				{#each ECKEN as ecke (ecke.name)}
					<button
						type="button"
						class="griff"
						aria-label="{ecke.name} des Zuschnitts verschieben (Pfeiltasten)"
						style:left={prozent(kanten[ecke.griff.x])}
						style:top={prozent(kanten[ecke.griff.y])}
						onpointerdown={(e) => zugBeginnen(e, ecke.griff)}
						onpointermove={zugBewegen}
						onpointerup={zugBeenden}
						onpointercancel={zugBeenden}
						onkeydown={(e) => taste(e, ecke.griff)}
					></button>
				{/each}
			</div>
		{:else if !fehler}
			<p class="leise">Lädt …</p>
		{/if}

		{#if fehler}
			<p class="fehler" role="alert">{fehler}</p>
		{/if}
		<div class="knopfzeile">
			<button type="button" class="sekundaer" disabled={!bild} onclick={() => drehen(false)} aria-label="Um 90° nach links drehen">↺ Links</button>
			<button type="button" class="sekundaer" disabled={!bild} onclick={() => drehen(true)} aria-label="Um 90° nach rechts drehen">↻ Rechts</button>
			<button type="button" class="sekundaer" disabled={istGanzesBild(kanten)} onclick={() => (kanten = GANZES_BILD)}>Ganzes Bild</button>
		</div>
		<div class="knopfzeile">
			<button type="button" disabled={!bild || laeuft} onclick={uebernehmen}>{laeuft ? 'Übernehmen …' : 'Übernehmen'}</button>
			<button type="button" class="sekundaer" onclick={abbrechen}>Abbrechen</button>
		</div>
	</div>
</dialog>
