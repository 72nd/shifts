#import "lib.typ": template, show_if, fmt_duration, truncate, parse_iso_datetime, fs

#let data = if "data" in sys.inputs {
  json(sys.inputs.data)
} else {
  json("plan.json")
}

#let day_section(day) = [
  == Day #day.day_count -- #parse_iso_datetime(day.date).display("[weekday], [day].[month].[year]")
  #let participants_count = day.shifts.map(
    it => it.required
  )
  #let max_count = calc.max(..participants_count) 
  #if max_count < 1 {
    max_count = 1
  }

  #let has_comments = calc.max(
    ..day.shifts.map(it => it.comment_en)
  ) != ""

  #table(
    columns: (
      auto, if has_comments{50mm} else {0mm},
      ..array.range(max_count).map(_ => 1fr)
    ),
    rows: (auto, 20mm),
    align: horizon,
    table.header(
      [Time],
      if has_comments{[Comment]},
      ..array.range(max_count).map(
        it => text("Person " + str(it+1))
      )
    ),
    ..for shift in day.shifts {
      let participants = shift.at("participants", default: ())
      let start_time = parse_iso_datetime(shift.start_time)
      let duration = duration(minutes: shift.duration)
      let end_time = start_time + duration
      
      (
        [
          #start_time.display("[hour]:[minute]") -- #end_time.display("[hour]:[minute]")\ 
          _#text(fs.n1, fmt_duration(duration))_
        ],
        if has_comments {
          text(size: fs.n3, truncate(shift.comment_en))
        },
        ..if shift.required > 0 {
          array.range(max_count).map(it => {
            if it < participants.len() {
              participants.at(it).name
            }
            if shift.required != -1 and it + 1 > shift.required {
              table.cell(align: center, fill: gray, sym.crossmark.heavy)
            }
          })
        },
        ..if shift.required < 1 {
          (table.cell(
            colspan: max_count,
            align: bottom,
            sym.infinity
          ),) 
        }
        /*
        if shift.required < 1 {
          table.cell(colspan: , "")
        }
        */
      )
    }
  )
]

#let type_section(type) = [
  = #type.title_en
  
  #text(fs.n2, [
    #show_if(type.description_en)
    _#show_if(type.location_en, prefix: "@")_
  ])
  #for day in type.shift_days {
    day_section(day)
    pagebreak()
  }
]

#show: template.with(
  title: "Shiftplan Poster",
  plan_name: data.plan.title_en,
  exported_at: data.exported_at,
  title_block: false,
)

#set text(size: 16pt)

#set page(
  flipped: true,
  margin: 15mm,
  header: context{
    set text(size: fs.n3)
    h(1fr)
    [Shiftplan Poster -- #data.plan.title_en]
    h(1fr)
  },
  footer: context{
    set text(size: fs.n3)
    [
      #datetime.today().display("[day].[month].[year]") -- 
      p. #counter(page).display("1/1", both: true)
    ]
    []
    h(1fr)
    [
      #query(heading.where(level: 1)).at(0).body
      #sym.arrow.r
      #query(heading.where(level: 2)).at(0).body
    ]
  },
)

#let header_top = stroke(1.0pt)
#let header_bottom = stroke(1.0pt)
#let content_top = stroke(0.5pt)
#let content_vertical = stroke(thickness: 0.4pt, dash: "dashed")
#set table(
  stroke: (x, y) => (
    left: if x > 1 { content_vertical },
    right: { 0pt },
    // First row --- Second row -- third to last row.
    top: if y < 1 { header_top } else if y == 1 { none } else { content_top },
    // First row -- Last row
    bottom: if y < 1 { header_bottom } else { stroke(1.5pt) },
  )
)

// CONTENT

#for type in data.shift_types {
  type_section(type)
}