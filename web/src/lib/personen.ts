import { api, type Person } from './api.ts';

export async function ladePersonen(): Promise<Person[]> {
	return (await api<{ personen: Person[] }>('GET', '/api/personen')).personen;
}
