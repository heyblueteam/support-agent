# Customer not receiving emails from Blue

When customers report not receiving emails from Blue (sign-in codes, invitations, notifications), the cause is almost always spam-foldering or provider-side filtering. Have them check spam and whitelist our sending addresses.

## Blue's sending email addresses

When asking customers to whitelist or check spam, reference these addresses:

| Address | Purpose |
|---------|---------|
| `noreply@system.blue.cc` | Sign-in codes, invitations, company deletion, account changes (blue.cc accounts) |
| `noreply@notify.blue.cc` | General notifications to users |
| `notifications@messages.blue.cc` | Default outbound notifications |
| `notifications@automations.blue.cc` | Emails sent via automation workflows |
| `notifications@mail.process-system.app` | **White-label only** — OTP/system emails for white-label tenants. Never used for `blue.cc` accounts (gated by `!isBlue` in `otpSender` strategy). Don't ask blue.cc customers to whitelist this. |

## What to tell the customer

1. Check the spam/junk folder for a message from the relevant address above. Sign-in codes come from `noreply@system.blue.cc`.
2. Add that address to their contacts / safe-sender list so future emails land in the inbox.
3. Confirm they're using the exact email tied to their account — a typo or a different address (e.g. a domain alias) won't receive anything. Look the account up by email in the DB to confirm the address on file (see the `investigate-customer` skill).

## If whitelisting doesn't help

Repeated hard bounces (invalid address, domain doesn't exist) can get an address blocked at the email provider. If a customer still can't receive anything after checking spam and confirming the address is correct, escalate to Manny — the block has to be cleared on the sending provider's side.
