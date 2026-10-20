package assets

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

const (
	TypeDomain     = "domain"
	TypeIP         = "ip"
	TypeCIDR       = "cidr"
	TypeURL        = "url"
	TypeEmail      = "email"
	TypeService    = "service"
	TypeTechnology = "technology"
)

// Relations read src -[relation]-> dst with the source as the subject:
// dev.acme.com -[subdomain_of]-> acme.com, acme.com -[resolves_to]-> 192.0.2.10.
const (
	RelationSubdomainOf = "subdomain_of"
	RelationResolvesTo  = "resolves_to"
	RelationMemberOf    = "member_of"
	RelationServes      = "serves"
	RelationHosts       = "hosts"
	RelationUses        = "uses"
	RelationSameAs      = "same_as"
	RelationRelatedTo   = "related_to"
)

const (
	KindTarget     = "target"
	KindHypothesis = "hypothesis"
	KindAction     = "action"
	KindFinding    = "finding"
	KindNote       = "note"
)

const (
	StatusPending  = "pending"
	StatusActive   = "active"
	StatusExplored = "explored"
	StatusDeadEnd  = "dead_end"
	StatusBlocked  = "blocked"
)

const (
	ExplorationRelationDerives   = "derives"
	ExplorationRelationTests     = "tests"
	ExplorationRelationDependsOn = "depends_on"
	ExplorationRelationSupports  = "supports"
)

func ValidType(t string) bool {
	switch t {
	case TypeDomain, TypeIP, TypeCIDR, TypeURL, TypeEmail, TypeService, TypeTechnology:
		return true
	}
	return false
}

func ValidRelation(r string) bool {
	switch r {
	case RelationSubdomainOf, RelationResolvesTo, RelationMemberOf, RelationServes,
		RelationHosts, RelationUses, RelationSameAs, RelationRelatedTo:
		return true
	}
	return false
}

func ValidKind(k string) bool {
	switch k {
	case KindTarget, KindHypothesis, KindAction, KindFinding, KindNote:
		return true
	}
	return false
}

func ValidStatus(s string) bool {
	switch s {
	case StatusPending, StatusActive, StatusExplored, StatusDeadEnd, StatusBlocked:
		return true
	}
	return false
}

func ValidExplorationRelation(r string) bool {
	switch r {
	case ExplorationRelationDerives, ExplorationRelationTests,
		ExplorationRelationDependsOn, ExplorationRelationSupports:
		return true
	}
	return false
}

// Normalize returns the canonical display form of a value and the dedup key it
// is stored under. The dedup key is what a second observation of the same thing
// collides with, so every type-specific spelling difference (case, trailing
// dot, default port, host bits below the mask, ...) has to collapse here and
// nowhere else.
func Normalize(assetType, value string) (string, string, error) {
	switch assetType {
	case TypeDomain:
		host, err := normalizeHostname(value)
		if err != nil {
			return "", "", err
		}
		if net.ParseIP(host) != nil {
			return "", "", fmt.Errorf("value %q is an ip address, use type %q", value, TypeIP)
		}
		return host, host, nil
	case TypeIP:
		ip, err := normalizeIP(value)
		if err != nil {
			return "", "", err
		}
		return ip, ip, nil
	case TypeCIDR:
		network, err := normalizeCIDR(value)
		if err != nil {
			return "", "", err
		}
		return network, network, nil
	case TypeURL:
		canonical, err := normalizeURL(value)
		if err != nil {
			return "", "", err
		}
		return canonical, canonical, nil
	case TypeEmail:
		canonical, err := normalizeEmail(value)
		if err != nil {
			return "", "", err
		}
		return canonical, canonical, nil
	case TypeService:
		canonical, err := normalizeService(value)
		if err != nil {
			return "", "", err
		}
		return canonical, canonical, nil
	case TypeTechnology:
		canonical, err := normalizeTechnology(value)
		if err != nil {
			return "", "", err
		}
		return canonical, canonical, nil
	default:
		return "", "", fmt.Errorf("unsupported asset type %q", assetType)
	}
}

// RegistrableDomain returns the eTLD+1 of a host: "a.b.example.co.uk" becomes
// "example.co.uk". A host that is an ip address or has no registrable domain
// ("localhost") is returned unchanged, so the caller can compare the result
// against the input to decide whether there is a parent to link.
func RegistrableDomain(host string) string {
	h := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if h == "" || net.ParseIP(h) != nil {
		return h
	}
	domain, err := publicsuffix.EffectiveTLDPlusOne(h)
	if err != nil {
		return h
	}
	return domain
}

// NormalizeCompanyName returns the trimmed display name and the case-folded
// dedup key of a company.
func NormalizeCompanyName(name string) (string, string, error) {
	canonical := strings.Join(strings.Fields(name), " ")
	if canonical == "" {
		return "", "", errors.New("company name is empty")
	}
	return canonical, strings.ToLower(canonical), nil
}

