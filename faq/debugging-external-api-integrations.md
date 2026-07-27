# An external integration (Pabbly / Zapier / Make) stopped creating records

When a customer reports that records/orders stopped being created automatically from an
external tool (Pabbly Connect, Zapier, Make, a custom script, etc.), the cause is almost
always a **stale API token on the integration side**, not a bug in Blue. These integrations
call the Blue API with a Personal Access Token (PAT: `X-Bloo-Token-ID` + `X-Bloo-Token-Secret`
headers) and create records **as the user who owns the token**. Diagnose it like this.

## 1. Find the company and the integration's identity

```sql
SELECT c.id, c.name, c.slug
FROM User u
JOIN CompanyUser cu ON cu.user = u.id
JOIN Company c ON c.id = cu.company
WHERE u.email = '<customer-email>';
```

The integration runs as *some* user — usually the OWNER or whoever set it up, not
necessarily the person who emailed. You identify that user in step 3 (it's the `createdBy`
on the auto-created records).

## 2. ⚠️ `PersonalAccessToken.lastUsedAt` is USELESS — the API never writes it

```sql
SELECT pat.name, pat.uid, pat.createdAt, pat.expiredAt, pat.lastUsedAt, u.email
FROM PersonalAccessToken pat
JOIN CompanyUser cu ON cu.user = pat.user
JOIN User u ON u.id = pat.user
WHERE cu.company = '<companyId>'
ORDER BY pat.createdAt DESC;
```

**`lastUsedAt` is always NULL.** The auth path (`api/src/lib/auth.ts`) validates the token
(lookup by `uid`, `bcrypt.compare` the secret, check `expiredAt`) but **never updates
`lastUsedAt`** — there is no `lastUsedAt =` write anywhere in the source. So a NULL value
does **not** mean the token is unused, and you cannot use this field to judge token activity.
(Note: the `investigate-customer` skill lists `lastUsedAt` in its API-keys query as if it
were meaningful — it isn't. Ignore it.)

What the token list **does** tell you: if the org's **only / newest** token was created
right around the time the integration broke, the working token was almost certainly
regenerated or deleted — which invalidates the credentials the integration still has stored.
That is the whole bug.

## 3. The decisive signal — record-creation timeline by `createdBy`

Since records are created *as the token owner*, filter creation by that user to find the
exact date it stopped. First find the affected workspace and who's been creating in it:

```sql
SELECT DATE(t.createdAt) as day, COUNT(*) as created, u.email
FROM Todo t
JOIN TodoList tl ON t.todoList = tl.id
JOIN Project p ON tl.project = p.id
LEFT JOIN User u ON u.id = t.createdBy
WHERE p.id = '<workspaceId>' AND t.createdAt >= NOW() - INTERVAL 90 DAY
GROUP BY day, t.createdBy ORDER BY day DESC;
```

Then confirm that the integration's identity user created **nothing anywhere in the org**
after the break date (a clean "the identity went dark", not "they just had no new orders"):

```sql
SELECT DATE(t.createdAt) as day, p.name as workspace, COUNT(*) as created, u.email
FROM Todo t
JOIN TodoList tl ON t.todoList = tl.id
JOIN Project p ON tl.project = p.id
LEFT JOIN User u ON u.id = t.createdBy
WHERE p.company = '<companyId>' AND t.createdAt >= NOW() - INTERVAL 14 DAY
GROUP BY day, p.name, u.email ORDER BY day DESC, created DESC;
```

If that user logs into the web app fine (`User.lastActiveAt` is recent) but creates **zero
records via the API** after the break date, the web session is healthy and only the **token**
is broken — a textbook stale-credential symptom.

## 4. Rule out a platform-wide regression BEFORE blaming the account

```sql
SELECT DATE(t.createdAt) as day, COUNT(*) as total_records,
       COUNT(DISTINCT p.company) as distinct_companies
FROM Todo t
JOIN TodoList tl ON t.todoList = tl.id
JOIN Project p ON tl.project = p.id
WHERE t.createdAt >= NOW() - INTERVAL 6 DAY
GROUP BY day ORDER BY day DESC;
```

Healthy = 100+ distinct companies creating records every day, including today. If that holds,
record creation works platform-wide and the problem is this account's token. (Don't bother
with Loki for token-auth failures: prod `LOG_LEVEL=error` plus `formatError` skipping
intentional GraphQL errors means a 401/FORBIDDEN/PROJECT_NOT_FOUND leaves no log line.)

## 5. Common findings

- **Org's newest/only token created around the break date + identity went dark after it** →
  token was regenerated/deleted; the integration still holds the old credentials. Fix is on
  the integration side. Customer reply: re-issue or confirm a token in Blue (Settings → API),
  copy **both** Token ID and Secret (secret shown once), paste into the integration's Blue
  connection, confirm the target workspace, send a test.
- **Token has a non-NULL `expiredAt` in the past** → expired token. Same fix.
- **Records still being created by that user after the "break" date** → not a token issue;
  the integration just hasn't fired (no new source events) or is targeting a different
  workspace/list. Ask what the source system shows.
- **`createdBy` is NULL on the records** → those aren't PAT-created. NULL `createdBy` comes
  from public **form submissions** or **automations** (recurring/scheduled record creation),
  which have no authenticated user — a different mechanism, diagnose accordingly.

## 6. Request-shape traps (token is fine, the call is wrong)

If the integration is **new** or was just reconfigured and **never** worked, it's usually a
request-shape problem rather than a stale token — see
[`pat-mutations-header-vs-input.md`](pat-mutations-header-vs-input.md): create-style mutations
(`createTodo`) take project/company context from the `X-Bloo-Project-ID` / `X-Bloo-Company-ID`
headers, not from the GraphQL input.

**Note:** Never share specific code lines, filenames, or database details with customers.
Summarize findings in plain language and give them the steps to fix it on their side.
