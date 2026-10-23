<script lang="ts">
	import { VERBLEIBE, type Verbleib } from './api.ts';

	let { wert = $bindable() }: { wert: Verbleib | null } = $props();
	const gruppe = $props.id();
</script>

<fieldset class="chips">
	<legend>Verbleib</legend>
	<label class="chip">
		<input type="radio" name={gruppe} checked={wert === null} onchange={() => (wert = null)} />
		Offen
	</label>
	{#each VERBLEIBE as v (v.wert)}
		<label class="chip">
			<input type="radio" name={gruppe} checked={wert === v.wert} onchange={() => (wert = v.wert)} />
			{v.name}
		</label>
	{/each}
</fieldset>
