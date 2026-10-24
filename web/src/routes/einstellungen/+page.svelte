<script lang="ts">
	import { fehlerText } from '#lib/api.ts';
	import { darstellung, DESIGNS, type Design } from '#lib/design.svelte.ts';
	import { designSpeichern } from '#lib/sitzung.svelte.ts';

	let fehler = $state('');

	// Das Design wirkt sofort; misslingt das Speichern am Konto, gilt wieder das vorherige.
	async function waehlen(d: Design) {
		fehler = '';
		try {
			await designSpeichern(d);
		} catch (e) {
			fehler = `Darstellung nicht gespeichert: ${fehlerText(e)}`;
		}
	}
</script>

<main class="seite">
	<header class="kopf">
		<a class="knopf sekundaer" href="/">← Liste</a>
		<h1>Einstellungen</h1>
	</header>

	<section>
		<!-- Echte Radio-Buttons in Karten; Pfeiltasten wechseln wie gewohnt. -->
		<fieldset class="designwahl" aria-describedby="darstellung-hinweis">
			<legend><h2>Darstellung</h2></legend>
			<p id="darstellung-hinweis" class="leise">Gilt für dein Konto auf allen Geräten.</p>
			<div class="designkarten">
				{#each DESIGNS as d (d.wert)}
					<label class="designkarte">
						<span class="designvorschau" data-vorschau={d.wert} aria-hidden="true">
							<span></span><span></span><span></span>
						</span>
						<span class="designkarte-zeile">
							<input
								type="radio"
								name="design"
								value={d.wert}
								checked={darstellung.design === d.wert}
								onchange={() => waehlen(d.wert)}
							/>
							<span class="designkarte-text">
								<span class="designkarte-name">{d.name}</span>
								<span class="leise klein-text">{d.beschreibung}</span>
							</span>
						</span>
					</label>
				{/each}
			</div>
		</fieldset>

		{#if fehler}
			<p class="fehler" role="alert">{fehler}</p>
		{/if}
	</section>
</main>
