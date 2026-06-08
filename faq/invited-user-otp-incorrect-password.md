# Invited teammate can't finish login — "incorrect password" on the one-time code

A customer adds a teammate, the teammate **gets the email and the code** but reports the
code "doesn't work" / says **"incorrect password"**, and they can't get in. This is almost
always a **half-finished invitation**, not a code-generator bug and not a password problem.

Different from [`invite-email-hard-bounce.md`](invite-email-hard-bounce.md) (invitee never
*receives* the email) and from [`otp-codes-rejected-zombie-firebase.md`](otp-codes-rejected-zombie-firebase.md)
(an *existing* account whose Firebase email got detached). Use this one when the invite was
received and the account was **never created**.

## What's actually happening

Blue logins are **passwordless** — there is no password to get wrong. New teammates go
invite link → enter the 4-digit code → fill a short profile form → account created
(`invitationSignUp`). If they stop before submitting that form, **no `User` row is ever
written** and they stay stuck, re-trying the code.

Why "incorrect password" then? Two common reasons, neither a real fault:

- **Wording.** The OTP-failure string is *"Invalid one time password"* (German UI:
  *"Ungültiges Einmalpasswort"*). Non-native speakers routinely relay this as "incorrect
  password" — there is no password field in this flow.
- **Wrong page.** If they use **Log in** (instead of the invite link) and type their email,
  they hit *"You have a pending invitation… accept it before signing in"*, not the code form
  at all.

The code itself is fine: a 4-digit `[1000–9999]` value, valid for 7 days. If it exists and
hasn't expired, the rejection is environmental or user-side — **don't go hunting for a code
bug.**

## Confirm (db1 read — `investigate-customer` patterns)

Three checks for the stuck teammate's email:

```sql
-- 1. No completed account → empty result = never finished signup
SELECT id, email FROM User WHERE email = '<email>';

-- 2. A still-pending invitation
SELECT id AS invitationId, email, accessLevel, expiredAt, createdAt
FROM Invitation WHERE email = '<email>';

-- 3. An unconsumed sign-up code (deleted on successful signup, so its presence = stuck)
SELECT email, category, code, expiredAt, createdAt
FROM SecurityCode WHERE email = '<email>' AND category = 'SIGN_UP';
```

The signature of this issue: **no `User` row** + **a pending `Invitation`** + **an unconsumed
`SIGN_UP` code whose `expiredAt` is in the future.** Cross-check against teammates the same
customer added who *did* get in (they'll have `User` rows) — that proves the flow works and
isolates the problem to these people.

The sign-up / invite flow now logs to the **`auth` channel** in Loki (filter
`{job="docker"} |~ "channel.*auth"` for the email): you'll see `invite: created`,
`signup: code sent`, `signup: code verified|rejected`, `invite: signup completed`. A `Click`
webhook followed by a ~1-second analytics session and **no** `invite: signup completed` is the
fingerprint of "reached the page, never submitted" (or a corporate link-scanner — see Notes).

## Resolve — hand them a working link (no email needed)

The invitation link is self-contained (see [`invite-email-hard-bounce.md`](invite-email-hard-bounce.md)
for the mechanics). Pull `invitationId` and `code` from the queries above and give the customer
one of:

```
# Drops straight onto the profile form with the code pre-filled — they just type their name:
https://blue.app/sign-up?email=<url-encoded-email>&code=<code>

# Or the canonical entry point (re-issues the code email, then lands on the same form):
https://blue.app/accept-invitation?id=<invitationId>&email=<url-encoded-email>
```

They **must submit the name/profile form** to finish — opening the link alone doesn't create
the account. If the invitation or code has expired, **re-send the invitation** (resets a fresh
7-day code) rather than handing out a dead link.

**Don't suggest "Continue with Google"** — the social path skips `invitationSignUp` and spins
up a brand-new org, orphaning the invitation.

## What to tell the customer

1. The two were partway through setup, not blocked by a real error — Blue sign-in uses a
   one-time **code**, never a password (that's usually the "incorrect password" confusion).
2. Have them open the invite link and **complete the short name form** in one sitting; the code
   from the email is all they need.
3. If it still fails, ask *which page* they're on and for a screenshot of the exact message —
   that distinguishes "wrong page" from a genuine code rejection.

## Notes

- **Suspect corporate email security** (Microsoft SafeLinks, Proofpoint, etc.) when the link
  works for us in incognito but not for the customer. Enterprise scanners pre-fetch invite
  links to vet them — that shows in Loki as a `Click` + a 1-second bot session with no
  completion, and can consume or rewrite the URL before the human clicks. Big orgs (the kind
  with a deliverability-savvy IT dept) are where this bites.
- **Completing it for them is a real fix but creates the account under whatever name you type**
  (and signs *your* browser in as them) — only do it deliberately, and tell them to fix their
  profile name afterward.
