# Bundled fallback fonts

Downloaded 2026-09-21 from the following upstream repositories. The exact bundled
files are identified by SHA256SUMS. All fonts use the SIL Open Font License 1.1;
family-specific copyright notices and licenses are included beside the fonts.

- Noto Sans, Noto Sans Arabic, Noto Sans Hebrew, Noto Sans Devanagari, Noto Sans
  Thai, Noto Sans Symbols 2, and Noto Emoji: https://github.com/google/fonts/tree/main/ofl
  (respectively `notosans`, `notosansarabic`, `notosanshebrew`,
  `notosansdevanagari`, `notosansthai`, `notosanssymbols2`, and `notoemoji`).
  Variable fonts use weight 400 and default values for other axes.
- Noto Sans CJK JP Regular: https://github.com/notofonts/noto-cjk/blob/main/Sans/OTF/Japanese/NotoSansCJKjp-Regular.otf
  Covers Japanese, Chinese, and Korean; shared Han characters use Japanese forms.
- The existing `../IBMPlexMono-Text.ttf` remains the primary font. Its license is
  included as `ibmplexmono-OFL.txt`, from https://github.com/google/fonts/tree/main/ofl/ibmplexmono.

The fonts are embedded into the executable, require no runtime download, and add
about 23 MB. Emoji are monochrome to match the white labels and black shadows.
This set does not cover every Unicode script or every future emoji sequence.
To extend coverage, add a licensed font and register it in `pkg/workers/text.go`.
