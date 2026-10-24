// Rotating and cropping a photo in the browser (canvas). The crop is free, without a fixed aspect ratio, and
// is given relative to the rotated image: 0 is left or top, 1 is right or bottom.

import { canvasToJpeg, MAX_EDGE } from './shrinkPhoto.ts';

/** Edges of the crop: left, top, right, bottom. */
export type Edges = { l: number; t: number; r: number; b: number };

/** What is being dragged: the whole crop or a corner. */
export type Handle = 'whole' | { x: 'l' | 'r'; y: 't' | 'b' };

/** Quarter turns clockwise, 0–3. */
export type Rotation = 0 | 1 | 2 | 3;

export const WHOLE_IMAGE: Edges = { l: 0, t: 0, r: 1, b: 1 };

/** Smallest crop, so the handles don't lie on top of each other. */
const MIN_SIZE = 0.05;

// iOS Safari can only handle canvases up to about 16.7 million pixels; larger results are scaled down.
const MAX_PIXELS = 16_000_000;

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

/** Moves the crop or a corner by dx, dy (fractions of the image width and height). */
export function move(e: Edges, handle: Handle, dx: number, dy: number): Edges {
	if (handle === 'whole') {
		dx = clamp(dx, -e.l, 1 - e.r);
		dy = clamp(dy, -e.t, 1 - e.b);
		return { l: e.l + dx, t: e.t + dy, r: e.r + dx, b: e.b + dy };
	}
	const n = { ...e };
	if (handle.x === 'l') n.l = clamp(e.l + dx, 0, e.r - MIN_SIZE);
	else n.r = clamp(e.r + dx, e.l + MIN_SIZE, 1);
	if (handle.y === 't') n.t = clamp(e.t + dy, 0, e.b - MIN_SIZE);
	else n.b = clamp(e.b + dy, e.t + MIN_SIZE, 1);
	return n;
}

/** Rotates the crop together with the image by 90° so it keeps the same part of the image. */
export function rotateEdges(e: Edges, clockwise: boolean): Edges {
	return clockwise ? { l: 1 - e.b, t: e.l, r: 1 - e.t, b: e.r } : { l: e.t, t: 1 - e.r, r: e.b, b: 1 - e.l };
}

export function isWholeImage(e: Edges): boolean {
	return e.l === 0 && e.t === 0 && e.r === 1 && e.b === 1;
}

/** Width and height of the image after rotation. */
export function rotatedSize(image: HTMLImageElement, r: Rotation): [number, number] {
	const w = image.naturalWidth;
	const h = image.naturalHeight;
	return r % 2 === 0 ? [w, h] : [h, w];
}

/**
 * Draws the rotated image into a context, at the scale of the rotated image. Modern browsers already take the
 * EXIF orientation of phone photos into account (image-orientation: from-image); naturalWidth/naturalHeight
 * also refer to the upright image.
 */
function drawRotated(ctx: CanvasRenderingContext2D, image: HTMLImageElement, r: Rotation) {
	const [w, h] = rotatedSize(image, r);
	const offset: [number, number][] = [
		[0, 0],
		[w, 0],
		[w, h],
		[0, h]
	];
	ctx.translate(...offset[r]);
	ctx.rotate((r * Math.PI) / 2);
	ctx.drawImage(image, 0, 0);
}

/** Draws the whole rotated image scaled down into the preview canvas. */
export function drawPreview(canvas: HTMLCanvasElement, image: HTMLImageElement, r: Rotation, edge: number) {
	const [w, h] = rotatedSize(image, r);
	const f = Math.min(1, edge / Math.max(w, h));
	canvas.width = Math.max(1, Math.round(w * f));
	canvas.height = Math.max(1, Math.round(h * f));
	const ctx = canvas.getContext('2d')!;
	ctx.scale(f, f);
	drawRotated(ctx, image, r);
}

/** Produces the rotated and cropped photo as JPEG, at most MAX_EDGE on the longer edge. */
export async function editedPhoto(image: HTMLImageElement, r: Rotation, e: Edges): Promise<File> {
	const [w, h] = rotatedSize(image, r);
	const width = Math.max(1, Math.round((e.r - e.l) * w));
	const height = Math.max(1, Math.round((e.b - e.t) * h));
	// The result has the same size limit as any other uploaded photo (shrinkPhoto.ts).
	const f = Math.min(1, MAX_EDGE / Math.max(width, height), Math.sqrt(MAX_PIXELS / (width * height)));
	const canvas = document.createElement('canvas');
	canvas.width = Math.max(1, Math.round(width * f));
	canvas.height = Math.max(1, Math.round(height * f));
	const ctx = canvas.getContext('2d')!;
	// JPEG has no transparency; as in the server's thumbnail, transparent areas become white.
	ctx.fillStyle = '#ffffff';
	ctx.fillRect(0, 0, canvas.width, canvas.height);
	ctx.scale(f, f);
	ctx.translate(-e.l * w, -e.t * h);
	ctx.imageSmoothingQuality = 'high';
	drawRotated(ctx, image, r);
	return canvasToJpeg(canvas);
}
