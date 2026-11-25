/**
 * Decode an iPhone HEIC/HEIF file to an ImageBitmap in the browser.
 *
 * The decoder is loaded only when needed. Normal JPEG/PNG uploads stay on the
 * native browser path and do not download the HEIC WebAssembly bundle.
 */
export async function decodeHeicBitmap(file: File): Promise<ImageBitmap | null> {
  const { heicTo, isHeic } = await import('heic-to/csp')
  if (!(await isHeic(file))) return null

  return heicTo({
    blob: file,
    type: 'bitmap',
    options: { imageOrientation: 'from-image' }
  })
}
