import { api, type Apartment, type Move, type Named } from './api.ts';

export async function loadMove(): Promise<Move | null> {
	return (await api<{ move: Move | null }>('GET', '/api/move')).move;
}

export async function loadApartments(): Promise<Apartment[]> {
	return (await api<{ apartments: Apartment[] }>('GET', '/api/apartments')).apartments;
}

/** The rooms of the old and the new apartment of a move. */
export async function loadMoveRooms(m: Move): Promise<{ old: Named[]; new: Named[] }> {
	const apartments = await loadApartments();
	const rooms = (id: number) => apartments.find((a) => a.id === id)?.rooms ?? [];
	return { old: rooms(m.oldApartment.id), new: rooms(m.newApartment.id) };
}

/** Body for move details: only the given fields change, null resets them. */
export type MoveDetailsChange = { originRoomId?: number | null; targetRoomId?: number | null; boxId?: number | null };
