package patterns

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kwkkd/arcana/pkg/anonymizer"
)

func TestLoadPatterns_UnknownKind_IsRefused(t *testing.T) {
	_, err := LoadPatterns(PatternListType(1))
	require.Error(t, err)
}

func TestLoadPatterns_ReturnsACopyTheCallerCannotPollute(t *testing.T) {
	first, err := LoadPatterns(PatternListTypeAll)
	require.NoError(t, err)
	require.NotEmpty(t, first.Patterns)
	first.Patterns[0].Name = "mutated"

	second, err := LoadPatterns(PatternListTypeAll)
	require.NoError(t, err)
	assert.NotEqual(t, "mutated", second.Patterns[0].Name)
}

func TestCatalogue_EveryExpressionCompilesAndNamesAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(catalogue))
	for _, p := range catalogue {
		_, err := regexp.Compile(p.Regex)
		require.NoError(t, err, p.Name)
		assert.False(t, seen[p.Name], "duplicate name %q", p.Name)
		seen[p.Name] = true
	}
}

func TestCatalogue_MasksEveryKindOfSecretItNames(t *testing.T) {
	list, err := LoadPatterns(PatternListTypeAll)
	require.NoError(t, err)

	r, err := anonymizer.NewReplacer(list.Regexes(), list.Names())
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		text   string
		secret string
		keeps  string
	}{
		{
			name:   "Private Key Block",
			text:   "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAfakefakefake\n-----END RSA PRIVATE KEY-----",
			secret: "MIIEowIBAAKCAQEAfakefakefake",
		},
		{
			name:   "AWS Access Key ID",
			text:   "the key is AKIAIOSFODNN7EXAMPLE here",
			secret: "AKIAIOSFODNN7EXAMPLE",
		},
		{
			name:   "AWS Secret Access Key",
			text:   "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			secret: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			keeps:  "aws_secret_access_key = ",
		},
		{
			name:   "Google API Key",
			text:   "key AIzaSyA12345678901234567890123456789012 end",
			secret: "AIzaSyA12345678901234567890123456789012",
		},
		{
			name:   "Google OAuth Token",
			text:   "token ya29.a0AfH6SMBBx1234567890abcdefghijklmnopqrst here",
			secret: "ya29.a0AfH6SMBBx1234567890abcdefghijklmnopqrst",
		},
		{
			name:   "OpenAI API Key",
			text:   "OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyz123456",
			secret: "sk-abcdefghijklmnopqrstuvwxyz123456",
		},
		{
			name:   "OpenAI Project Key",
			text:   "key sk-proj-abcdefghijklmnopqrstuvwxyz123456 here",
			secret: "sk-proj-abcdefghijklmnopqrstuvwxyz123456",
		},
		{
			name:   "Anthropic API Key",
			text:   "key sk-ant-api03-abcdefghijklmnopqrstuvwxyz123456 here",
			secret: "sk-ant-api03-abcdefghijklmnopqrstuvwxyz123456",
		},
		{
			name:   "GitHub Token",
			text:   "remote https://ghp_abcdefghijklmnopqrstuvwxyz1234567890@github.com",
			secret: "ghp_abcdefghijklmnopqrstuvwxyz1234567890",
		},
		{
			name:   "GitHub Fine-Grained PAT",
			text:   "token github_pat_11ABCDEFG0123456789_abcdefghijklmnopqrstuvwxyz end",
			secret: "github_pat_11ABCDEFG0123456789_abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:   "GitLab Token",
			text:   "token glpat-abcdefghijklmnopqrst end",
			secret: "glpat-abcdefghijklmnopqrst",
		},
		{
			name:   "Slack Token",
			text:   "token xoxb-123456789012-abcdefghijklmnop end",
			secret: "xoxb-123456789012-abcdefghijklmnop",
		},
		{
			name:   "Stripe Key",
			text:   "key sk_live_abcdefghijklmnop1234 end",
			secret: "sk_live_abcdefghijklmnop1234",
		},
		{
			name:   "SendGrid Key",
			text:   "key SG.abcdefghijklmnopqrstuv.ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQ end",
			secret: "SG.abcdefghijklmnopqrstuv.ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQ",
		},
		{
			name:   "npm Token",
			text:   "token npm_abcdefghijklmnopqrstuvwxyz1234567890 end",
			secret: "npm_abcdefghijklmnopqrstuvwxyz1234567890",
		},
		{
			name:   "PyPI Token",
			text:   "token pypi-AgEIcHlwaS5vcmcabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN end",
			secret: "pypi-AgEIcHlwaS5vcmcabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN",
		},
		{
			name:   "Docker Hub Token",
			text:   "token dckr_pat_abcdefghijklmnopqrstuvwxyz1 end",
			secret: "dckr_pat_abcdefghijklmnopqrstuvwxyz1",
		},
		{
			name:   "HashiCorp Vault Token",
			text:   "token hvs.CAESIJabcdefghijklmnopqrstuv end",
			secret: "hvs.CAESIJabcdefghijklmnopqrstuv",
		},
		{
			name:   "JSON Web Token",
			text:   "header eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdefghijklmnopqrst end",
			secret: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdefghijklmnopqrst",
		},
		{
			name:   "Bearer Token",
			text:   "Authorization: Bearer abcDEF1234567890xyz",
			secret: "abcDEF1234567890xyz",
			keeps:  "Authorization: Bearer ",
		},
		{
			name:   "Connection String",
			text:   "postgres://admin:hunter2@db.internal:5432/app",
			secret: "admin:hunter2@",
			keeps:  ":5432/app",
		},
		{
			name:   "URL Password",
			text:   "https://alice:s3cr3tpassword@example.com",
			secret: "s3cr3tpassword",
			keeps:  "https://alice:",
		},
		{
			name:   "Secret Assignment",
			text:   "api_key = SECRETVALUE12345",
			secret: "SECRETVALUE12345",
			keeps:  "api_key = ",
		},
		{
			name:   "Email Address",
			text:   "reach alice@corp.example today",
			secret: "alice@corp.example",
			keeps:  "reach ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			masked := r.ReplaceString(tc.text)

			assert.Contains(t, masked, "§*"+tc.name+"*§")
			assert.NotContains(t, masked, tc.secret)
			if tc.keeps != "" {
				assert.Contains(t, masked, tc.keeps)
			}
		})
	}
}
