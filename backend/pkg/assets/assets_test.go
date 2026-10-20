package assets

import "testing"

func TestAssets_Normalize(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		value     string
		want      string
		wantErr   bool
	}{
		{"domain lowercases and strips the trailing dot", TypeDomain, "Example.COM.", "example.com", false},
		{"domain strips a wildcard prefix", TypeDomain, "*.Dev.Acme.com", "dev.acme.com", false},
		{"domain punycodes unicode", TypeDomain, "bÜcher.de", "xn--bcher-kva.de", false},
		{"domain keeps underscores", TypeDomain, "bad_host.example.com", "bad_host.example.com", false},
		{"domain rejects an ip address", TypeDomain, "1.2.3.4", "", true},
		{"domain rejects an empty label", TypeDomain, "a..b", "", true},
		{"domain rejects a space", TypeDomain, "exa mple.com", "", true},
		{"domain rejects an empty value", TypeDomain, "", "", true},

		{"ip canonicalizes ipv4", TypeIP, " 10.0.0.1 ", "10.0.0.1", false},
		{"ip unwraps ipv4-mapped ipv6", TypeIP, "::ffff:1.2.3.4", "1.2.3.4", false},
		{"ip lowercases ipv6", TypeIP, "2001:DB8::1", "2001:db8::1", false},
		{"ip rejects an out-of-range address", TypeIP, "300.1.1.1", "", true},

		{"cidr masks host bits", TypeCIDR, "192.168.1.5/24", "192.168.1.0/24", false},
		{"cidr keeps an ipv6 network", TypeCIDR, "2001:db8:1::/64", "2001:db8:1::/64", false},
		{"cidr rejects a bare address", TypeCIDR, "192.168.1.0", "", true},
		{"cidr rejects an oversized prefix", TypeCIDR, "10.0.0.0/33", "", true},

		{"url lowercases the scheme and host", TypeURL, "HTTPS://Example.COM:443/Path", "https://example.com/Path", false},
		{"url drops a default port", TypeURL, "http://Example.com:80", "http://example.com", false},
		{"url keeps a non-default port", TypeURL, "http://example.com:8080/", "http://example.com:8080", false},
		{"url drops the fragment and the bare slash", TypeURL, "https://example.com/?a=b#frag", "https://example.com?a=b", false},
		{"url punycodes the host", TypeURL, "https://bücher.de/x", "https://xn--bcher-kva.de/x", false},
		{"url rejects a non-http scheme", TypeURL, "ftp://example.com", "", true},
		{"url rejects a missing host", TypeURL, "https://", "", true},

		{"email lowercases both sides", TypeEmail, "User@Example.COM", "user@example.com", false},
		{"email keeps a subdomain domain", TypeEmail, "user@sub.Example.com", "user@sub.example.com", false},
		{"email rejects a second at-sign", TypeEmail, "bad@@example.com", "", true},
		{"email rejects an ip domain", TypeEmail, "a@1.2.3.4", "", true},
		{"email rejects a missing at-sign", TypeEmail, "no-at-sign", "", true},

		{"service defaults the protocol", TypeService, "Example.com:443", "example.com:443/tcp", false},
		{"service lowercases the protocol", TypeService, "1.2.3.4:53/UDP", "1.2.3.4:53/udp", false},
		{"service brackets an ipv6 host", TypeService, "[2001:db8::1]:80", "[2001:db8::1]:80/tcp", false},
		{"service rejects a missing port", TypeService, "example.com", "", true},
		{"service rejects port zero", TypeService, "example.com:0", "", true},
		{"service rejects an unknown protocol", TypeService, "example.com:443/sctp", "", true},

		{"technology lowercases and collapses spaces", TypeTechnology, "  NGINX   1.24 ", "nginx 1.24", false},
		{"technology rejects an empty value", TypeTechnology, "   ", "", true},

		{"unsupported type is refused", "company", "acme", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, dedup, err := Normalize(tt.assetType, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q, %q) = %q, %q, nil; want an error", tt.assetType, tt.value, canonical, dedup)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q, %q) returned an error: %v", tt.assetType, tt.value, err)
			}
			if canonical != tt.want {
				t.Errorf("Normalize(%q, %q) = %q; want %q", tt.assetType, tt.value, canonical, tt.want)
			}
			if dedup != canonical {
				t.Errorf("dedup key of (%q, %q) = %q; want the canonical form %q", tt.assetType, tt.value, dedup, canonical)
			}
		})
	}
}

func TestAssets_RegistrableDomain(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{"multi-label public suffix", "a.b.example.co.uk", "example.co.uk"},
		{"already registrable", "example.com", "example.com"},
		{"uppercase and trailing dot", "EXAMPLE.COM.", "example.com"},
		{"ip is returned unchanged", "1.2.3.4", "1.2.3.4"},
		{"no registrable domain is returned unchanged", "localhost", "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RegistrableDomain(tt.host); got != tt.want {
				t.Errorf("RegistrableDomain(%q) = %q; want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestAssets_NormalizeCompanyName(t *testing.T) {
	name, dedup, err := NormalizeCompanyName("  Acme   Corp ")
	if err != nil {
		t.Fatalf("NormalizeCompanyName returned an error: %v", err)
	}
	if name != "Acme Corp" {
		t.Errorf("name = %q; want %q", name, "Acme Corp")
	}
	if dedup != "acme corp" {
		t.Errorf("dedup key = %q; want %q", dedup, "acme corp")
	}

	if _, _, err := NormalizeCompanyName("   "); err == nil {
		t.Error("NormalizeCompanyName accepted a blank name")
	}
}

func TestAssets_NodeDedupKey(t *testing.T) {
	if got := NodeDedupKey("  Find  Subdomains "); got != "find subdomains" {
		t.Errorf("NodeDedupKey = %q; want %q", got, "find subdomains")
	}
}

func TestAssets_Coverage_Percentages(t *testing.T) {
	if got := (Coverage{}).NodePercent(); got != 0 {
		t.Errorf("NodePercent with no nodes = %v; want 0", got)
	}
	if got := (Coverage{}).AssetPercent(); got != 0 {
		t.Errorf("AssetPercent with no assets = %v; want 0", got)
	}

	c := Coverage{TotalNodes: 4, CompletedNodes: 3, AnchoredAssets: 1, TotalAssets: 2}
	if got := c.NodePercent(); got != 75 {
		t.Errorf("NodePercent = %v; want 75", got)
	}
	if got := c.AssetPercent(); got != 50 {
		t.Errorf("AssetPercent = %v; want 50", got)
	}
}

func TestAssets_Validators(t *testing.T) {
	if !ValidType(TypeDomain) || ValidType("subdomain") || ValidType("") {
		t.Error("ValidType accepts the wrong set of types")
	}
	if !ValidRelation(RelationSubdomainOf) || ValidRelation("part_of") || ValidRelation("") {
		t.Error("ValidRelation accepts the wrong set of relations")
	}
	if !ValidKind(KindTarget) || ValidKind("todo") || ValidKind("") {
		t.Error("ValidKind accepts the wrong set of kinds")
	}
	if !ValidStatus(StatusExplored) || ValidStatus("open") || ValidStatus("") {
		t.Error("ValidStatus accepts the wrong set of statuses")
	}
	if !ValidExplorationRelation(ExplorationRelationDerives) || ValidExplorationRelation("owns") || ValidExplorationRelation("") {
		t.Error("ValidExplorationRelation accepts the wrong set of relations")
	}
}
