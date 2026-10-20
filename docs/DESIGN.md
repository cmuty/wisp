# Design

wisp keeps a persistent input below a quiet listening space. The W and the
current playback progress carry the strongest color; supporting text stays
readable. Navigation shares the identity header rather than becoming another
toolbar. English and Russian use the same layout and command syntax.

| Theme | Character | Accent range |
| --- | --- | --- |
| Mist | Glacial blue fading into lilac | `#87DCEF` → `#B5A3E8` |
| Ember | Peach light through evening smoke | `#F3B68C` → `#D58EAF` |
| Haze | Silver fog with an ice-blue signal | `#BDE8F2` → `#B9C0CE` |
| Aurora | Turquoise, sky blue, iris | `#63F6DF` → `#78DEFF` → `#888CF4` → `#CA9BFA` |
| Afterglow | Peach, rose, lavender | `#FFCF97` → `#F7A9B7` → `#D594E7` → `#9394EF` |

Gradients are rendered per visible cell, preserving Unicode display width.
The compact Aurora and Afterglow W blend color along both axes, rather than
assigning one color to each horizontal row.
Progress color follows the entire bar's coordinate system, so a short fill does
not compress the whole palette. The W uses solid pixels without a shadow halo.
Neutral text and quiet borders are intentional; not every glyph needs a gradient.

The butterfly uses a compact braille sprite with changing wing poses and headings.
Its flight alternates with resting on the playback frame. `/settings` can hide
the butterfly entirely and saves that preference across restarts. Reduced motion
keeps decoration quiet; ASCII mode replaces terminal-dependent art.

Use a Unicode monospace font. Keep input behavior predictable, avoid command
palette popovers, and provide commands for terminal-dependent shortcuts.
