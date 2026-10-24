<script lang="ts">
	// Asks for the label (optional) and the target room of a new box, as a modal <dialog>, and creates it.
	// done receives the new box, or null on cancel.
	import { onMount } from 'svelte';
	import { api, errorText, type Box, type Named } from './api.ts';
	import type { BoxChangeBody } from './move.ts';

	let { rooms, done }: { rooms: Named[]; done: (box: Box | null) => void } = $props();

	let dialog: HTMLDialogElement;
	let label = $state('');
	let targetRoom = $state('');
	let error = $state('');
	let busy = $state(false);

	onMount(() => dialog.showModal());

	async function create(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		try {
			const body: BoxChangeBody = { label: label.trim(), targetRoomId: targetRoom ? Number(targetRoom) : null };
			const box = await api<Box>('POST', '/api/move/boxes', body);
			dialog.close();
			done(box);
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}

	function cancel() {
		dialog.close();
		done(null);
	}
</script>

<dialog class="box-dialog" bind:this={dialog} aria-labelledby="new-box-title" oncancel={(e) => (e.preventDefault(), cancel())}>
	<form onsubmit={create}>
		<h2 id="new-box-title">Neue Box</h2>
		<label>
			Bezeichnung (optional)
			<input bind:value={label} placeholder="z. B. Box A, Bücher" />
		</label>
		<label>
			Zielraum
			<select bind:value={targetRoom}>
				<option value="">Noch offen</option>
				{#each rooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
			</select>
		</label>
		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
		<div class="button-row">
			<button type="submit" disabled={busy}>Anlegen</button>
			<button type="button" class="secondary" onclick={cancel}>Abbrechen</button>
		</div>
	</form>
</dialog>
