<script lang="ts">
	import { DISPOSITIONS, type Disposition } from './api.ts';

	let { value = $bindable() }: { value: Disposition | null } = $props();
	const group = $props.id();
</script>

<fieldset class="chips">
	<legend>Verbleib</legend>
	<label class="chip">
		<input type="radio" name={group} checked={value === null} onchange={() => (value = null)} />
		Offen
	</label>
	{#each DISPOSITIONS as d (d.value)}
		<label class="chip">
			<input type="radio" name={group} checked={value === d.value} onchange={() => (value = d.value)} />
			{d.name}
		</label>
	{/each}
</fieldset>
