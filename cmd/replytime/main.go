package main

// One-off analyzer: average reply time to support email for a date window.
// Efficient path — one metadata-only Threads.Get per thread (not per message),
// paginated. Reuses the support-agent OAuth/client.
//
//   go run ./cmd/replytime -after 2026/05/01 -before 2026/06/01
//
// "Us" = any sender on an internal domain (blue.cc / blue.app). Everyone else
// is a customer. First Response Time (FRT) = time from a customer's first
// message in a thread to our first reply. Cohort = threads whose first customer
// message falls inside the window.

import (
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/blue/support-agent/common"
	"google.golang.org/api/gmail/v1"
)

var internalDomains = []string{"blue.cc", "blue.app"}

// senders we never "reply" to — automated noise that would otherwise look like
// an unanswered inbound thread (they self-exclude from FRT anyway, but skip the
// fetch where cheap).
var noiseSenders = []string{
	"notifications@github.com", "noreply@github.com",
	"no-reply@", "noreply@", "mailer-daemon@", "postmaster@",
}

type msg struct {
	from string
	t    time.Time
	us   bool
}

func isInternal(from string) bool {
	f := strings.ToLower(from)
	for _, d := range internalDomains {
		if strings.Contains(f, "@"+d) {
			return true
		}
	}
	return false
}

func isNoise(from string) bool {
	f := strings.ToLower(from)
	for _, n := range noiseSenders {
		if strings.Contains(f, n) {
			return true
		}
	}
	return false
}

// Gmail Date headers come in many shapes; try the common ones.
var dateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
	"2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
}

