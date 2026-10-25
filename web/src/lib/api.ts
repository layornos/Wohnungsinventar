import { goto } from '$app/navigation';
import type { Theme } from './themes.ts';

/** Domain error of the API, e.g. code "login_failed". */
export class ApiError extends Error {
	constructor(
		readonly status: number,
		readonly code: string,
		message: string,
		/** The full error response, e.g. with the list of items without a target room. */
		readonly data: Record<string, unknown> = {}
	) {
		super(message);
	}
}

export const LOGIN = '/anmeldung';

export function toLogin() {
	return goto(LOGIN, { replaceState: true });
}

/** theme is null as long as the user has never chosen one; reminderMail: gets the monthly reminder mail. */
export type User = { personId: number; name: string; email: string; theme: Theme | null; reminderMail: boolean };

export type Person = { id: number; name: string; isUser: boolean };

/** Open invitation; personId is null if the person is only created on acceptance. */
export type Invitation = {
	id: number;
	personId: number | null;
	name: string;
	createdBy: string;
	createdAt: string;
	expiresAt: string;
};

/** Public page for invitation links; the token is in the fragment. */
export const INVITATION = '/einladung';

export function formatDateTime(iso: string): string {
	return new Date(iso).toLocaleString('de-DE', { dateStyle: 'medium', timeStyle: 'short' });
}

export function formatDate(iso: string): string {
	return new Date(iso).toLocaleDateString('de-DE');
}

/** null means: disposition still undecided. */
export type Disposition = 'keep' | 'sell' | 'sell_later' | 'give_away';

export const DISPOSITIONS: { value: Disposition; name: string }[] = [
	{ value: 'keep', name: 'Behalten' },
	{ value: 'sell', name: 'Verkaufen' },
	{ value: 'sell_later', name: 'Später verkaufen' },
	{ value: 'give_away', name: 'Verschenken' }
];

export function dispositionName(d: Disposition | null): string {
	return DISPOSITIONS.find((e) => e.value === d)?.name ?? 'Offen';
}

export type Photo = { id: number; mainPhoto: boolean; url: string; thumbnailUrl: string };

export type Named = { id: number; name: string };

/** Where something is: apartment › room › storage locations from outside to inside. */
export type Path = { apartment: Named; room: Named; storageLocations: Named[] };

export function pathText(p: Path): string {
	return [p.apartment, p.room, ...p.storageLocations].map((o) => o.name).join(' › ');
}

/** A room or storage location as the place for an item or a storage location. */
export type Place = { kind: 'room' | 'storageLocation'; id: number };
/** apartmentId: the apartment the room or storage location is in. */
export type PlaceEntry = Place & { apartmentId: number; path: string };

export type StorageLocationNode = Named & { furniture: Named | null; children: StorageLocationNode[] };
/** subtree: IDs of the storage location and all of its nested storage locations. */
export type StorageLocation = Named & { path: Path; furniture: Named | null; children: Named[]; subtree: number[] };
export type Room = Named & { apartment: Named; storageLocations: StorageLocationNode[] };
export type Apartment = Named & { rooms: Named[] };

export type Item = {
	id: number;
	name: string;
	description: string;
	disposition: Disposition | null;
	createdAt: string;
	updatedAt: string;
	owners: Person[];
	photos: Photo[];
	placement: Path | null;
	furnitureStorageLocations: Named[];
	move: MoveDetails | null;
	/** Planned reminders and those that became due, the newest first (spec 07). */
	schedules: Schedule[];
	reminders: Reminder[];
} & Sale;

export type ReminderKind = 'keepReview' | 'sellLater' | 'priceReminder' | 'giveAwayReminder';
export type Reaction =
	| 'keep'
	| 'sell'
	| 'sellNow'
	| 'oneMoreYear'
	| 'keepAfterAll'
	| 'changePrice'
	| 'keepPrice'
	| 'keepOffering'
	| 'dispositionChanged'
	| 'handedOver'
	| 'priceChanged';

/** The next planned due date (YYYY-MM-DD) of a reminder for an item. */
export type Schedule = { kind: ReminderKind; dueOn: string };

/**
 * A reminder that became due; it applies to the whole household. doneAt, doneBy and reaction are null while open;
 * doneBy also when it was settled automatically, e.g. by a change of the disposition.
 */
export type Reminder = {
	id: number;
	item: { id: number; name: string; thumbnailUrl: string; owners: Person[]; askingPrice: number | null };
	kind: ReminderKind;
	dueOn: string;
	triggeredAt: string;
	doneAt: string | null;
	doneBy: Named | null;
	reaction: Reaction | null;
};

