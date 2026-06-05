# How do I debug HTTP request automations?

When a customer reports HTTP request automations not working, follow these steps:

## 1. Find the customer's company

```sql
SELECT id, slug, name FROM Company WHERE slug = 'company-slug';
```

## 2. Find their HTTP request automations

```sql
SELECT
  aa.id as action_id,
  p.name as project_name,
  ho.url,
  ho.method,
  a.isActive
FROM AutomationAction aa
JOIN Automation a ON aa.automation = a.id
JOIN Project p ON a.project = p.id
JOIN AutomationActionHttpOption ho ON ho.automationActionId = aa.id
WHERE p.company = 'COMPANY_ID'
AND aa.type = 'MAKE_HTTP_REQUEST';
```

## 3. Check HTTP logs for those automations

```sql
SELECT
  l.url,
  l.statusCode,
  l.statusText,
  l.createdAt
FROM AutomationActionHttpLog l
JOIN AutomationActionHttpOption ho ON l.automationActionHttpOption = ho.id
WHERE ho.automationActionId IN ('ACTION_ID_1', 'ACTION_ID_2')
ORDER BY l.createdAt DESC
LIMIT 20;
```

## 4. Common findings

- **200 OK responses**: Blue is sending correctly. Issue is on the receiving end.
- **Google Apps Script returning HTML**: This is normal. Look for "Data logged successfully" in the response.
- **No recent logs**: Usually the automation just hasn't been triggered — but **first rule out a platform-wide stall (Section 5)** before telling the customer it's their config.
- **`timeout of 30000ms exceeded` / 500 with no response**: the endpoint took >30s or was unreachable; the request was bounded and aborted (this row type only exists since the timeout fix — previously such requests hung silently and wrote nothing). One-off → the customer's endpoint. Many, platform-wide → Section 5.
- **4xx/5xx errors**: Check the endpoint URL, authentication, or payload format.

## 5. Rule out a platform-wide HTTP-egress stall

`AutomationActionHttpLog` is written on **every** HTTP attempt that *settles* — success *and* failure (the handler logs in a `finally`). A hung request never settles, so historically it wrote **nothing**: **silent zero rows meant the request never completed**, not an endpoint problem. Since the timeout fix (below) a failed request is bounded and now **does** write a row (status 500, `statusText` = the transport error, e.g. `timeout of 30000ms exceeded`) — so a recurrence is *visible* instead of silent. When a customer reports HTTP automations "stopped working," check whether it's just them or the whole platform **before** drilling into their account.

**Root cause — resolved June 2026 (Bonfire Revenue).** The automation HTTP handler issued outbound requests with **no timeout**. When a destination was slow/unresponsive the request hung indefinitely; because the log row is written in a `finally` *after* the request settles, a hung request recorded nothing and silently never fired. Under sustained load these hangs accumulated and starved outbound HTTP for **all** automations platform-wide — while executions and email (a different transport) kept working, so nothing looked broken. A restart only *cleared* the backlog; load rebuilt it within the hour, which is why an immediate restart-plus-test looked fixed and then re-stalled (twice — the first "it's fixed" reply went out wrongly on the back of exactly this). The durable fix (PR #424) adds a **30s request timeout** so every request settles, releases its socket, and records a row. The deeper instability that makes outbound HTTP slow in the first place is the API OOM/crash (issue #388).

**Decisive query — is HTTP egress alive platform-wide right now?**

```sql
SELECT NOW() AS now_db,
  (SELECT MAX(createdAt) FROM AutomationActionHttpLog) AS http_max,
  (SELECT COUNT(*) FROM AutomationActionHttpLog WHERE createdAt >= NOW() - INTERVAL 1 HOUR) AS http_1h,
  (SELECT MAX(createdAt) FROM AutomationEmailLog)      AS email_max,
  (SELECT MAX(startedAt) FROM AutomationExecution)     AS exec_max;
```

- **`http_max` hours stale while `email_max` / `exec_max` are current** → platform-wide HTTP stall. It's an operational/process failure, **not** the customer.
- To gauge how anomalous a gap is, compare the dead window to the same UTC hours on the prior day (HTTP volume is steady hour-to-hour; peak ~50/hr). A multi-hour zero during daytime hours is a real outage, not quiet traffic.
- **Post-fix reading:** if HTTP is stalled but you now see `timeout`/500 rows, requests are at least firing and failing fast — look at the destinations and the #388 OOM. **Silent zero rows again** means a no-timeout path has regressed.

**Confirm the cause is operational, not code:**
- Check `cobalt deployments list --project api` — if **nothing deployed at the break time**, it's not a code regression. (The June 2026 stall began with *no* deploy — it was triggered by outbound destinations slowing, which the no-timeout handler then turned into a persistent hang.)
- From an api container, confirm outbound egress works (`getent hosts <host>`; a quick `axios.get` to a benign URL) — rules out DNS/network. Note the few requests that *do* get through will return 200, so "egress works" does **not** mean HTTP automations are healthy.

**Immediate mitigation (stop-gap, not a cure):** a rolling restart clears the accumulated hangs and HTTP resumes — but on a process *without* the timeout fix it rebuilds under load. Workers run **in-process** with the GraphQL server (no separate worker service):

```bash
docker service update --force --update-parallelism 1 --update-delay 10s api-<N>-web
```

(Zero-downtime: cycles the 4 replicas one at a time, same image — no redeploy.)

**Verifying recovery — a single test-fire is NOT proof.** A just-restarted or just-deployed process always passes one test-fire (the bug is load-accumulative) — that false signal is exactly how the first "it's fixed" reply went out wrongly. To confirm a real recovery, watch **log volume under daytime load** for ~an hour, not one request. (To smoke-test the *path* itself: `blue --company blue automations create … --trigger-type TODO_CREATED --action-type MAKE_HTTP_REQUEST --http-url https://example.com/…`, fire with `blue records create`, confirm a fresh `AutomationActionHttpLog` row, then delete the test workspace. Use a benign URL like `example.com`, not a callback catcher.)

**Note:** Never share specific code lines or database details with customers. Summarize findings in plain language.