func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	// strip trailing "(MST)" style comment that breaks RFC1123Z
	if i := strings.Index(s, " ("); i > 0 {
		s2 := s[:i]
		for _, l := range dateLayouts {
			if t, err := time.Parse(l, s2); err == nil {
				return t, true
			}
		}
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func header(m *gmail.Message, name string) string {
	if m.Payload == nil {
		return ""
	}
	for _, h := range m.Payload.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

func fmtDur(d time.Duration) string {
	h := d.Hours()
	if h < 1 {
		return fmt.Sprintf("%.0f min", d.Minutes())
	}
	if h < 48 {
		return fmt.Sprintf("%.1f h", h)
	}
	return fmt.Sprintf("%.1f days", h/24)
}

func pct(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p/100*float64(len(sorted)-1) + 0.5)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func main() {
	after := flag.String("after", "2026/05/01", "Gmail after: date (YYYY/MM/DD)")
	before := flag.String("before", "2026/06/01", "Gmail before: date (YYYY/MM/DD)")
	verbose := flag.Bool("v", false, "list every thread's FRT")
	flag.Parse()

	monthStart, _ := time.Parse("2006/01/02", *after)
	monthEnd, _ := time.Parse("2006/01/02", *before)

	client, err := common.NewGmailClient()
	if err != nil {
		fmt.Println("auth error:", err)
		return
	}

	q := fmt.Sprintf("after:%s before:%s", *after, *before)
	fmt.Printf("Query: %s\n", q)

	// 1) list thread IDs in window (paginated)
	var threadIDs []string
	pageToken := ""
	for {
		call := client.Service.Users.Threads.List(client.UserID).Q(q).MaxResults(500)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			fmt.Println("threads.list error:", err)
			return
		}
		for _, t := range resp.Threads {
			threadIDs = append(threadIDs, t.Id)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	fmt.Printf("Threads in window: %d\n", len(threadIDs))

	// 2) per thread: one metadata Get, compute FRT + per-reply latencies
	type frtRow struct {
		subject string
		from    string
		frt     time.Duration
		first   time.Time
	}
	var frts []frtRow
	var replyLatencies []time.Duration // every customer->us adjacent pair

	initiatedByUs := 0
	noReply := 0
	analyzed := 0

	for i, id := range threadIDs {
		th, err := client.Service.Users.Threads.Get(client.UserID, id).
			Format("metadata").
			MetadataHeaders("From", "Date", "Subject").Do()
		if err != nil {
			fmt.Printf("  thread %s get error: %v\n", id, err)
			continue
		}
		if i%50 == 49 {
			fmt.Printf("  ...%d/%d threads\n", i+1, len(threadIDs))
		}

		var msgs []msg
		subject := ""
		for _, m := range th.Messages {
			d := header(m, "Date")
			t, ok := parseDate(d)
			if !ok {
				continue
			}
			from := header(m, "From")
			if subject == "" {
				subject = header(m, "Subject")
			}
			msgs = append(msgs, msg{from: from, t: t, us: isInternal(from)})
		}
		if len(msgs) == 0 {
			continue
		}
		sort.Slice(msgs, func(a, b int) bool { return msgs[a].t.Before(msgs[b].t) })

		// per-reply latencies: each time an us-message directly follows a
		// customer message in time order.
		for j := 1; j < len(msgs); j++ {
			if msgs[j].us && !msgs[j-1].us {
				replyLatencies = append(replyLatencies, msgs[j].t.Sub(msgs[j-1].t))
			}
		}

		// FRT: first customer message, then first us-message after it.
		var firstCust *msg
		for k := range msgs {
			if !msgs[k].us && !isNoise(msgs[k].from) {
				firstCust = &msgs[k]
				break
			}
		}
		if firstCust == nil {
			initiatedByUs++ // no genuine customer inbound (we initiated, or pure noise)
			continue
		}
		// cohort gate: first customer message must fall inside the window
		if firstCust.t.Before(monthStart) || !firstCust.t.Before(monthEnd) {
			continue
		}
		var firstReply *msg
		for k := range msgs {
			if msgs[k].us && msgs[k].t.After(firstCust.t) {
				firstReply = &msgs[k]
				break
			}
		}
		if firstReply == nil {
			noReply++
			continue
		}
		analyzed++
		frts = append(frts, frtRow{
			subject: subject,
			from:    firstCust.from,
			frt:     firstReply.t.Sub(firstCust.t),
			first:   firstCust.t,
		})
	}

	// ---- report ----
	fmt.Printf("\n=== Cohort: customer-initiated threads with a reply, first msg in window ===\n")
	fmt.Printf("Analyzed (have FRT):        %d\n", analyzed)
	fmt.Printf("Initiated by us / noise:    %d\n", initiatedByUs)
	fmt.Printf("Customer msg, no reply yet: %d\n", noReply)

	if len(frts) == 0 {
		fmt.Println("\nNo FRT data points.")
		return
	}

	durs := make([]time.Duration, len(frts))
	var sum time.Duration
	within1h, within4h, within24h := 0, 0, 0
	for i, r := range frts {
		durs[i] = r.frt
		sum += r.frt
		if r.frt <= time.Hour {
			within1h++
		}
		if r.frt <= 4*time.Hour {
			within4h++
		}
		if r.frt <= 24*time.Hour {
			within24h++
		}
	}
	sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
	mean := time.Duration(int64(sum) / int64(len(durs)))
	median := durs[len(durs)/2]

	fmt.Printf("\n--- First Response Time (n=%d) ---\n", len(durs))
	fmt.Printf("Mean:   %s\n", fmtDur(mean))
	fmt.Printf("Median: %s\n", fmtDur(median))
	fmt.Printf("p90:    %s\n", fmtDur(pct(durs, 90)))
	fmt.Printf("Min:    %s\n", fmtDur(durs[0]))
	fmt.Printf("Max:    %s\n", fmtDur(durs[len(durs)-1]))
	fmt.Printf("Within 1h:  %d (%.0f%%)\n", within1h, 100*float64(within1h)/float64(len(durs)))
	fmt.Printf("Within 4h:  %d (%.0f%%)\n", within4h, 100*float64(within4h)/float64(len(durs)))
	fmt.Printf("Within 24h: %d (%.0f%%)\n", within24h, 100*float64(within24h)/float64(len(durs)))

	if len(replyLatencies) > 0 {
		sort.Slice(replyLatencies, func(a, b int) bool { return replyLatencies[a] < replyLatencies[b] })
		var rsum time.Duration
		for _, d := range replyLatencies {
			rsum += d
		}
		fmt.Printf("\n--- Every reply (customer->us pairs, n=%d) ---\n", len(replyLatencies))
		fmt.Printf("Mean:   %s\n", fmtDur(time.Duration(int64(rsum)/int64(len(replyLatencies)))))
		fmt.Printf("Median: %s\n", fmtDur(replyLatencies[len(replyLatencies)/2]))
		fmt.Printf("p90:    %s\n", fmtDur(pct(replyLatencies, 90)))
	}

	// slowest FRTs
	sort.Slice(frts, func(a, b int) bool { return frts[a].frt > frts[b].frt })
	fmt.Printf("\n--- Slowest 8 first responses ---\n")
	for i := 0; i < 8 && i < len(frts); i++ {
		s := frts[i].subject
		if len(s) > 45 {
			s = s[:45]
		}
		fmt.Printf("  %-9s  %-45s  %s\n", fmtDur(frts[i].frt), s, frts[i].from)
	}

	if *verbose {
		sort.Slice(frts, func(a, b int) bool { return frts[a].first.Before(frts[b].first) })
		fmt.Printf("\n--- All FRTs ---\n")
		for _, r := range frts {
			fmt.Printf("  %s  %-9s  %s\n", r.first.Format("Jan 02 15:04"), fmtDur(r.frt), r.subject)
		}
	}
}
