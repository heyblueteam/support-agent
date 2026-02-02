# How do I restore a deleted record?

When customers accidentally delete a record (todo, file, discussion, document, or todo list), we can often restore it from the Trash table.

## How trash works in Blue

When records are deleted, Blue stores a complete JSON snapshot in the `Trash` table before removing them. This allows manual restoration.

**Records that can be restored:**
- Todos
- Files
- Discussions
- Documents
- Todo Lists

**Records that CANNOT be restored:**
- Comments (soft-deleted, content is cleared)
- Automations (permanently deleted)
- Folders (permanently deleted)

## 1. Find the customer's company

```sql
SELECT id, slug, name FROM Company WHERE slug = 'company-slug';
```

## 2. Find the deleted record in Trash

```sql
-- Search by record type and company (check the JSON data)
SELECT
  id,
  type,
  data,
  createdAt,
  userId
FROM Trash
WHERE type = 'Todo'  -- or 'File', 'Discussion', 'Document', 'TodoList'
AND JSON_EXTRACT(data, '$.company') = 'COMPANY_ID'
ORDER BY createdAt DESC
LIMIT 20;

-- Or search by record ID if known
SELECT * FROM Trash WHERE id = 'RECORD_ID';

-- Search by name/title in the JSON data
SELECT
  id,
  type,
  JSON_EXTRACT(data, '$.title') as title,
  createdAt
FROM Trash
WHERE type = 'Todo'
AND JSON_EXTRACT(data, '$.title') LIKE '%search term%'
ORDER BY createdAt DESC;
```

## 3. Restore the record

### Restoring a Todo

```sql
-- First, inspect the trash record
SELECT data FROM Trash WHERE id = 'TODO_ID';

-- Insert back into Todo table (adjust fields based on actual data)
INSERT INTO Todo (
  id, title, description, project, todoList, company,
  createdAt, updatedAt, createdBy, status, position
)
SELECT
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.id')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.title')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.description')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.project')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.todoList')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.company')),
  JSON_EXTRACT(data, '$.createdAt'),
  NOW(),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.createdBy')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.status')),
  JSON_EXTRACT(data, '$.position')
FROM Trash WHERE id = 'TODO_ID';

-- Remove from trash after successful restore
DELETE FROM Trash WHERE id = 'TODO_ID';
```

### Restoring a TodoList

TodoLists use a soft-delete pattern with `projectId = 'trash'`:

```sql
-- Find soft-deleted todo lists
SELECT id, name, projectId FROM TodoList
WHERE projectId = 'trash'
AND company = 'COMPANY_ID';

-- Restore by setting the correct project ID
UPDATE TodoList
SET projectId = 'ORIGINAL_PROJECT_ID'
WHERE id = 'TODOLIST_ID';

-- Remove the trash record
DELETE FROM Trash WHERE id = 'TODOLIST_ID';
```

### Restoring a File

```sql
-- Inspect the file data
SELECT data FROM Trash WHERE id = 'FILE_ID';

-- Insert back into File table
INSERT INTO File (
  id, name, url, size, mimeType, project, company,
  createdAt, updatedAt, createdBy
)
SELECT
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.id')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.name')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.url')),
  JSON_EXTRACT(data, '$.size'),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.mimeType')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.project')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.company')),
  JSON_EXTRACT(data, '$.createdAt'),
  NOW(),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.createdBy'))
FROM Trash WHERE id = 'FILE_ID';

-- Remove from trash
DELETE FROM Trash WHERE id = 'FILE_ID';
```

**Note:** Files may fail to restore if the actual file was deleted from S3 storage. The database record can be restored but the file itself may be gone.

### Restoring a Discussion

```sql
INSERT INTO Discussion (
  id, title, html, project, company,
  createdAt, updatedAt, createdBy
)
SELECT
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.id')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.title')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.html')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.project')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.company')),
  JSON_EXTRACT(data, '$.createdAt'),
  NOW(),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.createdBy'))
FROM Trash WHERE id = 'DISCUSSION_ID';

DELETE FROM Trash WHERE id = 'DISCUSSION_ID';
```

### Restoring a Document

```sql
INSERT INTO Document (
  id, title, html, project, company,
  createdAt, updatedAt, createdBy
)
SELECT
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.id')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.title')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.html')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.project')),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.company')),
  JSON_EXTRACT(data, '$.createdAt'),
  NOW(),
  JSON_UNQUOTE(JSON_EXTRACT(data, '$.createdBy'))
FROM Trash WHERE id = 'DOCUMENT_ID';

DELETE FROM Trash WHERE id = 'DOCUMENT_ID';
```

## 4. After restoring

- The record will appear back in Blue but won't trigger notifications
- Search index may take a few minutes to update
- Activity feed won't show the restoration
- Related items (comments, attachments) may not be restored

## Important caveats

1. **Time limit**: Trash records are stored indefinitely, but very old records may have stale references
2. **Related data**: Comments on todos, file attachments, etc. are NOT stored in trash and cannot be restored
3. **Foreign keys**: Restore will fail if the parent project or user has been deleted
4. **S3 files**: File database records can be restored, but if the S3 object was deleted, the file itself is gone

---

## Customer reply template

### If we CAN restore the record:

```
Hi [Name],

Good news! I was able to restore the deleted [record type] for you.

You should now see "[record name]" back in your [project name] project. Please note that any comments or attachments that were on the original record may not have been recovered.

Let me know if you have any trouble finding it or if anything looks off.

Best regards,
Team Blue
```

### If we CANNOT restore the record:

```
Hi [Name],

I looked into this for you, but unfortunately I wasn't able to restore the deleted [record type].

[Choose the appropriate reason:]
- The record was deleted more than [X] days ago and is no longer in our recovery system.
- The record type ([type]) is permanently deleted and cannot be recovered.
- The parent project has also been deleted, which prevents restoration.

I'm sorry I couldn't help more with this one. To prevent accidental deletions in the future, you might consider adjusting user permissions in your project settings to limit who can delete records.

Let me know if there's anything else I can help with.

Best regards,
Team Blue
```

### If we need more information:

```
Hi [Name],

I'd be happy to help restore the deleted record. To locate it in our system, could you provide a bit more detail?

- What type of record was it? (task, file, message board post, document, or task list)
- What was the name or title of the record?
- Which project was it in?
- Approximately when was it deleted?

Once I have these details, I'll look into recovering it for you.

Best regards,
Team Blue
```
