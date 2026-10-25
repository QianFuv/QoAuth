# Bundled font provenance

QoAuth bundles unchanged JetBrainsLxgwNerdMono v1.3 WOFF2 payloads: 344 Regular
and 324 Bold shards. The split CSS retains unicode ranges and `font-display: swap`.
The notices are available at `/assets/licenses/FONTS.md` and in binary archives
under `LICENSES/`. The font's name and glyphs have not been changed.

The official v1.3 split ZIP SHA-256 is
`535c724568f24cad90bde45765fcc98ab623246fdf4f39ba3e9d1391ed7d2f2a`.
All 668 payload hashes were compared against that archive, separately for each
weight. The fixed merge source is
[`a1d8b32ecf0c0fcffcf5df01ada5c2e135754799`](https://github.com/lvbibir/JetBrainsLxgwNerdMono/tree/a1d8b32ecf0c0fcffcf5df01ada5c2e135754799).

| Input | Version | Source TTF SHA-256 |
|---|---|---|
| Regular CJK | LXGW WenKai Mono GB Screen v1.521 | `61f581b3ed47f0d7537421635dc5bae645147ce0d752de7aed778d5418e502ca` |
| Bold CJK | LXGW ZhenKai GB v0.819 | `b413151b90f3c96c0d8905eb55c7078f0e3aca9cfb0d962f3826ab185f2e6439` |
| Regular Latin/Nerd | Nerd Fonts v3.4.0, JetBrains Mono NL 2.304 | `d83b3639f2a78e6d6da2cc2a7de4c2fc6817819a2199a6213790b3f7573710d1` |
| Bold Latin/Nerd | Same version | `100877cb3c17f18e45108b6e769abaf8b7129b3b678fd0ce0908d966d35d1846` |

The LXGW source hashes match the corresponding official release assets. The Nerd
inputs match fixed Nerd Fonts commit `fa7b859994228a9c8759f99c55a8d31ee92a1b5e`.

The following original license bytes are preserved:

| Bundled notice | Fixed official source | Original SHA-256 |
|---|---|---|
| `LXGW-WenKai-Screen-OFL.txt` | [WenKai Screen v1.521 OFL](https://raw.githubusercontent.com/lxgw/LxgwWenKai-Screen/b2e2193c58b67ef499f257ca584946aeca73e9d3/OFL.txt) | `4d72e3080cdb3be63638bb7197c7caae55b6e4cbe1bab1ded2830bdff0c9b438` |
| `LXGW-ZhenKai-OFL.txt` | [ZhenKai v0.819 OFL](https://raw.githubusercontent.com/lxgw/LxgwZhenKai/f98cd0b197e80211e7f77e9171d29d0659b430bf/OFL.txt) | `71651b1dc30e0c8ab1768640db78e8139c86ca7af096d9b1d8c46eda67eeb524` |
| `NerdFonts-JetBrainsMono-OFL.txt` | [Nerd Fonts patched-font OFL](https://raw.githubusercontent.com/ryanoasis/nerd-fonts/fa7b859994228a9c8759f99c55a8d31ee92a1b5e/patched-fonts/JetBrainsMono/NoLigatures/Regular/OFL.txt) | `30f0c136e3c88e422d0791acd97238870f9054a9729bc34cf2ff0d4ed8cac4ad` |

Neither historical LXGW license declares Reserved Font Names. The standard OFL
restriction clause is not itself a declaration of reserved names. Font data is
OFL-1.1; Nerd Fonts tooling source has separate MIT terms. This record establishes
version and distribution evidence and does not offer a legal guarantee.

QoAuth itself still has no selected source license. Its publication is authorized
by its maintainer; SPDX `NOASSERTION` does not grant an additional source license.
