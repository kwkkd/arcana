package graph

import (
	"github.com/kwkkd/arcana/pkg/config"
	"github.com/kwkkd/arcana/pkg/controller"
	"github.com/kwkkd/arcana/pkg/database"
	"github.com/kwkkd/arcana/pkg/database/knowledge"
	"github.com/kwkkd/arcana/pkg/graph/subscriptions"
	"github.com/kwkkd/arcana/pkg/providers"
	"github.com/kwkkd/arcana/pkg/server/auth"
	"github.com/kwkkd/arcana/pkg/server/update"
	"github.com/kwkkd/arcana/pkg/templates"
	"github.com/kwkkd/arcana/pkg/timezone"

	"github.com/kwkkd/arcana/pkg/anonymizer"
	"github.com/sirupsen/logrus"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB              database.Querier
	Config          *config.Config
	Logger          *logrus.Entry
	TokenCache      *auth.TokenCache
	DefaultPrompter templates.Prompter
	ProvidersCtrl   providers.ProviderController
	Controller      controller.FlowController
	Subscriptions   subscriptions.SubscriptionsController
	Knowledge       knowledge.KnowledgeStore
	Replacer        anonymizer.Replacer
	Updates         *update.Service
	Timezones       *timezone.Catalog
}
