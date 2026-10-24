<script lang="ts">
	import { untrack } from 'svelte';
	import { suggest, suggestionsEnabled, type Suggestion } from './suggestions.ts';

	// Proposes name and description from the first photo of a new item while the name is still
	// empty. Runs in the background: it never blocks saving and only fills fields that are still empty.
	let {
		photo,
		name = $bindable(),
		description = $bindable()
	}: { photo: File | undefined; name: string; description: string } = $props();

	let enabled = $state(false);
	let requestedFor: File | undefined;
	let latest = 0;
	let busy = $state(false);
	let failed = $state(false);
	let suggested = $state<Suggestion | null>(null);

	suggestionsEnabled().then((e) => (enabled = e));

	$effect(() => {
		const file = photo;
		if (!enabled || !file || file === requestedFor) return;
		requestedFor = file;
		if (untrack(() => name.trim() !== '')) return;
		run(file);
	});

	async function run(file: File) {
		const request = ++latest;
		busy = true;
		failed = false;
		suggested = null;
		try {
			const s = await suggest(file);
			// Stale if the photo changed meanwhile, e.g. because the item was saved.
			if (photo !== file) return;
			if (name.trim() === '') name = s.name;
			if (description.trim() === '') description = s.description;
			suggested = s;
		} catch {
			if (photo === file) failed = true;
		} finally {
			if (request === latest) busy = false;
		}
	}

	// The mark disappears as soon as the name is changed by hand.
	const marked = $derived(suggested !== null && name !== '' && name === suggested.name);
</script>

{#if busy}
	<p class="suggestion muted" role="status">Vorschlag wird erstellt …</p>
{:else if marked}
	<p class="suggestion muted"><span class="badge">Vorschlag</span> aus dem Foto, bitte prüfen.</p>
{:else if failed}
	<p class="suggestion muted">Kein Vorschlag möglich.</p>
{/if}

<style>
	.suggestion {
		margin: 0;
		font-size: 0.85em;
	}
</style>
