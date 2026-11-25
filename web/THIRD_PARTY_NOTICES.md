# Third-party notices

## heic-to 1.6.7

Domus uses [heic-to](https://github.com/hoppergee/heic-to) to decode HEIC/HEIF photos
locally in the browser before resizing them for receipt OCR.

License: GNU Lesser General Public License v3.0 (LGPL-3.0). The license text is
included in `web/public/licenses/heic-to-LGPL-3.0.txt` and is served with the
web application at `/licenses/heic-to-LGPL-3.0.txt`.

The dependency is loaded only for HEIC/HEIF files; regular JPEG and PNG processing
continues to use the browser's native image decoder.
