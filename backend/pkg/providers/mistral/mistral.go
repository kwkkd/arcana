package mistral

import (
	"embed"

	"github.com/kwkkd/arcana/pkg/config"
	"github.com/kwkkd/arcana/pkg/providers/openaicompat"
	"github.com/kwkkd/arcana/pkg/providers/pconfig"
	"github.com/kwkkd/arcana/pkg/providers/provider"
	"github.com/kwkkd/arcana/pkg/system"

	"github.com/vxcontrol/langchaingo/llms"
)

//go:embed config.yml models.yml
var configFS embed.FS

const MistralAgentModel = "mistral-small-latest"

const MistralToolCallIDTemplate = "{r:9:x}"

func BuildProviderConfig(configData []byte) (*pconfig.ProviderConfig, error) {
	defaultOptions := []llms.CallOption{
		llms.WithModel(MistralAgentModel),
		llms.WithN(1),
		llms.WithMaxTokens(4000),
	}

	providerConfig, err := pconfig.LoadConfigData(configData, defaultOptions)
	if err != nil {
		return nil, err
	}

	return providerConfig, nil
}

func DefaultProviderConfig() (*pconfig.ProviderConfig, error) {
	configData, err := configFS.ReadFile("config.yml")
	if err != nil {
		return nil, err
	}

	return BuildProviderConfig(configData)
}

func DefaultModels() (pconfig.ModelsConfig, error) {
	configData, err := configFS.ReadFile("models.yml")
	if err != nil {
		return nil, err
	}

	return pconfig.LoadModelsConfigData(configData)
}

func New(
	cfg *config.Config,
	providerName provider.ProviderName,
	providerConfig *pconfig.ProviderConfig,
) (provider.Provider, error) {
	httpClient, err := system.GetHTTPClient(cfg)
	if err != nil {
		return nil, err
	}

	models, err := DefaultModels()
	if err != nil {
		return nil, err
	}

	return openaicompat.New(openaicompat.Spec{
		Type:               provider.ProviderMistral,
		Model:              MistralAgentModel,
		ToolCallIDTemplate: MistralToolCallIDTemplate,
		APIKey:             cfg.MistralAPIKey,
		ServerURL:          cfg.MistralServerURL,
		Prefix:             cfg.MistralProvider,
		PreserveReasoning:  true,
		LegacyMaxTokens:    true,
	}, httpClient, models, providerName, providerConfig)
}
