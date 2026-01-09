# How do I manage email suppressions?

When customers report not receiving emails from Blue, their email may be suppressed due to previous delivery failures.

## Check if an email is suppressed

```bash
./support-agent suppressions --check customer@example.com
```

If suppressed, you'll see:
```
Email 'customer@example.com' IS suppressed:
  ID: 1234567
  Reason: too many soft fails
```

## Remove a suppression

```bash
./support-agent suppressions --remove customer@example.com
```

## List all suppressions

```bash
./support-agent suppressions --list
```

## Common suppression reasons

- **too many soft fails** - Temporary delivery issues (full mailbox, server timeout). Usually safe to remove.
- **too many hard fails** - Permanent delivery issues (invalid email, domain doesn't exist). Verify the email is correct before removing.

## After removing a suppression

Let the customer know:
1. Their email has been unblocked
2. They should check their spam/junk folder
3. Ask them to add our sending address to their contacts

**Note:** If the same email keeps getting suppressed, there may be an underlying issue with their email provider or the address itself.
