# What is Extended Privacy?

Extended Privacy is an organization setting that stops Blue from sharing
purchase conversion data with advertising platforms (Meta/Facebook and
LinkedIn). The organization owner turns it on in **Org Settings → General**.
It is available on the **Pro and Enterprise** plans (including the Pro add-on
on a lifetime plan). Once enabled, it stays on — even if the plan later
downgrades. It exists for organizations with strict data policies — schools,
government agencies, and similar — where the rules require zero data flowing
to advertising platforms.

## What is shared with ad platforms, and when

Only **purchase conversion events**, and only for purchases made from outside
the EU, EEA, UK, or Switzerland. Blue sends them server-side so ad platforms
can measure ad campaign performance:

- A one-way **hash** of the buyer's email address (the email cannot be recovered)
- IP address and browser user agent at the time of purchase (Meta only)
- A click identifier, if the buyer arrived from a Meta or LinkedIn ad
- The amount, currency, and event type (purchase, subscription, refund)

Blue places **no ad tracking pixel** on the website and sets **no ad cookies**.
LinkedIn receives data on initial plan purchases only.

## What is never shared

**Workspace data** — projects, records, files, comments, and everything stored
in Blue — is never sent to any advertising platform, on any plan, with or
without Extended Privacy.

## What Blue sends by default

- **EU / EEA / UK / Switzerland**: excluded from this data sharing entirely.
  Location comes from the billing country on file with our payment processor.
- **Everywhere else**: purchase conversion events are shared, unless
  Extended Privacy is on.

Extended Privacy gives organizations outside those regions the same treatment:
no conversion data leaves Blue's servers for that organization, regardless of
location. The setting covers the purchase events described above — workspace
data was never shared on any plan to begin with.

## Customers on a lifetime (LTD) plan

Lifetime tiers do not include Extended Privacy on their own. Adding the Pro
add-on subscription unlocks it, along with the other Pro features.
