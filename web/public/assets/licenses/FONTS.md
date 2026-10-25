# Bundled font: JetBrainsLxgwNerdMono 1.3

The web UI self-hosts JetBrainsLxgwNerdMono version 1.3 (Regular 400, Bold 700) as
cn-font-split@7.6.6 WOFF2 slices with `unicode-range`, so browsers fetch only the
slices for characters on the page. The slices and generated `result.css` files are
byte-identical copies of the LitRadar project's `app/assets/JetBrainsLxgwNerdMono-*`
(LitRadar commits 70096be0 and 80075fab); no font data was modified.

- Font: JetBrainsLxgwNerdMono 1.3, Copyright (c) 2024 lvbibir,
  https://github.com/lvbibir/JetBrainsLxgwNerdMono (tag v1.3, a1d8b32ecf).
  Font metadata: "licensed under the SIL Open Font License, Version 1.1.
  JetBrains Mono: OFL-1.1, LXGW WenKai: OFL-1.1, Nerd Fonts: MIT."
- Latin glyphs: JetBrains Mono (Nerd Font patched), SIL OFL 1.1 — `JetBrainsMono-OFL.txt`
  from https://github.com/JetBrains/JetBrainsMono (OFL.txt).
- Regular CJK glyphs: LXGW WenKai Mono GB Screen v1.521, Copyright 2021–2024 LXGW
  and Copyright 2020 The Klee Project Authors. See `LXGW-WenKai-Screen-OFL.txt`.
- Bold CJK glyphs: LXGW ZhenKai GB v0.819, Copyright 2022, 2024 LXGW and
  Copyright 2020 The Klee Project Authors. See `LXGW-ZhenKai-OFL.txt`.
- Latin and patched glyphs: JetBrains Mono NL 2.304 from Nerd Fonts v3.4.0.
  See the matching patched-font `NerdFonts-JetBrainsMono-OFL.txt`.
  `NerdFonts-LICENSE.txt` retains the project-wide notice: font data is OFL,
  while MIT applies separately to tooling source, not the patched font data.

The two version-matched historical LXGW OFL texts do not declare Reserved Font
Names. Later licenses are not substituted for these versions. The source-font
names, glyphs and all 668 font payloads remain unchanged. Network-font tests disable
local fonts so a same-name system installation cannot hide a failed web-font load.
Version-specific URLs and SHA-256 evidence are in `docs/font-licensing.md` in the
QoAuth source repository. These are provenance notes, not a legal guarantee.

The slices are delivered only as web fonts inside this application, not as installable
desktop fonts. Upstream's build repository README describes that repository as for
personal use and defers to the source font licenses listed above.
