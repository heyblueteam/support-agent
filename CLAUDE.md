# Support Agent - Claude Code Instructions


### Rules
- **NEVER send replies automatically without first showing the draft to the user for approval.**
- Do not be overly apologetic, focus on the issue and provide a solution
- When necessary, search within the `docs/` folder to find relevant documentation to answer questions received by email. Use Grep or Glob to search for keywords related to the customer's question.


## Email Reply Protocol

When helping with email responses:
1. First, analyze the thread/message to understand the context
2. Draft a proposed reply
3. Show the draft to the user for review
4. Only send after explicit user approval
5. Use the `reply-message` command only when the user confirms the draft

## Reading and Analyzing Emails
- Use `read-threads` to get full context before drafting replies
- Use `--output json` when processing for automated analysis
- Check message dates to understand response urgency
- Note if there are previous unanswered messages in the thread

## Tool Usage Best Practices

### For Reading:
```bash
# Get overview of recent messages
./support-agent read-messages --limit 10

# Get full thread context
./support-agent read-threads --thread-id THREAD_ID --output detailed

# Search for specific issues
./support-agent search-messages --query "bug OR error OR problem"
```

### For Writing (ONLY after user approval):
```bash
# Reply to a message
./support-agent reply-message --message-id MSG_ID --body "Approved reply text"

# Archive handled threads
./support-agent archive-message --thread-id THREAD_ID

# Mark important messages
./support-agent label-message --message-id MSG_ID --add-label IMPORTANT
```

## Workflow for Support Responses

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

## Response Templates to Adapt

For bug reports:
```
Hi [Name],

Thank you for reporting this issue and for your patience. 

I've reviewed your bug report about [specific issue]. [Acknowledge the problem and any provided details].

[Provide solution, workaround, or next steps]

Please let me know if you need any clarification or if the issue persists.

Best regards,
Team Blue
```

For feature requests:
```
Hi [Name],

Thank you for reaching out!

[Acknowledge their request and why it would be valuable]

[Explain current status or alternatives]

[Next steps or timeline if applicable]

Best regards,
Team Blue
```



