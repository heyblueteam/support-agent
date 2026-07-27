# "Cannot GET /authorize" when connecting Blue's MCP server to Claude/ChatGPT

A customer trying to connect Blue's MCP server reports an error, and the screenshot/URL shows
something like:

```
mcp.blue.app/authorize?response_type=code&client_id=...&redirect_uri=https%3A%2F%2F...
Cannot GET /authorize
```

This is **not a bug and not a token problem.** The customer is adding Blue through the wrong UI.

## Root cause

The in-app **"Add custom connector"** screen in Claude (Desktop and web) and ChatGPT only speaks
**OAuth 2.1** — it performs a login-style handshake and has *no field for a static API token,
bearer token, or custom header*. It tries to discover/redirect to the server's `/authorize`
endpoint.

Blue's MCP server uses **Personal Access Token (PAT) auth via custom headers**, not OAuth. It
exposes only `POST/GET/DELETE /mcp` — there is **no `/authorize`, no `/token`, no
`/.well-known/oauth-*` discovery, and no Dynamic Client Registration**. So the connector's OAuth
handshake hits `mcp.blue.app/authorize`, which doesn't exist, and the server returns
`Cannot GET /authorize`.

(OAuth support for the MCP server is a parked plan — `plans/mcp-oauth.md`, 2026-06-25 — not yet
built. Until it ships, the custom-connector screen cannot work with Blue.)

## The fix: use the config-file method, not the connector screen

Blue connects via the documented `mcp-remote` config block (Claude Desktop) or `claude mcp add -t http`
(Claude Code), passing three headers:

| Header              | Value                                   |
| ------------------- | --------------------------------------- |
| `blue-token-id`     | Personal Access Token ID                |
| `blue-token-secret` | Personal Access Token secret            |
| `blue-org-id`       | Organisation slug or ID                 |

(Legacy names `x-bloo-token-id` / `x-bloo-token-secret` / `x-bloo-company-id` still work but are
deprecated.)

Token is created in Blue at **Account → API → Create Token** — the customer needs **both** the
Token ID and the Token Secret (the secret is shown only once). The `blue-org-id` is their org slug
(the name in their Blue web address, `blue.app/<org-slug>/...`).

Full published guide: <https://blue.app/docs/integrations/mcp>
(source: `app/src/content/docs/integrations/mcp.md`).

## Client support, at a glance

- **Claude Code, Cursor, Windsurf** — connect directly over Streamable HTTP (header-based config).
- **Claude Desktop** — only speaks local stdio, so it needs the `mcp-remote` bridge (the JSON config
  block below). Requires **Node.js 18+** (Node 20+ recommended); a startup `SyntaxError` almost
  always means an outdated Node on `npx`.
- **Claude web / ChatGPT custom-connector UI** — **not supported** (needs OAuth, which Blue's MCP
  server doesn't implement yet).

---

## Customer reply template

```
Hi [Name],

The screen you're using ("Add custom connector") won't work with Blue. That option expects a sign-in/login style connection, but Blue connects with an API token instead — that's why you're seeing the "Cannot GET /authorize" error. Claude is looking for a login page that Blue doesn't use. Skip that screen entirely.

The working way to connect Blue to Claude Desktop is through Claude's config file:

1. Get your token from Blue
   - In Blue: Account -> API -> Create Token
   - Copy both the Token ID and the Token Secret (you need both — the secret is only shown once)
   - You'll also need your organization slug — it's the name in your Blue web address, e.g. blue.app/your-org-name/...

2. Open Claude Desktop's config file
   - Mac: ~/Library/Application Support/Claude/claude_desktop_config.json
   - Windows: %APPDATA%\Claude\claude_desktop_config.json
   - (In Claude Desktop you can also open it via Settings -> Developer -> Edit Config. If the file is empty, paste the whole block below.)

3. Paste this in, replacing the three placeholders with your values:

{
  "mcpServers": {
    "blue": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://mcp.blue.app/mcp",
        "--header",
        "blue-token-id:YOUR_TOKEN_ID",
        "--header",
        "blue-token-secret:YOUR_TOKEN_SECRET",
        "--header",
        "blue-org-id:YOUR_ORG_SLUG"
      ]
    }
  }
}

4. Save the file and fully quit and reopen Claude Desktop.

One requirement: this needs Node.js 18 or newer installed on your computer (Node 20+ recommended). If Claude shows a startup error, that's almost always an outdated Node version.

Full guide (same steps, plus Claude Code, Cursor, and Windsurf): https://blue.app/docs/integrations/mcp

Best regards,
Manny
Founder of Blue
```
