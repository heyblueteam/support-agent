# Invited user never gets the email (hard bounce on their mailbox)

A customer invites someone to a workspace/board and reports the invitee never receives the invite — checked spam, nothing. Often the sender also can't find any trace of the email in the provider dashboard.

This is usually **not** a Blue bug. The invite is sent fine and then **hard-bounces** at the recipient's mail server (mailbox doesn't exist, isn't provisioned yet, wrong address, or blocks outside mail). Hard-bounced addresses get **suppressed** at the email provider, so they don't show in the normal sent/contacts view — which is why "no records" in the dashboard is misleading. The provider logs (shipped to Loki) are the source of truth.

See also [`email-deliverability.md`](email-deliverability.md) for the general spam/whitelist case and Blue's sending addresses.

## How to confirm (Loki)

Logging access is in [`docs/grafana.md`](../../docs/grafana.md). The email provider's relay/bounce/delivery events are in the Docker logs. Query Loki for the invitee's address over the day of the report:

```bash
ssh root@server.blue.cc "curl -s -G 'http://localhost:3100/loki/api/v1/query_range' \
  --data-urlencode 'query={job=\"docker\"} |~ \"<invitee-email>\"' \
  --data-urlencode 'start=<YYYY-MM-DD>T00:00:00Z' \
  --data-urlencode 'end=<YYYY-MM-DD>T23:59:59Z' \
  --data-urlencode 'limit=200'"
```

What the lines mean:

| Log line | Meaning |
|---|---|
| `Email relayed: Blue <…> → <addr>` | We sent it — leaves Blue fine |
| `[WEBHOOK] Permanent bounce received for <addr>` | **Hard bounce** — recipient server rejected it (the problem) |
| `[WEBHOOK] Delivery confirmed for <addr>` | Delivered successfully |
| `[WEBHOOK] Transient bounce …` | Soft/temporary — not counted, usually retried |

**Key diagnostic:** query the whole domain (`|~ "@theirdomain.com"`) and compare. If the invitee's address shows `Permanent bounce` while their **colleagues on the same domain show `Delivery confirmed` / `Open received`**, the problem is isolated to that one mailbox — not Blue, not a domain-wide block. That's the conclusive signal.

Cross-check the account state with the `investigate-customer` skill: a brand-new invitee has **no `User` row**, and the `Invitation` row will have `createdAt == updatedAt` (created once; `resendInvitation` bumps `updatedAt`, so equal timestamps mean it was never resent — rules out a rate-limit/retry issue on our side).

## Workaround — the invite link needs no email

The invitation link is **self-contained**: `/accept-invitation?id=<invitationId>&email=<email>` fetches a one-time code from the invitation id server-side, so clicking it lets the invitee create their account and join the board **without ever receiving the email**. Hand the link to the customer to pass along (Slack, text, etc.):

```
https://blue.app/accept-invitation?id=<invitationId>&email=<url-encoded-email>
```

Get `<invitationId>` from the `Invitation` row (`investigate-customer`). New invitee just enters their name — done. (Link is valid until the invitation's `expiredAt`, 7 days from creation.)

**Gotcha — don't tell them to use "Continue with Google":** the Google/social sign-up path does **not** honor invitations. It skips `invitationSignUp` and routes the user into creating their own new organization, leaving the invitation orphaned. The plain link (above) is the reliable path — it goes through the invitation profile step, which has no social button anyway.

## What to tell the customer

1. The invite is sending correctly; their mail server is rejecting it as undeliverable for that person specifically (note that their colleagues on the same domain receive Blue email fine).
2. Ask them to confirm the exact, active address with whoever manages their email, and that the mailbox can receive outside mail.
3. To unblock immediately, send them the `blue.app/accept-invitation` link to pass to the invitee — no email needed.
4. If the correct address turns out to be different, re-issue the invite to the right one.

## If the mailbox gets fixed but invites still bounce

Once an address has hard-bounced it may stay **suppressed** at the provider, so the first re-send after the customer fixes their mailbox can still be dropped. Escalate to Manny to clear the suppression on the provider side.
