# Why does the record field-search highlight only *some* field names?

Symptom: a customer searches fields inside a record and the yellow highlight lands
on some field titles (text, assignee) but not others (dropdown/select, lookup,
percentage, currency, …). Reported repeatedly by Dickey Boats; fixed in PR #528.

**Root cause:** highlighting was split across two mechanisms. Most renderers paint
the match structurally with `<HighlightedText>` (a real `<mark>`), but field
titles that didn't were meant to be caught by a global CSS-Custom-Highlight DOM
scanner (`useFieldSearchHighlight`). That scanner is racy — it walks the DOM on a
rAF + MutationObserver and silently misses fields whose row settles late (select
chip measurement, async lookup data, the percentage bar). So the gap moved around
field-type to field-type, which is why it took several "fix one more type" passes.

**The durable fix / how to diagnose the next gap:**

- Every field **title** now renders through one shared component,
  `app/src/components/record/layout/FieldLabel.vue` — it self-injects the search
  query and paints a real `<mark>`. If a new field type's name isn't highlighting,
  it's rendering the title as bare `{{ field.name }}` instead of
  `<FieldLabel :name="field.name" />`. That's the fix — swap it.
- The global scanner is kept **only** for the description editor (contenteditable,
  where DOM `<mark>`s can't be inserted). Don't lean on it for new titles/values —
  highlight structurally where the text is rendered.
- Field **values** highlight via `<HighlightedText :text="..." :query="searchQuery" />`
  (query from `useRecordFieldSearchQuery()`). Same rule: a value not highlighting
  means that renderer skipped `<HighlightedText>`.

Not a browser/CSS limitation — the CSS Custom Highlight API paints fine; the issue
was always the scanner's timing, which is why structural marks are the answer.
