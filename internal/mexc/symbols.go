package mexc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Resolver struct {
	cachePath string
	ttl       time.Duration
	fetch     func() ([]Contract, error)
}

type cacheFile struct {
	FetchedAt int64      `json:"fetchedAt"`
	Contracts []Contract `json:"contracts"`
}

type UnknownSymbolError struct {
	Input       string
	Suggestions []string
}

func (e *UnknownSymbolError) Error() string {
	if len(e.Suggestions) > 0 {
		return fmt.Sprintf("UNKNOWN_SYMBOL %q (closest: %s)", e.Input, strings.Join(e.Suggestions, ", "))
	}
	return fmt.Sprintf("UNKNOWN_SYMBOL %q", e.Input)
}

func NewResolver(client *Client) *Resolver {
	cacheDir := os.ExpandEnv("$HOME/.cache/mexctrade")
	_ = os.MkdirAll(cacheDir, 0o700)
	r := &Resolver{
		cachePath: filepath.Join(cacheDir, "contracts.json"),
		ttl:       24 * time.Hour,
	}
	r.fetch = client.fetchContracts
	return r
}

func (r *Resolver) load() ([]Contract, error) {
	body, err := os.ReadFile(r.cachePath)
	if err != nil {
		return nil, err
	}
	var f cacheFile
	if err := json.Unmarshal(body, &f); err != nil {
		return nil, err
	}
	if time.Since(time.Unix(f.FetchedAt, 0)) > r.ttl {
		return nil, fmt.Errorf("cache stale")
	}
	return f.Contracts, nil
}

func (r *Resolver) save(contracts []Contract) error {
	f := cacheFile{FetchedAt: time.Now().Unix(), Contracts: contracts}
	body, _ := json.Marshal(f)
	return os.WriteFile(r.cachePath, body, 0o644)
}

func (r *Resolver) all() ([]Contract, error) {
	if cs, err := r.load(); err == nil {
		return cs, nil
	}
	cs, err := r.fetch()
	if err != nil {
		return nil, err
	}
	_ = r.save(cs)
	return cs, nil
}

func (r *Resolver) Resolve(input string) (*Contract, error) {
	contracts, err := r.all()
	if err != nil {
		return nil, fmt.Errorf("load contracts: %w", err)
	}
	want := strings.ToUpper(input)
	if !strings.Contains(want, "_") {
		want += "_USDT"
	}
	for i := range contracts {
		if strings.EqualFold(contracts[i].Symbol, want) {
			return &contracts[i], nil
		}
	}
	return nil, &UnknownSymbolError{Input: input, Suggestions: suggest(want, contracts)}
}

// suggest returns up to 3 closest symbol names by edit distance <= 2.
func suggest(target string, contracts []Contract) []string {
	type cand struct {
		sym  string
		dist int
	}
	var cs []cand
	for _, c := range contracts {
		d := levenshtein(target, c.Symbol)
		if d <= 2 {
			cs = append(cs, cand{c.Symbol, d})
		}
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].dist < cs[j].dist })
	out := []string{}
	for i := 0; i < len(cs) && i < 3; i++ {
		out = append(out, cs[i].sym)
	}
	return out
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
