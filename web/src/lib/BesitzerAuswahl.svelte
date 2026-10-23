<script lang="ts">
	import type { Person } from './api.ts';

	let { personen, ausgewaehlt = $bindable() }: { personen: Person[]; ausgewaehlt: number[] } = $props();

	function umschalten(id: number) {
		ausgewaehlt = ausgewaehlt.includes(id) ? ausgewaehlt.filter((a) => a !== id) : [...ausgewaehlt, id];
	}
</script>

<fieldset class="chips">
	<legend>Besitzer</legend>
	{#each personen as p (p.id)}
		<label class="chip">
			<input type="checkbox" checked={ausgewaehlt.includes(p.id)} onchange={() => umschalten(p.id)} />
			{p.name}
		</label>
	{/each}
</fieldset>
