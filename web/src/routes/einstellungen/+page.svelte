<script lang="ts">
	import { errorText } from '#lib/api.ts';
	import { appearance, THEMES, type Theme } from '#lib/theme.svelte.ts';
	import { saveTheme } from '#lib/session.svelte.ts';

	let error = $state('');

	// The theme takes effect immediately; if saving it for the user fails, the previous one applies again.
	async function choose(t: Theme) {
		error = '';
		try {
			await saveTheme(t);
		} catch (e) {
			error = `Darstellung nicht gespeichert: ${errorText(e)}`;
		}
	}
</script>

<main class="page">
	<header class="header">
		<a class="button secondary" href="/">← Liste</a>
		<h1>Einstellungen</h1>
	</header>

	<section>
		<!-- Real radio buttons in cards; arrow keys switch as usual. -->
		<fieldset class="theme-picker" aria-describedby="theme-note">
			<legend><h2>Darstellung</h2></legend>
			<p id="theme-note" class="muted">Gilt für dich auf allen Geräten.</p>
			<div class="theme-cards">
				{#each THEMES as t (t.value)}
					<label class="theme-card">
						<span class="theme-preview" data-preview={t.value} aria-hidden="true">
							<span></span><span></span><span></span>
						</span>
						<span class="theme-card-row">
							<input
								type="radio"
								name="theme"
								value={t.value}
								checked={appearance.theme === t.value}
								onchange={() => choose(t.value)}
							/>
							<span class="theme-card-text">
								<span class="theme-card-name">{t.name}</span>
								<span class="muted small-text">{t.description}</span>
							</span>
						</span>
					</label>
				{/each}
			</div>
		</fieldset>

		{#if error}
			<p class="error" role="alert">{error}</p>
		{/if}
	</section>
</main>
