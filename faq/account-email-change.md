# How do I change a customer's account email / recover a locked-out owner?

Use this when a customer can't access the email tied to their Blue account — most
commonly an AppSumo LTD owner whose Google Workspace / custom-domain email lapsed,
so they can no longer receive OTP login codes.

The fix is an **email change**, applied to **two** places that must stay in sync:

1. **Firebase Auth** — the login identity (OTP + password authenticate here)
2. **MySQL `User.email`** (db2 write primary) — the app-side record

Change only one and login breaks: the app matches the MySQL row, Firebase
authenticates the email.

## 1. Verify ownership before changing anything

Don't move an account on an unverified claim. For AppSumo LTDs, ask for the
**AppSumo invoice PDF** and confirm every field matches production:

```sql
-- db1 (read replica)
SELECT u.id AS user_id, u.email, u.firstName, u.lastName
FROM User u WHERE u.email = '<old-email>';

SELECT cl.source, cl.planId, cl.licenseId, cl.activationEmail, cl.createdAt,
       c.name, c.slug, c.bannedAt, cu.level
FROM CompanyLicense cl
JOIN Company c ON c.id = cl.company
JOIN CompanyUser cu ON cu.company = c.id AND cu.user = '<user_id>'
WHERE cu.user = '<user_id>';
```

The invoice's **license key** must equal `CompanyLicense.licenseId`, and its
**activation email** the old account email. Confirm the requester is `OWNER` and
`Company.bannedAt` is NULL. Also confirm the **new** email has no existing Blue
user (a collision would block the change):

```sql
SELECT id, email FROM User WHERE email = '<new-email>';  -- expect 0 rows
```

## 2. Change the Firebase Auth email

Blue login is Firebase-backed. The Firebase UID is `User.firebaseUid ?? User.id`.

**Important:** accounts created before Google Sign-In existed are keyed purely by
UID and often have **no email set** on the Firebase record (`email=undefined`,
empty `providerData`). `getUserByEmail("<old-email>")` returns
`auth/user-not-found` for these — that's expected, not a dead end. Look the user
up **by UID** instead. This script is idempotent (updates if present, creates if
not):

```bash
cobalt run --project api --no-tty 'node -e "
const admin = require(\"firebase-admin\");
admin.initializeApp({ credential: admin.credential.cert({ clientEmail: process.env.FIREBASE_CLIENT_EMAIL, privateKey: process.env.FIREBASE_PRIVATE_KEY.replace(/\\\\n/g, \"\\n\"), projectId: process.env.FIREBASE_PROJECT_ID }) });
const UID = \"<USER_ID>\";
const NEW = \"<new-email>\";
(async () => {
  let u = null;
  try { u = await admin.auth().getUser(UID); } catch (e) { if (e.code !== \"auth/user-not-found\") throw e; }
  if (u) { console.log(\"before email=\" + u.email + \" providers=\" + u.providerData.map(p=>p.providerId).join(\",\")); await admin.auth().updateUser(UID, { email: NEW, emailVerified: true }); console.log(\"action=updated\"); }
  else { console.log(\"before: no firebase user\"); await admin.auth().createUser({ uid: UID, email: NEW, emailVerified: true }); console.log(\"action=created\"); }
  const a = await admin.auth().getUser(UID);
  console.log(\"after email=\" + a.email + \" verified=\" + a.emailVerified);
})().catch(e => { console.error(\"ERR\", e.code || e.message); process.exit(1); });
"'
```

Escaping: single quotes wrap the shell arg, `\"` for JS strings, `\\\\n` for the
private-key newline (survives the shell → string-literal pipeline). Always set
`emailVerified: true` so the customer isn't stuck on a verification gate.

## 3. Change the MySQL `User.email` (db2 write primary)

Writes go to **db2 only** (db2 is the write primary; db1 is a read replica). The
`AND email = '<old-email>'` guard makes a re-run a safe no-op.

```bash
cd /Users/manny/blue/infra/ansible
DB2_PASS=$(ansible-vault view host_vars/db2.blue.cc/vault.yml 2>/dev/null | grep mysql_root_password | cut -d'"' -f2)
ssh root@db2.blue.cc "mysql -u root -p${DB2_PASS} blue_production -e \"UPDATE User SET email='<new-email>' WHERE id='<USER_ID>' AND email='<old-email>'; SELECT ROW_COUNT(); SELECT id, email FROM User WHERE id='<USER_ID>';\" 2>/dev/null"
```

Expect `ROW_COUNT() = 1` and the SELECT showing the new email.

## 4. Verify consistency

Firebase `after email=` and the MySQL `SELECT` must both show the new email. If
they differ, login will fail — fix before replying.

## Important notes

- **Production writes.** Steps 2–3 modify prod auth + DB. They require explicit
  operator approval and are run by the operator, not autonomously — investigation
  reads (Step 1) are db1-only.
- **Both sides or neither.** A half-applied change locks the user out worse than
  before. Verify Step 4 before sending the reply.
- **OTP is passwordless.** The customer doesn't need a password — once the email
  is correct on both sides they sign in at `blue.cc/sign-in` via the one-time
  code. Only use `reset-customer-password` if they specifically want a password.
- **Banned accounts:** refuse and route to `ban-customer` instead.

## Customer reply template

```
Hi [Name],

Thanks for sending the invoice — I was able to verify your purchase and confirm
you're the owner of the [Workspace] workspace.

I've moved your account from [old-email] over to this address ([new-email]), so
the lapsed domain is no longer in the way. Your workspace, license, and all your
data are untouched.

To get back in:
1. Go to https://blue.cc/sign-in
2. Enter [new-email]
3. You'll receive a one-time login code at this address — enter it and you're in

Everything should be exactly as you left it. Let me know if you hit any trouble
signing in.

Best regards,
Manny
Founder of Blue
```
