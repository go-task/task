// find-adopters scans GitHub for every public repository containing a Taskfile
// and produces a ranked list of adopter candidates.
//
// GitHub Code Search caps results at 1000 per query. This tool partitions
// queries by star buckets (and, if needed, by pushed-date ranges) to cover the
// full population, then enriches every hit with GraphQL (stars, description,
// owner type, language) before sorting by popularity.
//
// Usage:
//
//	find-adopters [flags]
//
// Auth: set GITHUB_TOKEN, or have `gh auth login` configured locally.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// ----- Config / flags -----

type config struct {
	output          string
	emitJSON        bool
	minStars        int
	includeForks    bool
	includeArchived bool
	ownerType       string
	verbose         bool
}

func parseFlags() config {
	var c config
	flag.StringVar(&c.output, "o", "adopters-scan.tsv", "output path")
	flag.BoolVar(&c.emitJSON, "json", false, "emit JSON instead of TSV")
	flag.IntVar(&c.minStars, "min-stars", 0, "filter results below threshold")
	flag.BoolVar(&c.includeForks, "include-forks", false, "include forked repos")
	flag.BoolVar(&c.includeArchived, "include-archived", false, "include archived repos")
	flag.StringVar(&c.ownerType, "owner-type", "any", "filter by owner type: org|user|any")
	flag.BoolVar(&c.verbose, "v", false, "verbose progress logging")
	flag.Parse()
	return c
}

// ----- Auth -----

