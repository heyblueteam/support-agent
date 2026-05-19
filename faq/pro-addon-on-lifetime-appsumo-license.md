# Customer on an AppSumo / legacy lifetime tier asks if adding Pro affects their lifetime deal

A common question from legacy/AppSumo LTD customers (Tier 1–9): "Can I add Pro for $500/year, and what does it do to my lifetime license?"

**Short answer: yes, $500/year, and it has zero impact on the lifetime license.** Pro is purely additive.

## Why — the two orthogonal billing dimensions

Blue billing has two independent axes:

- **tier** — the scale limits (users, workspaces, records/org, automations, storage, API rate). For a lifetime customer this comes from their `CompanyLicense.planId` (`bloo_tier1`–`bloo_tier9` → `legacy_tier1`–`legacy_tier9`). The LTD *is* the tier.
- **plan** — `null` (base) | `pro` | `enterprise`. Determines feature flags and overrides a few limits. Pro is a separate $500/year Stripe subscription.

`getOrgLimits(tier, plan)` in `api/src/plans/registry.ts` combines them. Adding Pro only sets `plan = 'pro'`; the `tier` (the lifetime license row) is never touched, converted, or consumed. Cancelling Pro later just drops `plan` back to base — the LTD is unaffected.

**Pro is per-organisation, not per-license.** Legacy tiers allow many orgs per license (Tier 9 = up to 100). Pro is a per-org subscription, so it upgrades only the one organisation it's applied to. Pro on N orgs = N × $500/year.

## What Pro changes on any legacy tier

Flat overrides (same for every tier under Pro):

- Records per workspace: **20,000** (base 2,000)
- Custom fields per workspace: **150** (base 30)
- Custom roles per workspace: 50
- Dashboards: 50
- All 18 Pro feature flags: record-level access, saved views, records in multiple workspaces, conditional/scheduled automations, automation chaining, bulk actions, audit logs, public-view filters/password/custom-URL, reports, priority support, white label, compliance, HTML email templates, rollup fields

Multipliers on the tier's own numbers:

- Records per org: tier base **×10** (legacy tiers: 25,000 → 250,000)
- Automations per month: tier base **×10**
- API rate limits (per key / user / org): tier base **×2**
- Workspaces: ×5 — but legacy tiers are already *unlimited*, so this is a no-op for them

Legacy automation bases (for the ×10): T1–5 = 500, T6 = 750, T7 = 1,200, T8 = 1,800, T9 = 2,500. All legacy tiers share rate limits 200 / 400 / 600 req/s (per key / user / org) → **400 / 800 / 1,200** with Pro.

### Worked example — Tier 9 + Pro

- 20,000 records/workspace (up from 2,000)
- 150 custom fields/workspace (up from 30)
- 25,000 automation runs/month (up from 2,500)
- API rate limits: 400 req/s per key, 800/s per user, 1,200/s per org (up from 200 / 400 / 600)
- All other Tier 9 limits unchanged

To answer for a different tier: flat overrides stay the same; recompute automations (tier base ×10) and rate limits (×2 → 400/800/1,200 for any legacy tier).

## Source of truth

- `api/src/plans/registry.ts` — `TIER_LIMITS`, `PRO_OVERRIDES`, multipliers, `PRO_FEATURES`, `getOrgLimits()`
- Public pricing page: blue.cc/pricing (Pro card) and blue.cc/pro
- Don't quote these internals to the customer — give product-level numbers only.

## Customer reply template

```
Hi [Name],

Happy to clarify both points.

Yes — you can add Pro for $500/year. It's a separate annual subscription that sits on top of your existing plan.

It has no impact on your [Tier N] lifetime license. Your lifetime deal stays exactly as it is — Pro doesn't replace it, convert it, or change it in any way. Pro is purely additive: it unlocks advanced features (record-level permissions, saved views, scheduled automations, reports, white label, and more) and raises your limits. On your [Tier N] plan, Pro takes you to:

- 20,000 records per workspace (up from 2,000)
- 150 custom fields per workspace (up from 30)
- [tier automations ×10] automation runs per month (up from [tier base])
- API rate limits doubled: 400 requests/second per API key, 800/second per user, and 1,200/second per organisation (up from 200 / 400 / 600)

All of your other [Tier N] limits stay the same.

One important note: Pro applies per organisation, not per license. Your [Tier N] license can run multiple organisations, but adding Pro upgrades only the specific organisation you apply it to — not all of them. If you want Pro on more than one organisation, each is a separate $500/year.

Best regards,
Manny
Founder of Blue
```
