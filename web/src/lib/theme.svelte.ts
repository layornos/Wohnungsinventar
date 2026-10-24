// The theme is set as data-theme on <html>. app.html already sets it from localStorage before the first render so
// nothing flashes; here it is only switched and the theme colour is updated.
// The choice is stored with the user on the server (session.svelte.ts); localStorage only mirrors it for the next load.

export type Theme = 'minimal' | 'neon' | 'soft';

export const DEFAULT_THEME: Theme = 'minimal';

export const THEMES: { value: Theme; name: string; description: string }[] = [
	{ value: 'minimal', name: 'Minimal', description: 'Schlicht, hell oder dunkel wie das System' },
	{ value: 'neon', name: 'Neon', description: 'Immer dunkel, mit leuchtendem Mint' },
	{ value: 'soft', name: 'Weich & farbig', description: 'Immer hell, rund und bunt' }
];

export function isTheme(value: unknown): value is Theme {
	return THEMES.some((t) => t.value === value);
}

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

/** Sets the browser bar colour to --theme-color of the active theme (Minimal light or dark). */
export function updateThemeColor() {
	const color = getComputedStyle(document.documentElement).getPropertyValue('--theme-color').trim();
	if (color) document.querySelector('meta[name="theme-color"]')?.setAttribute('content', color);
}
