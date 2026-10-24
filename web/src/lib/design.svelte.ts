// Das Design steht als data-design am <html>. app.html setzt es schon vor dem ersten Rendern aus
// localStorage, damit nichts aufblitzt; hier wird nur noch gewechselt und die Themenfarbe nachgezogen.
// Gespeichert wird die Wahl am Konto (sitzung.svelte.ts); localStorage spiegelt sie nur fürs nächste Laden.

export type Design = 'minimal' | 'neon' | 'weich';

export const DESIGNS: { wert: Design; name: string; beschreibung: string }[] = [
	{ wert: 'minimal', name: 'Minimal', beschreibung: 'Schlicht, hell oder dunkel wie das System' },
	{ wert: 'neon', name: 'Neon', beschreibung: 'Immer dunkel, mit leuchtendem Mint' },
	{ wert: 'weich', name: 'Weich & farbig', beschreibung: 'Immer hell, rund und bunt' }
];

export function istDesign(wert: unknown): wert is Design {
	return DESIGNS.some((d) => d.wert === wert);
}

// Muss zum Schlüssel im Inline-Script in app.html passen.
const SCHLUESSEL = 'inventar.design';

const amStart = document.documentElement.dataset.design;

/** Das aktive Design; geteilt, damit alle Stellen dieselbe Wahl zeigen. */
export const darstellung = $state<{ design: Design }>({
	design: istDesign(amStart) ? amStart : 'minimal'
});

/** Wendet das Design an und merkt es sich für dieses Gerät; am Konto speichert es designSpeichern. */
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
