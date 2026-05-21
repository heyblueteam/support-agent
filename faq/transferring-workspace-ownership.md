# How do I transfer ownership of a workspace (or several)?

When a customer wants to transfer ownership of one or more workspaces (e.g., an employee is leaving and their workspaces need to move to someone else), this is a manual DB operation. There is no in-app self-serve flow today.

This is the workspace-level analogue of [transferring organization ownership](transferring-organization-ownership.md). The mechanism is similar but on a different table.

## How workspace ownership works in Blue

A workspace is a `Project` row. There is no `Project.owner` column. Ownership is a role on `ProjectUser`:

```
enum level: OWNER | ADMIN | MEMBER | CLIENT | COMMENT_ONLY | VIEW_ONLY
```

- One `OWNER` per workspace (not enforced in DB, but anything else is unsupported)
- The previous owner should be demoted to `ADMIN` unless the customer asks otherwise
- A workspace can legitimately have no OWNER (e.g., the original creator left and was removed org-wide — their `ProjectUser` row was deleted along with their `CompanyUser` row)

## 1. Find the workspaces and confirm membership

Get the company first using the customer's email (see `investigate-customer` skill). Then for each workspace name the customer listed:

```sql
SELECT p.id as project_id, p.name, p.slug, p.isTemplate, p.archived
FROM Project p
WHERE p.company = '<companyId>'
  AND (
    p.name LIKE '%Workspace A%' OR
    p.name LIKE '%Workspace B%' OR
    p.name LIKE '%Template Foo%'
  )
ORDER BY p.isTemplate, p.name;
```

Then confirm the current OWNER and whether the new owner is already a member:

```sql
SELECT p.name as workspace, p.isTemplate, pu.id as pu_id, pu.level, u.email
FROM Project p
JOIN ProjectUser pu ON pu.project = p.id
JOIN User u ON u.id = pu.user
WHERE p.id IN ('<projectId1>','<projectId2>',...)
  AND (u.id IN ('<oldOwnerUserId>','<newOwnerUserId>') OR pu.level = 'OWNER')
ORDER BY p.name, pu.level, u.email;
```

For each workspace, you want to know:
- Who is the current OWNER (if any)
- Whether the new owner already has a `ProjectUser` row (and what level)

## 2. Apply the transfer

Three cases per workspace:

**Case A — new owner already exists on the workspace:**

```sql
-- Promote new owner
UPDATE ProjectUser SET level = 'OWNER', updatedAt = NOW()
WHERE id = '<newOwnerProjectUserId>';

-- Demote current owner to ADMIN (skip if there is no current OWNER)
UPDATE ProjectUser SET level = 'ADMIN', updatedAt = NOW()
WHERE id = '<currentOwnerProjectUserId>';
```

**Case B — new owner is not yet on the workspace:**

```sql
-- Add the new owner as OWNER (use cuid-like id; uid is the public-facing id)
INSERT INTO ProjectUser (id, uid, project, user, level, createdAt, updatedAt, allowNotification)
VALUES ('<newId>', '<newUid>', '<projectId>', '<newOwnerUserId>', 'OWNER', NOW(), NOW(), 1);

-- Demote current owner if present
UPDATE ProjectUser SET level = 'ADMIN', updatedAt = NOW()
WHERE id = '<currentOwnerProjectUserId>';
```

Prefer Case A — ask the customer to first invite the new owner to the workspace via the UI if they're not already a member. Avoids hand-rolling IDs.

**Bulk transfers** (e.g., several workspaces at once) — batch the updates with `IN (...)` lists for readability:

```sql
UPDATE ProjectUser SET level = 'OWNER', updatedAt = NOW()
WHERE id IN ('<pu1>','<pu2>','<pu3>');

UPDATE ProjectUser SET level = 'ADMIN', updatedAt = NOW()
WHERE id IN ('<oldOwner1>','<oldOwner2>','<oldOwner3>');
```

## 3. Verify on db1

After running the writes on db2, verify the post-state on the replica:

```sql
SELECT p.name as workspace, pu.level, u.email
FROM Project p
JOIN ProjectUser pu ON pu.project = p.id
JOIN User u ON u.id = pu.user
WHERE p.id IN ('<projectId1>','<projectId2>',...)
  AND u.id IN ('<oldOwnerUserId>','<newOwnerUserId>')
ORDER BY p.name, u.email;
```

## Important notes

- **Writes go to db2; reads go to db1** (per monorepo CLAUDE.md). The investigate-customer skill defaults to db1.
- **Demote to ADMIN by default.** If the customer's stated goal is to *remove* the previous owner entirely (e.g., they're a departing employee), that's a separate operation — usually done by removing them from the *organization* via the UI, which cascades to remove all their `ProjectUser` rows. You don't need to delete `ProjectUser` rows manually.
- **Templates are workspaces too.** `Project.isTemplate = 1` distinguishes them, but ownership works identically.
- **No app-level mutation exists.** Don't tell the customer "you can do this in Settings" — they can't. Frame the reply as a manual transfer you've already done.

## Customer reply template

```
Hi [Name],

All done. I've transferred ownership of these workspaces to [New Owner Email]:

  1. [Workspace A]
  2. [Workspace B]
  ...

Where [Previous Owner] was the previous owner, they've been demoted to Admin so they retain access until you remove their account. Once you remove them from the organization, their remaining access will be cleaned up automatically.

There's no in-app way to do this today — it's on our list to add a workspace ownership transfer option in the UI.

Let me know if there's anything else.

Best regards,
Manny
Founder of Blue
```
