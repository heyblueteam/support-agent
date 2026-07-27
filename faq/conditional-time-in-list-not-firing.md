# Why isn't my "time in list" automation firing?

When a customer reports that an automation meant to act on a record after it's been
sitting in a list for a while (e.g. "after 24 hours move it", "follow-up after 3 days",
"escalate stale cards") **isn't firing, or fires inconsistently**, the cause is almost
always the trigger *type* they chose — not their config. This is a real product trap, not
user error.

## The mechanism (the root cause)

The "Time in list" / "Time tracking" condition (`TIME_IN_LIST`) can be added to two
different trigger types, and they behave **completely differently** over time:

- **Conditional automation** ("When conditions met", `CONDITIONAL` trigger) — re-evaluated
  **only when the record (or related data) is mutated**: edited, moved, a field/tag change,
  a comment, etc. **Pure passage of time never re-checks it.** A `CHECK_CONDITIONAL` job is
  enqueued from todo mutations only (the `todo-change-observer` Prisma extension,
  `updateTodos`, automation-chaining) — **there is no cron / periodic tick** for conditional
  automations. So a record that just sits untouched in a list is *never* re-evaluated, and
  the automation won't notice the moment it crosses 24h / 48h / N days.

- **Scheduled automation** (`SCHEDULED` trigger) — registers a real BullMQ cron
  (`AutomationDataSource.ts`, `pattern: cronExpression`) and re-runs on a recurring clock,
  re-querying *all* matching records each tick. `TIME_IN_LIST` is supported there as a filter
  (`automation-filter-conversion.ts`). **This is the reliable path for anything time-based.**

### Why time *windows* make it worse

If they built a window — `TIME_IN_LIST GTE 24h AND LTE 48h` — on a **conditional**
automation, the upper bound (LTE) is a silent failure point. "Starts matching" only fires on
a not-matching → matching edge, computed against a Redis cache of prior state
(`conditional-automation-cache.ts`, keyed `cond-auto:state:{automationId}`). With no idle
re-check, a record can pass straight *through* the window (evaluated at 23h = no, next touched
at 49h = no) and **never be seen inside it** — that step never runs at all.

Common shape of this ticket: a chain of conditional automations with overlapping windows
(24–48h, 48–72h, 72–96h …) gated by a status custom field, intended as drip/staged
follow-ups. It *looks* correct in the UI and silently under-fires.

## Both trigger types need Pro

`conditional_automations` and `scheduled_automations` are both **PRO_FEATURES**
(`api/src/plans/registry.ts`). A base-tier org (tier set, no Pro add-on / no `compedPlan` /
no active Pro sub on `prod_UC8GcJittpyE93`) can't use either reliably. Confirm the org's
plan before assuming a bug — see `derive.ts` / the entitlement query in
`debugging-automation-emails.md`. (Legacy AppSumo `bloo_tierN` sets the *tier*, **not** the
plan — Pro is a separate add-on.)

## What to tell the customer (the workaround)

Steer them off conditional and onto **Scheduled** for anything keyed on elapsed time:

- Use a **Scheduled automation** that runs on a recurring clock (hourly or daily) with the
  "Time in list" condition as a filter.
- Use only a **lower bound** ("in list for at least 24 hours"), never a window.
- Have each step's action **move the record to the next list** (or set a status field/tag) so
  it leaves the match set and the step fires **once** — the next stage keys off that new
  list/status. (A scheduled automation re-fires on every matching record each tick, so without
  an action that removes the record from the set it will repeat every run.)

## DB signal (read-only → db1)

Inspect what they actually built — trigger type + filterGroups tell the whole story:

```sql
SELECT t.automation, t.type, t.conditionMode, t.filterGroups
FROM AutomationTrigger t
JOIN Automation a ON t.automation = a.id
JOIN Project p ON a.project = p.id
WHERE p.company = '<COMPANY_ID>'
ORDER BY a.updatedAt DESC LIMIT 30;
```

`type = CONDITIONAL` with a `filterGroups` entry of `"type": "TIME_IN_LIST"` (especially a
GTE **and** an LTE on the same list) = this issue. `type = SCHEDULED` with the same filter =
the reliable setup.

**Never share trigger names, table names, or internal mechanism with the customer.** Explain
in plain product terms: conditional automations only re-check when the record changes, so use
a Scheduled automation for time-based steps.
