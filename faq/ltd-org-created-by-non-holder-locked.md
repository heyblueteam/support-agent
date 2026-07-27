# LTD customer: "my license includes N organizations, but my new org is locked"

A team member of a licensed (AppSumo/LTD) organization creates a second organization from **their own account** and expects the lifetime license to cover it. It doesn't — the new org starts a plain 7-day free trial, expires, and locks with the "Pick a plan" upgrade prompt. Meanwhile the billing page still shows e.g. "1 of 100 organizations used", which is what confuses them.

## How the license actually works

- A Tier N license pre-allocates **N `CompanyLicense` slot rows**, all carrying the same `licenseId` and the same `activationEmail` (the license holder's email).
- A slot auto-attaches **only at org creation**, and only when the **creator's email exactly matches `activationEmail`**. This is intentional: otherwise any invited member (or vendor/client) could create organizations under someone else's license.
- An org created by anyone else gets no slot → ordinary free trial → locks at trial end. Transferring ownership afterwards does **not** attach a license by itself (attach only happens at creation); a transfer must be paired with a manual license attach on our side.

## Diagnosis (db1, read-only)

```sql
-- Who created/owns what
SELECT c.id, c.name, c.slug, c.createdAt, c.freeTrialExpiredAt, cu.level, u.email
FROM User u JOIN CompanyUser cu ON cu.user = u.id JOIN Company c ON c.id = cu.company
WHERE u.email IN ('<requester>', '<suspected holder>');

-- The slots: all share activationEmail = the holder
SELECT id, company, planId, licenseId, activationEmail
FROM CompanyLicense WHERE activationEmail = '<holder email>';
```

Confirmed when: the locked org's OWNER email ≠ `activationEmail`, the locked org's `CompanyLicense` lookup is empty, and free slots (`company IS NULL`) exist.

## Resolution — two paths, offer both

1. **License holder creates the org** (self-serve, preferred when the locked org has no data): the holder's account creates the new organization, a slot auto-attaches, they invite the team. The locked org can be deleted from the lock prompt.
2. **Keep the existing locked org**: get **explicit confirmation from the license holder on the thread** (they're often already CC'd), then internally (a) transfer the org's ownership to the holder and (b) attach a free slot on **db2** (write primary):

```sql
UPDATE CompanyLicense
SET company = '<lockedCompanyId>', updatedAt = NOW(3)
WHERE id = '<a free slot id>' AND company IS NULL;
-- ROW_COUNT() should be 1; tier recomputes on read, org unlocks immediately
```

The customer **cannot** transfer ownership themselves while the org is locked — don't ask them to.

## Customer-reply notes

- Don't expose slot rows / activationEmail mechanics. Phrase it as: "the N organizations included apply to organizations created by the license holder."
- Explain the why in one line ("otherwise members who aren't the license holder could create organizations under that license").
- The "X of N used" counter is correct — the locked org was never on the license.

Example thread: "Question About Lifetime License and Organization Access" (sama.younis@rlglobal.com, 2026-06-10).
