<script lang="ts">
	import type { Person } from './api.ts';

	let { persons, selected = $bindable() }: { persons: Person[]; selected: number[] } = $props();

	function toggle(id: number) {
		selected = selected.includes(id) ? selected.filter((s) => s !== id) : [...selected, id];
	}
</script>

<fieldset class="chips">
	<legend>Besitzer</legend>
	{#each persons as p (p.id)}
		<label class="chip">
			<input type="checkbox" checked={selected.includes(p.id)} onchange={() => toggle(p.id)} />
			{p.name}
		</label>
	{/each}
</fieldset>
