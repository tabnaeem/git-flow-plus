# Git Flow Plus — Presentation Assets

Everything needed to run the team training session, generated from the
current codebase and from [`../Diagrams.md`](../Diagrams.md).

| Asset | Contents |
|---|---|
| `GitFlowPlus-Team-Training.pptx` | The 22-slide deck (title + 20 content slides + closing). Every slide has embedded speaker notes (View → Notes Page, or Presenter View). |
| `../PresentationScript.md` | The same narration as a standalone script, with per-slide timing guidance, transition cues, and an anticipated Q&A section. |
| `icons/` | 26 high-resolution (512×512) PNG icons used throughout the deck — colored circle + white Font Awesome 6 glyph, one per role/concept (developer, QA engineer, git-branch, git-tag, manifest, etc.). Reusable in other GitFlowPlus materials. |
| `diagrams/` | SVG + PNG (2× scale) exports of all 22 Mermaid diagram blocks (20 numbered diagrams, 2 of which — Branch Relationship and Branch Resolver — carry a second illustrative diagram) from [`../Diagrams.md`](../Diagrams.md). Filenames are `NN-slug.svg` / `.png`, numbered to match the section headings in that file. |

## Regenerating

The deck is built with `pptxgenjs`; the diagram exports with
`@mermaid-js/mermaid-cli`. Both were generated from a scratch Node.js
project (not checked into this repo — only the outputs are). To
regenerate after editing `Diagrams.md` or the deck content, recreate a
Node project with `pptxgenjs`, `react-icons`, `react`, `react-dom`,
`sharp`, and `@mermaid-js/mermaid-cli`, then re-run the icon generator,
`slides.js`, and a Mermaid-block extractor against `Diagrams.md`.

## Known gap: rendered slide screenshots

The deck was validated with `scripts/office/validate.py` (schema/
relationship/content-type checks — all passed) and a programmatic
geometry audit (every element's bounding box checked against the
13.33"×7.5" canvas, plus pairwise text-box overlap detection — all
flagged issues were fixed). It was **not** rendered to PDF/JPEG for a
frame-by-frame visual QA pass, because this machine has neither
LibreOffice nor Microsoft PowerPoint installed for headless conversion.
Before presenting, do a quick manual read-through in PowerPoint (or
LibreOffice Impress) to confirm rendering — the automated checks catch
off-canvas placement and overlapping text, but not font-substitution or
rendering-engine quirks.
