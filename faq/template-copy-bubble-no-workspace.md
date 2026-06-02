# "Created a workspace from a template, the bubble closed, but no workspace appeared"

A recurring report: the customer creates a workspace (project) from a template,
sees the "processing"/"creating workspace" bubble in the bottom-right, the bubble
closes — and no new workspace shows up. Sometimes phrased as "it's been
processing for ages" or "I think there are duplicates pending."

## What's actually happening

Creating a workspace from a template runs as a **background copy job** inside a
single database transaction. The new project is created at the *start* of that
transaction and everything (lists, records, fields, files) is copied into it.

- If the job **throws anywhere**, the whole transaction rolls back — the project
  and all its contents disappear. Nothing partial is left behind, so there are
  **no orphaned or "pending" duplicates** (you can reassure the customer of this).
- The only progress signal is a Redis status key the frontend polls. On failure
  that key is deleted, so the bubble just **vanishes with no error toast** — which
  is why it looks like nothing happened.

Most failures are **transient** (a hiccup in the copy job, queue contention, a
momentarily-missing file). The copy usually succeeds on a retry shortly after —
which is why the workspace often already exists by the time you investigate.

> A fix shipped in PR #385 (`onCopyProjectFailed` subscription) so a failed copy
> now shows the user an error + retry instead of silently vanishing. Before that
> deploy, the failure was completely invisible to the customer.

## How to diagnose

Find the company by the customer's email (see the `investigate-customer` skill),
then look for the workspace — **it has often already been created** (sometimes
under a slightly different name than the customer remembers). Include templates
and archived projects:

```sql
-- All projects for the company, newest first — look for the target name
SELECT id, name, slug, isTemplate, archived, createdAt,
  (SELECT COUNT(*) FROM TodoList tl WHERE tl.project = p.id) AS lists,
  (SELECT COUNT(*) FROM File f WHERE f.project = p.id) AS files
FROM Project p
WHERE p.company = 'COMPANY_ID'
  AND (p.name LIKE '%SEARCH_TERM%' OR p.createdAt >= NOW() - INTERVAL 7 DAY)
ORDER BY p.createdAt DESC;
```

A healthy result has non-zero `lists`/`files` — the copy completed. Then confirm
the customer can actually see it (membership), since a MEMBER only sees projects
they belong to:

```sql
SELECT u.firstName, u.lastName, u.email
FROM ProjectUser pu JOIN User u ON pu.user = u.id
WHERE pu.project = 'PROJECT_ID';
```

If the customer is **not** in `ProjectUser` for a project that clearly exists,
that's the "I can't see it" case — they were left off the new project's member
list. (Owners/admins see everything; members don't.)

If no workspace exists at all, the copy failed and rolled back cleanly — tell
them to simply retry (and, once PR #385 is live, they'll get a clear error if it
fails again).

## Customer reply template

```
Hi [Name],

Good news — your "[Workspace Name]" workspace is there now, fully built from the
template, and you[ and your team] have access. There are no duplicate or
"pending" copies sitting in the system — just the one clean workspace.

What happened: the earlier attempt hit a temporary hiccup in our template-copy
process and closed without showing you an error. That's on us, and we're
improving it so a failure surfaces clearly with a retry instead of leaving you
guessing.

Nothing you need to do on your end — but if you ever see that bubble close with
no workspace again, a simple retry will create it.

Best regards,
Manny
Founder of Blue
```
