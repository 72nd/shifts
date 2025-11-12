#import "lib.typ": group_by, parse_iso_datetime, template

/// Height of the timetable row.
#let ROW_HEIGHT = 1.0em

// How many cells per hour.
#let CELLS_PER_HOUR = 4

#let DEBUG = false

// ================================================
// DATA
// ================================================

#let data = if "data" in sys.inputs {
  json(sys.inputs.data)
} else {
  json("plan.json")
  // json("confidential.json")
}

// Count of shift types.
#let types_count = data.shift_types.len()
// Total number of cells per hour.
#let cells_per_hour = CELLS_PER_HOUR * data.shift_types.len()
// Duration of one cell.
#let cell_duration = duration(minutes: int(60 / CELLS_PER_HOUR))

/// Creates a sorted array of all days in the data.
///
/// -> array
#let days = (
  data
    .shift_types
    .map(it => it.shift_days.map(it => it.date))
    .flatten()
    .dedup()
    .sorted()
    .map(it => parse_iso_datetime(it))
)
#let by_day_and_type = {
  let flat = data
    .shift_types
    .map(
      tp => tp.shift_days.map(
        sd => sd.shifts.map(
          shift => (
            shift
              + (
                shift_type: tp.title_en,
                day: parse_iso_datetime(sd.date),
                // day_iso: sd.date,
              )
          ),
        ),
      ),
    )
    .flatten()
  let shift_types = data.shift_types.map(it => it.title_en)

  days.map(day => (
    day: day,
    by_type: shift_types.map(tp => (
      shift_type: tp,
      shifts: flat
        .filter(it => it.shift_type == tp and it.day == day)
        .map(shift => (
          shift
            + (
              starts: parse_iso_datetime(shift.start_time),
              ends: parse_iso_datetime(shift.start_time) + duration(minutes: shift.duration),
            )
        )),
    )),
  ))
}

// ================================================
// RENDERING: CELL
// ================================================

#let status_gradient = gradient.linear(
  color.hsl(0deg, 100%, 43%),
  color.hsl(42deg, 100%, 51%),
  color.hsl(115deg, 100%, 42%),
)

#let shift_count(shift) = {
  let participants = if "participants" in shift { shift.participants.len() } else { 0 }
  let percentage = if shift.required == -1 { 100% } else { participants / shift.required * 100% }
  let required = if shift.required == -1 { sym.infinity } else { shift.required }
  let participants_list = if "participants" in shift [ --- #shift.participants.map(it => it.name).join(", ") ] else { "" }
  [
    #participants/#required#participants_list
    #place(
      top + right,
      rect(width: 5mm, height: 5mm, fill: status_gradient.sample(percentage)),
    )
  ]
}

#let shift_cell(shift, cell_day) = {
  // Calculate effective event duration for display in the current day's table.
  // For single-day events, use full duration. For multi-day events, calculate
  // only the portion that belongs to the current day being displayed.
  let duration_in_minutes = 0
  let cell_label = ""

  if shift.starts.day() == shift.ends.day() {
    // Single-day event: use full duration
    duration_in_minutes = shift.duration
  } else {
    let duration_on_start_day = 24 * 60 - shift.starts.hour() * 60 - shift.starts.minute()
    if shift.starts.day() == cell_day {
      // Event starts on this day: show time until midnight
      duration_in_minutes = duration_on_start_day
    } else {
      // Event started on previous day: show remaining time
      duration_in_minutes = shift.duration - duration_on_start_day
    }
  }

  let span = calc.floor(duration_in_minutes / int(60 / CELLS_PER_HOUR))
  if DEBUG {
    return table.cell[
      *#shift.shift_type* --- #shift.starts.display("[hour]:[minute]") -- #shift.ends.display("[hour]:[minute]")

    ]
  }
  table.cell(
    rowspan: span,
    stroke: (
      top: (thickness: .9pt, dash: none),
      bottom: (thickness: .9pt, dash: none),
    ),
    if shift.starts.day() == cell_day.day() [
      *#shift.shift_type*\
      #shift.starts.display("[hour]:[minute]") -- #shift.ends.display("[hour]:[minute]") --- #shift_count(shift)
    ] else [
      Cond. #shift.shift_type
    ],
  )
}

// ================================================
// RENDERING:TIME ARRAY
// ================================================

