<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, boxName, errorText, dispositionName, type Photo, type Item, type Move, type Apartment, type Person, type Place, type PlaceEntry, type Disposition } from '#lib/api.ts';
	import DispositionPicker from '#lib/DispositionPicker.svelte';
	import PhotoEditor from '#lib/PhotoEditor.svelte';
	import PhotoInput from '#lib/PhotoInput.svelte';
	import { loadPlaces, placementBody, confirmRemoval } from '#lib/places.ts';
	import { loadMove, loadApartments, type MoveDetailsChange } from '#lib/move.ts';
	import PathView from '#lib/PathView.svelte';
	import PlacePicker from '#lib/PlacePicker.svelte';
	import { loadPersons } from '#lib/persons.ts';

	const path = $derived(`/api/items/${page.params.id}`);

	let item = $state<Item | null>(null);
	let persons = $state<Person[]>([]);
	let shown = $state<number | null>(null);
	let error = $state('');
	let busy = $state(false);
	let editing = $state(false);
	let draft = $state({ name: '', description: '', disposition: null as Disposition | null });

	const photo = $derived(item?.photos.find((p) => p.id === shown) ?? item?.photos.find((p) => p.mainPhoto));
	const otherPersons = $derived(persons.filter((p) => !item?.owners.some((o) => o.id === p.id)));

	$effect(() => {
		const p = path;
		item = null;
		error = '';
		api<Item>('GET', p)
			.then((i) => (item = i))
			.catch((e) => (error = errorText(e)));
	});

	loadPersons()
		.then((p) => (persons = p))
		.catch((e) => (error = errorText(e)));

	/** Performs a change that returns the item, and shows errors. */
	async function change(method: string, url: string, body?: unknown) {
		error = '';
		busy = true;
		try {
			item = await api<Item>(method, url, body);
			return true;
		} catch (e) {
			error = errorText(e);
			return false;
		} finally {
			busy = false;
		}
	}

	function startEditing() {
		if (!item) return;
		draft = { name: item.name, description: item.description, disposition: item.disposition };
		editing = true;
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (await change('PUT', path, draft)) editing = false;
	}

	function uploadPhotos(files: File[]) {
		const form = new FormData();
		for (const f of files) form.append('photo', f);
		change('POST', `${path}/photos`, form);
	}

	// Saved photo currently being rotated or cropped, with its loaded original.
	let photoEdit = $state<{ photoId: number; source: Blob } | null>(null);

	async function startPhotoEdit(p: Photo) {
		error = '';
		busy = true;
		try {
			const res = await fetch(p.url);
			if (!res.ok) throw new Error(res.statusText);
			photoEdit = { photoId: p.id, source: await res.blob() };
		} catch {
			error = 'Das Foto ließ sich nicht laden.';
		} finally {
			busy = false;
		}
	}

	// The edited photo replaces the original; the server gives it a new URL.
	function photoEditDone(file: File | null) {
		const id = photoEdit?.photoId;
		photoEdit = null;
		if (id === undefined || !file) return;
		const form = new FormData();
		form.append('photo', file);
		change('PUT', `/api/photos/${id}`, form);
	}

	async function deletePhoto(id: number) {
		if (!confirm('Dieses Foto löschen?')) return;
		if (await change('DELETE', `/api/photos/${id}`)) shown = null;
	}

	let newOwner = $state('');
	async function addOwner() {
		if (!newOwner) return;
		if (await change('PUT', `${path}/owners/${newOwner}`)) newOwner = '';
	}

	let places = $state<PlaceEntry[]>([]);
	let choosingPlace = $state(false);
	let place = $state<Place | null>(null);

	async function startPlaceChange() {
		try {
			places = await loadPlaces();
		} catch (e) {
			error = errorText(e);
			return;
		}
		const p = item?.placement;
		const innermost = p?.storageLocations.at(-1);
		place = innermost ? { kind: 'storageLocation', id: innermost.id } : p ? { kind: 'room', id: p.room.id } : null;
		choosingPlace = true;
	}

	async function savePlace() {
		if (await change('PUT', `${path}/placement`, placementBody(place))) choosingPlace = false;
	}

	// Move details, while a move is active.
	let move = $state<Move | null>(null);
	let apartments = $state<Apartment[]>([]);
	const newRooms = $derived(apartments.find((a) => a.id === move?.newApartment.id)?.rooms ?? []);
	let details = $state({ origin: '', target: '', box: '' });
	let detailsBefore = { origin: '', target: '', box: '' };

	loadMove()
		.then(async (m) => {
			move = m;
			if (m) apartments = await loadApartments();
		})
		.catch((e) => (error = errorText(e)));

	$effect(() => {
		const d = item?.move;
		detailsBefore = {
			origin: d?.originRoom ? String(d.originRoom.id) : '',
			target: d?.targetRoom ? String(d.targetRoom.id) : '',
			box: d?.box ? String(d.box.id) : ''
		};
		details = { ...detailsBefore };
	});

	// Only send what changed: otherwise an empty field would clear the preset origin room.
	function saveDetails() {
		const id = (v: string) => (v ? Number(v) : null);
		const body: MoveDetailsChange = {};
		if (details.origin !== detailsBefore.origin) body.originRoomId = id(details.origin);
		if (details.target !== detailsBefore.target) body.targetRoomId = id(details.target);
		if (details.box !== detailsBefore.box) body.boxId = id(details.box);
		if (Object.keys(body).length > 0) change('PATCH', `${path}/move`, body);
	}

	async function deleteItem() {
		if (!item) return;
		error = '';
		try {
			const question = `„${item.name}“ mit allen Fotos löschen?`;
			const confirmed =
				item.furnitureStorageLocations.length > 0
					? await confirmRemoval(`${path}/removal`, `${question} Seine Lagerorte (${item.furnitureStorageLocations.map((s) => s.name).join(', ')}) fallen weg.`)
					: confirm(question);
			if (!confirmed) return;
			await api('DELETE', path);
			await goto('/', { replaceState: true });
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<main class="page wide">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		{#if item && !editing}
			<button class="secondary" onclick={startEditing}>Bearbeiten</button>
		{/if}
	</header>

	{#if error}
		<p class="error" role="alert">{error}</p>
	{/if}

	{#if item}
		<div class="two-column">
			<div class="photo-column">
				{#if photo}
					<figure class="large-photo">
						<a href={photo.url} target="_blank" rel="noopener"><img src={photo.thumbnailUrl} alt={item.name} /></a>
						{#if item.photos.length > 1}
							<button type="button" class="photo-delete" aria-label="Foto löschen" disabled={busy} onclick={() => deletePhoto(photo.id)}>
								×
							</button>
						{/if}
					</figure>
				{/if}
				<ul class="photo-strip">
					{#each item.photos as p (p.id)}
						<li>
							<button type="button" class="photo-choice" class:active={p.id === photo?.id} onclick={() => (shown = p.id)}>
								<img src={p.thumbnailUrl} alt={p.mainPhoto ? 'Hauptfoto' : 'Foto'} loading="lazy" />
							</button>
						</li>
					{/each}
				</ul>
				{#if photo}
					<div class="button-row">
						{#if !photo.mainPhoto}
							<button class="secondary" disabled={busy} onclick={() => change('PUT', `${path}/main-photo`, { photoId: photo.id })}>
								Als Hauptfoto
							</button>
						{:else}
							<span class="badge">Hauptfoto</span>
						{/if}
						<button class="secondary" disabled={busy} onclick={() => startPhotoEdit(photo)}>Drehen / Zuschneiden</button>
					</div>
				{/if}
				<PhotoInput chosen={uploadPhotos} disabled={busy} />
				{#if photoEdit}
					<PhotoEditor source={photoEdit.source} done={photoEditDone} />
				{/if}
			</div>

			<div class="details-column">
				{#if editing}
					<form onsubmit={save}>
						<label>
							Name
							<input bind:value={draft.name} required />
						</label>
						<label>
							Beschreibung
							<textarea bind:value={draft.description} rows="3"></textarea>
						</label>
						<DispositionPicker bind:value={draft.disposition} />
						<div class="button-row">
							<button type="submit" disabled={busy}>Speichern</button>
							<button type="button" class="secondary" onclick={() => (editing = false)}>Abbrechen</button>
						</div>
					</form>
				{:else}
					<h1>{item.name}</h1>
					{#if item.description}
						<p class="description">{item.description}</p>
					{/if}
					<p>Verbleib: <span class="disposition" data-disposition={item.disposition ?? 'undecided'}>{dispositionName(item.disposition)}</span></p>
				{/if}

				<section>
					<h2>Platz</h2>
					{#if choosingPlace}
						<PlacePicker bind:places bind:value={place} create />
						<div class="button-row">
							<button disabled={busy} onclick={savePlace}>Speichern</button>
							<button class="secondary" onclick={() => (choosingPlace = false)}>Abbrechen</button>
						</div>
					{:else}
						{#if item.placement}
							<PathView path={item.placement} />
							{#if item.placement.storageLocations.length === 0}<p class="muted">Direkt im Raum, ohne Lagerort.</p>{/if}
						{:else}
							<p class="muted">Noch nicht platziert.</p>
						{/if}
						<button class="secondary small" onclick={startPlaceChange}>Platz ändern</button>
					{/if}
					{#if item.furnitureStorageLocations.length > 0}
						<p>
							Lagerorte dieses Möbelstücks:
							{#each item.furnitureStorageLocations as s, i (s.id)}{i > 0 ? ', ' : ''}<a href="/lagerorte/{s.id}">{s.name}</a>{/each}
						</p>
					{/if}
				</section>

				{#if move}
					<section>
						<h2>Umzug</h2>
						<div class="filter">
							<label>
								Herkunftsraum
								<select bind:value={details.origin}>
									<option value="">–</option>
									{#each apartments as a (a.id)}
										<optgroup label={a.name}>
											{#each a.rooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
										</optgroup>
									{/each}
								</select>
							</label>
							<label>
								Zielraum
								<select bind:value={details.target}>
									<option value="">–</option>
									{#each newRooms as r (r.id)}<option value={String(r.id)}>{r.name}</option>{/each}
								</select>
							</label>
							<label class="span-all">
								Box
								<select bind:value={details.box}>
									<option value="">Keine</option>
									{#each move.boxes as b (b.id)}<option value={String(b.id)}>{boxName(b)}</option>{/each}
								</select>
							</label>
						</div>
						<button class="secondary" disabled={busy} onclick={saveDetails}>Umzugsangaben speichern</button>
					</section>
				{/if}

				<section>
					<h2>Besitzer</h2>
					<ul class="rows">
						{#each item.owners as o (o.id)}
							<li>
								<span>{o.name}</span>
								<button class="secondary small" disabled={busy} onclick={() => change('DELETE', `${path}/owners/${o.id}`)} aria-label="{o.name} als Besitzer entfernen">
									Entfernen
								</button>
							</li>
						{/each}
					</ul>
					{#if otherPersons.length > 0}
						<div class="button-row">
							<select aria-label="Besitzer hinzufügen" bind:value={newOwner}>
								<option value="">Person wählen …</option>
								{#each otherPersons as p (p.id)}
									<option value={String(p.id)}>{p.name}</option>
								{/each}
							</select>
							<button class="secondary" disabled={busy || !newOwner} onclick={addOwner}>Hinzufügen</button>
						</div>
					{/if}
				</section>

				<section>
					<button class="danger" onclick={deleteItem}>Item löschen</button>
				</section>
			</div>
		</div>
	{:else if !error}
		<p class="muted">Lädt …</p>
	{/if}
</main>
