<script lang="ts">
	import { api, ApiError, datumText, fehlerText, type Einladung, type Nutzer, type Person } from '#lib/api.ts';
	import { ladePersonen } from '#lib/personen.ts';

	let nutzer = $state<Nutzer[]>([]);
	let einladungen = $state<Einladung[]>([]);
	let personen = $state<Person[]>([]);
	let fehler = $state('');
	let laeuft = $state(false);

	// "" heißt: neue Person, sonst die ID einer bestehenden Person ohne Nutzerzugang.
	let personWahl = $state('');
	let name = $state('');
	let link = $state<{ name: string; url: string } | null>(null);
	let kopiert = $state(false);

	const ohneZugang = $derived(personen.filter((p) => !p.nutzer));
	const kannTeilen = typeof navigator !== 'undefined' && 'share' in navigator;

	async function neuLaden() {
		try {
			const [n, e, p] = await Promise.all([
				api<{ nutzer: Nutzer[] }>('GET', '/api/nutzer'),
				api<{ einladungen: Einladung[] }>('GET', '/api/einladungen'),
				ladePersonen()
			]);
			nutzer = n.nutzer;
			einladungen = e.einladungen;
			personen = p;
		} catch (e) {
			fehler = fehlerText(e);
		}
	}
	neuLaden();

	async function einladen(event: SubmitEvent) {
		event.preventDefault();
		fehler = '';
		laeuft = true;
		try {
			const body = personWahl ? { personId: Number(personWahl) } : { name };
			const e = await api<Einladung & { link: string }>('POST', '/api/einladungen', body);
			link = { name: e.name, url: e.link };
			kopiert = false;
			personWahl = '';
			name = '';
			await neuLaden();
		} catch (e) {
			fehler =
				e instanceof ApiError && e.code === 'bereits_nutzer'
					? 'Diese Person hat schon einen Nutzerzugang.'
					: fehlerText(e);
		} finally {
			laeuft = false;
		}
	}

	async function kopieren() {
		if (!link) return;
		try {
			await navigator.clipboard.writeText(link.url);
			kopiert = true;
		} catch {
			fehler = 'Kopieren nicht möglich. Bitte den Link markieren und selbst kopieren.';
		}
	}

	async function teilen() {
		if (!link) return;
		try {
			await navigator.share({ title: 'Einladung zum Inventar', text: `Einladung für ${link.name}`, url: link.url });
		} catch {
			// Abbrechen im Teilen-Menü ist kein Fehler.
		}
	}

	async function zurueckziehen(e: Einladung) {
		if (!confirm(`Einladung für ${e.name} zurückziehen? Der Link funktioniert dann nicht mehr.`)) return;
		fehler = '';
		laeuft = true;
		try {
			await api('DELETE', `/api/einladungen/${e.id}`);
			await neuLaden();
		} catch (err) {
			fehler = fehlerText(err);
		} finally {
			laeuft = false;
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/">← Liste</a>
		<h1>Nutzer</h1>
	</header>

	{#if fehler}
		<p class="fehler" role="alert">{fehler}</p>
	{/if}

	<ul class="zeilen">
		{#each nutzer as n (n.personId)}
			<li>
				<span>{n.name}</span>
				<span class="leise">{n.email}</span>
			</li>
		{/each}
	</ul>

	<section>
		<h2>Einladen</h2>
		<p class="leise">
			Du bekommst einen Link, den du selbst weiterschickst. Er funktioniert einmal und läuft nach 7 Tagen ab.
		</p>
		<form onsubmit={einladen}>
			<label>
				Person
				<select bind:value={personWahl}>
					<option value="">Neue Person</option>
					{#each ohneZugang as p (p.id)}
						<option value={String(p.id)}>{p.name}</option>
					{/each}
				</select>
			</label>
			{#if !personWahl}
				<label>
					Name
					<input placeholder="z. B. Kim" bind:value={name} required />
				</label>
			{/if}
			<button type="submit" disabled={laeuft}>Einladung erstellen</button>
		</form>

		{#if link}
			<div class="hinweis">
				<p>Einladungslink für {link.name}. Er wird nur jetzt angezeigt.</p>
				<input readonly aria-label="Einladungslink" value={link.url} onfocus={(e) => e.currentTarget.select()} />
				<div class="knopfzeile">
					<button type="button" onclick={kopieren}>{kopiert ? 'Kopiert' : 'Kopieren'}</button>
					{#if kannTeilen}
						<button type="button" class="sekundaer" onclick={teilen}>Teilen …</button>
					{/if}
				</div>
			</div>
		{/if}
	</section>

	<section>
		<h2>Offene Einladungen</h2>
		{#if einladungen.length === 0}
			<p class="leise">Keine offenen Einladungen.</p>
		{:else}
			<ul class="zeilen">
				{#each einladungen as e (e.id)}
					<li>
						<span>
							{e.name}
							<span class="leise">· von {e.ersteller} · läuft ab {datumText(e.laeuftAbAm)}</span>
						</span>
						<button class="sekundaer klein" disabled={laeuft} onclick={() => zurueckziehen(e)}>Zurückziehen</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</main>
