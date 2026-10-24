<script lang="ts">
	// Rotates a photo in 90° steps and crops it freely, as a modal <dialog> above the page.
	// The crop can be moved with finger or mouse (pointer events) and with the arrow keys.
	// done receives the edited photo as JPEG, or null on cancel or without changes.
	import { onMount } from 'svelte';
	import {
		editedPhoto,
		WHOLE_IMAGE,
		rotatedSize,
		isWholeImage,
		rotateEdges,
		move,
		drawPreview,
		type Rotation,
		type Handle,
		type Edges
	} from '#lib/crop.ts';

	let { source, done }: { source: Blob; done: (file: File | null) => void } = $props();

	// Longer edge of the preview in the dialog; the result is made from the full image.
	const PREVIEW_EDGE = 1600;
	// One arrow key press moves by this fraction of the image width or height.
	const STEP = 0.02;

	const CORNERS: { handle: Exclude<Handle, 'whole'>; name: string }[] = [
		{ handle: { x: 'l', y: 't' }, name: 'Linke obere Ecke' },
		{ handle: { x: 'r', y: 't' }, name: 'Rechte obere Ecke' },
		{ handle: { x: 'l', y: 'b' }, name: 'Linke untere Ecke' },
		{ handle: { x: 'r', y: 'b' }, name: 'Rechte untere Ecke' }
	];

	let dialog: HTMLDialogElement;
	let stage = $state<HTMLDivElement>();
	let canvas = $state<HTMLCanvasElement>();
	let image = $state<HTMLImageElement | null>(null);
	let rotation = $state<Rotation>(0);
	let edges = $state<Edges>(WHOLE_IMAGE);
	let error = $state('');
	let busy = $state(false);

	const aspectRatio = $derived.by(() => {
		if (!image) return 1;
		const [w, h] = rotatedSize(image, rotation);
		return w / h;
	});
	const changed = $derived(rotation !== 0 || !isWholeImage(edges));

	onMount(() => {
		dialog.showModal();
		const url = URL.createObjectURL(source);
		const img = new Image();
		img.src = url;
		img.decode()
			.then(() => (image = img))
			.catch(() => (error = 'Das Foto lässt sich nicht öffnen.'));
		return () => URL.revokeObjectURL(url);
	});

	$effect(() => {
		if (image && canvas) drawPreview(canvas, image, rotation, PREVIEW_EDGE);
	});

	function rotate(clockwise: boolean) {
		rotation = ((rotation + (clockwise ? 1 : 3)) % 4) as Rotation;
		edges = rotateEdges(edges, clockwise);
	}

	let drag: { handle: Handle; pointerId: number; x: number; y: number; edges: Edges; width: number; height: number } | null = null;

	function dragStart(e: PointerEvent, handle: Handle) {
		if (!stage) return;
		e.preventDefault();
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		const r = stage.getBoundingClientRect();
		drag = { handle, pointerId: e.pointerId, x: e.clientX, y: e.clientY, edges, width: r.width, height: r.height };
	}

	function dragMove(e: PointerEvent) {
		if (!drag || e.pointerId !== drag.pointerId) return;
		edges = move(drag.edges, drag.handle, (e.clientX - drag.x) / drag.width, (e.clientY - drag.y) / drag.height);
	}

	function dragEnd(e: PointerEvent) {
		if (drag?.pointerId === e.pointerId) drag = null;
	}

	function key(e: KeyboardEvent, handle: Handle) {
		const direction: Record<string, [number, number]> = {
			ArrowLeft: [-1, 0],
			ArrowRight: [1, 0],
			ArrowUp: [0, -1],
			ArrowDown: [0, 1]
		};
		const d = direction[e.key];
		if (!d) return;
		e.preventDefault();
		edges = move(edges, handle, d[0] * STEP, d[1] * STEP);
	}

	function cancel() {
		dialog.close();
		done(null);
	}

	async function apply() {
		if (!image || !changed) return cancel();
		busy = true;
		try {
			const file = await editedPhoto(image, rotation, edges);
			dialog.close();
			done(file);
		} catch {
			error = 'Das Foto ließ sich nicht bearbeiten.';
		} finally {
			busy = false;
		}
	}

	const percent = (value: number) => `${value * 100}%`;
</script>

<dialog
	class="photo-editor"
	aria-label="Foto drehen und zuschneiden"
	bind:this={dialog}
	oncancel={(e) => {
		e.preventDefault();
		cancel();
	}}
>
	<div class="editor-content">
		<h2>Drehen und zuschneiden</h2>
		{#if image}
			<div
				class="stage"
				bind:this={stage}
				style:aspect-ratio={aspectRatio}
				style:width="min(100%, calc(62dvh * {aspectRatio}))"
			>
				<canvas bind:this={canvas} aria-hidden="true"></canvas>
				<div class="dimming" aria-hidden="true">
					<div
						class="crop-shadow"
						style:left={percent(edges.l)}
						style:top={percent(edges.t)}
						style:width={percent(edges.r - edges.l)}
						style:height={percent(edges.b - edges.t)}
					></div>
				</div>
				<button
					type="button"
					class="crop"
					aria-label="Zuschnitt verschieben (Pfeiltasten)"
					style:left={percent(edges.l)}
					style:top={percent(edges.t)}
					style:width={percent(edges.r - edges.l)}
					style:height={percent(edges.b - edges.t)}
					onpointerdown={(e) => dragStart(e, 'whole')}
					onpointermove={dragMove}
					onpointerup={dragEnd}
					onpointercancel={dragEnd}
					onkeydown={(e) => key(e, 'whole')}
				></button>
				{#each CORNERS as corner (corner.name)}
					<button
						type="button"
						class="handle"
						aria-label="{corner.name} des Zuschnitts verschieben (Pfeiltasten)"
						style:left={percent(edges[corner.handle.x])}
						style:top={percent(edges[corner.handle.y])}
						onpointerdown={(e) => dragStart(e, corner.handle)}
						onpointermove={dragMove}
						onpointerup={dragEnd}
						onpointercancel={dragEnd}
						onkeydown={(e) => key(e, corner.handle)}
					></button>
				{/each}
			</div>
		{:else if !error}
			<p class="muted">Lädt …</p>
		{/if}

		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
		<div class="button-row">
			<button type="button" class="secondary" disabled={!image} onclick={() => rotate(false)} aria-label="Um 90° nach links drehen">↺ Links</button>
			<button type="button" class="secondary" disabled={!image} onclick={() => rotate(true)} aria-label="Um 90° nach rechts drehen">↻ Rechts</button>
			<button type="button" class="secondary" disabled={isWholeImage(edges)} onclick={() => (edges = WHOLE_IMAGE)}>Ganzes Bild</button>
		</div>
		<div class="button-row">
			<button type="button" disabled={!image || busy} onclick={apply}>{busy ? 'Übernehmen …' : 'Übernehmen'}</button>
			<button type="button" class="secondary" onclick={cancel}>Abbrechen</button>
		</div>
	</div>
</dialog>
