import type { Handle } from '@sveltejs/kit/hooks';
import { bootData } from '#lib/themes.ts';

// The app is a static single-page app, so this only runs while building index.html (and in the dev server):
// it hands the theme list to the inline script in app.html, which applies the stored theme before the first render.
const themes = JSON.stringify(bootData());

export const handle: Handle = ({ event, resolve }) =>
	resolve(event, { transformPageChunk: ({ html }) => html.replace('%inventar.themes%', themes) });
