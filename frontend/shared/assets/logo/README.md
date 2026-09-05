# Logo

`prova_logo.png` is the source artwork: a line engraving of a bird on a branch,
grayscale on a transparent ground. Every other file here and in the two apps is
derived from it, so replacing the logo means replacing this file and
regenerating the rest.

| File | Where it is used |
| --- | --- |
| `prova_logo.png` | Source only, never imported by the interface. |
| `prova-mark.png` | `ProvaMark` / `ProvaLogo` in `shared/components/prova/logo.tsx`. Trimmed, 256px on the long edge. |
| `web/src/app/{favicon.ico,icon.png,apple-icon.png}` | Browser tab and home-screen icons of the admin panel. |
| `desktop/src/app/{favicon.ico,icon.png}` | Renderer icons inside the Electron window. |
| `desktop/build/icon.png` | electron-builder input; the `.icns` and `.ico` of the packaged app come from this 1024×1024 file. |

The icons sit on an opaque paper ground (`#eaefef`, the `--surface` token) so
the dark engraving stays visible against dark browser and OS chrome. Next's
icon pipeline decodes them as RGBA, so those files must not be palette PNGs —
the interface mark may be, and is, since it ships to the client.

Regenerating (run from `frontend/`, `sharp` comes from the workspace):

```js
import sharp from "sharp"
const trimmed = await sharp("shared/assets/logo/prova_logo.png")
  .trim({ threshold: 1 })
  .toBuffer()
// mark: resize to 256 inside, png({ palette: true, colors: 128 })
// icons: composite onto a square #eaefef canvas, ~10% inset, png({ palette: false })
```
