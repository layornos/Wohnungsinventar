// The photo a page is currently rotating or cropping in PhotoEditor.svelte, for the capture page (photos not yet
// uploaded) as well as the item page (saved photos). The page shows the editor while current is set and passes done
// to it; apply only receives an actually edited photo, a cancel or no change just closes the editor.

export type PhotoEdit<K> = {
	readonly current: { key: K; source: Blob } | null;
	start(key: K, source: Blob): void;
	done(file: File | null): void;
};

/** key says which photo is being edited, e.g. its index or its ID. */
export function photoEdit<K>(apply: (key: K, file: File) => void): PhotoEdit<K> {
	let current = $state<{ key: K; source: Blob } | null>(null);
	return {
		get current() {
			return current;
		},
		start(key, source) {
			current = { key, source };
		},
		done(file) {
			const edited = current;
			current = null;
			if (edited && file) apply(edited.key, file);
		}
	};
}
