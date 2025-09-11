# Support Agent - Gmail Integration Tools

A collection of command-line tools for Gmail integration designed for use by Claude Code and other AI agents. Provides clean, structured access to Gmail operations without preprocessing or analysis.

## Features

- **Read Operations**: List messages, read threads, get message details, search
- **Write Operations**: Send replies, archive messages, manage labels
- **OAuth2 Authentication**: Secure Gmail API access with token management
- **Multiple Output Formats**: Simple, detailed, and JSON outputs for different use cases
- **Agent-Friendly**: Structured JSON output perfect for AI agent consumption

## Installation

1. Clone or navigate to the support-agent directory:
```bash
cd /Users/manny/Blue/support-agent
```

2. Build the tool:
```bash
go build -o support-agent
```

3. Set up Gmail API credentials (see detailed Authentication Setup below)

4. Create `.env` file (optional):
```bash
cp .env.example .env
# Edit .env with your settings
```

## Authentication Setup

### Step 1: Create Google Cloud Project and Enable Gmail API

1. **Go to Google Cloud Console**: https://console.cloud.google.com

2. **Create or Select a Project**:
   - Click the project dropdown at the top
   - Either select an existing project or click "New Project"
   - Name it something like "support-agent" or "gmail-tools"

3. **Enable Gmail API**:
   - In the left sidebar, go to "APIs & Services" > "Library"
   - Search for "Gmail API"
   - Click on it and press "Enable"

### Step 2: Create OAuth 2.0 Credentials

1. **Go to Credentials**:
   - Navigate to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth Client ID"

2. **Configure OAuth Consent Screen** (if prompted):
   - Choose "External" (or "Internal" if using Google Workspace)
   - Fill in required fields:
     - App name: "Support Agent" or similar
     - User support email: Your email
     - Developer contact: Your email
   - Add scopes: `https://www.googleapis.com/auth/gmail.modify`
   - Add your email to test users

3. **Create OAuth Client**:
   - Application Type: **Desktop app** (important!)
   - Name: "support-agent" or similar
   - Click "Create"

4. **Download Credentials**:
   - Click the download button (⬇️) next to your OAuth 2.0 Client ID
   - Save as `gmail.json` in the support-agent directory

### Step 3: First-Time Authentication

1. **Run any command** to trigger authentication:
```bash
./support-agent read-messages --limit 1
```

2. **You'll see a URL** like:
```
Go to the following link in your browser:
https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=...
Enter authorization code:
```

3. **Copy and open the URL** in your browser

4. **Sign in and authorize** with the Gmail account you want to access

5. **Get the authorization code**:
   - After approving, you'll be redirected to `http://localhost`
   - You'll see "This site can't be reached" - **this is normal!**
   - Look at the URL in your browser's address bar
   - It will contain: `http://localhost/?code=4/0AcvDMrD...&scope=...`
   - Copy everything between `code=` and `&scope` (or end of URL)

6. **Paste the code** back in the terminal and press Enter

7. **Success!** The token is saved to `~/.support-agent/token.json` and will auto-refresh

### Authentication Troubleshooting

