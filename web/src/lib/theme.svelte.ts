// The theme is set as data-theme on <html>, together with data-item-view for list or tiles. app.html already sets
// both from localStorage before the first render so nothing flashes; here they are only switched and the theme
// colour is updated.
// The choice is stored with the user on the server (session.svelte.ts); localStorage only mirrors it for the next load.
import { DEFAULT_THEME, isTheme, themeInfo, type Theme } from './themes.ts';

// Must match the key in the inline script in app.html.
const STORAGE_KEY = 'inventar.theme';

const atStart = document.documentElement.dataset.theme;

/** The active theme; shared so that every place shows the same choice. */
export const appearance = $state<{ theme: Theme }>({
	theme: isTheme(atStart) ? atStart : DEFAULT_THEME
});

/** Applies the theme and remembers it for this device; saveTheme stores it with the user. */
export function applyTheme(t: Theme) {
	appearance.theme = t;
	document.documentElement.dataset.theme = t;
	document.documentElement.dataset.itemView = themeInfo(t).tiles ? 'tiles' : 'list';
	try {
		localStorage.setItem(STORAGE_KEY, t);
	} catch {
		// Without storage the choice only lasts until the next reload.
	}
	updateThemeColor();
}

/** Back to the default theme and forget the choice on this device, e.g. after logging out. */
export function resetTheme() {
	applyTheme(DEFAULT_THEME);
	try {
		localStorage.removeItem(STORAGE_KEY);
	} catch {
		// Nothing stored, nothing to forget.
	}
}

/** Sets the browser bar colour of the active theme (Minimal light or dark like the system). */
export function updateThemeColor() {
	const { color } = themeInfo(appearance.theme);
	const dark = matchMedia('(prefers-color-scheme: dark)').matches;
	document.querySelector('meta[name="theme-color"]')?.setAttribute('content', dark ? color.dark : color.light);
}
