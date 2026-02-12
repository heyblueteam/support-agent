# How do I extend a company's free trial?

When customers need more time to evaluate Blue, we can extend their free trial period directly in the database.

## How free trials work in Blue

Free trials are tracked on the `Company` table. When a company is created, it automatically receives a 14-day trial (configurable via `FREE_TRIAL_DURATION` environment variable).

**Key fields:**
- `freeTrialStartedAt` - When the trial originally began
- `freeTrialExpiredAt` - **The expiration date/time** (this is the key field that determines if trial is active)
- `freeTrialExtendedAt` - Timestamp of the last extension (for audit tracking)
- `freeTrialExtendedById` - User ID who extended it (for audit tracking)

**Trial status logic:**
- The system checks `freeTrialExpiredAt` in real-time to determine if a company is still in trial
- If `freeTrialExpiredAt` is in the future → trial is active
- If `freeTrialExpiredAt` is in the past → trial has expired and company is locked (unless they have an active subscription)

## 1. Find the company

```sql
-- Search by company name
SELECT id, slug, name, freeTrialStartedAt, freeTrialExpiredAt
FROM Company
WHERE name LIKE '%CompanyName%'
ORDER BY createdAt DESC;

-- Or search by company slug
SELECT id, slug, name, freeTrialStartedAt, freeTrialExpiredAt
FROM Company
WHERE slug = 'company-slug';

-- Or search by user email
SELECT c.id, c.slug, c.name, c.freeTrialStartedAt, c.freeTrialExpiredAt
FROM Company c
JOIN CompanyUser cu ON cu.company = c.id
JOIN User u ON u.id = cu.user
WHERE u.email = 'user@example.com';
```

## 2. Check current trial status

```sql
-- Get detailed trial information
SELECT
  id,
  name,
  slug,
  freeTrialStartedAt,
  freeTrialExpiredAt,
  freeTrialExtendedAt,
  freeTrialExtendedById,
  CASE
    WHEN freeTrialExpiredAt > NOW() THEN 'Active'
    ELSE 'Expired'
  END as trialStatus,
  DATEDIFF(freeTrialExpiredAt, NOW()) as daysRemaining
FROM Company
WHERE id = 'COMPANY_ID';
```

## 3. Extend the free trial

### Extend by a specific number of days (e.g., 30 days)

```sql
-- Extend from NOW (if trial is already expired)
UPDATE Company
SET
  freeTrialExpiredAt = DATE_ADD(NOW(), INTERVAL 30 DAY),
  freeTrialExtendedAt = NOW()
WHERE id = 'COMPANY_ID';

-- Or extend from current expiration (if trial is still active)
UPDATE Company
SET
  freeTrialExpiredAt = DATE_ADD(freeTrialExpiredAt, INTERVAL 30 DAY),
  freeTrialExtendedAt = NOW()
WHERE id = 'COMPANY_ID';
```

### Extend to a specific date

```sql
UPDATE Company
SET
  freeTrialExpiredAt = '2026-03-15 23:59:59',
  freeTrialExtendedAt = NOW()
WHERE id = 'COMPANY_ID';
```

### Extend multiple companies at once

```sql
-- Useful for bulk operations (e.g., all companies belonging to one customer)
UPDATE Company
SET
  freeTrialExpiredAt = DATE_ADD(NOW(), INTERVAL 30 DAY),
  freeTrialExtendedAt = NOW()
WHERE name IN ('CompanyA', 'CompanyB', 'CompanyC');

-- Or by company owner
UPDATE Company c
JOIN CompanyUser cu ON cu.company = c.id
JOIN User u ON u.id = cu.user
SET
  c.freeTrialExpiredAt = DATE_ADD(NOW(), INTERVAL 30 DAY),
  c.freeTrialExtendedAt = NOW()
WHERE u.email = 'owner@example.com'
AND cu.role = 'OWNER';
```

## 4. Verify the extension

```sql
-- Check that the trial was extended successfully
SELECT
  id,
  name,
  freeTrialExpiredAt,
  freeTrialExtendedAt,
  DATEDIFF(freeTrialExpiredAt, NOW()) as daysRemaining
FROM Company
WHERE id = 'COMPANY_ID';
```

