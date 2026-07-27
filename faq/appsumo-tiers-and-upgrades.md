# AppSumo / legacy lifetime tiers — limits, pricing, and upgrades

Use this when a lifetime-deal (AppSumo or direct "Blue/bloo" LTD) customer asks
about their plan limits, reports **"automations stopped working"**, or wants to
upgrade their tier. All reads are read-only → **db1 read replica**.

The single most common "automations stopped" cause for a lifetime customer is
**hitting the monthly automation cap** (Section 3) — check that first.

## 1. Identify the license

```sql
SELECT cl.source, cl.planId, cl.licenseId, cl.activationEmail, cl.createdAt
FROM CompanyLicense cl
JOIN Company c ON cl.company = c.id
WHERE c.id = '<COMPANY_ID>';
```

- `source` = `appsumo` (AppSumo) or `blue` / `bloo` (direct lifetime).
- `planId` = `bloo_tierN` for **every** lifetime tier regardless of source —
  there are no `appsumo_tier*` rows. `bloo_tierN` maps 1:1 to `legacy_tierN`.
- The LTD **is** the tier. An expired free trial / `subscribedAt = NULL` is
  irrelevant — the license grants the tier on read (`deriveCompanyTierAndPlan`).
  Don't diagnose a lifetime customer as "locked" because the trial expired.

## 2. Tier catalog

Limits are in `api/src/plans/registry.ts` (`TIER_LIMITS`); the one-time prices are
in `app/src/composables/billing/useLicenseBilling.ts` (`LICENSE_TIERS`).

| Tier | One-time price | Orgs | Users | Automations/mo | Webhooks | Dashboards |
|------|---------------:|-----:|-------|---------------:|---------:|-----------:|
| 1 | $49  | 1   | 30   | 500   | 3  | 3  |
| 2 | $98  | 2   | 75   | 500   | 5  | 4  |
| 3 | $147 | 3   | 150  | 500   | 8  | 5  |
| 4 | $196 | 4   | 300  | 500   | 12 | 6  |
| 5 | $245 | 5   | ∞    | 500   | 18 | 8  |
| 6 | $399 | 10  | ∞    | 750   | 25 | 10 |
| 7 | $499 | 15  | ∞    | 1,200 | 35 | 12 |
| 8 | $599 | 25  | ∞    | 1,800 | 45 | 14 |
| 9 | $999 | 100 | ∞    | 2,500 | 50 | 15 |

**Identical across all legacy tiers** (so they never change on upgrade):
3,000 GB (3 TB) storage, 25,000 records/org, unlimited workspaces, 5,000 webhook
deliveries/mo, API rate limits 200/400/600 req/s (per key/user/org), 1 GB max
upload, 5 custom roles/workspace. Upgrading a tier only buys **more automation
runs + more organizations** (and a few more webhooks/dashboards).

## 3. Monthly automation cap — the #1 "automations stopped" cause

Automation runs are metered **per company per UTC calendar month**. The check is
`assertCanRunAutomation` (`api/src/services/LimitService.ts`): it counts
`AutomationExecution` rows since the 1st of the UTC month and **blocks once
`count >= automationsPerMonth`** (`PlanLimitError`, kind `over_limit`). The
counter resets to 0 at **00:00 UTC on the 1st** — automations resume on their own,
no action needed.

```sql
-- Runs used this UTC month vs the tier cap in the table above
SELECT COUNT(*) AS execs_this_month
FROM AutomationExecution ae
JOIN Automation a ON ae.automationId = a.id
JOIN Project p ON a.project = p.id
WHERE p.company = '<COMPANY_ID>'
  AND ae.startedAt >= DATE_FORMAT(UTC_DATE(), '%Y-%m-01');
```

If they're at/over the cap, that's the answer — automations are paused until the
1st, or they raise the cap (Section 4 or 5). If they're well under, it's not the
limit → work it as `debugging-automation-emails.md` / `debugging-http-automations.md`.

## 4. Upgrading a lifetime tier (one-time, permanent)

A lifetime upgrade is a **one-time** Stripe charge of the **price difference**
between tiers — not a new full purchase, not a subscription.
`upgradeCompanyLicense` charges `newTierPrice − currentTierPrice` (the listed
Stripe price per `planId`, so it's the same difference shown in the table even if
they bought on an AppSumo promo). You can only move **up** (`availableUpgrades` =
higher tiers only).

Upgrade cost from a given tier = (target price) − (current price). E.g. from
**Tier 5 ($245)**: → T6 **$154**, → T7 **$254**, → T8 **$354**, → T9 **$754**.

- **Who:** only the org **OWNER** can run the upgrade checkout.
- **Where:** in-app at **Settings → Billing → License**, *if* LTD upgrades are
  enabled. Visibility is gated by the admin setting `hideUpgradeLTD`
  (`Company.isUpgradableLTD`, read from redis `blue:admin-settings`). If it's
  hidden, the button won't show — fall back to generating a checkout link.

## 5. Pro add-on — the bigger automation lever ($500/yr)

Pro is a **separate $500/year subscription** that stacks on top of the lifetime
tier (it never touches/consumes the LTD). It multiplies the tier's monthly
automation cap **×10** (Tier 5 → 5,000/mo; Tier 9 → 25,000/mo), doubles API rate
limits, and unlocks the 18 Pro features (scheduled/conditional automations,
automation chaining, saved views, reports, white label, record-level access, …).
Full breakdown: `pro-addon-on-lifetime-appsumo-license.md`.

For a customer who only needs more automation runs, Pro (×10) is often a bigger
jump than a tier upgrade — e.g. Tier 5 + Pro = 5,000/mo vs Tier 9 = 2,500/mo —
but it's recurring, whereas a tier upgrade is one-time.

## 6. Customer-facing answer (adapt)

Never mention code, tables, or `planId`s. Plain product language:

> You're on the Tier N lifetime plan, which includes **[cap] automation runs per
> month**. You've reached that for [month], so automations are paused until they
> reset on the 1st (UTC) — they'll resume automatically. To raise the cap you can
> either upgrade your lifetime tier (one-time) or add Pro ($500/year, which takes
> you to [cap×10]/month plus advanced features). [tier table if they asked].
