import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// Statischer Build als Single-Page-App; der Go-Server liefert ihn eingebettet aus
			// und beantwortet unbekannte Pfade mit index.html.
			adapter: adapter({ fallback: 'index.html' })
		})
	],
	server: {
		// In der Entwicklung läuft der Go-Server auf localhost:8080.
		proxy: { '/api': 'http://localhost:8080' }
	}
});
