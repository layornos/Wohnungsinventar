// Drehen und Zuschneiden eines Fotos im Browser (Canvas). Der Zuschnitt ist frei, ohne festes
// Seitenverhältnis, und wird relativ zum gedrehten Bild angegeben: 0 ist links bzw. oben, 1 rechts bzw. unten.

/** Kanten des Zuschnitts: links, oben, rechts, unten. */
export type Kanten = { l: number; o: number; r: number; u: number };

/** Was gezogen wird: der ganze Zuschnitt oder eine Ecke. */
export type Griff = 'ganz' | { x: 'l' | 'r'; y: 'o' | 'u' };

/** Vierteldrehungen im Uhrzeigersinn, 0–3. */
export type Drehung = 0 | 1 | 2 | 3;

export const GANZES_BILD: Kanten = { l: 0, o: 0, r: 1, u: 1 };

/** Kleinster Zuschnitt, damit die Griffe nicht aufeinanderliegen. */
const MINDESTGROESSE = 0.05;

// iOS Safari kann Canvas nur bis etwa 16,7 Mio. Pixel; größere Ergebnisse werden verkleinert.
const MAX_PIXEL = 16_000_000;

const begrenzen = (wert: number, min: number, max: number) => Math.min(max, Math.max(min, wert));

/** Verschiebt den Zuschnitt oder eine Ecke um dx, dy (Anteile der Bildbreite und -höhe). */
export function verschieben(k: Kanten, griff: Griff, dx: number, dy: number): Kanten {
	if (griff === 'ganz') {
		dx = begrenzen(dx, -k.l, 1 - k.r);
		dy = begrenzen(dy, -k.o, 1 - k.u);
		return { l: k.l + dx, o: k.o + dy, r: k.r + dx, u: k.u + dy };
	}
	const n = { ...k };
	if (griff.x === 'l') n.l = begrenzen(k.l + dx, 0, k.r - MINDESTGROESSE);
	else n.r = begrenzen(k.r + dx, k.l + MINDESTGROESSE, 1);
	if (griff.y === 'o') n.o = begrenzen(k.o + dy, 0, k.u - MINDESTGROESSE);
	else n.u = begrenzen(k.u + dy, k.o + MINDESTGROESSE, 1);
	return n;
}

/** Dreht den Zuschnitt mit dem Bild um 90°, damit er denselben Bildausschnitt behält. */
export function kantenDrehen(k: Kanten, imUhrzeigersinn: boolean): Kanten {
	return imUhrzeigersinn ? { l: 1 - k.u, o: k.l, r: 1 - k.o, u: k.r } : { l: k.o, o: 1 - k.r, r: k.u, u: 1 - k.l };
}

export function istGanzesBild(k: Kanten): boolean {
	return k.l === 0 && k.o === 0 && k.r === 1 && k.u === 1;
}

/** Breite und Höhe des Bildes nach der Drehung. */
export function gedrehteGroesse(bild: HTMLImageElement, d: Drehung): [number, number] {
	const w = bild.naturalWidth;
	const h = bild.naturalHeight;
	return d % 2 === 0 ? [w, h] : [h, w];
}

/**
 * Zeichnet das gedrehte Bild in einen Kontext, im Maßstab des gedrehten Bildes. Moderne Browser
 * berücksichtigen dabei schon die EXIF-Ausrichtung von Handy-Fotos (image-orientation: from-image),
 * naturalWidth/naturalHeight beziehen sich ebenfalls auf das aufrecht stehende Bild.
 */
function gedrehtZeichnen(ctx: CanvasRenderingContext2D, bild: HTMLImageElement, d: Drehung) {
	const [w, h] = gedrehteGroesse(bild, d);
	const verschiebung: [number, number][] = [
		[0, 0],
		[w, 0],
		[w, h],
		[0, h]
	];
	ctx.translate(...verschiebung[d]);
	ctx.rotate((d * Math.PI) / 2);
	ctx.drawImage(bild, 0, 0);
}

/** Zeichnet das ganze gedrehte Bild verkleinert in die Leinwand der Vorschau. */
export function vorschauZeichnen(leinwand: HTMLCanvasElement, bild: HTMLImageElement, d: Drehung, kante: number) {
	const [w, h] = gedrehteGroesse(bild, d);
	const f = Math.min(1, kante / Math.max(w, h));
	leinwand.width = Math.max(1, Math.round(w * f));
	leinwand.height = Math.max(1, Math.round(h * f));
	const ctx = leinwand.getContext('2d')!;
	ctx.scale(f, f);
	gedrehtZeichnen(ctx, bild, d);
}

/** Erzeugt das gedrehte und zugeschnittene Foto als JPEG. */
export async function bearbeitetesFoto(bild: HTMLImageElement, d: Drehung, k: Kanten): Promise<File> {
	const [w, h] = gedrehteGroesse(bild, d);
	const breite = Math.max(1, Math.round((k.r - k.l) * w));
	const hoehe = Math.max(1, Math.round((k.u - k.o) * h));
	const f = Math.min(1, Math.sqrt(MAX_PIXEL / (breite * hoehe)));
	const leinwand = document.createElement('canvas');
	leinwand.width = Math.max(1, Math.round(breite * f));
	leinwand.height = Math.max(1, Math.round(hoehe * f));
	const ctx = leinwand.getContext('2d')!;
	// JPEG kennt keine Transparenz; wie in der Vorschau des Servers werden transparente Flächen weiß.
	ctx.fillStyle = '#ffffff';
	ctx.fillRect(0, 0, leinwand.width, leinwand.height);
	ctx.scale(f, f);
	ctx.translate(-k.l * w, -k.o * h);
	gedrehtZeichnen(ctx, bild, d);
	const blob = await new Promise<Blob | null>((fertig) => leinwand.toBlob(fertig, 'image/jpeg', 0.92));
	if (!blob) throw new Error('Foto konnte nicht erzeugt werden');
	return new File([blob], 'foto.jpg', { type: 'image/jpeg' });
}
