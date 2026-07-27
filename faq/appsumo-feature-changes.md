# "You're nerfing AppSumo customers" — what actually changed

Use this when a lifetime/AppSumo (or direct "Blue/bloo" LTD) customer asks whether
their features have been **reduced / "nerfed"** — most often storage, but the
AppSumo grapevine bundles several unrelated changes together. The job is to
**separate three different things** they've conflated, because only one of them is
actually AppSumo-specific.

Sister FAQ for tier limits / upgrades / the automation cap mechanics:
`appsumo-tiers-and-upgrades.md`. Pricing context: `pro-addon-on-lifetime-appsumo-license.md`,
`white-label-ltd.md`.

## The one-line truth

**The only thing being reduced *specifically* for AppSumo accounts is file
storage.** Everything else is either a platform-wide change that applies to all
customers (including brand-new paying ones), a feature that was never part of the
lifetime deal, or a misunderstanding.

Sort every complaint into one of these buckets:

| Bucket | Items |
|--------|-------|
| **AppSumo-specific reduction** | Storage allowance (storage GB + max file size) |
| **Platform-wide (all customers)** | Automation run limits · API rate limits · removal of Blue branding (white-label-only) · per-workspace record/field caps |
| **Never part of the deal / misunderstanding** | Custom CNAMEs · the per-*org* record limit shown on public pricing |
| **Removed for everyone** | Status Updates |

## 1. Storage — the only AppSumo-specific change

- Original deal: **3 TB per org, 5 GB max file** — effectively unlimited.
- Reason for the change: a small number of accounts abused it (generic file
  backup, large media libraries; one org held ~1 TB in a few hundred files;
  several hundred orgs stored copyright-infringing content, a terms breach).
- New allowances **scale with tier** (a total-per-org cap **and** a max-single-file
  cap):

  | Tier | Total | Max file |
  |------|------:|---------:|
  | 1 | 10 GB | 100 MB |
  | 2 | 20 GB | 250 MB |
  | 3 | 30 GB | 500 MB |
  | 4 | 50 GB | 500 MB |
  | 5 | 75 GB | 750 MB |
  | 6 | 150 GB | 1 GB |
  | 7 | 200 GB | 1 GB |
  | 8 | 300 GB | 2 GB |
  | 9 | 500 GB | 2.5 GB |

- **Grandfathered:** existing uploads are never touched — the cap applies to
  **new uploads only**, once an org is over.
- **~98%** of active AppSumo orgs are already under the new limits.
- **30% upgrade discount** for anyone over: code `APPSUMO30`.
- Agreed with the AppSumo partner team before announcing.

> **Status (as of 2026-06):** announced, **not yet enforced in code** —
> `api/src/plans/registry.ts` still shows `storageGB: 3_000` / `maxUploadSizeMB:
> 1_024` for every legacy tier. Rollout is **later in Q3**, with advance notice to
> affected orgs. So quote these numbers as the *announced policy*, not current
> enforcement.

## 2. Custom domains (CNAMEs) — a misunderstanding, never in the deal

Custom CNAMEs were **never** part of the AppSumo deal. They were a separate,
standalone, manually-configured feature offered to **all** customers. In the
rebuilt Blue that approach was retired in favour of **full white-label**, which
lives in **Pro**. Nothing AppSumo-specific was removed.

## 3. Record & field limits — per-org limit ≠ AppSumo; per-workspace caps are platform-wide

- The **per-organization record limit** shown on public pricing does **not** apply
  to AppSumo accounts.
- What applies **to everyone** is a **per-workspace cap: 2,000 records and 30
  fields** (`BASE_LIMITS` in `registry.ts`). This is a **performance** measure
  across the whole platform, not an AppSumo nerf — especially because nearly every
  field type (formulas, cross-workspace references, lookups — all heavy) is open to
  all customers.
- AppSumo includes **unlimited workspaces**, so effective totals are very large.

> Internal note: `registry.ts` currently also carries `recordsPerOrg: 25_000` on
> legacy tiers. Policy per founder is that the per-org cap does **not** bind AppSumo
> accounts; the per-workspace caps are the operative limit. Reconcile if a customer
> pushes on the exact number.

## 4. Automation limits — platform-wide, generous for LTD

- Unlimited automations were **never** promised. Previously monitored manually;
  now metered per the industry standard, because every run has real cost.
- Cap is **per org per UTC calendar month**, scaling with tier (Tier 1–4: 500 …
  Tier 9: **2,500/org/mo**). Tier 9 × 100 orgs = up to **250,000 runs/month**.
- Set so only a tiny fraction of customers are affected — targets accounts
  programmatically firing tens of thousands of runs.
- Mechanics (counter, reset, `assertCanRunAutomation`): see
  `appsumo-tiers-and-upgrades.md` §3. Pro multiplies the cap ×10.

## 5. API rate limits — platform-wide, LTD is at the top tier

- Introduced across **all** customers.
- Legacy/AppSumo tiers get **200 req/min per key, 400 per user, 600 per org** —
  the **highest of any non-Pro/Enterprise plan** (identical to the top "Scale"
  plan). Pro doubles it; Enterprise is higher still.
- Window is **per minute** (`api/src/lib/rate-limit.ts`). (The older
  `appsumo-tiers-and-upgrades.md` says "req/s" — that's a typo; it's per minute.)

## 6. Branding removal — platform-wide, white-label-only

Removing Blue branding from **forms, shared links, and emails** is a **white-label
(Pro)** capability for **all** customers — not something taken from AppSumo
specifically.

## 7. Status Updates — the only feature fully removed (for everyone)

The single feature fully retired. A **product-vision** decision, not AppSumo-related.
Everyone was emailed in advance and offered a **full data export**.

## 8. Where the limits go from here (use to reassure)

Frame the non-storage limits as **conservative starting points, not the ceiling**.
Blue is partway through rewriting a 2M+ line legacy codebase, and there's a lot of
optimization still ahead. As that lands and the data comes in, the plan is to
**raise limits over time — automations and records/custom fields especially.**
Storage is the **exception**: it's the one place with **no structural advantage** —
it can't be engineered around, the costs are simply what they are. So most numbers
trend **up**, not down.

## 9. The bigger picture (use to close)

`blue.app/changelog` shows the platform improving almost daily — whether someone
bought in 2020/21 or 2024/25. The **entire platform was rebuilt at no extra cost**,
and **95%+ of every new feature is available to AppSumo users**; only a handful are
reserved for Pro/Enterprise.

## Customer-facing answer (adapt — never mention code/tables/`planId`s)

Lead with the one-line truth, then address only what they raised. Personalise the
storage/automation numbers to their tier. Keep the four buckets straight:
AppSumo-specific = storage only; the rest is platform-wide, never-in-the-deal, or
removed-for-everyone.
