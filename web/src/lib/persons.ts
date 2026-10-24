import { api, type Person } from './api.ts';

export async function loadPersons(): Promise<Person[]> {
	return (await api<{ persons: Person[] }>('GET', '/api/persons')).persons;
}
