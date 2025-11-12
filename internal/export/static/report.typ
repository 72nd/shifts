#import "lib.typ": template, show_if, fmt_duration, fs, truncate, parse_iso_datetime

#let data = json("plan.json")

#let day_section(day) = [
  == Day #day.day_count -- #parse_iso_datetime(day.date).display("[weekday], [day].[month].[year]")
  #table(
    columns: 5,
    table.header(
      [Time],
      [Duration],
      [Status],
      [Participants],
      [Comment]
    ),
    ..for shift in day.shifts {
      let participants = shift.at("participants", default: ())
      let start_time = parse_iso_datetime(shift.start_time)
      let duration = duration(minutes: shift.duration)
      let end_time = start_time + duration
      
      (
        [
          #start_time.display("[hour]:[minute]") -- #end_time.display("[hour]:[minute]")
        ],
        [#fmt_duration(duration)],
        [
          #participants.len()
          \/ 
          #if shift.required == -1 {
            sym.infinity
          } else {
            shift.required
          }
        ],
        participants.map(it => it.name).join(", "),
        text(size: fs.n2, truncate(shift.comment_en)),
      )
    }
  )
]

#let type_section(type) = [
  = #type.title_en
  #show_if(type.description_en) _#show_if(type.location_en, prefix: "@")_
  #for day in type.shift_days {
    day_section(day)
  }
]

#show: template.with(
  title: "Report",
  plan_name: data.plan.title_en,
  exported_at: data.exported_at,
  logo: "logo.svg",
)

= Overview

#show_if(data.plan.description_en, prefix: [_Description:_])

#for type in data.shift_types {
  type_section(type)
}