## Alternative: Using GraphQL (Admin mutation)

Instead of direct database access, you can use the admin GraphQL mutation:

```graphql
mutation ExtendTrial {
  updateCompany(input: {
    id: "COMPANY_ID"
    freeTrialExpiredAt: "2026-03-15T23:59:59.000Z"
  }) {
    id
    name
    freeTrialExpiredAt
  }
}
```

**Benefits of using GraphQL:**
- Automatically syncs with Stripe if the company has a subscription
- Includes audit logging
- Validates the new date (must be after or equal to current expiration)

**When to use direct database access:**
- Bulk operations (extending multiple companies at once)
- Trial has already expired and needs immediate restoration
- GraphQL API is unavailable

## Important notes

### 1. Two trial systems exist in parallel
- **Free trial** on `Company.freeTrialExpiredAt` - For users without paid subscriptions
- **Stripe trial** on `CompanySubscriptionPlan.trialEnd` - For users with Stripe subscriptions

If a company has a Stripe subscription, you may need to update both:

```sql
-- Check if company has a Stripe subscription
SELECT
  c.id,
  c.name,
  c.freeTrialExpiredAt,
  csp.trialEnd,
  csp.status
FROM Company c
LEFT JOIN CompanySubscriptionPlan csp ON csp.company = c.id
WHERE c.id = 'COMPANY_ID';

-- Update both trial periods
UPDATE Company
SET freeTrialExpiredAt = DATE_ADD(NOW(), INTERVAL 30 DAY)
WHERE id = 'COMPANY_ID';

UPDATE CompanySubscriptionPlan
SET trialEnd = DATE_ADD(NOW(), INTERVAL 30 DAY)
WHERE company = 'COMPANY_ID';
```

### 2. User-facing extension feature

Regular users can extend their own trial through the Blue UI:
- Adds **7 days** to the trial
- Can only be used once every **3 months** (rate limited)
- Only company **OWNER** can extend
- Located in: Company Settings → Billing

### 3. No automatic expiration
- The system doesn't actively expire trials with a cron job
- Trial status is evaluated at query time by comparing `freeTrialExpiredAt` to the current time
- Companies with expired trials are marked as "locked" and cannot access features

### 4. Audit tracking
Consider updating `freeTrialExtendedById` for better audit trails:

```sql
-- Find your own user ID
SELECT id, email FROM User WHERE email = 'support@blue.com';

-- Update with audit info
UPDATE Company
SET
  freeTrialExpiredAt = DATE_ADD(NOW(), INTERVAL 30 DAY),
  freeTrialExtendedAt = NOW(),
  freeTrialExtendedById = 'YOUR_USER_ID'
WHERE id = 'COMPANY_ID';
```

---

## Customer reply templates

### When extending a trial:

```
Hi [Name],

I've extended your free trial for [Company Name] by [X] days. Your new trial expiration date is [Date].

You should now have full access to all Blue features until then. If you have any questions about your trial or would like to discuss subscription options, feel free to reach out.

Best regards,
Team Blue
```

### When a customer requests an extension:

```
Hi [Name],

Thank you for reaching out! I'd be happy to extend your free trial so you have more time to evaluate Blue.

I've added [X] days to your trial period. Your account will now remain active until [Date].

If you need any help getting the most out of Blue during your trial, or if you'd like to discuss which plan works best for your team, just let me know.

Best regards,
Team Blue
```

### When explaining the standard extension:

```
Hi [Name],

I see you're looking to extend your trial period. Great news – you can actually do this yourself right from your Blue account!

Go to Company Settings → Billing, and you'll see an option to extend your trial by 7 days. This feature is available once every 3 months.

If you need a longer extension or have already used this feature recently, let me know and I can help extend it manually.

Best regards,
Team Blue
```

### For bulk extension (multiple companies):

```
Hi [Name],

I've extended the free trial for all three of your companies:
- [Company A] - extended until [Date]
- [Company B] - extended until [Date]
- [Company C] - extended until [Date]

All accounts should now have full access. Let me know if you have any questions.

Best regards,
Team Blue
```
