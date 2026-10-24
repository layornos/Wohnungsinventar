<script lang="ts">
	// Floating ☰ button at the bottom left, within thumb reach; opens the side menu on narrow screens (hidden from
	// 1024px, desktop.css). Rendered once by the layout.
	import { openSideMenu, sideMenu } from './sideMenu.svelte.ts';
</script>

<button type="button" class="secondary menu-button" aria-label="Menü" aria-expanded={sideMenu.open} aria-controls="side-menu" onclick={openSideMenu}>☰</button>
