import { api } from './api.ts';

/** Name and description proposed by Claude from a photo (ADR-0006). */
export type Suggestion = { name: string; description: string };

let enabled: Promise<boolean> | null = null;

/** Whether the server has suggestions configured; asked once per page load. */
export function suggestionsEnabled(): Promise<boolean> {
	enabled ??= api<{ suggestions: boolean }>('GET', '/api/features')
		.then((f) => f.suggestions)
		.catch(() => {
			// Ask again next time, e.g. after being offline.
			enabled = null;
			return false;
		});
	return enabled;
}

/** Sends the photo to the server, which asks Claude; the photo is not stored. */
export function suggest(file: File): Promise<Suggestion> {
	const form = new FormData();
	form.set('photo', file);
	return api<Suggestion>('POST', '/api/suggestions', form);
}
