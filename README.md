# mdtable-lint

Markdown tables are forgiving right up until they aren't. Miscount a column,
forget a leading pipe on one row out of ten, botch the delimiter row - most
renderers just do something plausible instead of failing, so the mistake
ships and someone notices weeks later when a cell is in the wrong column.

`mdtable-lint` parses the GFM-style tables in a markdown file and reports
problems with a file and line number, like a normal linter. It's strict by
default. If you've got a legacy file full of tables that are "wrong" only in
ways the spec actually allows, pass `--lenient` instead of fixing them all.

## Usage

```
go build -o mdtable-lint .
./mdtable-lint README-example.md
```

Given a file containing:

```markdown
| Name  | Age | City     |
|-------|-----|----------|
| Alice | 30  | New York |
| Bob   | 41
```

Strict mode (the default) reports:

```
README-example.md:4: error: [column-count] row has 2 column(s), header has 3
README-example.md:4: error: [pipe-style] row's leading/trailing pipe style does not match the header row
```

`Bob`'s row is missing a column and its trailing pipe, both of which strict
mode treats as errors worth fixing. With `--lenient`, the missing column is
allowed (GFM treats it as an implicit empty cell) but the exit code and
message set still reflect that dropping data silently is worse than being
short a cell.

## Rules

| Rule | What it catches | Disabled by `--lenient`? |
|------|------------------|---------------------------|
| `delimiter-row` | Delimiter row's column count doesn't match the header | No |
| `column-count` (too many) | A row has more cells than the header; GFM drops the extras | No |
| `column-count` (too few) | A row has fewer cells than the header | Yes - this is legal GFM |
| `empty-table` | Header and delimiter row with zero data rows | Yes - a header-only table is valid GFM |
| `pipe-style` | A row's leading/trailing pipe usage doesn't match the header | Yes - cosmetic only |

Every finding is one line: `file:line: severity: [rule] message`. The
process exits `1` if any error-level finding was reported, `2` on a usage or
file I/O problem, `0` otherwise.

## Known limitations

The table scanner doesn't understand fenced code blocks yet, so a
table-shaped block of text inside a ``` fence will be linted as if it were a
real table. See the roadmap below.

## License

MIT, see [LICENSE](LICENSE).
