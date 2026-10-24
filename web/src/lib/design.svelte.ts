// Das Design steht als data-design am <html>. app.html setzt es schon vor dem ersten Rendern aus
// localStorage, damit nichts aufblitzt; hier wird nur noch gewechselt und die Themenfarbe nachgezogen.

export type Design = 'minimal' | 'neon';

export const DESIGNS: { wert: Design; name: string }[] = [
	{ wert: 'minimal', name: 'Minimal' },
	{ wert: 'neon', name: 'Neon' }
];

// Muss zum Schlüssel im Inline-Script in app.html passen.
const SCHLUESSEL = 'inventar.design';

/** Das aktive Design; geteilt, damit Menü-Dialog und Seitenleiste dieselbe Wahl zeigen. */
export const darstellung = $state<{ design: Design }>({
	design: document.documentElement.dataset.design === 'neon' ? 'neon' : 'minimal'
});

/** Wechselt das Design und merkt es sich für dieses Gerät. */
export function designWaehlen(d: Design) {
	darstellung.design = d;
	document.documentElement.dataset.design = d;
	try {
		localStorage.setItem(SCHLUESSEL, d);
	} catch {
		// Ohne Speicher gilt die Wahl eben nur bis zum Neuladen.
	}
	themenfarbeAnpassen();
}

/** Setzt die Farbe der Browser-Leiste auf --themenfarbe des aktiven Designs (Minimal hell oder dunkel). */
export function themenfarbeAnpassen() {
	const farbe = getComputedStyle(document.documentElement).getPropertyValue('--themenfarbe').trim();
	if (farbe) document.querySelector('meta[name="theme-color"]')?.setAttribute('content', farbe);
}
