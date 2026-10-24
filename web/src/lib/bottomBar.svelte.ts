// Lets a page put its own controls into the bottom bar next to the ☰ button (e.g. the search on the start page).
import type { Snippet } from 'svelte';

export const bottomBar = $state<{ content: Snippet | null }>({ content: null });

/** Shows the snippet in the bottom bar while the calling component is mounted; call inside $effect. */
export function useBottomBar(content: Snippet) {
	bottomBar.content = content;
	return () => {
		if (bottomBar.content === content) bottomBar.content = null;
	};
}
