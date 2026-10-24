<script lang="ts">
	import { aktuellesDesign, designWaehlen, DESIGNS } from './design.ts';

	let design = $state(aktuellesDesign());
</script>

<!-- Zwei echte Radio-Buttons als segmentierter Knopf; Pfeiltasten wechseln wie gewohnt. -->
<fieldset class="darstellung">
	<legend>Darstellung</legend>
	<div class="segmente">
		{#each DESIGNS as d (d.wert)}
			<label>
				<input type="radio" name="design" value={d.wert} bind:group={design} onchange={() => designWaehlen(d.wert)} />
				{d.name}
			</label>
		{/each}
	</div>
</fieldset>