// NodeDedupKey is the identity of an exploration node inside one flow.
func NodeDedupKey(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

// Coverage is the per-flow share of the asset graph touched by the exploration
// graph and of the exploration nodes that reached a terminal status.
type Coverage struct {
	TotalNodes     int64
	CompletedNodes int64
	AnchoredAssets int64
	TotalAssets    int64
}

func (c Coverage) NodePercent() float64 {
	if c.TotalNodes == 0 {
		return 0
	}
	return 100 * float64(c.CompletedNodes) / float64(c.TotalNodes)
}

func (c Coverage) AssetPercent() float64 {
	if c.TotalAssets == 0 {
		return 0
	}
	return 100 * float64(c.AnchoredAssets) / float64(c.TotalAssets)
}

func normalizeHostname(value string) (string, error) {
	host := strings.TrimSpace(value)
	host = strings.TrimSuffix(host, ".")
	// A wildcard certificate name is an observation about the parent; the
	// "*." is not part of any host that resolves.
	host = strings.TrimPrefix(host, "*.")
	if host == "" {
		return "", errors.New("host is empty")
	}
	host = strings.ToLower(host)
	if ip := net.ParseIP(host); ip != nil {
		return normalizeIP(host)
	}
	if !isASCII(host) {
		ascii, err := idna.Lookup.ToASCII(host)
		if err != nil {
			return "", fmt.Errorf("invalid host %q: %w", value, err)
		}
		host = strings.ToLower(ascii)
	}
	if len(host) > 253 {
		return "", fmt.Errorf("host %q is longer than 253 characters", value)
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" {
			return "", fmt.Errorf("host %q has an empty label", value)
		}
		if len(label) > 63 {
			return "", fmt.Errorf("host %q has a label longer than 63 characters", value)
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if c == '-' || c == '_' || ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') {
				continue
			}
			return "", fmt.Errorf("host %q has an invalid character %q", value, c)
		}
	}
	return host, nil
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func normalizeIP(value string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return "", fmt.Errorf("invalid ip address %q", value)
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String(), nil
	}
	return ip.String(), nil
}

func normalizeCIDR(value string) (string, error) {
	_, network, err := net.ParseCIDR(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid network %q: %w", value, err)
	}
	return network.String(), nil
}

func normalizeURL(value string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid url %q: %w", value, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("url %q must be http or https", value)
	}
	if u.Host == "" {
		return "", fmt.Errorf("url %q has no host", value)
	}
	host, err := normalizeHostname(u.Hostname())
	if err != nil {
		return "", err
	}
	port := u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	hostPort := host
	if port != "" {
		hostPort = host + ":" + port
	}
	path := u.EscapedPath()
	if path == "/" {
		path = ""
	}
	canonical := scheme + "://" + hostPort + path
	if u.RawQuery != "" {
		canonical += "?" + u.RawQuery
	}
	return canonical, nil
}

func normalizeEmail(value string) (string, error) {
	canonical := strings.ToLower(strings.TrimSpace(value))
	at := strings.LastIndex(canonical, "@")
	if at <= 0 || at == len(canonical)-1 {
		return "", fmt.Errorf("invalid email %q", value)
	}
	local := canonical[:at]
	if strings.ContainsAny(local, " \t\"'<>@") {
		return "", fmt.Errorf("invalid email %q", value)
	}
	host, err := normalizeHostname(canonical[at+1:])
	if err != nil {
		return "", err
	}
	if net.ParseIP(host) != nil {
		return "", fmt.Errorf("invalid email %q: the domain is an ip address", value)
	}
	return local + "@" + host, nil
}

func normalizeService(value string) (string, error) {
	rest := strings.TrimSpace(value)
	proto := "tcp"
	if slash := strings.LastIndex(rest, "/"); slash >= 0 {
		proto = strings.ToLower(rest[slash+1:])
		rest = rest[:slash]
	}
	if proto != "tcp" && proto != "udp" {
		return "", fmt.Errorf("service %q has an unsupported protocol %q", value, proto)
	}
	host, portString, err := net.SplitHostPort(rest)
	if err != nil {
		return "", fmt.Errorf("service %q must be host:port[/tcp|/udp]: %w", value, err)
	}
	host, err = normalizeHostname(host)
	if err != nil {
		return "", err
	}
	port, err := strconv.Atoi(portString)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("service %q has an invalid port %q", value, portString)
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return host + ":" + strconv.Itoa(port) + "/" + proto, nil
}

func normalizeTechnology(value string) (string, error) {
	canonical := strings.ToLower(strings.Join(strings.Fields(value), " "))
	if canonical == "" {
		return "", errors.New("technology is empty")
	}
	return canonical, nil
}