/**
 * What an item has from spec 06; amounts are cents. handedOverOn (YYYY-MM-DD) and saleProceeds are only set while
 * the item is in the archive. offeredSince is when it last got the disposition sell or give_away.
 */
export type Sale = {
	askingPrice: number | null;
	offeredSince: string | null;
	handedOverOn: string | null;
	saleProceeds: number | null;
};

/** Whether an item with this disposition can have an asking price. */
export function forSale(d: Disposition | null): boolean {
	return d === 'sell' || d === 'sell_later';
}

/** Whether an item with this disposition gets a listing text. */
export function hasListing(d: Disposition | null): boolean {
	return forSale(d) || d === 'give_away';
}

/** targetRoom: the room of the new apartment everything in the box goes to, unless an item has its own. */
export type BoxSummary = { id: number; number: number; label: string; targetRoom: Named | null };
export type Box = BoxSummary & { count: number };

/**
 * Origin room, target room and box of an item in the active move. targetRoom is the effective one: the item's
 * own target room, otherwise that of its box (then targetRoomFromBox is true).
 */
export type MoveDetails = {
	originRoom: Named | null;
	targetRoom: Named | null;
	targetRoomFromBox: boolean;
	box: BoxSummary | null;
};

export type Move = {
	id: number;
	oldApartment: Named;
	newApartment: Named;
	active: boolean;
	createdAt: string;
	completedAt: string | null;
	boxes: Box[];
};

export function boxName(b: BoxSummary): string {
	return b.label ? `Box ${b.number} · ${b.label}` : `Box ${b.number}`;
}

/** Name of the box with its target room, e.g. "Box 3 · Bücher → Küche". */
export function boxTitle(b: BoxSummary): string {
	return b.targetRoom ? `${boxName(b)} → ${b.targetRoom.name}` : boxName(b);
}

/** Path of the unpack mode of a box; QR code and NFC tag lead there. */
export function unpackPath(boxId: number): string {
	return `/boxen/${boxId}/auspacken`;
}

export type ItemListEntry = {
	id: number;
	name: string;
	disposition: Disposition | null;
	thumbnailUrl: string;
	/** Name of the storage location or, without one, of the room; empty without placement. */
	place: string;
	/** The storage location the item is placed in directly; null if it is directly in a room or unplaced. */
	storageLocationId: number | null;
	move: MoveDetails | null;
} & Sale;

/** Item in the packing list of a box; previousPlace is the path of its placement before unpacking. */
export type PackedItem = ItemListEntry & { previousPlace: string };

/** Answer of packing a storage location into a box or unpacking items from it. */
export type BoxChange = { count: number; box: Box };

/** One page of the item list; more says whether further pages follow. */
export type ItemPage = { items: ItemListEntry[]; more: boolean };

/**
 * Change for several items at once (POST /api/items/bulk); only the fields given are changed.
 * disposition null sets it back to undecided; storageLocationId or roomId sets the placement.
 */
export type ItemsChange = {
	disposition?: Disposition | null;
	addOwnerId?: number;
	storageLocationId?: number;
	roomId?: number;
};

/** Applies the change to all items, all or nothing, and returns how many were changed. */
export async function changeItems(itemIds: number[], change: ItemsChange): Promise<number> {
	return (await api<{ count: number }>('POST', '/api/items/bulk', { itemIds, ...change })).count;
}

/** Message for an error, whether from the API or because the server cannot be reached. */
export function errorText(e: unknown): string {
	return e instanceof ApiError ? e.message : 'Der Server ist nicht erreichbar.';
}

/**
 * Calls the API; FormData is sent as multipart, everything else as JSON.
 * On 401 it redirects to the login page unless `noRedirect` is set.
 */
export async function api<T>(
	method: string,
	path: string,
	body?: unknown,
	opts: { noRedirect?: boolean } = {}
): Promise<T> {
	const asJSON = body !== undefined && !(body instanceof FormData);
	const res = await fetch(path, {
		method,
		headers: asJSON ? { 'Content-Type': 'application/json' } : {},
		body: asJSON ? JSON.stringify(body) : (body as FormData | undefined)
	});
	if (res.status === 204) return undefined as T;
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		if (res.status === 401 && !opts.noRedirect) {
			await toLogin();
		}
		throw new ApiError(res.status, data.code ?? 'unknown', data.message ?? res.statusText, data);
	}
	return data as T;
}
