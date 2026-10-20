// Package patterns holds the built-in catalogue of sensitive-value shapes the
// anonymizer masks.
//
// The catalogue is deliberately conservative about what it hides. Credentials
// and personal data are masked; hosts, IP addresses and URLs are not, because
// the product's agents reason about them in every message they carry — masking
// them would leave the agent recalling "§*Host*§ is up" from its own memory.
package patterns

import (
	"fmt"
	"regexp"
)

// Pattern couples a human-readable name with the expression that finds one kind
// of sensitive value. A pattern may mark the exact span to mask with the named
// group replace; without it the whole match is masked.
type Pattern struct {
	Name  string
	Regex string
}

// PatternListType names a catalogue to load.
type PatternListType int

const (
	PatternListTypeAll PatternListType = iota
)

// PatternList is a catalogue about to be handed to the anonymizer. Patterns is
// exported and mutable by contract: callers append their own entries — the
// configured secrets, for one — before asking for Regexes and Names.
type PatternList struct {
	Patterns []Pattern
}

// Regexes compiles the catalogue in order. An expression that fails to compile
// becomes a stand-in that never matches rather than being dropped: names and
// regexes are paired positionally and a dropped entry would shift every pair
// after it.
func (l *PatternList) Regexes() []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(l.Patterns))
	for _, p := range l.Patterns {
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			re = neverMatches
		}
		out = append(out, re)
	}
	return out
}

func (l *PatternList) Names() []string {
	out := make([]string, 0, len(l.Patterns))
	for _, p := range l.Patterns {
		out = append(out, p.Name)
	}
	return out
}

var neverMatches = regexp.MustCompile(`\b\B`)

// LoadPatterns returns a fresh copy of the named catalogue, so a caller's
// appends stay its own.
func LoadPatterns(kind PatternListType) (*PatternList, error) {
	switch kind {
	case PatternListTypeAll:
		return &PatternList{Patterns: append([]Pattern(nil), catalogue...)}, nil
	default:
		return nil, fmt.Errorf("unknown pattern list type %d", kind)
	}
}

// Order matters where shapes overlap: Connection String runs before URL
// Password so a database URL masks user and password together, and Secret
// Assignment runs after the credential-shape patterns so it only catches values
// they did not already name.
var catalogue = []Pattern{
	{"Private Key Block", `(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`},
	{"AWS Access Key ID", `(?:A3T[A-Z0-9]|AKIA|ASIA|ABIA|ACCA)[A-Z0-9]{16}`},
	{"AWS Secret Access Key", `(?i)aws[_-]?(?:secret)?[_-]?access[_-]?key["']?\s*[:=]\s*["']?(?P<replace>[A-Za-z0-9/+=]{40})`},
	{"Google API Key", `AIza[0-9A-Za-z_-]{35}`},
	{"Google OAuth Token", `ya29\.[0-9A-Za-z_-]{20,}`},
	{"OpenAI API Key", `sk-[A-Za-z0-9]{32,}`},
	{"OpenAI Project Key", `sk-proj-[A-Za-z0-9_-]{32,}`},
	{"Anthropic API Key", `sk-ant-[A-Za-z0-9_-]{32,}`},
	{"GitHub Token", `gh[pousr]_[A-Za-z0-9]{36,}`},
	{"GitHub Fine-Grained PAT", `github_pat_[A-Za-z0-9_]{30,}`},
	{"GitLab Token", `glpat-[A-Za-z0-9_-]{20,}`},
	{"Slack Token", `xox[baprs]-[A-Za-z0-9-]{10,}`},
	{"Stripe Key", `[rs]k_(?:live|test)_[A-Za-z0-9]{16,}`},
	{"SendGrid Key", `SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}`},
	{"npm Token", `npm_[A-Za-z0-9]{36}`},
	{"PyPI Token", `pypi-[A-Za-z0-9_-]{50,}`},
	{"Docker Hub Token", `dckr_pat_[A-Za-z0-9_-]{27,}`},
	{"HashiCorp Vault Token", `hvs\.[A-Za-z0-9_-]{24,}`},
	{"JSON Web Token", `eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`},
	{"Bearer Token", `(?i)bearer\s+(?P<replace>[A-Za-z0-9._~+/=-]{16,})`},
	{"Connection String", `(?i)(?:postgres(?:ql)?|mysql|mariadb|mongodb(?:\+srv)?|redis|rediss|amqp|amqps|mssql|clickhouse)://(?P<replace>[^:/\s]+:[^@/\s]+@)`},
	{"URL Password", `\b[a-zA-Z][A-Za-z0-9+.-]*://[^/\s:@]{1,128}:(?P<replace>[^/\s:@]{1,256})@`},
	{"Secret Assignment", `(?i)(?:password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|secret[_-]?key|auth[_-]?token|client[_-]?secret)["']?\s*[:=]\s*["']?(?P<replace>[^\s"',;(){}§]{8,})`},
	{"Email Address", `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`},
}
