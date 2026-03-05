# How do I transfer organization ownership?

When a customer wants to transfer ownership of their Blue organization to another user (e.g., to a new billing contact or executive), this must be done manually via the database.

## How ownership works in Blue

Ownership is stored on the `CompanyUser` table via the `level` enum field:

```
enum level: OWNER | ADMIN | MEMBER | CLIENT | COMMENT_ONLY | VIEW_ONLY
```

- Only one user should hold the `OWNER` level per organization at a time
- The OWNER has full access including billing and subscription management
- The previous owner should be demoted to `ADMIN` (not `MEMBER`) to retain elevated access unless the customer specifies otherwise

## 1. Verify both users belong to the organization

```sql
SELECT c.id as company_id, c.name, cu.id as cu_id, cu.level, u.email, u.id as user_id
FROM Company c
JOIN CompanyUser cu ON cu.company = c.id
JOIN User u ON u.id = cu.user
WHERE c.name LIKE '%OrgName%'
   OR u.email IN ('current-owner@example.com', 'new-owner@example.com')
ORDER BY c.name, cu.level;
```

Confirm:
- Both users are members of the same company
- The current owner has `OWNER` level
- The new owner is already a member (if not, they need to be invited first)

## 2. Transfer ownership

Run both updates together to avoid leaving the org without an owner:

```sql
-- Promote new owner
UPDATE CompanyUser
SET level = 'OWNER', updatedAt = NOW()
WHERE id = 'NEW_OWNER_COMPANYUSER_ID';

-- Demote current owner to ADMIN
UPDATE CompanyUser
SET level = 'ADMIN', updatedAt = NOW()
WHERE id = 'CURRENT_OWNER_COMPANYUSER_ID';
```

## 3. Verify the change

```sql
SELECT cu.level, u.email
FROM CompanyUser cu
JOIN User u ON u.id = cu.user
WHERE cu.id IN ('NEW_OWNER_COMPANYUSER_ID', 'CURRENT_OWNER_COMPANYUSER_ID');
```

## Important notes

- **The new owner must already be a member** of the organization before the transfer. If they aren't, they need to accept an invitation first.
- **Demote to ADMIN, not MEMBER** — unless the customer explicitly requests the previous owner be fully downgraded. ADMIN retains the ability to manage projects and members.
- **Only one OWNER per org** — having two OWNERs is not technically blocked by the DB but is not supported by the app and can cause unexpected behavior.
- **Billing access** — the OWNER role is required to manage billing and subscriptions in Blue. Once transferred, the new owner can immediately upgrade or manage the subscription.

## Customer reply template

```
Hi [Name],

I've transferred ownership of the [Organization Name] workspace to [New Owner Email].

[New Owner] now has full owner access, including billing and subscription management. [Previous Owner Email] has been updated to Admin.

Let me know if you need anything else — happy to help with the upgrade when you're ready.

Best regards,
Manny
Founder of Blue
```