**"invalid_grant" error**:
- The authorization code expired (they're only valid for a few minutes)
- Solution: Run the command again and use the new auth URL immediately

**"This site can't be reached" page**:
- This is expected! The authorization code is in the URL
- Look at the address bar for `code=` parameter

**Token expired errors**:
```bash
# Delete the old token and re-authenticate
rm ~/.support-agent/token.json
./support-agent read-messages --limit 1
```

**"redirect_uri_mismatch" error**:
- Make sure you selected "Desktop app" when creating OAuth credentials
- Verify you're using the correct `gmail.json` file

## Usage

### Basic Command Structure
```bash
./support-agent <command> [options]
```

### Read Messages
List messages from inbox with various filters:
```bash
# List unread messages
./support-agent read-messages --unread --limit 10

# Filter by sender
./support-agent read-messages --from customer@example.com

# Get JSON output for agents
./support-agent read-messages --output json --limit 5
```

### Read Threads
Get complete conversation threads:
```bash
# Get thread in simple format
./support-agent read-threads --thread-id THREAD_ID

# Get detailed thread with full message bodies
./support-agent read-threads --thread-id THREAD_ID --output detailed

# Get JSON for agent processing
./support-agent read-threads --thread-id THREAD_ID --output json
```

### Read Message Details
Get complete message with body and attachments:
```bash
./support-agent read-message-detail --message-id MESSAGE_ID
./support-agent read-message-detail --message-id MESSAGE_ID --output json
```

### Search Messages
Use Gmail's powerful search syntax:
```bash
# Search by sender and date
./support-agent search-messages --query "from:user@example.com after:2024/1/1"

# Search for attachments
./support-agent search-messages --query "has:attachment"

# Complex queries
./support-agent search-messages --query "subject:invoice OR subject:receipt is:unread"
```

### Reply to Messages
Send replies maintaining thread context:
```bash
./support-agent reply-message \
  --message-id MESSAGE_ID \
  --body "Thank you for your email. We'll look into this issue."
```

### Archive Messages
Remove messages from inbox:
```bash
# Archive single message
./support-agent archive-message --message-id MESSAGE_ID

# Archive entire thread
./support-agent archive-message --thread-id THREAD_ID
```

### Manage Labels
Add or remove labels:
```bash
# Add label to message
./support-agent label-message --message-id MESSAGE_ID --add-label IMPORTANT

# Remove label from thread
./support-agent label-message --thread-id THREAD_ID --remove-label INBOX

# Multiple labels
./support-agent label-message --message-id MESSAGE_ID --add-label "IMPORTANT,STARRED"
```

## Output Formats

### Simple (default)
Basic information for quick viewing:
```
MSG_ID | sender@example.com | Email Subject | 2024-01-09 10:30:00
```

### Detailed
Full message content with all headers and body:
```
=== Message Details ===
ID: MESSAGE_ID
From: sender@example.com
Subject: Email Subject
Date: 2024-01-09 10:30:00
Body: Full message content...
```

### JSON
Structured data for agent processing:
```json
{
  "id": "MESSAGE_ID",
  "thread_id": "THREAD_ID",
  "from": "sender@example.com",
  "to": "recipient@example.com",
  "subject": "Email Subject",
  "date": "2024-01-09T10:30:00Z",
  "body": "Full message content...",
  "labels": ["INBOX", "UNREAD"]
}
```

## Integration with Claude Code / AI Agents

This tool is designed for easy integration with AI agents:

1. **Structured Output**: Use `--output json` for parseable data
2. **No Preprocessing**: Raw Gmail data without sentiment analysis or categorization
3. **Clean Interface**: Consistent command structure across all tools
4. **Error Handling**: Clear error messages for debugging

### Example Agent Usage

```bash
# Agent checks for new support emails
messages=$(./support-agent read-messages --unread --output json)

# Agent reads full thread for context
thread=$(./support-agent read-threads --thread-id THREAD_ID --output json)

# Agent sends reply
./support-agent reply-message --message-id MSG_ID --body "AI-generated response"

# Agent archives handled thread
./support-agent archive-message --thread-id THREAD_ID
```

## Gmail Search Query Examples

The search-messages command supports Gmail's full query syntax:

- `from:sender@example.com` - Messages from specific sender
- `to:me` - Messages sent directly to you
- `subject:"important update"` - Exact phrase in subject
- `has:attachment` - Messages with attachments
- `is:unread` - Unread messages
- `label:important` - Messages with specific label
- `after:2024/1/1 before:2024/2/1` - Date range
- `larger:10M` - Messages larger than 10MB
- `"exact phrase"` - Exact phrase anywhere
- `OR` operator - `from:alice OR from:bob`
- `-` operator - `from:example.com -from:noreply`

## Environment Variables

Optional configuration via `.env` file or environment:

- `GMAIL_CREDENTIALS_PATH`: Path to gmail.json (default: `./gmail.json`)
- `GMAIL_CREDENTIALS`: Raw JSON credentials (overrides file)
- `TOKEN_DIR`: Directory for token storage (default: `~/.support-agent`)
- `USER_EMAIL`: Default user email for operations

## Security

- OAuth2 tokens stored with 0600 permissions
- Automatic token refresh
- Credentials never logged or exposed
- Uses Gmail API's modify scope (not full access)

## Troubleshooting

### Authentication Issues
- Ensure gmail.json has valid OAuth2 credentials
- Check token permissions: `ls -la ~/.support-agent/token.json`
- Delete token to re-authenticate: `rm ~/.support-agent/token.json`

### API Errors
- Verify Gmail API is enabled in Google Cloud Console
- Check OAuth2 consent screen is configured
- Ensure credentials have correct scopes

### Build Issues
- Requires Go 1.22 or later
- Run `go mod download` to fetch dependencies
- Check for conflicting package versions

## Development

### Adding New Tools

1. Create new tool in `tools/` directory
2. Implement `Run<ToolName>(args []string) error` function
3. Add command routing in `main.go`
4. Update README documentation

### Testing

```bash
# Build and test
go build -o support-agent
./support-agent help

# Test authentication
./support-agent read-messages --limit 1

# Test with different outputs
./support-agent read-messages --output json
./support-agent read-messages --output detailed
```

## License

Internal Blue tool - not for public distribution.