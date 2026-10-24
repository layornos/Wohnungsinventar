// Shrinks photos in the browser before the upload: phone photos have 8–12 MB, the inventory needs far less.
// The result is an upright JPEG (the EXIF orientation is applied to the pixels), which the server accepts.

/** Longest edge of an uploaded photo in pixels. */
export const MAX_EDGE = 2048;

/** JPEG quality of shrunk and edited photos. */
export const JPEG_QUALITY = 0.85;

// Files up to this size are uploaded unchanged as long as their edges fit and the server accepts their type.
const SMALL_ENOUGH = 1_500_000;

// Types the server takes (internal/photo); anything else (e.g. HEIC) is always converted to JPEG.
const ACCEPTED = ['image/jpeg', 'image/png', 'image/webp'];

/** Scale factor so the longer edge is at most MAX_EDGE; never enlarges. */
export function shrinkFactor(width: number, height: number): number {
	return Math.min(1, MAX_EDGE / Math.max(width, height));
}

/** Encodes a canvas as JPEG file. */
export async function canvasToJpeg(canvas: HTMLCanvasElement, name = 'photo.jpg'): Promise<File> {
	const blob = await new Promise<Blob | null>((done) => canvas.toBlob(done, 'image/jpeg', JPEG_QUALITY));
	if (!blob) throw new Error('Photo could not be encoded');
	return new File([blob], name, { type: 'image/jpeg' });
}

type Decoded = { source: CanvasImageSource; width: number; height: number; close: () => void };

// Decodes the photo upright. createImageBitmap applies the EXIF orientation with imageOrientation 'from-image';
// where it fails, an <img> does the same (image-orientation: from-image is the default in modern browsers).
async function decode(file: Blob): Promise<Decoded> {
	if (typeof createImageBitmap === 'function') {
		try {
			const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
			return { source: bitmap, width: bitmap.width, height: bitmap.height, close: () => bitmap.close() };
		} catch {
			// Fall back to <img> below.
		}
	}
	const url = URL.createObjectURL(file);
	const img = new Image();
	img.src = url;
	try {
		await img.decode();
	} catch (e) {
		URL.revokeObjectURL(url);
		throw e;
	}
	return { source: img, width: img.naturalWidth, height: img.naturalHeight, close: () => URL.revokeObjectURL(url) };
}

function jpegName(name: string): string {
	return (name.replace(/\.[^.]*$/, '') || 'photo') + '.jpg';
}

/**
 * Shrinks a photo to at most MAX_EDGE on the longer edge as JPEG. Photos that are already small enough are returned
 * unchanged, and so is the original if anything goes wrong (the server then decides).
 */
export async function shrinkPhoto(file: File): Promise<File> {
	const accepted = ACCEPTED.includes(file.type);
	if (accepted && file.size <= SMALL_ENOUGH / 3) return file;
	let decoded: Decoded | null = null;
	try {
		decoded = await decode(file);
		const { width, height } = decoded;
		const f = shrinkFactor(width, height);
		if (accepted && f === 1 && file.size <= SMALL_ENOUGH) return file;
		const canvas = document.createElement('canvas');
		canvas.width = Math.max(1, Math.round(width * f));
		canvas.height = Math.max(1, Math.round(height * f));
		const ctx = canvas.getContext('2d')!;
		// JPEG has no transparency; as in the server's thumbnail, transparent areas become white.
		ctx.fillStyle = '#ffffff';
		ctx.fillRect(0, 0, canvas.width, canvas.height);
		ctx.imageSmoothingQuality = 'high';
		ctx.drawImage(decoded.source, 0, 0, canvas.width, canvas.height);
		const shrunk = await canvasToJpeg(canvas, jpegName(file.name));
		// A re-encoded photo without shrinking can turn out larger than the original; then the original stays.
		return accepted && f === 1 && shrunk.size >= file.size ? file : shrunk;
	} catch {
		return file;
	} finally {
		decoded?.close();
	}
}

/** Shrinks several photos one after another, so a phone does not hold several decoded photos at once. */
export async function shrinkPhotos(files: File[]): Promise<File[]> {
	const result: File[] = [];
	for (const file of files) result.push(await shrinkPhoto(file));
	return result;
}
