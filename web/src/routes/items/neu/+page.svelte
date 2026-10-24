<script lang="ts">
	import { page } from '$app/state';
	import { api, errorText, type Item, type Person, type Place, type PlaceEntry, type Disposition } from '#lib/api.ts';
	import DispositionPicker from '#lib/DispositionPicker.svelte';
	import OwnerPicker from '#lib/OwnerPicker.svelte';
	import PhotoEditor from '#lib/PhotoEditor.svelte';
	import PhotoInput from '#lib/PhotoInput.svelte';
	import NameSuggestion from '#lib/NameSuggestion.svelte';
	import { loadPlaces, lastPlace, rememberLastPlace } from '#lib/places.ts';
	import { loadPersons } from '#lib/persons.ts';
	import PlacePicker from '#lib/PlacePicker.svelte';
	import { session } from '#lib/session.svelte.ts';

	type Chosen = { file: File; preview: string };

	let persons = $state<Person[]>([]);
	let photos = $state<Chosen[]>([]);
	let name = $state('');
	let description = $state('');
	let owners = $state<number[]>(session.user ? [session.user.personId] : []);
	let disposition = $state<Disposition | null>(null);
	let error = $state('');
	let busy = $state(false);
	let saved = $state<Item | null>(null);
	// From the packing mode new items go straight into the active box.
	const box = page.url.searchParams.get('box');
	let places = $state<PlaceEntry[]>([]);
	let place = $state<Place | null>(null);

	loadPersons()
		.then((p) => (persons = p))
		.catch((e) => (error = errorText(e)));

	// The last chosen place is preselected so a shelf can be worked through quickly.
	loadPlaces()
		.then((p) => {
			places = p;
			const last = lastPlace();
			place = p.some((e) => e.kind === last?.kind && e.id === last?.id) ? last : null;
		})
		.catch((e) => (error = errorText(e)));

	function add(files: File[]) {
		photos = [...photos, ...files.map((file) => ({ file, preview: URL.createObjectURL(file) }))];
	}

	// Index of the photo currently being rotated or cropped.
	let editing = $state<number | null>(null);

	function editDone(file: File | null) {
		const i = editing;
		editing = null;
		if (i === null || !file) return;
		URL.revokeObjectURL(photos[i].preview);
		photos[i] = { file, preview: URL.createObjectURL(file) };
	}

	function remove(i: number) {
		URL.revokeObjectURL(photos[i].preview);
		photos = photos.filter((_, j) => j !== i);
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		if (photos.length === 0) {
			error = 'Bitte mindestens ein Foto aufnehmen.';
			return;
		}
		if (owners.length === 0) {
			error = 'Bitte mindestens einen Besitzer wählen.';
			return;
		}
		const form = new FormData();
		form.set('name', name);
		form.set('description', description);
		if (disposition) form.set('disposition', disposition);
		if (place) form.set(place.kind, String(place.id));
		if (box) form.set('box', box);
		for (const id of owners) form.append('owner', String(id));
		for (const p of photos) form.append('photo', p.file);

		busy = true;
		try {
			saved = await api<Item>('POST', '/api/items', form);
			rememberLastPlace(place);
			// Ready for the next item; owners and place stay, because similar items often follow.
			for (const p of photos) URL.revokeObjectURL(p.preview);
			photos = [];
			name = '';
			description = '';
			disposition = null;
		} catch (e) {
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<main class="page wide">
	<header class="header">
		<a class="button secondary" href={box ? '/umzug/packen?box=' + box : '/'}>{box ? '← Packen' : '← Liste'}</a>
		<h1>Erfassen</h1>
	</header>
	{#if box}<p class="notice">Neue Items kommen direkt in die aktive Box.</p>{/if}

	{#if saved}
		<p class="notice" role="status">
			Gespeichert: <a href="/items/{saved.id}">{saved.name}</a>
		</p>
	{/if}

	<form class="two-column" onsubmit={save}>
		<div class="photo-column">
			{#if photos.length > 0}
				<ul class="photo-strip">
					{#each photos as p, i (p.preview)}
						<li>
							<img src={p.preview} alt="Foto {i + 1}" />
							<button type="button" class="secondary small" onclick={() => remove(i)} aria-label="Foto {i + 1} entfernen">×</button>
							<button type="button" class="secondary small photo-edit-button" onclick={() => (editing = i)} aria-label="Foto {i + 1} drehen oder zuschneiden">✎</button>
						</li>
					{/each}
				</ul>
			{/if}
			<PhotoInput chosen={add} />
			{#if editing !== null}
				<PhotoEditor source={photos[editing].file} done={editDone} />
			{/if}
		</div>

		<div class="details-column">
			<label>
				Name
				<input bind:value={name} required autocomplete="off" />
			</label>
			<label>
				Beschreibung (optional)
				<textarea bind:value={description} rows="2"></textarea>
			</label>
			<NameSuggestion photo={photos[0]?.file} bind:name bind:description />
			<OwnerPicker {persons} bind:selected={owners} />
			<PlacePicker bind:places bind:value={place} create />
			<DispositionPicker bind:value={disposition} />

			{#if error}
				<p class="error" role="alert">{error}</p>
			{/if}
			<button type="submit" disabled={busy}>{busy ? 'Speichern …' : 'Speichern'}</button>
		</div>
	</form>
</main>
