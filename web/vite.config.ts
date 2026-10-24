import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for our own code, libraries excluded (no longer needed with Svelte 6).
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// Static build as a single-page app; the Go server serves it embedded
			// and answers unknown paths with index.html.
			adapter: adapter({ fallback: 'index.html' })
		})
	],
	server: {
		// In development the Go server runs on localhost:8080.
		proxy: { '/api': 'http://localhost:8080' }
	}
});
