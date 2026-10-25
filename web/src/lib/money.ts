// Amounts are whole cents in the API (spec 06); the UI shows and takes euros the German way.

const euro = new Intl.NumberFormat('de-DE', { minimumFractionDigits: 0, maximumFractionDigits: 2 });

/** "15 €", "45,50 €", "1.200 €"; whole euros without decimals, like the listing text of the server. */
export function formatEuro(cents: number): string {
	const text = cents % 100 === 0 ? euro.format(cents / 100) : (cents / 100).toLocaleString('de-DE', { minimumFractionDigits: 2 });
	return `${text} €`;
}

/** Reads an amount like "45", "45,5", "45,50 €" or "1.200"; null for an empty field, undefined if unreadable. */
export function parseEuro(text: string): number | null | undefined {
	const t = text.replace(/€/g, '').replace(/\s/g, '');
	if (t === '') return null;
	const m = /^(\d{1,3}(?:\.\d{3})*|\d+)(?:,(\d{1,2}))?$/.exec(t);
	if (!m) return undefined;
	return Number(m[1].replace(/\./g, '')) * 100 + Number((m[2] ?? '').padEnd(2, '0'));
}

/** Cents as text for an input field, e.g. "45,50", empty for none. */
export function euroInput(cents: number | null): string {
	if (cents === null) return '';
	return formatEuro(cents).replace(' €', '');
}

/** Today in the household's time zone as YYYY-MM-DD, the format of handedOverOn. */
export function today(): string {
	return new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Berlin' }).format(new Date());
}

/** A date like "2026-03-31" as "31.03.2026". */
export function formatDay(day: string): string {
	const [y, m, d] = day.split('-');
	return `${d}.${m}.${y}`;
}
