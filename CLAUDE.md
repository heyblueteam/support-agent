# Support Agent - Claude Code Instructions

## Important Guidelines

### Email Reply Protocol
**NEVER send replies automatically without first showing the draft to the user for approval.**

When helping with email responses:
1. First, analyze the thread/message to understand the context
2. Draft a proposed reply
3. Show the draft to the user for review
4. Only send after explicit user approval
5. Use the `reply-message` command only when the user confirms the draft

### Reading and Analyzing Emails
- Use `read-threads` to get full context before drafting replies
- Use `--output json` when processing for automated analysis
- Check message dates to understand response urgency
- Note if there are previous unanswered messages in the thread

### Tool Usage Best Practices

#### For Reading:
```bash
# Get overview of recent messages
./support-agent read-messages --limit 10

# Get full thread context
./support-agent read-threads --thread-id THREAD_ID --output detailed

# Search for specific issues
./support-agent search-messages --query "bug OR error OR problem"
```

#### For Writing (ONLY after user approval):
```bash
# Reply to a message
./support-agent reply-message --message-id MSG_ID --body "Approved reply text"

# Archive handled threads
./support-agent archive-message --thread-id THREAD_ID

# Mark important messages
./support-agent label-message --message-id MSG_ID --add-label IMPORTANT
```

### Workflow for Support Responses

1. **Identify the Issue**
   - Read the full thread for context
   - Check if customer has sent follow-ups
   - Note any screenshots or links provided

2. **Draft Response**
   - Address the specific problem
   - Acknowledge any delays in response
   - Provide clear next steps or solutions
   - Be professional and empathetic

3. **Get Approval**
   - Present the draft to the user
   - Make any requested changes
   - Confirm before sending

4. **Post-Send Actions**
   - Consider labeling for follow-up if needed
   - Archive if issue is resolved
   - Document any bugs or feature requests found

### Response Templates to Adapt

For bug reports:
```
Hi [Name],

Thank you for reporting this issue and for your patience. I apologize for the delayed response.

I've reviewed your bug report about [specific issue]. [Acknowledge the problem and any provided details/screenshots].

[Provide solution, workaround, or next steps]

Please let me know if you need any clarification or if the issue persists.

Best regards,
Team Blue
```

For feature requests:
```
Hi [Name],

Thank you for reaching out and for your patience with our response.

[Acknowledge their request and why it would be valuable]

[Explain current status or alternatives]

[Next steps or timeline if applicable]

Best regards,
Team Blue
```

### Important Reminders
- Always check the date of the original message to acknowledge if response is delayed
- Look for follow-up messages that indicate urgency or frustration
- Never send automated responses without human review
- Consider timezone differences when sending replies
- Be extra careful with messages marked IMPORTANT or UNREAD

### Testing vs Production
When testing the tool:
- Use `--limit 1` or `--limit 2` to avoid processing too many messages
- Test with `--output detailed` first to understand the data
- Always verify the correct message ID before replying
- Consider using labels to mark test messages

### Error Handling
If you encounter errors:
- Token issues: Delete `~/.support-agent/token.json` and re-authenticate
- API errors: Check if Gmail API quota is exceeded
- Connection issues: Verify internet connectivity and Gmail service status