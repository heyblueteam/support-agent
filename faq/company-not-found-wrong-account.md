# "Company not found" / "Failed to load organization" — user is signed into the wrong account

A customer reports they "can't log in" to a specific organization and sees **"Company
not found"** or **"Failed to load organization"**. They usually *can* log in fine — the
error appears **after** authentication, when the app tries to open the org. The common
cause is not a broken org and not a broken login: the customer has **two Blue accounts**
and is signed into the one that isn't a member of the org they're trying to reach.

## Why this produces the error

Org URLs are `blue.app/org/<slug>/...`. After auth, `currentOrg.ts` fetches the company
by slug, scoped in `auth.ts` to the **logged-in user's `CompanyUser` rows**. If the
authenticated user isn't a member of that company, the query returns nothing →
`currentOrg.ts` throws "Organization not found" / falls into "Failed to load
organization". The org itself is fine; it's simply invisible to the account they're on.

A frequent giveaway: the org slug in their URL is a real, healthy company, but the email
they're *currently* signed in with has no `CompanyUser` row for it.

## How customers end up with two accounts

- **Different domains.** A work account on a custom/org domain (e.g. `name@org-domain`)
  *and* a personal Gmail. The org membership lives on the work account; they sign in with
  the personal one.
- **Gmail dot-aliasing.** `may.zamuco@gmail.com` and `mayzamuco@gmail.com` are the *same
  inbox* to Gmail but **different `User` rows** in Blue. They may have created/used both at
  different times. (See `project_gmail_dot_aliasing` in memory.)
- **In-app feedback reveals the live session.** A "Feedback Form" email
  (`notifications@automations.blue.cc`) carries the logged-in user as `reply_to` — compare
  that against the account they *say* they use to confirm which one they're actually on.

## Diagnosis (db1, read-only)

Find every account for the person — check the address they emailed from, the one they say
they use, and dot-variants:

```sql
-- All matching users (try the stated login, the From address, and dot-variants)
SELECT id, email, firstName, lastName, firebase_uid, lastActiveAt, createdAt
FROM User
WHERE email IN ('<stated-login>', '<from-address>', '<dot-variant>')
   OR email LIKE '%<localpart>%';
```

Then list each account's memberships and confirm which one belongs to the target org:

```sql
SELECT cu.user, u.email, cu.company, c.name, c.slug, cu.level, c.bannedAt
FROM CompanyUser cu
JOIN User u ON u.id = cu.user
LEFT JOIN Company c ON c.id = cu.company
WHERE cu.user IN ('<userId-A>', '<userId-B>')
ORDER BY u.email;
```

Confirmed when: the target org's slug is healthy (`bannedAt IS NULL`, members + projects
present), one account is a `CompanyUser` of it, and the account they've been *using* is
not. `lastActiveAt` usually shows the member account stale and the other active recently.

## Resolution

Almost always **no code change and no DB write** — they just need to sign in with the
account that's actually a member:

1. Go to **blue.app** (not blue.cc — the domain migrated; old bookmarks point at blue.cc).
2. Sign out.
3. Sign in with the **member** email.
4. Open the target org.

**Then confirm OTP reachability.** If the member account is on a domain they may no longer
control (lapsed work email, custom domain), they might not receive the one-time code there
— that's the *real* blocker, not the "Company not found" symptom. Ask them to confirm the
code arrives at the member email. If it doesn't, escalate to an email change
(`account-email-change.md`).

**Watch for the collision trap.** Moving the member account's email to their other inbox
(e.g. serf.me → gmail) **fails if that gmail already has its own Blue `User`** —
`User.email` is unique. In the two-accounts case both addresses usually already exist, so a
straight email change is blocked. Options then: (a) have them keep using the member account
once OTP is sorted, or (b) the org OWNER re-invites their preferred account as a new member
(membership is granted at the org level by an admin/owner, not by us moving emails). A
plain MEMBER can't self-resolve either path.

## Customer-reply notes

- Frame it plainly: "You have two separate accounts; access to <Org> is on <member-email>,
  and you're currently signed in with <other-email>." Don't expose `CompanyUser`, slugs, or
  internals.
- Give the exact steps (blue.app → sign out → sign in with the member email).
- Ask them to confirm the login code arrives at the member email — branch the next step on
  their answer. Don't promise a proactive follow-up; wait for their reply (leave the thread
  in the inbox, don't archive).

Example thread: "Still having issues with log ins" (may.zamuco@gmail.com, 2026-06-22) —
member account `may.zamuco@serf.me` in PACCTX, signed in as `mayzamuco@gmail.com`.
