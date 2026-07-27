package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/mail"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

// Concurrency default. Threads.Get costs 10 quota units and Gmail allows 250
// units/second/user, so 8 in flight at roughly two round-trips a second lands
// near 160 units/second — fast, with headroom for the retry path to spend.
const defaultTriageConcurrency = 8

// TriageMessage is one message inside a thread, flattened for triage.
type TriageMessage struct {
	ID      string    `json:"id"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	ReplyTo string    `json:"reply_to,omitempty"`
	Subject string    `json:"subject"`
	Date    string    `json:"date"`
	Time    time.Time `json:"time"`
	Labels  []string  `json:"labels"`
	IsDraft bool      `json:"is_draft"`
	IsSent  bool      `json:"is_sent"`
	FromUs  bool      `json:"from_us"`
	Snippet string    `json:"snippet,omitempty"`
	Body    string    `json:"body,omitempty"`
}

// TriageThread is a whole conversation plus the state the Phase 1 drop rules
// need. Every field below is derived from the same single Threads.Get response,
// which is the point: inbox, sent and draft state cannot disagree with each
// other when they are read off one object.
type TriageThread struct {
	ID           string   `json:"id"`
	Subject      string   `json:"subject"`
	MessageCount int      `json:"message_count"`
	Participants []string `json:"participants"`

	LastInbound *time.Time `json:"last_inbound,omitempty"`
	LastSent    *time.Time `json:"last_sent,omitempty"`
	DraftDate   *time.Time `json:"draft_date,omitempty"`
	AgeDays     float64    `json:"age_days"`

	HasUnsentDraft bool `json:"has_unsent_draft"`
	DraftIsStale   bool `json:"draft_is_stale"`
	RepliedSince   bool `json:"replied_since_last_inbound"`

	NeedsWork  bool   `json:"needs_work"`
	DropReason string `json:"drop_reason,omitempty"`

	Messages []TriageMessage `json:"messages,omitempty"`
}

// RunTriageThreads fetches whole threads for a query in one pass.
//
// The per-message commands cannot answer "has this been dealt with?". A sent
// reply carries no INBOX label, so an in:inbox search does not return it, and a
// thread answered but never archived is indistinguishable from one nobody has
// touched. Threads.Get returns every message in the conversation — inbound,
// sent and draft alike — so the question is answerable from one response
// instead of a join across three queries.
func RunTriageThreads(args []string) error {
	fs := flag.NewFlagSet("triage-threads", flag.ExitOnError)

	query := fs.String("query", "in:inbox", "Gmail search query selecting the threads")
	limit := fs.Int64("limit", 0, "Max threads (0 = every match, following pagination)")
	concurrency := fs.Int("concurrency", defaultTriageConcurrency, "Parallel thread fetches")
	needsWorkOnly := fs.Bool("needs-work", false, "Emit only threads that still need working")
	withBodies := fs.Bool("bodies", false, "Include full message bodies (large)")
	withMessages := fs.Bool("messages", true, "Include per-message detail")
	output := fs.String("output", "simple", "Output format: simple or json")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *concurrency < 1 {
		*concurrency = 1
	}

	client, err := common.NewGmailClient()
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %v", err)
	}

	stubs, err := client.ListThreads(*query, *limit)
	if err != nil {
		return fmt.Errorf("failed to list threads: %v", err)
	}
	if len(stubs) == 0 {
		fmt.Println("No threads found matching query.")
		return nil
	}

	threads, failures := fetchThreads(client, stubs, *concurrency)

	results := make([]TriageThread, 0, len(threads))
	for _, t := range threads {
		if t == nil {
			continue
		}
		results = append(results, buildTriageThread(t, *withBodies, *withMessages))
	}

	sort.Slice(results, func(i, j int) bool { return results[i].AgeDays > results[j].AgeDays })

	shown := results
	if *needsWorkOnly {
		shown = shown[:0:0]
		for _, r := range results {
			if r.NeedsWork {
				shown = append(shown, r)
			}
		}
	}

	// Report dropped threads as a count rather than silence: a run that claims
	// to cover the whole inbox has to show what it did not return.
	needsWork := 0
	for _, r := range results {
		if r.NeedsWork {
			needsWork++
		}
	}

	if *output == "json" {
		payload, err := json.MarshalIndent(shown, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}
		fmt.Println(string(payload))
	} else {
		printTriageTable(shown)
	}

	fmt.Printf("\n%d threads · %d need work · %d dropped\n",
		len(results), needsWork, len(results)-needsWork)
	if len(failures) > 0 {
		fmt.Printf("WARNING: %d thread(s) could not be fetched and are missing from this list:\n", len(failures))
		for _, f := range failures {
			fmt.Printf("  %s\n", f)
		}
	}

	return nil
}

// fetchThreads pulls every thread concurrently, preserving input order.
func fetchThreads(client *common.GmailClient, stubs []*gmail.Thread, concurrency int) ([]*gmail.Thread, []string) {
	out := make([]*gmail.Thread, len(stubs))
	errs := make([]string, len(stubs))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, stub := range stubs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			thread, err := client.GetThreadWithRetry(id, 4)
			if err != nil {
				errs[i] = fmt.Sprintf("%s: %v", id, err)
				return
			}
			out[i] = thread
		}(i, stub.Id)
	}
	wg.Wait()

	failures := make([]string, 0)
	for _, e := range errs {
		if e != "" {
			failures = append(failures, e)
		}
	}
	return out, failures
}

// buildTriageThread flattens a Gmail thread and derives its triage state.
func buildTriageThread(thread *gmail.Thread, withBodies, withMessages bool) TriageThread {
	out := TriageThread{
		ID:           thread.Id,
		MessageCount: len(thread.Messages),
	}

	participants := make(map[string]bool)
	msgs := make([]TriageMessage, 0, len(thread.Messages))

	for _, msg := range thread.Messages {
		if msg.Payload == nil {
			continue
		}
		headers := common.ExtractHeaders(msg)
		labels := common.GetLabelNames(msg.LabelIds)

		tm := TriageMessage{
			ID:      msg.Id,
			From:    headers["from"],
			To:      headers["to"],
			ReplyTo: headers["reply-to"],
			Subject: headers["subject"],
			Date:    headers["date"],
			Labels:  labels,
			IsDraft: hasLabel(labels, "DRAFT"),
			IsSent:  hasLabel(labels, "SENT"),
			FromUs:  common.IsInternalAddress(headers["from"]),
			Snippet: msg.Snippet,
		}
		if withBodies {
			tm.Body = common.ExtractMessageBody(msg)
		}
		// InternalDate is epoch milliseconds and is set by Gmail, so it stays
		// sortable when a sender's Date header is malformed or misconfigured.
		if msg.InternalDate > 0 {
			tm.Time = time.UnixMilli(msg.InternalDate).UTC()
		} else if parsed, err := mail.ParseDate(headers["date"]); err == nil {
			tm.Time = parsed.UTC()
		}

		if !tm.FromUs {
			if addr := addressOf(tm.From); addr != "" {
				participants[addr] = true
			}
		}
		msgs = append(msgs, tm)
	}

	sort.Slice(msgs, func(i, j int) bool { return msgs[i].Time.Before(msgs[j].Time) })

	for _, m := range msgs {
		t := m.Time
		switch {
		case m.IsDraft:
			out.DraftDate = laterOf(out.DraftDate, t)
			out.HasUnsentDraft = true
		case m.IsSent || m.FromUs:
			out.LastSent = laterOf(out.LastSent, t)
		default:
			out.LastInbound = laterOf(out.LastInbound, t)
		}
		if out.Subject == "" && m.Subject != "" {
			out.Subject = m.Subject
		}
	}

	for p := range participants {
		out.Participants = append(out.Participants, p)
	}
	sort.Strings(out.Participants)

	if out.LastInbound != nil {
		out.AgeDays = round1(time.Since(*out.LastInbound).Hours() / 24)
	} else if len(msgs) > 0 {
		out.AgeDays = round1(time.Since(msgs[len(msgs)-1].Time).Hours() / 24)
	}

	// We have answered if a send is newer than the newest inbound message. A
	// thread we sent on with no inbound at all is ours too — outbound-only.
	out.RepliedSince = out.LastSent != nil &&
		(out.LastInbound == nil || out.LastSent.After(*out.LastInbound))

	// A draft written before the customer's latest message has been overtaken
	// by it. Chasing is the +2 waiting signal in the score, so this is exactly
	// the thread that must not be dropped for "already drafted".
	out.DraftIsStale = out.HasUnsentDraft && out.LastInbound != nil &&
		out.DraftDate != nil && out.LastInbound.After(*out.DraftDate)

	switch {
	case out.RepliedSince:
		out.DropReason = "replied since last inbound"
	case out.HasUnsentDraft && !out.DraftIsStale:
		out.DropReason = "draft pending human review"
	default:
		out.NeedsWork = true
	}

	if !withMessages {
		out.Messages = nil
	} else {
		out.Messages = msgs
	}
	return out
}

func printTriageTable(threads []TriageThread) {
	if len(threads) == 0 {
		fmt.Println("No threads to show.")
		return
	}
	fmt.Printf("%-6s %-5s %-38s %-52s %s\n", "AGE", "STATE", "FROM", "SUBJECT", "NOTE")
	for _, t := range threads {
		state := "work"
		if !t.NeedsWork {
			state = "drop"
		}
		note := t.DropReason
		if t.DraftIsStale {
			note = "draft stale — customer wrote again"
		}
		who := strings.Join(t.Participants, ",")
		fmt.Printf("%-6s %-5s %-38.38s %-52.52s %s\n",
			fmt.Sprintf("%.1fd", t.AgeDays), state, who, t.Subject, note)
	}
}

func hasLabel(labels []string, want string) bool {
	return slices.Contains(labels, want)
}

// addressOf pulls the bare address out of an RFC 5322 header value, falling
// back to the raw string so an unparseable From still identifies a participant.
func addressOf(header string) string {
	if header == "" {
		return ""
	}
	if parsed, err := mail.ParseAddress(header); err == nil {
		return strings.ToLower(parsed.Address)
	}
	return strings.ToLower(strings.TrimSpace(header))
}

func laterOf(current *time.Time, candidate time.Time) *time.Time {
	if candidate.IsZero() {
		return current
	}
	if current == nil || candidate.After(*current) {
		c := candidate
		return &c
	}
	return current
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
