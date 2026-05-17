# Customer deleted their org and lost their AppSumo / LTD license — now everything shows the free trial

When a customer deletes their Blue organization (often to "start fresh"), their AppSumo/LTD license becomes **detached, not lost**. Any new organization they create then defaults to the free trial. The license is fully recoverable.

## What actually happened

A lifetime/AppSumo license is a `CompanyLicense` row attached to a *company*, not a user. When the org is deleted, the license's `company` foreign key is set to **NULL** — the row survives intact (`source`, `planId`, `licenseId`, `activationEmail` are all still there), it's just unattached. Entitlements (tier, user/org limits) are computed **on-read from the license**, so a license with `company = NULL` grants nothing → free trial everywhere.

A common second symptom: "logging in with `info@…` says no account exists." That address is usually the **AppSumo purchase/activation email**, which is frequently *not* a Blue login at all. Verify with `SELECT id FROM User WHERE email = '<that email>'` — if empty, the error is correct; the customer's real logins are their team accounts.

## Why the customer usually can't self-recover

The self-serve mutation `applyCompanyLicense` matches `WHERE activationEmail = <logged-in user's email> AND company IS NULL`. For AppSumo deals the `activationEmail` is often the billing mailbox (e.g. `info@…`) that has no Blue user. No logged-in user's email matches → it always throws. **Recovery is manual.**

## 1. Find the orphaned license and confirm the invoice matches

Read replica (db1):

```sql
SELECT id, company, source, planId, licenseId, activationEmail, invoiceId, createdAt
FROM CompanyLicense
WHERE licenseId = '<license key from invoice>'
   OR activationEmail = '<activation email from invoice>';
```

Confirm: `company IS NULL`, `planId` matches the invoiced tier (e.g. `bloo_tier2` = Tier 2, 75 users, 2 orgs), `licenseId` matches the AppSumo invoice key.

## 2. Pick the target company

Find the customer's current live orgs (the new free-trial ones they created):

```sql
SELECT u.email, cu.level, c.id AS companyId, c.name, c.slug, c.createdAt
FROM User u
JOIN CompanyUser cu ON cu.user = u.id
JOIN Company c ON c.id = cu.company
WHERE u.email IN ('<their team emails>')
ORDER BY u.email, cu.createdAt;
```

Attach to the org the customer asked for (default: the account of the person who emailed, if they're an OWNER). Confirm that target company has **no existing license** (the `company` column is `@unique` — one license per company).

## 3. Re-link the license (production write — db2 only)

This is exactly what the admin `linkCompanyLicense` mutation does (single column update, no other side effects — tier recomputes on read). Run on **db2** (write primary). The `AND company IS NULL` guard makes it idempotent and prevents clobbering an already-linked license:

```sql
UPDATE CompanyLicense
SET company = '<targetCompanyId>', updatedAt = NOW(3)
WHERE id = '<licenseRowId>' AND company IS NULL;
SELECT ROW_COUNT();   -- expect 1
```

## 4. Verify

```sql
SELECT id, company, planId, activationEmail, updatedAt
FROM CompanyLicense WHERE id = '<licenseRowId>';
```

`company` should now be the target company id. The tier is live immediately on the next read — no cache to clear, no `Company` columns to touch (tier/limits are derived, not stored).

## Important notes

- **The license is never actually lost** by an org deletion — it's detached. Reassure the customer of this.
- **Don't write to db1.** All recovery `UPDATE`s go to **db2** (write primary); reads go to db1.
- **One license per company** — `CompanyLicense.company` is unique. If the target company already has a license, pick a different org or ask the customer.
- **Don't infer plan/paid status from `Company` columns** (`subscribedAt`, `isEnterprise`, `subscriptionPlan`) — they don't map cleanly. The license row + `planId` is the source of truth.

## Customer reply template

```
Hi [Name],

Thanks for sending the invoice — that made this quick to sort out.

Here's what happened: your AppSumo Lifetime Deal ([Tier — limits]) is valid and was never lost. When you deleted the old organization, that detached the license from it, so every new organization defaulted to the free trial — which is what you were seeing.

[If applicable:] The [purchase email] address was the email used to buy the AppSumo deal — it was never set up as a Blue login, which is why logging in with it reports no account. Your actual Blue logins are unaffected and still work.

What I've done: I've reinstated your license onto your "[Workspace Name]" workspace under [their login email]. Log in there and you'll see [Tier] active, exactly as purchased. You can now re-invite your team and rebuild from a clean slate.

If you'd rather have the license on a different workspace or account, just say the word and I'll move it over.

Best regards,
Manny
Founder of Blue
```
