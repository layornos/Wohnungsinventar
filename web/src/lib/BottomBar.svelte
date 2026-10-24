<script lang="ts">
	// Bar at the bottom of every logged-in page on narrow screens, within thumb reach: the ☰ button plus whatever
	// the current page puts there (bottomBar.svelte.ts). Hidden from 1024px, where the sidebar takes over (desktop.css).
	import MenuButton from './MenuButton.svelte';
	import { bottomBar } from './bottomBar.svelte.ts';
</script>

<div class="bottom-bar">
	<MenuButton />
	{@render bottomBar.content?.()}
</div>
