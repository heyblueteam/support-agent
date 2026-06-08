# Why are a customer's OTP login codes always rejected as "invalid"?

Use this when a customer **receives** login codes fine but every code comes back
**"Invalid one time password"** — on any sign-in page (`blue.app/sign-in`,
`app.blue.cc`), entered immediately, no reuse. This is **not** a deliverability
problem (the email arrives) and **not** an expiry problem (it fails even on a
fresh code).

## Root cause: a zombie Firebase record (UID exists, email detached)

Blue login is Firebase-backed, and OTP validation matches the submitted code to a
Firebase user **looked up by email**. If the Firebase record for that UID has **no
email attached**, `getUserByEmail()` returns `auth/user-not-found` — so there's
nothing to match the code against, and every code is rejected.

This is the same class of orphaned/UID-only Firebase record described in
[`account-email-change.md`](account-email-change.md): accounts created before
Google Sign-In existed (or otherwise left in a half-initialized state) are keyed
purely by UID with `email=undefined` and empty `providerData`. OTP can't work
until the email is re-attached.

## Confirm

Find the user and note the Firebase UID (`User.firebase_uid ?? User.id`):

```sql
-- db1 (read replica)
SELECT id, email, firebase_uid FROM User WHERE email = '<customer-email>';
```

Then check Firebase. `getUserByEmail` returning `auth/user-not-found` while the
UID resolves is the tell:

```bash
cobalt run --project api --no-tty 'node -e "
const admin = require(\"firebase-admin\");
admin.initializeApp({ credential: admin.credential.cert({ clientEmail: process.env.FIREBASE_CLIENT_EMAIL, privateKey: process.env.FIREBASE_PRIVATE_KEY.replace(/\\\\n/g, \"\\n\"), projectId: process.env.FIREBASE_PROJECT_ID }) });
const UID = \"<USER_ID>\";
const EMAIL = \"<customer-email>\";
(async () => {
  try { await admin.auth().getUserByEmail(EMAIL); console.log(\"email attached — NOT this issue\"); }
  catch (e) { if (e.code === \"auth/user-not-found\") { const u = await admin.auth().getUser(UID); console.log(\"zombie: uid exists, email=\" + u.email + \" providers=\" + u.providerData.map(p=>p.providerId).join(\",\")); } else throw e; }
})().catch(e => { console.error(\"ERR\", e.code || e.message); process.exit(1); });
"'
```

## Fix: re-attach the email (one `updateUser` on the UID)

`updateUser(UID, { email, password, emailVerified: true })` attaches the email,
sets a password, and clears the verification gate in one call — restoring **both**
OTP and password login. This is exactly the zombie-UID path in the
**`reset-customer-password`** skill; run that skill, which handles
`auth/uid-already-exists` automatically:

```
zombie uid -> updateUser
after uid=<UID> email=<customer-email> verified=true providers=password
```

After this, the customer can sign in with the temp password **and** their OTP
codes will be accepted again. Hand the temp password back through the support
channel and tell them to change it in Account Settings.

## Notes

- **Production write.** The `updateUser`/create requires explicit operator
  approval (run by the operator, not autonomously); investigation reads are
  db1-only.
- **Email mismatch is common here.** The address a customer emails *from* often
  differs from their Blue login (e.g. a work address vs. the account email).
  Always confirm which email is the actual Blue account — and that they control
  the inbox where codes land — before resetting.
- **Don't reach for deliverability fixes.** The codes arriving is proof delivery
  works; the failure is on validation. See [`email-deliverability.md`](email-deliverability.md)
  only if codes aren't arriving at all.
