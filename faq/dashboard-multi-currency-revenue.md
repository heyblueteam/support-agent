# Dashboard charts "missing" foreign-currency revenue (multi-currency fields)

A customer reports their dashboard isn't counting invoices/deals paid in foreign currencies — only their main currency seems to move the charts. Often they've duplicated a chart per currency (e.g. "KRW Revenue", "USD Revenue", "EUR Revenue") and all copies show the same number.

This is usually **not** missing data. Two product semantics combine to look like it:

1. **Chart aggregation is currency-blind.** A SUM/AVG over a CURRENCY field adds the raw numbers of *every* value regardless of each value's currency (`generateChartSegments` in `api/src/lib/chart.ts` sums `TodoCustomField.number`; the segment-value worker does the same). A $290 value adds `290` next to millions of won — numerically present, visually invisible.
2. **The chart's "currency" setting is display formatting only** (`display.currency` on the Chart row). It picks the symbol; it does not filter or convert anything. That's why per-currency chart copies are identical.

There is also no currency-code filter in `TodoFilterInput`, so "one chart per currency" cannot be configured.

## The correct setup

A **Currency Conversion** field (type `CURRENCY_CONVERSION`) normalizing everything to one target currency ("Any → KRW" etc.), with charts pointed at *that* field:

- Bar/pie chart value pickers and stat cards all support Currency Conversion fields (bar/pie support shipped 2026-06-11).
- The conversion field can also be referenced from FORMULA fields (e.g. a per-record "total incl. domestic + converted foreign").
- Conversion uses the frankfurter.dev rate for the date configured on the field (`conversionDateType`: date of entry / specific date / from a date field).

## Gotcha: conversion fields created after the data

Conversion runs on source-value **edit**, plus an automatic **backfill on field creation** (shipped 2026-06-11, `backfill-currency-conversion-field` job on the `currency-conversion-v1` queue). Conversion fields **created before that ship date** may still have holes: values entered before the field existed and never re-saved have no converted value.

Diagnose on db1 — source values with no conversion:

```sql
SELECT t.title, fc.number, fc.currency, conv.number AS converted
FROM Todo t
JOIN TodoCustomField fc ON fc.todo = t.id AND fc.customField = '<source-currency-field-id>'
LEFT JOIN TodoCustomField conv ON conv.todo = t.id AND conv.customField = '<conversion-field-id>'
WHERE fc.number IS NOT NULL AND (conv.id IS NULL OR conv.number IS NULL);
```

Fix by enqueueing the backfill manually for the existing field (idempotent — only fills missing rows, rates resolved per the field's date config):

```bash
cobalt run "node -e 'import(\"./dist/queues/currency-conversion.js\").then(m=>m.currencyConversionQueue.add(\"backfill-currency-conversion-field\",{customFieldId:\"<conversion-field-id>\"})).then(j=>{console.log(\"enqueued\",j.id);process.exit(0)}).catch(e=>{console.error(e);process.exit(1)})'" --project api --no-tty
```

Then re-run the SQL above to confirm zero rows (mind db1 replica lag).

## What to tell the customer

- Their foreign values *are* recorded; charts just sum raw numbers, and the chart currency picker is formatting only.
- Point revenue charts at the Currency Conversion field so every record is counted in one currency (and/or fold it into their total formula).
- If they had the conversion field before mid-2026, run the manual backfill above for them rather than asking them to re-save records.

Real-world case: 10 Media (`10-media`), thread "Blue Dashboard Not Reflecting Foreign Currency Invoice Revenue", 2026-06-11 — 8 of 42 foreign invoices predated their conversion field; manual backfill filled them.
