// Common methods and template for all Shift Reports.

// The scale-factors as defined in LaTeX. See:
// https://www.tug.org/texinfohtml/latex2e.html#Font-sizes
#let fs = (
  n4: 0.5em,
  n3: 0.7em,
  n2: 0.8em,
  n1: 0.9em,
  p1: 1.2em,
  p2: 1.44em,
  p3: 1.728em,
  p4: 2.074em,
  p5: 2.488em,
)

#let show_if(
  prefix: none,
  value,
) = [
  #if value == none or value == "" {
    return
  }
  #if prefix != none and prefix != "" [
    #prefix
  ]
  #value
]

#let parse_iso_datetime(value) = {
  let pattern = "(\d{4})-([01]\d)-([0-3]\d)T([0-2]\d):([0-5]\d):([0-5]\d)"
  let captures = value.match(regex(pattern)).captures
  datetime(
    year: int(captures.at(0)),
    month: int(captures.at(1)),
    day: int(captures.at(2)),
    hour: int(captures.at(3)),
    minute: int(captures.at(4)),
    second: int(captures.at(5)),
  )
}

#let pad_zeros(n, width) = {
  let s = str(n)
  if s.len() < width {
    return "0" * (width - s.len()) + s
  }
  return s
}

#let fmt_duration(_duration) = {
  pad_zeros(calc.floor(_duration.minutes() / 60), 2)
  text(":")
  pad_zeros(calc.rem(_duration.minutes(), 60), 2)
}

#let truncate(content, max_words: 6) = {
  let words = content.split()
  if words.len() <= max_words {
    return content
  }
  words.slice(0, max_words).join(" ") + sym.dots
}

/// Groups elements from a list by a specified key, returning a dictionary
/// where each key maps to all elements that share that key value.
///
/// The function extracts unique values from the specified key field, sorts them,
/// and creates a dictionary mapping each unique key to its corresponding elements.
///
/// ```example
/// #let data = (
///   (name: "Alice", department: "Engineering"),
///   (name: "Bob", department: "Marketing"),
///   (name: "Charlie", department: "Engineering")
/// )
/// #group_by(data, "department")
/// ```
///
/// -> array
#let group_by(
  /// The list of dictionaries to group. -> array
  list,
  /// The key field by which to group the elements. -> string
  group_key,
) = {
  let keys = list.map(it => it.at(group_key)).dedup().sorted()
  keys.map(key => ({ key }: list.filter(it => it.at(group_key) == key))).join()
}


#let logo_author(logo,author) = [
  #set text(hyphenate: true)
  #if logo != none {
    box(
      height: 0.9em,
      image(logo),
    )
  }
  #author
]

#let rnd_title_block(
  subject_line: none,
  title_line: none,
  subtitle_line: none,
  logo: none,
) = {
  par(
    leading: 0.5em,
    logo_author(logo, subject_line),
  )
  v(1em, weak: true)
  par(
    spacing: 0.1em,
    text(fs.p5, weight: "bold", hyphenate: true, title_line),
  )
  v(1.2em, weak: true)
  par(
    justify: false,
    text(fs.p2, hyphenate: true, subtitle_line),
  )
  // text(fs.p2, align(left, subtitle_line))
  v(2em, weak: true)
}

#let template(
  title: none,
  plan_name: none,
  author: "Shifts – The friendly shift planning tool",
  title_block: true,
  exported_at: none,
  logo: none,
  body,
) = {
  // DOCUMENT
  set document(
    title: [#title #plan_name],
    author: author,
    date: datetime.today(),
  )
  // TEXT
  set text(
    font: "Noto Sans",
  )
  // PAGE
  set page(
    header: context {
      set text(size: fs.n1)
      if counter(page).get().first() > 1 or not title_block {
        logo_author(logo, author)
        h(1fr)
        [#title -- #plan_name]
      }
    },
    footer: context {
      set text(fs.n1)
      [#parse_iso_datetime(exported_at).display("[day].[month].[year] [hour]:[minute]:[second]")]
      h(1fr)
      [#counter(page).display("1/1", both: true)]
    },
  )
  // TABLE
  show table.cell.where(y: 0): strong
  let header_top = stroke(1.0pt)
  let header_bottom = stroke(1.0pt)
  let content_top = stroke(0.5pt)
  set table(
    stroke: (_, y) => (
      left: { 0pt },
      right: { 0pt },
      // First row --- Second row -- third to last row.
      top: if y < 1 { header_top } else if y == 1 { none } else { content_top },
      // First row -- Last row
      bottom: if y < 1 { header_bottom } else { stroke(1.5pt) },
    ),
    align: auto,
    inset: 5.4pt,
  )
  if title_block {
    rnd_title_block(
      subject_line: author,
      title_line: title,
      subtitle_line: plan_name,
      logo: logo,
    )
  }
  {
    body
  }
}