func resolveToken() (string, error) {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, nil
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("no GITHUB_TOKEN env var and `gh auth token` failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ----- HTTP client with rate limiting -----

type client struct {
	http    *http.Client
	token   string
	verbose bool

	searchMu   sync.Mutex
	searchLast time.Time
	searchGap  time.Duration // minimum gap between code-search requests
}

func newClient(token string, verbose bool) *client {
	return &client{
		http:      &http.Client{Timeout: 60 * time.Second},
		token:     token,
		verbose:   verbose,
		searchGap: 7 * time.Second, // ~8.5 req/min, under the 10/min cap
	}
}

func (c *client) logf(format string, args ...any) {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[find-adopters] "+format+"\n", args...)
	}
}

func (c *client) throttleSearch() {
	c.searchMu.Lock()
	defer c.searchMu.Unlock()
	if wait := c.searchGap - time.Since(c.searchLast); wait > 0 {
		time.Sleep(wait)
	}
	c.searchLast = time.Now()
}

func (c *client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	const attempts = 5
	for i := 0; i < attempts; i++ {
		resp, err := c.http.Do(req)
		if err != nil {
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		// Respect secondary rate limits + 5xx backoff.
		if resp.StatusCode == 403 || resp.StatusCode == 429 || resp.StatusCode >= 500 {
			resp.Body.Close()
			wait := time.Duration(1<<i) * time.Second
			if retry := resp.Header.Get("Retry-After"); retry != "" {
				var secs int
				fmt.Sscanf(retry, "%d", &secs)
				if secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}
			c.logf("backoff %s (status=%d)", wait, resp.StatusCode)
			time.Sleep(wait)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("giving up after %d attempts", attempts)
}

// ----- Code search (discovery) -----

type searchResp struct {
	TotalCount int  `json:"total_count"`
	Incomplete bool `json:"incomplete_results"`
	Items      []struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"items"`
}

func (c *client) searchCode(q string, page int) (*searchResp, error) {
	c.throttleSearch()
	url := fmt.Sprintf(
		"https://api.github.com/search/code?q=%s&per_page=100&page=%d",
		urlEscape(q), page,
	)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search %q: %s: %s", q, resp.Status, body)
	}
	var sr searchResp
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

// urlEscape is a tiny URL-query-safe encoder. GitHub accepts `+` as space and
// `%`-encodes special chars; we just need the common characters.
func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.' || r == '~':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('+')
		default:
			fmt.Fprintf(&b, "%%%02X", r)
		}
	}
	return b.String()
}

// paginateQuery pages through up to 1000 results. If total_count is above the
// cap reachable by pageLimit pages, it still paginates — callers that want to
// avoid wasted calls and subdivide instead should check total beforehand by
// passing pageLimit=1.
func (c *client) paginateQuery(q string, pageLimit int) (repos []string, total int, err error) {
	first, err := c.searchCode(q, 1)
	if err != nil {
		return nil, 0, err
	}
	total = first.TotalCount
	if total == 0 {
		return nil, 0, nil
	}
	for _, it := range first.Items {
		repos = append(repos, it.Repository.FullName)
	}
	pages := (total + 99) / 100
	if pages > pageLimit {
		pages = pageLimit
	}
	for page := 2; page <= pages; page++ {
		sr, err := c.searchCode(q, page)
		if err != nil {
			return repos, total, err
		}
		for _, it := range sr.Items {
			repos = append(repos, it.Repository.FullName)
		}
		if len(sr.Items) < 100 {
			break
		}
	}
	return repos, total, nil
}

// GitHub Code Search caps at 1000 results per query and is unreliable with
// the `size:` qualifier (total_count is non-monotone as ranges shrink), so
// partitioning tricks don't work cleanly. We instead combine two strategies:
//
//  1. Paginate each Taskfile variant directly — gets ~900 top-ranked hits per
//     variant (the "best match" slice GitHub surfaces).
//  2. Iterate a curated list of well-known organizations with an explicit
//     `org:` qualifier — gets full coverage inside big brands even when their
//     repos don't rank in the global top 900.
//
// The union is deduplicated and enriched via GraphQL.

// knownOrgs is a snapshot of organizations worth scanning explicitly. Adding
// here captures every Taskfile inside the org regardless of its global rank.
// Loosely ordered from most likely to least.
var knownOrgs = []string{
	// Hyperscalers / clouds
	"docker", "microsoft", "google", "GoogleCloudPlatform", "aws", "awslabs",
	"aws-samples", "amazon-science", "Azure", "Azure-Samples",
	// Infra / DevOps vendors
	"hashicorp", "hashicorp-forge", "vercel", "cloudflare", "digitalocean",
	"heroku", "JetBrains", "pulumi", "buildkite", "circleci", "dagger",
	"temporalio", "encoredev", "argoproj", "fluxcd", "flux-framework",
	// Dev tools / platforms
	"netflix", "shopify", "airbnb", "uber", "lyft", "stripe", "github",
	"gitlabhq", "atlassian", "RedHat", "RedHatOfficial", "openshift",
	// Communication / consumer
	"spotify", "slackapi", "discord", "figma", "linear", "twilio", "segmentio",
	// Data / ML
	"prisma", "supabase", "railwayapp", "superfly", "fly-apps", "planetscale",
	"tailscale", "coder", "anthropics", "openai", "huggingface",
	"pytorch", "tensorflow",
	// Observability / CNCF
	"grafana", "prometheus", "envoyproxy", "getsentry", "sentry", "cncf",
	"helm", "istio", "linkerd", "traefik", "caddyserver",
	// Frontend frameworks
	"vitejs", "biomejs", "sveltejs", "vuejs", "reactjs", "astro", "nuxt",
	// Databases
	"mongodb-labs", "redis", "neo4j", "elastic", "influxdata", "timescale",
	"clickhouse", "FerretDB",
	// Go ecosystem / popular OSS
	"goreleaser", "spf13", "urfave", "charmbracelet", "nodejs", "golang",
	"rust-lang", "python", "apache", "etcd-io", "grpc", "arduino",
	// Data eng
	"dbt-labs", "astronomer", "prefecthq",
}

// discover walks every Taskfile variant with global pagination plus per-org
// scans, and returns unique owner/name pairs.
func (c *client) discover() (map[string]struct{}, error) {
	uniq := make(map[string]struct{})

	variants := []string{
		"Taskfile.yml",
		"Taskfile.yaml",
		"Taskfile.dist.yml",
		"Taskfile.dist.yaml",
	}

	c.logf("phase: global pagination (best-match top ~900 per variant)")
	for _, v := range variants {
		q := fmt.Sprintf("filename:%s", v)
		c.logf("  query: %s", q)
		repos, total, err := c.paginateQuery(q, 10)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: variant %s: %v\n", v, err)
			continue
		}
		c.logf("    total=%d collected=%d", total, len(repos))
		for _, r := range repos {
			uniq[r] = struct{}{}
		}
	}

	c.logf("phase: per-org scan (%d orgs)", len(knownOrgs))
	for _, org := range knownOrgs {
		q := fmt.Sprintf("filename:Taskfile.yml org:%s", org)
		repos, total, err := c.paginateQuery(q, 10)
		if err != nil {
			// Orgs that don't exist return 404 — log once and move on.
			c.logf("  skip %s: %v", org, err)
			continue
		}
		if total == 0 {
			continue
		}
		c.logf("  org:%s total=%d collected=%d", org, total, len(repos))
		for _, r := range repos {
			uniq[r] = struct{}{}
		}
	}

	return uniq, nil
}

// ----- Enrichment (GraphQL) -----

type Adopter struct {
	FullName    string `json:"full_name"`
	Stars       int    `json:"stars"`
	IsFork      bool   `json:"is_fork"`
	IsArchived  bool   `json:"is_archived"`
	OwnerType   string `json:"owner_type"` // "Organization" or "User"
	Language    string `json:"language"`
	Description string `json:"description"`
	URL         string `json:"url"`
	PushedAt    string `json:"pushed_at"`
	Topics      []string `json:"topics"`
}

func (c *client) enrichBatch(repos []string) ([]Adopter, error) {
	var b strings.Builder
	b.WriteString("query {")
	for i, r := range repos {
		parts := strings.SplitN(r, "/", 2)
		if len(parts) != 2 {
			continue
		}
		owner := jsonEscape(parts[0])
		name := jsonEscape(parts[1])
		fmt.Fprintf(&b, ` r%d: repository(owner: "%s", name: "%s") {
  nameWithOwner stargazerCount isFork isArchived pushedAt url
  owner { __typename }
  description
  primaryLanguage { name }
  repositoryTopics(first: 10) { nodes { topic { name } } }
}`, i, owner, name)
	}
	b.WriteString(" }")

	payload := map[string]string{"query": b.String()}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("graphql: %s: %s", resp.Status, rb)
	}

	var g struct {
		Data map[string]*struct {
			NameWithOwner   string `json:"nameWithOwner"`
			StargazerCount  int    `json:"stargazerCount"`
			IsFork          bool   `json:"isFork"`
			IsArchived      bool   `json:"isArchived"`
			PushedAt        string `json:"pushedAt"`
			URL             string `json:"url"`
			Owner           struct {
				TypeName string `json:"__typename"`
			} `json:"owner"`
			Description     string `json:"description"`
			PrimaryLanguage *struct {
				Name string `json:"name"`
			} `json:"primaryLanguage"`
			RepositoryTopics struct {
				Nodes []struct {
					Topic struct {
						Name string `json:"name"`
					} `json:"topic"`
				} `json:"nodes"`
			} `json:"repositoryTopics"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return nil, err
	}

	var out []Adopter
	for _, v := range g.Data {
		if v == nil {
			continue
		}
		a := Adopter{
			FullName:    v.NameWithOwner,
			Stars:       v.StargazerCount,
			IsFork:      v.IsFork,
			IsArchived:  v.IsArchived,
			OwnerType:   v.Owner.TypeName,
			Description: v.Description,
			URL:         v.URL,
			PushedAt:    v.PushedAt,
		}
		if v.PrimaryLanguage != nil {
			a.Language = v.PrimaryLanguage.Name
		}
		for _, n := range v.RepositoryTopics.Nodes {
			a.Topics = append(a.Topics, n.Topic.Name)
		}
		out = append(out, a)
	}
	return out, nil
}

func jsonEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// enrichAll runs batched GraphQL enrichment with a small worker pool.
func (c *client) enrichAll(repos []string) []Adopter {
	const (
		batchSize = 50
		workers   = 4
	)

	batches := make([][]string, 0, (len(repos)+batchSize-1)/batchSize)
	for i := 0; i < len(repos); i += batchSize {
		end := i + batchSize
		if end > len(repos) {
			end = len(repos)
		}
		batches = append(batches, repos[i:end])
	}

	c.logf("enriching %d repos in %d batches (%d workers)", len(repos), len(batches), workers)

	var (
		out  []Adopter
		mu   sync.Mutex
		wg   sync.WaitGroup
		jobs = make(chan []string)
		done int
	)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range jobs {
				result, err := c.enrichBatch(batch)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warn: enrich batch: %v\n", err)
					continue
				}
				mu.Lock()
				out = append(out, result...)
				done++
				if c.verbose {
					fmt.Fprintf(os.Stderr, "[find-adopters] enriched %d/%d batches\n", done, len(batches))
				}
				mu.Unlock()
			}
		}()
	}
	for _, b := range batches {
		jobs <- b
	}
	close(jobs)
	wg.Wait()
	return out
}

