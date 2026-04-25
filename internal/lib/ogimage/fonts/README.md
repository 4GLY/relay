# Fonts shipped with the OG image generator

The OG image renderer embeds two open fonts. Their canonical SIL OFL 1.1
license texts ship alongside the `.ttf` binaries in this directory so that
the redistribution requirement of OFL §2 stays satisfied for any consumer
who pulls relay-api as a compiled binary. Both files are embedded into the
binary via `//go:embed` in `generator.go`, so any redistribution carries
the license texts automatically.

## Fraunces (project title)

- Source: <https://github.com/undercasetype/Fraunces>
- Copyright 2018 The Fraunces Project Authors
- License: SIL OFL 1.1 — see `Fraunces-OFL.txt`

## Nunito (subtitle)

- Source: <https://github.com/googlefonts/nunito>
- Copyright 2014 The Nunito Project Authors
- License: SIL OFL 1.1 — see `Nunito-OFL.txt`
