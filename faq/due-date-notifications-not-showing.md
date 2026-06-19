# Why isn't a customer getting due date / task notifications ("in app" / desktop)?

The usual cause is **not a bug** — it's that the customer isn't an **assignee** on the record. This is the single most common reason a due date notification "doesn't show."

## Key facts about Blue notifications

1. **There are only two channels: Email and Push.** Blue has **no separate in-app notification center / bell / inbox**. When a customer says the notification isn't showing "in app", they mean the **web/desktop push toast** (Firebase Cloud Messaging), not a feed inside the app. The Account → Notifications screen confirms this — every event has exactly an **Email** and a **Push** checkbox.

2. **Due date notifications go ONLY to the record's assignees.** The reminder (`TODO_REMINDER`) and overdue (`TODO_OVERDUED`) handlers build their recipient list from the record's assignees and send nothing if there are none. So if a record has a due date but the customer isn't assigned to it, they get **no due date notification on any channel** — even though their other emails (comments, mentions, assignments) keep arriving. "My email works but due date push doesn't" is very often actually "I'm not an assignee on the record I set a due date on."

3. **All-day due dates fire the next morning, not at midnight.** A due date with no specific time (end-of-day) defers the overdue notification to **~8:01am the next day**. A customer testing with an end-of-day due date and expecting an instant notification will see "nothing happened."

4. **Push requires three things together:** browser permission granted (green "Push notifications enabled" banner in Account → Notifications), a registered FCM token, and the per-event Push checkbox on (defaults on). Granting browser permission alone isn't enough if token registration didn't complete.

Push **does** work on the current app — it's not globally broken. ~3,300+ users carry current-app push tokens and there are no mass FCM send failures. If *you personally* have never seen one, check whether your own account ever registered a token (see query below) — most "I've never seen a push" cases are simply an account that never granted/registered.

## How to diagnose (db1, read-only)

Find the user + company:

```sql
SELECT c.id AS companyId, c.name, cu.level, u.id AS userId, u.email
FROM User u
JOIN CompanyUser cu ON cu.user = u.id
JOIN Company c ON c.id = cu.company
WHERE u.email = 'customer@example.com';
```

**Is the customer an assignee on ANY due-dated record?** (If 0, this is almost certainly the cause.)

```sql
SELECT COUNT(*) AS assigned_with_due
FROM Todo t
JOIN TodoUser tu ON tu.todo = t.id
JOIN User u ON u.id = tu.user
WHERE u.email = 'customer@example.com' AND t.duedAt IS NOT NULL;
```

**Do they have a push token, and how old is it?** (A token only from the old app, with no recent one from the current app, means push has nothing valid to deliver to.)

```sql
SELECT upt.createdAt, upt.updatedAt, LEFT(upt.token, 14) AS token_prefix
FROM UserPushToken upt
JOIN User u ON u.id = upt.user
WHERE u.email = 'customer@example.com';
```

**Are their per-event push/email preferences on?** (Defaults to allowed if no row exists.)

```sql
SELECT c.name AS company, no.name AS event, cuno.allowEmail, cuno.allowPush
FROM CompanyUserNotificationOption cuno
JOIN CompanyUser cu ON cu.id = cuno.companyUser
JOIN Company c ON c.id = cu.company
JOIN User u ON u.id = cu.user
JOIN NotificationOption no ON no.id = cuno.notificationOption
WHERE u.email = 'customer@example.com'
  AND no.name IN ('TODO_OVERDUED', 'TODO_REMINDER', 'TODO_DUE_DATE_CHANGED');
```

## Most common root cause

**Not an assignee on the record.** The fix is product usage, not code: add the user (or whoever should be notified) as an assignee on the record. Once assigned, both Email and Push fire on the due date.

---

## Customer reply template

```
Hi [Name],

Due date notifications are only sent to the people assigned to that record. If
a record has a due date but you're not one of its assignees, no due date
notification goes out — on any channel — even though your other email
notifications keep coming through normally.

To get it working:

1. Open the record and add yourself (or whoever should be notified) as an
   assignee.
2. In Account → Notifications, confirm the green "Push notifications enabled"
   banner is showing and that Task Overdue / Due Date Reminders have the Push
   checkbox turned on.
3. To test quickly, set a due date a few minutes into the future rather than
   end-of-day. An all-day due date sends the next morning rather than at
   midnight, so an end-of-day test can look like nothing happened.

Best regards,
Manny
Founder of Blue
```
