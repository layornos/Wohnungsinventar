<script lang="ts">
	import { darstellung, designWaehlen, DESIGNS } from './design.svelte.ts';

	// Eigener Gruppenname je Instanz: Dialog und Seitenleiste können gleichzeitig im DOM stehen.
	const gruppe = $props.id();
</script>

<!-- Zwei echte Radio-Buttons als segmentierter Knopf; Pfeiltasten wechseln wie gewohnt. -->
<fieldset class="darstellung">
	<legend>Darstellung</legend>
	<div class="segmente">
		{#each DESIGNS as d (d.wert)}
			<label>
				<input type="radio" name={gruppe} value={d.wert} checked={darstellung.design === d.wert} onchange={() => designWaehlen(d.wert)} />
				{d.name}
			</label>
		{/each}
	</div>
</fieldset>
