import { api, type Reaction, type Reminder, type ReminderKind } from './api.ts';

// Reminders (spec 07): names, the answers per kind and the number of open ones, shared by the side menu and the
// start page.

export const KIND_NAMES: Record<ReminderKind, string> = {
	keepReview: 'Behalte-Prüfung',
	sellLater: 'Später verkaufen',
	priceReminder: 'Preis prüfen',
	giveAwayReminder: 'Noch zu verschenken'
};

/** The answers a reminder of a kind offers, the first one is the main one. */
export const REACTIONS: Record<ReminderKind, { reaction: Reaction; name: string }[]> = {
	keepReview: [
		{ reaction: 'keep', name: 'Weiter behalten' },
		{ reaction: 'sell', name: 'Verkaufen' }
	],
	sellLater: [
		{ reaction: 'sellNow', name: 'Jetzt verkaufen' },
		{ reaction: 'oneMoreYear', name: 'Noch ein Jahr' },
		{ reaction: 'keepAfterAll', name: 'Doch behalten' }
	],
	// "Preis ändern" goes with the new price from its own field (erinnerungen page).
	priceReminder: [{ reaction: 'keepPrice', name: 'Preis lassen' }],
	// "Verschenkt" opens the handover dialog on the item page.
	giveAwayReminder: [{ reaction: 'keepOffering', name: 'Weiter anbieten' }]
};

const REACTION_NAMES: Record<Reaction, string> = {
	keep: 'Weiter behalten',
	sell: 'Verkaufen',
	sellNow: 'Jetzt verkaufen',
	oneMoreYear: 'Noch ein Jahr',
	keepAfterAll: 'Doch behalten',
	changePrice: 'Preis geändert',
	keepPrice: 'Preis gelassen',
	keepOffering: 'Weiter angeboten',
	dispositionChanged: 'Verbleib geändert',
	handedOver: 'Abgegeben',
	priceChanged: 'Preis geändert'
};

/** How a done reminder was settled, e.g. "Weiter behalten (Sandro)". */
export function outcome(r: Reminder): string {
	const name = r.reaction ? REACTION_NAMES[r.reaction] : 'Erledigt';
	return r.doneBy ? `${name} (${r.doneBy.name})` : name;
}

/** Whether the answer makes the item one to sell, so its asking price is next. */
export function leadsToSale(r: Reaction): boolean {
	return r === 'sell' || r === 'sellNow';
}

export const reminderCount = $state({ open: 0 });

/** Loads the number of open reminders of the household; errors only leave the old number. */
export async function refreshReminderCount() {
	try {
		reminderCount.open = (await api<{ open: number }>('GET', '/api/reminders/count')).open;
	} catch {
		// Shown again with the next refresh.
	}
}