// ----- Output -----

func writeTSV(w io.Writer, data []Adopter) error {
	fmt.Fprintln(w, "stars\tfull_name\towner_type\tlanguage\turl\tdescription")
	for _, a := range data {
		desc := strings.ReplaceAll(a.Description, "\t", " ")
		desc = strings.ReplaceAll(desc, "\n", " ")
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			a.Stars, a.FullName, a.OwnerType, a.Language, a.URL, desc)
	}
	return nil
}

func writeJSON(w io.Writer, data []Adopter) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// ----- Main -----

func main() {
	cfg := parseFlags()

	token, err := resolveToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cli := newClient(token, cfg.verbose)

	start := time.Now()
	cli.logf("phase: discover")
	uniq, err := cli.discover()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cli.logf("discovered %d unique repos in %s", len(uniq), time.Since(start).Round(time.Second))

	repos := make([]string, 0, len(uniq))
	for r := range uniq {
		repos = append(repos, r)
	}
	sort.Strings(repos)

	cli.logf("phase: enrich")
	enriched := cli.enrichAll(repos)

	// Filter
	filtered := enriched[:0]
	for _, a := range enriched {
		if !cfg.includeForks && a.IsFork {
			continue
		}
		if !cfg.includeArchived && a.IsArchived {
			continue
		}
		if a.Stars < cfg.minStars {
			continue
		}
		switch cfg.ownerType {
		case "org":
			if a.OwnerType != "Organization" {
				continue
			}
		case "user":
			if a.OwnerType != "User" {
				continue
			}
		}
		filtered = append(filtered, a)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Stars > filtered[j].Stars
	})

	f, err := os.Create(cfg.output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if cfg.emitJSON {
		err = writeJSON(f, filtered)
	} else {
		err = writeTSV(f, filtered)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "wrote %d rows to %s (total=%d, filtered=%d, elapsed=%s)\n",
		len(filtered), cfg.output, len(enriched), len(filtered), time.Since(start).Round(time.Second))
}
