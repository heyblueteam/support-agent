# How do I debug HTTP request automations?

When a customer reports HTTP request automations not working, follow these steps:

## 1. Find the customer's company

```sql
SELECT id, slug, name FROM Company WHERE slug = 'company-slug';
```

## 2. Find their HTTP request automations

```sql
SELECT
  aa.id as action_id,
  p.name as project_name,
  ho.url,
  ho.method,
  a.isActive
FROM AutomationAction aa
JOIN Automation a ON aa.automation = a.id
JOIN Project p ON a.project = p.id
JOIN AutomationActionHttpOption ho ON ho.automationActionId = aa.id
WHERE p.company = 'COMPANY_ID'
AND aa.type = 'MAKE_HTTP_REQUEST';
```

## 3. Check HTTP logs for those automations

```sql
SELECT
  l.url,
  l.statusCode,
  l.statusText,
  l.createdAt
FROM AutomationActionHttpLog l
JOIN AutomationActionHttpOption ho ON l.automationActionHttpOption = ho.id
WHERE ho.automationActionId IN ('ACTION_ID_1', 'ACTION_ID_2')
ORDER BY l.createdAt DESC
LIMIT 20;
```

## 4. Common findings

- **200 OK responses**: Blue is sending correctly. Issue is on the receiving end.
- **Google Apps Script returning HTML**: This is normal. Look for "Data logged successfully" in the response.
- **No recent logs**: The automation hasn't been triggered, not that it's failing.
- **4xx/5xx errors**: Check the endpoint URL, authentication, or payload format.

**Note:** Never share specific code lines or database details with customers. Summarize findings in plain language.
