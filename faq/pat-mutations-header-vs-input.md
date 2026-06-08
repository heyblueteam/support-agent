# Why does my Personal Access Token return FORBIDDEN / PROJECT_NOT_FOUND on a mutation?

Customers using Personal Access Tokens (PATs) regularly hit `FORBIDDEN` ("You are not authorized.") or `PROJECT_NOT_FOUND` on mutations that look correct. In every case so far the token is fine — the request shape is wrong. The two recurring traps:

1. **The auth context comes from request headers, not from the GraphQL input.** Many resolvers look up the project/company from `X-Bloo-Project-ID` / `X-Bloo-Company-ID` headers before the query body is even evaluated.
2. **For `inviteUser`, putting `companyId` in the input silently switches to a company-wide flow** that has different permission requirements than the project flow.

## How PAT auth context works

A request authenticated by a PAT (`X-Bloo-Token-ID` + `X-Bloo-Token-Secret` headers) runs as the user who created the token, with the same permissions that user has. The token is not separately scoped.

But for many mutations, the *project* and *company* the request operates on come from headers — not from arguments inside the GraphQL input:

- `X-Bloo-Company-ID` — the company context (CUID, not slug)
- `X-Bloo-Project-ID` — the project context (CUID or slug)

If a resolver needs project context and `X-Bloo-Project-ID` isn't set, it throws `PROJECT_NOT_FOUND` or `FORBIDDEN` even when the input has a `projectId` field. Always set both headers when calling project-scoped mutations.

## Known mutations that need the headers

- `createTodo` — needs `X-Bloo-Project-ID` and `X-Bloo-Company-ID`. Passing `projectId` in the input does not satisfy the auth check.
- `forms(filter: { projectId })`, `form(id:)` — need `X-Bloo-Project-ID`. The filter argument is for the query, not authorization.
- Anything that resolves a project from headers and not from a parent id (e.g. via `todoId`).

Rule of thumb: if a mutation works for queries that look up by id (`todoList(id:)`, `project(id:)`) but fails for create-style mutations, you're missing the headers.

## The `inviteUser` companyId trap

`inviteUser` has two flows hidden behind one mutation, and the input field you pass decides which flow runs:

| Input fields | Flow | Permission required |
|---|---|---|
| `projectId` only | Project invite | Project role above VIEW_ONLY / COMMENT_ONLY |
| `companyId` (with or without `projectId`) | Company-wide invite | Caller must be company **OWNER** |

If `companyId` is in the input, the resolver branches to the company flow **and ignores `projectId` entirely** (it only reads `projectIds: [String!]`). Customers who want to invite someone to a single project but pass both `projectId` and `companyId` get `FORBIDDEN` if they aren't a company OWNER.

**Fix:** for a project invite, drop `companyId` from the input. Pass only `projectId`. Keep the headers as-is.

## Permission nuance on `accessLevel`

Even with the correct shape, the inviter's role limits which `accessLevel` they can assign:

- `OWNER` — can invite at any level
- `ADMIN` — can invite up to MEMBER (not OWNER)
- `MEMBER` — can invite only at CLIENT
- `CLIENT` — can invite only at CLIENT
- `VIEW_ONLY` / `COMMENT_ONLY` — cannot invite at all

So a project MEMBER trying to invite at `accessLevel: MEMBER` will get `FORBIDDEN` even with the right input shape. They need to invite as `CLIENT`, or have an ADMIN/OWNER run the invite.

---

## Customer reply templates

### `inviteUser` returning FORBIDDEN with `companyId` in input

```
Hi [Name],

The FORBIDDEN is from the shape of the input, not the token, headers, or a plan restriction. Passing companyId in the input switches the mutation to a company-wide invite flow that ignores the projectId you also pass and requires you to be a company-level OWNER. That's why it fails even with the right headers.

For a project invite, drop companyId from the input and pass just projectId. That uses the project invite flow, which only requires you have invite permission in that specific project.

mutation InviteUserToProject($email: String!, $projectId: String!, $accessLevel: UserAccessLevel!) {
  inviteUser(input: { email: $email, projectId: $projectId, accessLevel: $accessLevel })
}

Keep all your existing headers — that part is right.

One nuance on accessLevel worth knowing: a project MEMBER can only invite at CLIENT level; an ADMIN can invite up to MEMBER; an OWNER can invite anyone. If you still hit FORBIDDEN after removing companyId, check your role in that project — if you're a MEMBER there, invite as CLIENT instead, or have an OWNER/ADMIN run the invite.

Best regards,
Manny
Founder of Blue
```

### Mutation returning PROJECT_NOT_FOUND despite a valid project id in the input

```
Hi [Name],

The project id in the input isn't used for the authorization check — that lookup runs first, against request headers. When the header isn't set, the API has no project context and throws PROJECT_NOT_FOUND before the input is even evaluated.

Add these headers to the request (using the same project id you're passing in the input):

X-Bloo-Project-ID: <your project id or slug>
X-Bloo-Company-ID: <your company id>

The other mutations you've used successfully resolve the project through a related id (e.g. a todo id), which is why they didn't need the header. Create-style mutations don't have that, so the headers are required.

Best regards,
Manny
Founder of Blue
```
