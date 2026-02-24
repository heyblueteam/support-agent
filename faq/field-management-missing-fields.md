# Field Management — Missing Fields or Folders

Customers report that their field management interface doesn't show all fields (and/or folders), but the fields still appear on the record content popup.

## What's happening

Each project has a `todoFields` JSON column on the `Project` table. This JSON array controls what appears in the **field management UI**. The record content popup has a fallback that displays all custom fields attached to a project regardless of this config, which is why fields can appear in the popup but not in field management.

When `todoFields` loses its `CUSTOM_FIELD` entries — but the `CustomField` rows still exist in the database — only the built-in fields are visible in the field management interface.

### Known root causes

**1. Project copy bug (most common)**
When a project is duplicated, `copy-project-service.ts` remaps old custom field IDs to new ones. If a field ID isn't found in the remap map, it's silently dropped with no error. The copied project ends up with only built-in fields in `todoFields`.

**2. Frontend state bug**
The field manager modal sends the *entire* `todoFields` array on every save. If the component's React/Vue state was out of sync at save time (e.g., a race condition, partial load), it overwrites the DB with whatever was in state — which may not include all fields.

In both cases the `CustomField` rows are untouched; only the JSON config is corrupted.

---

## Diagnosis

### 1. Find the company and project

```sql
-- Find user and company by email
SELECT u.email, c.name AS company_name, c.id AS company_id
FROM User u
JOIN CompanyUser cu ON cu.user = u.id
JOIN Company c ON c.id = cu.company
WHERE u.email LIKE '%customerdomain%';

-- Find the project by name
SELECT id, name, todoFields
FROM Project
WHERE company = 'COMPANY_ID'
AND name LIKE '%project name%';
```

### 2. Inspect the todoFields JSON

A broken `todoFields` looks like this — only built-in types, no `CUSTOM_FIELD` entries:

```json
[
  {"type": "TAG", "customFieldId": null},
  {"type": "ASSIGNEE", "customFieldId": null},
  {"type": "DUE_DATE", "customFieldId": null},
  {"type": "DESCRIPTION", "customFieldId": null},
  {"type": "TIME_TRACKING", "customFieldId": null},
  {"type": "CHECKLIST", "customFieldId": null}
]
```

### 3. Check what custom fields actually exist for the project

```sql
SELECT id, name, type, position
FROM CustomField
WHERE project = 'PROJECT_ID'
ORDER BY position;
```

If this returns rows but they're absent from `todoFields`, the config is corrupted and needs to be fixed.

---

## Fix

Rebuild `todoFields` by combining the built-in fields with all custom fields ordered by `position`.

**Step 1 — Generate the new JSON**

Take the IDs from the query above and build the array. Built-in fields always go first, then `CUSTOM_FIELD` entries in position order:

```json
[
  {"type": "TAG", "customFieldId": null},
  {"type": "ASSIGNEE", "customFieldId": null},
  {"type": "DUE_DATE", "customFieldId": null},
  {"type": "DESCRIPTION", "customFieldId": null},
  {"type": "TIME_TRACKING", "customFieldId": null},
  {"type": "CHECKLIST", "customFieldId": null},
  {"type": "CUSTOM_FIELD", "customFieldId": "FIELD_ID_1"},
  {"type": "CUSTOM_FIELD", "customFieldId": "FIELD_ID_2"},
  ...
]
```

**Step 2 — Apply the fix**

```sql
UPDATE Project
SET todoFields = '[
  {"type":"TAG","customFieldId":null},
  {"type":"ASSIGNEE","customFieldId":null},
  {"type":"DUE_DATE","customFieldId":null},
  {"type":"DESCRIPTION","customFieldId":null},
  {"type":"TIME_TRACKING","customFieldId":null},
  {"type":"CHECKLIST","customFieldId":null},
  {"type":"CUSTOM_FIELD","customFieldId":"FIELD_ID_1"},
  {"type":"CUSTOM_FIELD","customFieldId":"FIELD_ID_2"}
]'
WHERE id = 'PROJECT_ID';
```

**Step 3 — Verify**

```sql
SELECT todoFields FROM Project WHERE id = 'PROJECT_ID';
```

Confirm the JSON contains all expected field IDs. The customer can refresh and will see all fields immediately — no restart needed.

> **Note on folders:** If the customer had fields organised into folders (CUSTOM_FIELD_GROUP), that structure is lost and cannot be automatically recovered. They'll need to re-create their folder groupings manually in the field management UI after the fix. Let them know in your reply.

---

## Important caveats

- The fix restores fields as a **flat list** — any prior folder structure is lost
- `CustomField` rows are never affected by this bug; only the JSON config is broken
- If the project was copied and the *source* project also has broken `todoFields`, check and fix that one too
- If a field appears in `todoFields` but not in `CustomField`, remove it from the JSON — it's a ghost reference and will cause errors in the field manager

---

## Customer reply template

```
Hi [Name],

I've looked into this and found the issue — the field configuration for your "[Project Name]" project had become out of sync, which was causing the field management interface to only show a partial list of fields.

I've corrected the configuration and all your fields should now be visible in field management. Please give the page a refresh to pick up the changes.

One thing to be aware of: if you had your fields organised into folders/groups, you may need to recreate that grouping in the field management interface — the folder structure couldn't be recovered automatically, though all the fields themselves are intact.

Let me know if anything still looks off.

Best regards,
Team Blue
```
