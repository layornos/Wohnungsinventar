<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText, unpackPath, type Box } from '#lib/api.ts';
	import BoxLabel from '#lib/BoxLabel.svelte';
	import LabelActions from '#lib/LabelActions.svelte';

	const id = $derived(Number(page.params.id));
	const link = $derived(`${location.origin}${unpackPath(id)}`);

	let box = $state<Box | null>(null);
	let error = $state('');

	$effect(() => {
		const boxID = id;
		box = null;
		error = '';
		api<{ box: Box }>('GET', `/api/boxes/${boxID}`)
			.then((r) => (box = r.box))
			.catch((e) => (error = errorText(e)));
	});
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
		<LabelActions {link} />
	{:else if !error}
		<p class="muted">Lädt …</p>
	{/if}
</main>
