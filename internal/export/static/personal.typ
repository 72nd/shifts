#import "lib.typ": template, show_if, fmt_duration, fs, truncate, parse_iso_datetime

#set page(
  "a5",
  margin: 10mm,
)

#set text(9pt)

#show heading.where(level: 1): set text(fs.n1)

#show: template.with(
  title: "Anna's shifts",
  plan_name: "BeSTival 2025",
  exported_at: "2025-05-01T00:00:00Z"
)

= Day 1 -- Freitag, 10.05.2025

#table(
  columns: (auto, auto, auto, 1fr),
  align: (center, auto, auto, auto),
  table.header(
    [Status],
    [Time],
    [Type],
    [Location],
  ),
  [#sym.ballot.cross],
  [12:00 -- 14:00],
  [Entrance],
  [Entrance],
  [#sym.ballot],
  [12:00 -- 14:00],
  [Entrance],
  [Entrance],
)

= Day 2 -- Samstag, 11.05.2025

#table(
  columns: (auto, auto, auto, 1fr),
  align: (center, auto, auto, auto),
  table.header(
    [Status],
    [Time],
    [Type],
    [Location],
  ),
  [#sym.ballot],
  [12:00 -- 14:00],
  [Entrance],
  [Entrance],
  [#sym.ballot],
  [12:00 -- 14:00],
  [Entrance],
  [Entrance],
)


#set text(size: 8pt)

See next page for more information on your shift types.

#pagebreak()

= Information on your shifts

== Entance

#lorem(60)

Thanks for your help! #emoji.heart