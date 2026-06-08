# How do I debug automation emails not sending?

When a customer reports an automation email isn't arriving (e.g. "my form sends no
notification / no thank-you email", "automations aren't working"), work it in this
order. The three numbered traps below are where this investigation goes wrong — read
them before trusting any single signal.

All queries are read-only → **db1 read replica**. Get the password the usual way
(see the `investigate-customer` skill) and run over `ssh root@db1.blue.cc`.

## 1. Rule out the plan limit / lock first (it's usually NOT this)

Automations are gated per company per **UTC calendar month**. The check lives in
`api/src/services/LimitService.ts` (`assertCanRunAutomation`) and `api/src/lib/automation-limit-check.ts`;
the numbers are in `api/src/plans/registry.ts`.

- **Locked (tier = null):** a company with **no license and no active sub** runs
  **zero** automations — `getOrgLimits` returns null → `PlanLimitError(locked)`. A free
  trial does **not** grant a tier. Confirm with the entitlement query below.
- **Over limit:** legacy tiers = 500–2 500/mo; current tiers (starter…scale) = 250–5 000/mo;
  Pro multiplies the tier cap ×10; Enterprise = unlimited. Most "not working" reports are
  nowhere near the cap.

```sql
-- Entitlement: license tier, sub, comp, trial. NOTE: the sub FK lives on Company
-- (Company.subscriptionPlan -> CompanySubscriptionPlan.id), NOT a `company` column.
SELECT c.id, c.compedPlan, c.freeTrialStartedAt, c.freeTrialExpiredAt, c.subscribedAt,
       (SELECT COUNT(*) FROM CompanyLicense cl WHERE cl.company = c.id) AS license_rows,
       (SELECT GROUP_CONCAT(cl.planId) FROM CompanyLicense cl WHERE cl.company = c.id) AS license_planIds,
       sp.status AS sub_status, sp.stripeProductId AS sub_product
FROM Company c
LEFT JOIN CompanySubscriptionPlan sp ON c.subscriptionPlan = sp.id
WHERE c.id = '<COMPANY_ID>';

-- Usage this month (compare to the tier cap above)
SELECT COUNT(*) AS execs_this_month
FROM AutomationExecution ae
JOIN Automation a ON ae.automationId = a.id
JOIN Project p ON a.project = p.id
WHERE p.company = '<COMPANY_ID>'
  AND ae.startedAt >= DATE_FORMAT(UTC_DATE(), '%Y-%m-01');
```

`bloo_tierN` maps to `legacy_tierN`. If they're locked or genuinely over the cap, that's
the answer — stop here. Otherwise the automations are firing and the problem is downstream.

## 2. Did the automation fire? (and the COMPLETED trap)

```sql
SELECT a.id AS automation_id, a.isActive, p.name AS project,
       t.type AS trigger_type, t.todoList AS trigger_list, a.createdAt
FROM Automation a
JOIN Project p ON a.project = p.id
LEFT JOIN AutomationTrigger t ON t.automation = a.id
WHERE p.company = '<COMPANY_ID>'
ORDER BY a.createdAt;

-- Executions (did it run, and how often)
SELECT ae.automationId, t.type AS trigger_type, ae.status, ae.todosAffected, ae.startedAt, ae.errorMessage
FROM AutomationExecution ae
JOIN Automation a ON ae.automationId = a.id
JOIN Project p ON a.project = p.id
LEFT JOIN AutomationTrigger t ON t.automation = a.id
WHERE p.company = '<COMPANY_ID>'
ORDER BY ae.startedAt DESC LIMIT 50;
```

> **TRAP 1 — `AutomationExecution.status = COMPLETED` does NOT mean the email sent.**
> For single-record event triggers the execution row is written `COMPLETED`
> **unconditionally, before any action runs** (`AutomationDataSource.handleAutomation`,
> the metering block). It only means "the trigger matched and fired for N records." A
> COMPLETED execution with no email is normal when the action failed.

Note the trigger's `todoList` (form submissions create a record via `TODO_CREATED`; if the
form drops records in a different list than the trigger's, the automation never matches).

## 3. Did the email actually send? (the email-log traps)

