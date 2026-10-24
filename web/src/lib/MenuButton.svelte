<script lang="ts">
	// ☰ button at the left of the bottom bar (BottomBar.svelte); opens the side menu on narrow screens.
	import { openSideMenu, sideMenu } from './sideMenu.svelte.ts';
</script>

<button type="button" class="secondary menu-button" aria-label="Menü" aria-expanded={sideMenu.open} aria-controls="side-menu" onclick={openSideMenu}>☰</button>
