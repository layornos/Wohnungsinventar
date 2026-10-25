// The selectable themes: the only list of them in the client. Each theme's look lives in src/themes/<value>.css,
// which +layout.svelte loads; app.html receives this list through hooks.server.ts; the server only knows the
// allowed values (store.Theme). A new theme needs an entry here, its CSS file and the value in the server.
// Plain data without runes, so the build can read it outside the browser (hooks.server.ts).

export const THEMES = [
	{
		value: 'minimal',
		name: 'Minimal',
		description: 'Schlicht, hell oder dunkel wie das System',
		// Browser bar colour: the theme's background, for light and dark system mode.
		color: { light: '#ffffff', dark: '#09090b' },
		// Item overview as tiles instead of a list.
		tiles: false
	},
	{
		value: 'neon',
		name: 'Neon',
		description: 'Immer dunkel, mit leuchtendem Mint',
		color: { light: '#09090f', dark: '#09090f' },
		tiles: true
	},
	{
		value: 'soft',
		name: 'Weich & farbig',
		description: 'Immer hell, rund und bunt',
		color: { light: '#f3f0fa', dark: '#f3f0fa' },
		tiles: true
	}
] as const satisfies readonly {
	value: string;
	name: string;
	description: string;
	color: { light: string; dark: string };
	tiles: boolean;
}[];

export type Theme = (typeof THEMES)[number]['value'];

export const DEFAULT_THEME: Theme = 'minimal';

export function isTheme(value: unknown): value is Theme {
	return THEMES.some((t) => t.value === value);
}

export function themeInfo(t: Theme) {
	return THEMES.find((e) => e.value === t)!;
}

/** What the inline script in app.html needs to apply the stored theme before the first render. */
export function bootData() {
	return {
		default: DEFAULT_THEME,
		themes: Object.fromEntries(THEMES.map((t) => [t.value, { color: t.color, tiles: t.tiles }]))
	};
}
