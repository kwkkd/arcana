package providers

import (
	"reflect"
	"strings"
	"sync"

	"github.com/kwkkd/arcana/pkg/providers/anthropic"
	"github.com/kwkkd/arcana/pkg/providers/bedrock"
	"github.com/kwkkd/arcana/pkg/providers/deepseek"
	"github.com/kwkkd/arcana/pkg/providers/gemini"
	"github.com/kwkkd/arcana/pkg/providers/glm"
	"github.com/kwkkd/arcana/pkg/providers/kimi"
	"github.com/kwkkd/arcana/pkg/providers/minimax"
	"github.com/kwkkd/arcana/pkg/providers/mistral"
	"github.com/kwkkd/arcana/pkg/providers/openai"
	"github.com/kwkkd/arcana/pkg/providers/pconfig"
	"github.com/kwkkd/arcana/pkg/providers/provider"
	"github.com/kwkkd/arcana/pkg/providers/qwen"
	"github.com/kwkkd/arcana/pkg/providers/xai"
)

type bundledCatalog struct {
	Pkg  string
	Load func() (pconfig.ModelsConfig, error)
}

var bundledCatalogs = []bundledCatalog{
	{"anthropic", anthropic.DefaultModels},
	{"bedrock", bedrock.DefaultModels},
	{"deepseek", deepseek.DefaultModels},
	{"gemini", gemini.DefaultModels},
	{"glm", glm.DefaultModels},
	{"kimi", kimi.DefaultModels},
	{"minimax", minimax.DefaultModels},
	{"mistral", mistral.DefaultModels},
	{"openai", openai.DefaultModels},
	{"qwen", qwen.DefaultModels},
	{"xai", xai.DefaultModels},
}

// BundledCatalogs is shared by every caller for the life of the process:
// callers must not modify the returned catalogues.
var BundledCatalogs = sync.OnceValue(func() map[provider.ProviderType]pconfig.ModelsConfig {
	catalogs := make(map[provider.ProviderType]pconfig.ModelsConfig, len(bundledCatalogs))
	for _, catalog := range bundledCatalogs {
		if models, err := catalog.Load(); err == nil {
			catalogs[provider.ProviderType(catalog.Pkg)] = models
		}
	}

	return catalogs
})

var (
	declaredCapabilitiesOnce sync.Once
	declaredCapabilities     map[string]pconfig.ModelConfig
)

func bareModelName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func declaredModelCapabilities() map[string]pconfig.ModelConfig {
	declaredCapabilitiesOnce.Do(func() {
		declaredCapabilities = buildCapabilityIndex(bundledCatalogs)
	})

	return declaredCapabilities
}

func buildCapabilityIndex(catalogs []bundledCatalog) map[string]pconfig.ModelConfig {
	index := make(map[string]pconfig.ModelConfig)
	conflicting := make(map[string]struct{})

	for _, catalog := range catalogs {
		models, err := catalog.Load()
		if err != nil {
			continue
		}
		for _, m := range models {
			if m.Reasoning == nil && m.Thinking == nil {
				continue
			}
			key := bareModelName(m.Name)
			if existing, ok := index[key]; ok {
				if !sameCapability(existing, m) {
					conflicting[key] = struct{}{}
				}
				continue
			}
			index[key] = m
		}
	}

	for key := range conflicting {
		delete(index, key)
	}

	return index
}

func sameCapability(a, b pconfig.ModelConfig) bool {
	return reflect.DeepEqual(a.Reasoning, b.Reasoning) && reflect.DeepEqual(a.Thinking, b.Thinking)
}

func EnrichCatalogCapabilities(models pconfig.ModelsConfig) pconfig.ModelsConfig {
	if len(models) == 0 {
		return models
	}

	declared := declaredModelCapabilities()
	enriched := make(pconfig.ModelsConfig, len(models))

	for i, m := range models {
		enriched[i] = m

		if m.Reasoning != nil || (m.Thinking != nil && !*m.Thinking) {
			continue
		}

		known, ok := declared[bareModelName(m.Name)]
		if !ok {
			continue
		}

		enriched[i].Reasoning = known.Reasoning
		if m.Thinking == nil {
			enriched[i].Thinking = known.Thinking
		}
	}

	return enriched
}