```sql
-- Join via todoId, NOT via the action (see TRAP 3). dedupeKey IS NOT NULL = claim path.
SELECT el.subject, el.emailTo, el.dedupeKey IS NOT NULL AS has_dedupe, el.sentAt,
       el.createdAt, el.todoId
FROM AutomationEmailLog el
JOIN Todo t ON el.todoId = t.id
JOIN TodoList tl ON t.todoList = tl.id
JOIN Project p ON tl.project = p.id
WHERE p.company = '<COMPANY_ID>'
ORDER BY el.createdAt DESC LIMIT 50;

-- Proof an email action ran to completion (written AFTER the send attempt)
SELECT ta.todo, ta.newValue AS sent_to, ta.createdAt
FROM TodoAction ta
JOIN Todo t ON ta.todo = t.id
JOIN TodoList tl ON t.todoList = tl.id
JOIN Project p ON tl.project = p.id
WHERE p.company = '<COMPANY_ID>' AND ta.type = 'SEND_EMAIL'
ORDER BY ta.createdAt DESC LIMIT 50;
```

> **TRAP 2 — `AutomationEmailLog.sentAt = NULL` is ambiguous.**
> There are two write paths in `applySendEmail` (`communication-handlers.ts`):
> - **Legacy path** (no `bulkTriggerId`, `dedupeKey` NULL): the row is created **only after
>   `mailer.send` returns without throwing**, and it **never sets `sentAt`**. So `sentAt = NULL`
>   here means **the email was sent fine** (sentAt just isn't tracked on this path).
> - **Claim/dedupe path** (`dedupeKey` set): a pending row is written `sentAt = NULL`
>   **before** send, then updated to a timestamp on success. Here `sentAt = NULL` means
>   **the send threw / never completed** → not delivered.
>
> So: branch on `dedupeKey`. `emailTo` shows the resolved recipient ("" = empty recipient,
> e.g. a custom-field email the submitter left blank).

> **TRAP 3 — don't join `AutomationEmailLog` to `AutomationAction`.**
> If the customer edited or recreated the automation while testing, the action's `id`
> changes, so `el.automationActionId` points at a now-deleted action and the join returns
> **zero rows** — making it look like nothing sent. Always join via `todoId`.

## 4. Common root causes

- **Sent, but not received** → deliverability. The legacy-path log row exists and there's
  no "Email sending failed" in logs → it went out. Have them check spam and confirm the
  recipient address is one they actually monitor (watch for a different-domain or alias
  recipient configured in the action). See `email-deliverability.md`.
- **Empty recipient** → `emailTo = ""`. The action points at a custom-field email the form
  submitter didn't fill, or the field isn't on the record. Their config, not a bug.
- **Template-variable throw (real bug class)** → an email/HTTP template that references a
  custom-field variable (e.g. `{Naam}`) can throw during template rendering in `loadHTML`
  (`html-template-processor.ts`). One historical instance: an orphaned `labeledEntries`
  include after the LABELED_LIST→Table field swap caused a Prisma `Unknown field` throw for
  **every** custom-field-referencing automation, platform-wide (fixed, commit `0da6d8326`,
  Jun 2026). Because such a throw escaped the per-trigger action loop, it also **killed
  other healthy actions on the same trigger** (e.g. a valid owner email lost because a
  co-firing thank-you email threw) and left **no log row**. Per-action isolation was added
  (PR #448) so this class can't cascade again — but a new template throw can still drop the
  one affected email.

## 5. Confirm in logs (Loki)

The decisive evidence is usually the exception. Loki is on `server.blue.cc:3100`.

```bash
# Replace the time window with the execution's startedAt (UTC).
ssh root@server.blue.cc 'curl -sG "http://localhost:3100/loki/api/v1/query_range" \
  --data-urlencode "query={job=\"docker\"} |~ \"Unhandled rejection|Failed to send automation email|Email sending failed|<TODO_ID>\"" \
  --data-urlencode "start=2026-06-04T14:28:00Z" --data-urlencode "end=2026-06-04T14:31:00Z" \
  --data-urlencode "limit=80"'
```

- An `Unhandled rejection` at the exact execution time → a throw escaped the action (the
  detail/stack is usually on the **following** log lines — widen the window and read raw).
- `Failed to send automation email` (caught in `applySendEmail`) vs `Email sending failed`
  (caught inside `mailer.send`) tells you whether the throw was before or during the actual
  SMTP/SES send.
- Loki keeps ~30 days; for older incidents you're limited to the DB signals above.

**Never share code lines, table names, or internal addresses with the customer.** Summarize
in plain product language (what was wrong, that it's fixed/deployed, what to re-test).
