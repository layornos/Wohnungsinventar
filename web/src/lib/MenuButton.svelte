<script lang="ts">
	// ☰ button at the start of a page header; opens the side menu on narrow screens (hidden from 1024px, desktop.css).
	import { openSideMenu, sideMenu } from './sideMenu.svelte.ts';
</script>

<button type="button" class="secondary menu-button" aria-label="Menü" aria-expanded={sideMenu.open} aria-controls="side-menu" onclick={openSideMenu}>☰</button>
