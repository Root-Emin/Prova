# Packaging assets

`icon.png` is the application icon electron-builder turns into the macOS
`.icns` and the Windows `.ico`. It is generated from
`shared/assets/logo/prova_logo.png`; see the README next to that file before
editing it by hand.

Platform-specific signing material is still missing on purpose. The builder
configuration intentionally does not contain signing identities, notarization
credentials, or placeholder secrets.