#let properties_for_content_cell(
  index,
  types_for_day,
  hour,
) = {
  // Determine the shift type based on the index of the cell.
  let shift_type_i = calc.rem(index, types_count)
  let shift_type = data.shift_types.at(shift_type_i).title_en

  // Get all shifts for the current shift type and day.
  let shifts = types_for_day.by_type.filter(it => it.shift_type == shift_type)
  // Handle the case where there are no shifts for the current day.
  shifts = if shifts.len() > 0 {
    shifts.first().shifts
  } else {
    ()
  }

  // Determine the quarter based on the index of the cell.
  let quarter_i = int((index - shift_type_i) / types_count)

  // Calculate the start and end datetime of the cell.
  let cell_start = types_for_day.day + duration(hours: hour, minutes: quarter_i * 15)
  let cell_end = cell_start + cell_duration

  (
    // All shifts for the shift type associated with the current column. Thus
    // these are the potential candidates for this cell.
    shifts: shifts,
    // Start datetime of the cell.
    cell_start: cell_start,
    // End datetime of the cell.
    cell_end: cell_end,
  )
}

/// Find events that either start exactly at this cell's start time, or events
/// that span midnight (hour 0) and are still ongoing in this cell.
#let start_match(cell_properties) = {
  let match = cell_properties.shifts.filter(
    it => (
      it.starts == cell_properties.cell_start
        or (
          cell_properties.cell_start.hour() == 0
            and cell_properties.cell_start.minute() == 0
            and it.starts < cell_properties.cell_start
            and it.ends >= cell_properties.cell_end
        )
    ),
  )

  assert(
    match.len() >= 0 and match.len() <= 1,
    message: "only one event per cell allowed, got " + match.map(it => it.shift_type).join(", "),
  )

  return match
}

/// Check if any shift is "still running" in the cell.
#let in_match(cell_properties) = {
  let match = cell_properties.shifts.filter(
    it => it.starts < cell_properties.cell_start and it.ends >= cell_properties.cell_end,
  )
  return match
}

#let time_array(day_date) = {
  let types_for_day = by_day_and_type.filter(it => it.day == day_date).first()

  let rsl = ()
  for hour in range(24) {
    let time = datetime(hour: hour, minute: 0, second: 0)

    // Row header containing the hour.
    let time_row_header = table.cell(rowspan: CELLS_PER_HOUR, time.display("[hour]:[minute]"))

    // Cells containing shift information in 60 / CELLS_PER_HOUR min increments.
    let content_cells = range(cells_per_hour).map(i => {
      let cell_properties = properties_for_content_cell(i, types_for_day, hour)
      let cell_start_match = start_match(cell_properties)
      let cell_in_match = in_match(cell_properties)

      if cell_start_match.len() > 0 {
        shift_cell(cell_start_match.first(), types_for_day.day)
      } else if cell_in_match.len() == 0 {
        ([],)
      } else {
        // Cell is covered by rowspan - return empty tuple (vanishes on flatten)
        if DEBUG {
          ([Cond. »#cell_in_match.pop().shift_type«],)
        } else {
          ()
        }
      }
    })
    rsl.push(time_row_header)
    rsl.push(content_cells)
  }
  return rsl.flatten()
}

#let render_day(day) = [
  = #day.display("[weekday], [day].[month].[year]")

  #let header_footer = (
    [],
    ..data.shift_types.map(it => text(10.5pt, weight: "bold", style: "italic", it.title_en)),
  )
  #table(
    columns: (auto, ..range(data.shift_types.len()).map(it => 1fr)),
    rows: (auto, ..range(24 * CELLS_PER_HOUR).map(it => ROW_HEIGHT), auto),
    stroke: (x, y) => if x > 0 and y > 0 and calc.rem(y, 4) != 1 {
      (
        top: (thickness: .9pt, dash: "dotted"),
        right: .9pt,
        left: .9pt,
      )
    } else if y  == 24 * CELLS_PER_HOUR + 1 {
      (
        top: 1pt,
      )
    } else { .9pt },
    table.header(..header_footer),
    ..time_array(day),
  )
]

// ================================================
// CONTENT
// ================================================

#set page(
  "a3",
  flipped: true,
  margin: 10mm,
)

#show: template.with(
  title: "Timetable",
  logo: "logo.svg",
  plan_name: data.plan.title_en,
  exported_at: data.exported_at,
  title_block: false,
)

/*
#for day in by_day_and_type [
  -- *\[Day\]* #day.day.display("[weekday], [day].[month].[year]")\
  #for tp in day.by_type [
    -- -- *\[Type\]* #tp.shift_type\
    #for shift in tp.shifts [
      -- -- -- #shift.starts.display("[hour]:[minute]") -- #shift.ends.display("[hour]:[minute]")\
    ]
  ]
]
*/


#for (index, day) in days.enumerate() {
  render_day(day)
  if index < days.len() - 1 {
    pagebreak()
  }
}
