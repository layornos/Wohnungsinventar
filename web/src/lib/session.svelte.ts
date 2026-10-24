import { api, type User } from './api.ts';
import { appearance, applyTheme, isTheme, resetTheme, type Theme } from './theme.svelte.ts';

/** The logged-in user; null as long as nobody is logged in. */
export const session = $state<{ user: User | null; loaded: boolean }>({
	user: null,
	loaded: false
});

// If the user has chosen a theme, it applies. Otherwise the theme of this device is saved for the user once,
// so that all devices end up with the same one.
function adopt(user: User) {
	session.user = user;
	if (isTheme(user.theme)) applyTheme(user.theme);
	else saveTheme(appearance.theme).catch(() => {
		// Stays local; the next login tries again.
	});
}

export async function loadSession() {
	try {
		adopt(await api<User>('GET', '/api/me', undefined, { noRedirect: true }));
	} catch {
		session.user = null;
	} finally {
		session.loaded = true;
	}
}

export async function login(email: string, password: string) {
	adopt(await api<User>('POST', '/api/login', { email, password }, { noRedirect: true }));
}

/** Accepts an invitation; afterwards the new person is logged in. */
export async function acceptInvitation(token: string, email: string, password: string) {
	adopt(
		await api<User>(
			'POST',
			'/api/invitations/accept',
			{ token, email, password },
			{ noRedirect: true }
		)
	);
}

/** Goes back to the default theme so the next person on this device doesn't inherit the previous one. */
export async function logout() {
	await api('POST', '/api/logout', undefined, { noRedirect: true });
	session.user = null;
	resetTheme();
}

// Counts the choices so that with quick switching only the last one counts.
let lastChoice = 0;

/**
 * Applies the theme immediately and stores it with the user. If saving fails, the previous theme applies again and
 * the error goes to the caller – unless the theme has been switched again in the meantime.
 */
export async function saveTheme(t: Theme) {
	const before = appearance.theme;
	const nr = ++lastChoice;
	applyTheme(t);
	try {
		const user = await api<User>('PUT', '/api/me/theme', { theme: t });
		if (nr === lastChoice) session.user = user;
	} catch (e) {
		if (nr !== lastChoice) return;
		applyTheme(before);
		throw e;
	}
}
