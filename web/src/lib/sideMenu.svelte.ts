// Connects the ☰ button in a page header with the side menu dialog, which the layout renders once for all pages.

/** Whether the side menu dialog is open; for aria-expanded on the button. */
export const sideMenu = $state<{ open: boolean }>({ open: false });

let opener: (() => void) | null = null;

/** Called by the side menu dialog; returns the function that unregisters it again. */
export function registerSideMenu(open: () => void) {
	opener = open;
	return () => {
		if (opener === open) opener = null;
	};
}

export function openSideMenu() {
	opener?.();
}
