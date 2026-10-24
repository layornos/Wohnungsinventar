import type { Path, Room, StorageLocationNode } from './api.ts';

/** What a storage location label shows: its ID (for the QR code) and the path from the room down to it. */
export type StorageLocationLabelData = { id: number; path: string[] };

/** Label of a storage location from its path, e.g. ["Keller", "Regal 2", "Fach oben"]; the apartment is left out. */
export function labelOf(id: number, path: Path): StorageLocationLabelData {
	return { id, path: [path.room.name, ...path.storageLocations.map((s) => s.name)] };
}

/** Labels of all storage locations of a room, depth first in the order of the tree. */
export function labelsOfRoom(room: Room): StorageLocationLabelData[] {
	const out: StorageLocationLabelData[] = [];
	const walk = (nodes: StorageLocationNode[], parents: string[]) => {
		for (const n of nodes) {
			const path = [...parents, n.name];
			out.push({ id: n.id, path });
			walk(n.children, path);
		}
	};
	walk(room.storageLocations, [room.name]);
	return out;
}

/** Link the QR code of a storage location label leads to. */
export function storageLocationLink(id: number): string {
	return `${location.origin}/lagerorte/${id}`;
}
