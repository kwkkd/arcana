# Testing Tools

Companion CLIs for checking a deployment before and after a flow: `ctester` for provider and agent conformance, `etester` for embeddings and the vector store, and `ftester` for individual tool calls.

## Testing LLM Agents

ARCANA includes a powerful utility called `ctester` for testing and validating LLM agent capabilities. This tool helps ensure your LLM provider configurations work correctly with different agent types, allowing you to optimize model selection for each specific agent role.

The utility features parallel testing of multiple agents, detailed reporting, and flexible configuration options.

### Key Features

- **Parallel Testing**: Tests multiple agents simultaneously for faster results
- **Comprehensive Test Suite**: Evaluates basic completion, JSON responses, function calling, and penetration testing knowledge
- **Detailed Reporting**: Generates markdown reports with success rates and performance metrics
- **Flexible Configuration**: Test specific agents or test groups as needed
- **Specialized Test Groups**: Includes domain-specific tests for cybersecurity and penetration testing scenarios

### Usage Scenarios

#### For Developers (with local Go environment)

If you've cloned the repository and have Go installed:

```bash
# Default configuration with .env file
cd backend
go run cmd/ctester/*.go -verbose

# Custom provider configuration
go run cmd/ctester/*.go -config ../examples/configs/openrouter.provider.yml -verbose

# Generate a report file
go run cmd/ctester/*.go -config ../examples/configs/deepinfra.provider.yml -report ../test-report.md

# Test specific agent types only
go run cmd/ctester/*.go -agents simple,simple_json,primary_agent -verbose

# Test specific test groups only
go run cmd/ctester/*.go -groups basic,advanced -verbose
```

#### For Users (using Docker image)

If you prefer to use the pre-built Docker image without setting up a development environment:

```bash
# Using Docker to test with default environment
docker run --rm -v $(pwd)/.env:/opt/pentagi/.env vxcontrol/pentagi /opt/pentagi/bin/ctester -verbose

# Test with your custom provider configuration
docker run --rm \
  -v $(pwd)/.env:/opt/pentagi/.env \
  -v $(pwd)/my-config.yml:/opt/pentagi/config.yml \
  vxcontrol/pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/config.yml -agents simple,primary_agent,coder -verbose

# Generate a detailed report
docker run --rm \
  -v $(pwd)/.env:/opt/pentagi/.env \
  -v $(pwd):/opt/pentagi/output \
  vxcontrol/pentagi /opt/pentagi/bin/ctester -report /opt/pentagi/output/report.md
```

#### Using Pre-configured Providers

The Docker image comes with built-in support for major providers (OpenAI, Anthropic, Gemini, Ollama) and pre-configured provider files for additional services (OpenRouter, OpenCode, Atlas, OrcaRouter, DeepInfra, DeepSeek, Moonshot, Novita, xAI):

```bash
# Test with OpenRouter configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/openrouter.provider.yml

# Test with OpenCode Go plan configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/opencode.provider.yml

# Test with DeepInfra configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/deepinfra.provider.yml

# Test with DeepSeek configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type deepseek

# Test with GLM configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type glm

# Test with Kimi configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type kimi

# Test with Qwen configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type qwen

# Test with DeepSeek configuration file for custom provider
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/deepseek.provider.yml

# Test with Moonshot configuration file for custom provider
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/moonshot.provider.yml

# Test with Novita configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/novita.provider.yml

# Test with xAI configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/xai.provider.yml

# Test with OpenAI configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type openai

# Test with Anthropic configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type anthropic

# Test with Gemini configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type gemini

# Test with AWS Bedrock configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -type bedrock

# Test with Custom OpenAI configuration
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/custom-openai.provider.yml

# Test with Azure OpenAI configuration (needs LLM_SERVER_API_TYPE=azure, see Using Azure OpenAI)
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/azure-openai.provider.yml

# Test with Ollama configuration (local inference)
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/ollama-llama318b.provider.yml

# Test with Ollama Qwen3 32B configuration (requires custom model creation)
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/ollama-qwen332b-fp16-tc.provider.yml

# Test with Ollama QwQ 32B configuration (requires custom model creation and 71.3GB VRAM)
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/ollama-qwq32b-fp16-tc.provider.yml
```

To use these configurations, your `.env` file only needs to contain:

```
LLM_SERVER_URL=https://openrouter.ai/api/v1      # or https://api.deepinfra.com/v1/openai or https://api.openai.com/v1 or https://opencode.ai/zen/go/v1 or https://api.novita.ai/openai or https://api.atlascloud.ai/v1 or https://api.orcarouter.ai/v1 or https://api.x.ai/v1
LLM_SERVER_KEY=your_api_key
LLM_SERVER_MODEL=                                # Leave empty, as models are specified in the config
LLM_SERVER_CONFIG_PATH=/opt/pentagi/conf/openrouter.provider.yml  # or deepinfra.provider.ymll or opencode.provider.ymll or custom-openai.provider.yml or novita.provider.yml or atlas.provider.yml or orcarouter.provider.yml or xai.provider.yml
LLM_SERVER_PROVIDER=                             # Provider name for LiteLLM proxy (e.g., openrouter, deepseek, moonshot, novita, opencode, orcarouter, xai)
LLM_SERVER_PRESERVE_REASONING=false              # Preserve reasoning content in multi-turn conversations (required by Moonshot, default: false)

# For OpenAI (official API)
OPEN_AI_KEY=your_openai_api_key                  # Your OpenAI API key
OPEN_AI_SERVER_URL=https://api.openai.com/v1     # OpenAI API endpoint

# For Anthropic (Claude models)
ANTHROPIC_API_KEY=your_anthropic_api_key         # Your Anthropic API key
ANTHROPIC_SERVER_URL=https://api.anthropic.com/v1  # Anthropic API endpoint

# For Gemini (Google AI)
GEMINI_API_KEY=your_gemini_api_key               # Your Google AI API key
GEMINI_SERVER_URL=https://generativelanguage.googleapis.com  # Google AI API endpoint

# For AWS Bedrock (enterprise foundation models)
BEDROCK_REGION=us-east-1                         # AWS region for Bedrock service
# Authentication (choose one method, priority: DefaultAuth > BearerToken > AccessKey):
BEDROCK_DEFAULT_AUTH=false                       # Use AWS SDK credential chain (env vars, EC2 role, ~/.aws/credentials)
BEDROCK_BEARER_TOKEN=                            # Bearer token authentication (takes priority over static credentials)
BEDROCK_ACCESS_KEY_ID=your_aws_access_key        # AWS access key ID (static credentials)
BEDROCK_SECRET_ACCESS_KEY=your_aws_secret_key    # AWS secret access key (static credentials)
BEDROCK_SESSION_TOKEN=                           # AWS session token (optional, for temporary credentials with static auth)
BEDROCK_SERVER_URL=                              # Optional custom Bedrock endpoint (VPC endpoints, local testing)
BEDROCK_CONFIG_PATH=                             # Optional path to a custom YAML provider config (overrides built-in model/pricing definitions)

# For Ollama (local server or cloud)
OLLAMA_SERVER_URL=                               # Local: http://ollama-server:11434, Cloud: https://ollama.com
OLLAMA_SERVER_API_KEY=                           # Required for Ollama Cloud (https://ollama.com/settings/keys), leave empty for local
OLLAMA_SERVER_MODEL=
OLLAMA_SERVER_CONFIG_PATH=
OLLAMA_SERVER_PULL_MODELS_TIMEOUT=
OLLAMA_SERVER_PULL_MODELS_ENABLED=
OLLAMA_SERVER_LOAD_MODELS_ENABLED=

# For DeepSeek (Chinese AI with strong reasoning)
DEEPSEEK_API_KEY=                                # DeepSeek API key
DEEPSEEK_SERVER_URL=https://api.deepseek.com     # DeepSeek API endpoint
DEEPSEEK_PROVIDER=                               # Optional: LiteLLM prefix (e.g., 'deepseek')

# For GLM (Zhipu AI)
GLM_API_KEY=                                     # GLM API key
GLM_SERVER_URL=https://api.z.ai/api/paas/v4      # GLM API endpoint (international)
GLM_PROVIDER=                                    # Optional: LiteLLM prefix (e.g., 'zai')

# For Kimi (Moonshot AI)
KIMI_API_KEY=                                    # Kimi API key
KIMI_SERVER_URL=https://api.moonshot.ai/v1       # Kimi API endpoint (international)
KIMI_PROVIDER=                                   # Optional: LiteLLM prefix (e.g., 'moonshot')

# For Qwen Cloud or Alibaba Cloud Model Studio
QWEN_API_KEY=                                    # Direct provider or gateway API key
QWEN_SERVER_URL=https://dashscope-us.aliyuncs.com/compatible-mode/v1  # Direct DashScope or gateway URL
QWEN_PROVIDER=                                   # Gateway prefix: 'qwen_cloud' or 'dashscope'; empty for direct DashScope

# For Ollama (local inference) use variables above
OLLAMA_SERVER_URL=http://localhost:11434
OLLAMA_SERVER_MODEL=llama3.1:8b-instruct-q8_0
OLLAMA_SERVER_CONFIG_PATH=/opt/pentagi/conf/ollama-llama318b.provider.yml
OLLAMA_SERVER_PULL_MODELS_ENABLED=false
OLLAMA_SERVER_LOAD_MODELS_ENABLED=false
```

#### Using OpenAI with Unverified Organizations

For OpenAI accounts with unverified organizations that don't have access to the latest reasoning models (o1, o3, o4-mini), you need to use a custom configuration.

To use OpenAI with unverified organization accounts, configure your `.env` file as follows:

```bash
LLM_SERVER_URL=https://api.openai.com/v1
LLM_SERVER_KEY=your_openai_api_key
LLM_SERVER_MODEL=                                # Leave empty, models are specified in config
LLM_SERVER_CONFIG_PATH=/opt/pentagi/conf/custom-openai.provider.yml
```

This configuration uses the pre-built `custom-openai.provider.yml` file that maps all agent types to models available for unverified organizations, using `o3-mini` instead of models like `o1`, `o3`, and `o4-mini`.

You can test this configuration using:

```bash
# Test with custom OpenAI configuration for unverified accounts
docker exec -it pentagi /opt/pentagi/bin/ctester -config /opt/pentagi/conf/custom-openai.provider.yml
```

#### Using LiteLLM Proxy

When using LiteLLM proxy to access various LLM providers, model names are prefixed with the provider name (e.g., `moonshot/kimi-2.5` instead of `kimi-2.5`). To use the same provider configuration files with both direct API access and LiteLLM proxy, set the `LLM_SERVER_PROVIDER` variable:

```bash
# Direct access to Moonshot API
LLM_SERVER_URL=https://api.moonshot.ai/v1
LLM_SERVER_KEY=your_moonshot_api_key
LLM_SERVER_CONFIG_PATH=/opt/pentagi/conf/moonshot.provider.yml
LLM_SERVER_PROVIDER=                             # Empty for direct access

# Access via LiteLLM proxy
LLM_SERVER_URL=http://litellm-proxy:4000
LLM_SERVER_KEY=your_litellm_api_key
LLM_SERVER_CONFIG_PATH=/opt/pentagi/conf/moonshot.provider.yml
LLM_SERVER_PROVIDER=moonshot                     # Provider prefix for LiteLLM
```

With `LLM_SERVER_PROVIDER=moonshot`, the system automatically prefixes all model names from the configuration file with `moonshot/`, making them compatible with LiteLLM's model naming convention.

**LiteLLM Provider Name Mapping:**

When using LiteLLM proxy, set the corresponding `*_PROVIDER` variable to enable model prefixing:

- `deepseek` - for DeepSeek models (`DEEPSEEK_PROVIDER=deepseek` → `deepseek/deepseek-v4-flash`)
- `zai` - for GLM models (`GLM_PROVIDER=zai` → `zai/glm-4`)
- `moonshot` - for Kimi models (`KIMI_PROVIDER=moonshot` → `moonshot/kimi-k3`)
- `dashscope` - for Qwen models (`QWEN_PROVIDER=dashscope` → `dashscope/qwen-plus`)
- `openai`, `anthropic`, `gemini` - for major cloud providers
- `opencode` - for OpenCode Go plan
- `openrouter` - for OpenRouter aggregator
- `orcarouter` - for OrcaRouter aggregator
- `deepinfra` - for DeepInfra hosting
- `novita` - for Novita AI
- `xai` - for xAI (`grok-*` models)
- Any other provider name configured in your LiteLLM instance

**Example with LiteLLM:**
```bash
# Use DeepSeek models via LiteLLM proxy with model prefixing
DEEPSEEK_API_KEY=your_litellm_proxy_key
DEEPSEEK_SERVER_URL=http://litellm-proxy:4000
DEEPSEEK_PROVIDER=deepseek  # Models become deepseek/deepseek-v4-flash, deepseek/deepseek-v4-pro for LiteLLM

# Direct DeepSeek API usage (no prefix needed)
DEEPSEEK_API_KEY=your_deepseek_api_key
DEEPSEEK_SERVER_URL=https://api.deepseek.com
# Leave DEEPSEEK_PROVIDER empty
```

This approach allows you to:
- Use the same configuration files for both direct and proxied access
- Switch between providers without modifying configuration files
- Easily test different routing strategies with LiteLLM

#### Running Tests in a Production Environment

If you already have a running ARCANA container and want to test the current configuration:

```bash
# Run ctester in an existing container using current environment variables
docker exec -it pentagi /opt/pentagi/bin/ctester -verbose

# Test specific agent types with deterministic ordering
docker exec -it pentagi /opt/pentagi/bin/ctester -agents simple,primary_agent,pentester -groups basic,knowledge -verbose

# Generate a report file inside the container
docker exec -it pentagi /opt/pentagi/bin/ctester -report /opt/pentagi/data/agent-test-report.md

# Access the report from the host
docker cp pentagi:/opt/pentagi/data/agent-test-report.md ./
```

### Command-line Options

The utility accepts several options:

- `-env <path>` - Path to environment file (default: `.env`)
- `-type <provider>` - Provider type: `custom`, `openai`, `anthropic`, `ollama`, `bedrock`, `gemini` (default: `custom`)
- `-config <path>` - Path to custom provider config (default: from `LLM_SERVER_CONFIG_PATH` env variable)
- `-tests <path>` - Path to custom tests YAML file (optional)
- `-report <path>` - Path to write the report file (optional)
- `-agents <list>` - Comma-separated list of agent types to test (default: `all`)
- `-groups <list>` - Comma-separated list of test groups to run (default: `all`)
- `-verbose` - Enable verbose output with detailed test results for each agent

### Available Agent Types

Agents are tested in the following deterministic order:

1. **simple** - Basic completion tasks
2. **simple_json** - JSON-structured responses
3. **primary_agent** - Main reasoning agent
4. **assistant** - Interactive assistant mode
5. **generator** - Content generation
6. **refiner** - Content refinement and improvement
7. **adviser** - Expert advice and consultation
8. **reflector** - Self-reflection and analysis
9. **searcher** - Information gathering and search
10. **enricher** - Data enrichment and expansion
11. **coder** - Code generation and analysis
12. **installer** - Installation and setup tasks
13. **pentester** - Penetration testing and security assessment

### Available Test Groups

- **basic** - Fundamental completion and prompt response tests
- **advanced** - Complex reasoning and function calling tests
- **json** - JSON format validation and structure tests (specifically designed for `simple_json` agent)
- **knowledge** - Domain-specific cybersecurity and penetration testing knowledge tests

> **Note**: The `json` test group is specifically designed for the `simple_json` agent type, while all other agents are tested with `basic`, `advanced`, and `knowledge` groups. This specialization ensures optimal testing coverage for each agent's intended purpose.

### Example Provider Configuration

Provider configuration defines which models to use for different agent types:

```yaml
simple:
  model: "provider/model-name"
  temperature: 0.7
  top_p: 0.95
  n: 1
  max_tokens: 4000

simple_json:
  model: "provider/model-name"
  temperature: 0.7
  top_p: 1.0
  n: 1
  max_tokens: 4000
  json: true

# ... other agent types ...
```

### Optimization Workflow

1. **Create a baseline**: Run tests with default configuration to establish benchmark performance
2. **Analyze agent-specific performance**: Review the deterministic agent ordering to identify underperforming agents
3. **Test specialized configurations**: Experiment with different models for each agent type using provider-specific configs
4. **Focus on domain knowledge**: Pay special attention to knowledge group tests for cybersecurity expertise
5. **Validate function calling**: Ensure tool-based tests pass consistently for critical agent types
6. **Compare results**: Look for the best success rate and performance across all test groups
7. **Deploy optimal configuration**: Use in production with your optimized setup

This tool helps ensure your AI agents are using the most effective models for their specific tasks, improving reliability while optimizing costs.

## Embedding Configuration and Testing

ARCANA uses vector embeddings for semantic search, knowledge storage, and memory management. The system supports multiple embedding providers that can be configured according to your needs and preferences.

### Supported Embedding Providers

ARCANA supports the following embedding providers:

- **OpenAI** (default): Uses OpenAI's text embedding models
- **Ollama**: Local embedding model through Ollama
- **Mistral**: Mistral AI's embedding models
- **Jina**: Jina AI's embedding service
- **HuggingFace**: Models from HuggingFace
- **GoogleAI**: Google's embedding models
- **VoyageAI**: VoyageAI's embedding models

> **OpenAI-compatible third parties**: any provider exposing OpenAI's `/embeddings` API can be plugged in via `EMBEDDING_PROVIDER=openai` with a custom `EMBEDDING_URL`. For example, **Qwen DashScope** offers `text-embedding-v4` through the `/compatible-mode/v1` endpoint (International and Chinese Mainland regions only — the US region does not expose embeddings). See the [Qwen Alternative Integrations](providers.md#alternative-integrations) subsection for the full configuration snippet.

<details>
<summary><b>Embedding Provider Configuration</b> (click to expand)</summary>

### Environment Variables

To configure the embedding provider, set the following environment variables in your `.env` file:

```bash
# Primary embedding configuration
EMBEDDING_PROVIDER=openai       # Provider type (openai, ollama, mistral, jina, huggingface, googleai, voyageai)
EMBEDDING_MODEL=text-embedding-3-small  # Model name to use
EMBEDDING_URL=                  # Optional custom API endpoint
EMBEDDING_KEY=                  # API key for the provider (if required)
EMBEDDING_BATCH_SIZE=100        # Number of documents to process in a batch
EMBEDDING_STRIP_NEW_LINES=true  # Whether to remove new lines from text before embedding
EMBEDDING_MAX_TEXT_BYTES=8192   # Max bytes of text sent to embedding model per document (byte proxy for token limit)

# Advanced settings
PROXY_URL=                      # Optional proxy for all backend LLM, embedding, search and update-check calls
HTTP_CLIENT_TIMEOUT=600         # Timeout in seconds for external API calls (default: 600, 0 = no timeout)
TERMINAL_TOOL_TIMEOUT=1200      # Default timeout in seconds for terminal tool commands when timeout=0 or negative (range: 1–10800; values <= 0 or above 10800 are clamped to 10800 = 3 hours)

# SSL/TLS Certificate Configuration (for external communication with LLM backends and tool servers)
EXTERNAL_SSL_CA_PATH=           # Path to custom CA certificate file (PEM format) inside the container
                                # Must point to /opt/pentagi/ssl/ directory (e.g., /opt/pentagi/ssl/ca-bundle.pem)
EXTERNAL_SSL_INSECURE=false     # Skip certificate verification (use only for testing)
```

<details>
<summary><b>How to Add Custom CA Certificates</b> (click to expand)</summary>

If you see this error: `tls: failed to verify certificate: x509: certificate signed by unknown authority`

**Step 1:** Get your CA certificate bundle in PEM format (can contain multiple certificates)

**Step 2:** Place the file in the SSL directory on your host machine:
```bash
# Default location (if PENTAGI_SSL_DIR is not set)
cp ca-bundle.pem ./pentagi-ssl/

# Or custom location (if using PENTAGI_SSL_DIR in docker-compose.yml)
cp ca-bundle.pem /path/to/your/ssl/dir/
```

**Step 3:** Set the path in `.env` file (path must be inside the container):
```bash
# The volume pentagi-ssl is mounted to /opt/pentagi/ssl inside the container
EXTERNAL_SSL_CA_PATH=/opt/pentagi/ssl/ca-bundle.pem
EXTERNAL_SSL_INSECURE=false
```

**Step 4:** Restart ARCANA:
```bash
docker compose restart pentagi
```

**Notes:**
- The `pentagi-ssl` volume is mounted to `/opt/pentagi/ssl` inside the container
- You can change host directory using `PENTAGI_SSL_DIR` variable in docker-compose.yml
- File supports multiple certificates and intermediate CAs in one PEM file
- Use `EXTERNAL_SSL_INSECURE=true` only for testing (not recommended for production)

</details>

### Provider-Specific Limitations

Each provider has specific limitations and supported features:

- **OpenAI**: Supports all configuration options
- **Ollama**: Does not support `EMBEDDING_KEY` as it uses local models
- **Mistral**: Supports all configuration options
- **Jina**: Supports all configuration options
- **HuggingFace**: Requires `EMBEDDING_KEY` and supports all other options
- **GoogleAI**: Does not support `EMBEDDING_URL`, requires `EMBEDDING_KEY`
- **VoyageAI**: Supports all configuration options

If `EMBEDDING_URL` and `EMBEDDING_KEY` are both left empty, the system uses the corresponding LLM provider's server and key together (e.g., `OPEN_AI_SERVER_URL` and `OPEN_AI_KEY` when `EMBEDDING_PROVIDER=openai`). A key set in `EMBEDDING_KEY` without `EMBEDDING_URL` goes to the provider's public endpoint, and an `EMBEDDING_URL` set without a key receives the LLM provider's key only when it is that provider's own server. So an OpenAI-compatible endpoint of another vendor needs its key in `EMBEDDING_KEY`, and a keyless local server (LocalAI, vLLM, TEI) takes any non-empty value there. Whenever the Mistral embedder uses `MISTRAL_SERVER_URL`, it also names the model the way that server expects: `MISTRAL_PROVIDER=mistral` turns `mistral-embed` into `mistral/mistral-embed`, and a model that already starts with `mistral/` is sent as is.

### Why Consistent Embedding Providers Matter

It's crucial to use the same embedding provider consistently because:

1. **Vector Compatibility**: Different providers produce vectors with different dimensions and mathematical properties
2. **Semantic Consistency**: Changing providers can break semantic similarity between previously embedded documents
3. **Memory Corruption**: Mixed embeddings can lead to poor search results and broken knowledge base functionality

If you change your embedding provider, you should flush and reindex your entire knowledge base (see `etester` utility below).

</details>

### Embedding Tester Utility (etester)

ARCANA includes a specialized `etester` utility for testing, managing, and debugging embedding functionality. This tool is essential for diagnosing and resolving issues related to vector embeddings and knowledge storage.

<details>
<summary><b>Etester Commands</b> (click to expand)</summary>

```bash
# Test embedding provider and database connection
cd backend
go run cmd/etester/main.go test -verbose

# Show statistics about the embedding database
go run cmd/etester/main.go info

# Delete all documents from the embedding database (use with caution!)
go run cmd/etester/main.go flush

# Recalculate embeddings for all documents (after changing provider)
go run cmd/etester/main.go reindex

# Search for documents in the embedding database
go run cmd/etester/main.go search -query "How to install PostgreSQL" -limit 5
```

### Using Docker

If you're running ARCANA in Docker, you can use etester from within the container:

```bash
# Test embedding provider
docker exec -it pentagi /opt/pentagi/bin/etester test

# Show detailed database information
docker exec -it pentagi /opt/pentagi/bin/etester info -verbose
```

### Advanced Search Options

The `search` command supports various filters to narrow down results:

```bash
# Filter by document type
docker exec -it pentagi /opt/pentagi/bin/etester search -query "Security vulnerability" -doc_type guide -threshold 0.8

# Filter by flow ID
docker exec -it pentagi /opt/pentagi/bin/etester search -query "Code examples" -doc_type code -flow_id 42

# All available search options
docker exec -it pentagi /opt/pentagi/bin/etester search -help
```

Available search parameters:
- `-query STRING`: Search query text (required)
- `-doc_type STRING`: Filter by document type (answer, memory, guide, code)
- `-flow_id NUMBER`: Filter by flow ID (positive number)
- `-answer_type STRING`: Filter by answer type (guide, vulnerability, code, tool, other)
- `-guide_type STRING`: Filter by guide type (install, configure, use, pentest, development, other)
- `-limit NUMBER`: Maximum number of results (default: 3)
- `-threshold NUMBER`: Similarity threshold (0.0-1.0, default: 0.7)

### Memory Lifecycle Across Flows

ARCANA stores several kinds of vector documents, and they serve different purposes:

- `memory` captures flow-specific execution history such as tool results and agent observations
- `guide`, `answer`, and `code` are intended for reusable knowledge that can help future runs

If you want to inspect what happened in one engagement, search the vector store with the related `flow_id`. If you want knowledge to survive beyond a single run, store the durable result explicitly as a `guide`, `answer`, or `code` document instead of relying on execution memory alone.

For example, if a target has recurring setup notes, authentication quirks, or target-specific testing methodology, instruct the agent to save that information as a `guide` and search for it at the beginning of the next engagement. This is the safest current workflow when you want a new flow to start with reusable context.

Flow deletion removes the flow from normal queries through ARCANA's soft-delete mechanism, so reusable knowledge should be treated as a separate concern from per-flow execution history. If you enable the optional Graphiti knowledge graph described in [graphiti.md](graphiti.md), treat its current search context as scoped to the active flow or engagement unless you explicitly build a separate cross-flow reuse workflow.

### Common Troubleshooting Scenarios

1. **After changing embedding provider**: Always run `flush` or `reindex` to ensure consistency
2. **Poor search results**: Try adjusting the similarity threshold or check if embeddings are correctly generated
3. **Database connection issues**: Verify PostgreSQL is running with pgvector extension installed
4. **Missing API keys**: Check environment variables for your chosen embedding provider

</details>

### Troubleshooting: Flow Stalls or Hangs Without Progress

If a flow starts but then appears to wait indefinitely with no subtasks progressing, a common cause is an embedding provider that is misconfigured or unreachable. ARCANA uses the embedding provider to store and search vector memory while a flow runs, so embedding calls that fail or hang can leave a flow waiting instead of advancing.

**1. Check the container logs first.** Embedding errors surface in the ARCANA logs:

```bash
docker logs pentagi
```

Look for embedding-related failures such as authentication errors (401/403), wrong-model or not-found errors (404), connection timeouts, or TLS certificate errors. These point at the embedding provider configuration rather than at the flow itself.

**2. Validate the provider with etester.** The [Embedding Tester Utility (etester)](#embedding-tester-utility-etester) checks both the embedding provider and the database connection without starting a flow:

```bash
docker exec -it pentagi /opt/pentagi/bin/etester test -verbose
```

A failing `test` confirms the problem is in the embedding configuration rather than in the flow.

**3. Verify the configuration.** Check the following in your `.env` file against the [Supported Embedding Providers](#supported-embedding-providers) list and each provider's documented limitations:

- `EMBEDDING_PROVIDER` is one of the supported providers (default `openai`).
- `EMBEDDING_MODEL` is a valid model name for that provider.
- `EMBEDDING_URL` and `EMBEDDING_KEY` are correct for the provider. If both are left empty, ARCANA falls back to the matching LLM provider settings (for example `OPEN_AI_KEY` and `OPEN_AI_SERVER_URL` when `EMBEDDING_PROVIDER=openai`), so a missing or wrong key there can break embeddings too.
- The endpoint is reachable from inside the container. If outbound calls go through a proxy, confirm `PROXY_URL` is set; if calls hang rather than fail quickly, `HTTP_CLIENT_TIMEOUT` controls how long ARCANA waits on the provider before giving up.

> **Changing provider?** If you switch embedding providers after data has already been indexed, run `flush` or `reindex` with etester so old and new vectors are not mixed. See [Why Consistent Embedding Providers Matter](#why-consistent-embedding-providers-matter) above.

## Function Testing with ftester

ARCANA includes a versatile utility called `ftester` for debugging, testing, and developing specific functions and AI agent behaviors. While `ctester` focuses on testing LLM model capabilities, `ftester` allows you to directly invoke individual system functions and AI agent components with precise control over execution context.

### Key Features

- **Direct Function Access**: Test individual functions without running the entire system
- **Mock Mode**: Test functions without a live ARCANA deployment using built-in mocks
- **Interactive Input**: Fill function arguments interactively for exploratory testing
- **Detailed Output**: Color-coded terminal output with formatted responses and errors
- **Context-Aware Testing**: Debug AI agents within the context of specific flows, tasks, and subtasks
- **Observability Integration**: All function calls are logged to Langfuse and Observability stack

### Usage Modes

#### Command Line Arguments

Run ftester with specific function and arguments directly from the command line:

```bash
# Basic usage with mock mode
cd backend
go run cmd/ftester/main.go [function_name] -[arg1] [value1] -[arg2] [value2]

# Example: Test terminal command in mock mode
go run cmd/ftester/main.go terminal -command "ls -la" -message "List files"

# Using a real flow context
go run cmd/ftester/main.go -flow 123 terminal -command "whoami" -message "Check user"

# Testing AI agent in specific task/subtask context
go run cmd/ftester/main.go -flow 123 -task 456 -subtask 789 pentester -message "Find vulnerabilities"
```

#### Interactive Mode

Run ftester without arguments for a guided interactive experience:

```bash
# Start interactive mode
go run cmd/ftester/main.go [function_name]

# For example, to interactively fill browser tool arguments
go run cmd/ftester/main.go browser
```

<details>
<summary><b>Available Functions</b> (click to expand)</summary>

### Environment Functions
- **terminal**: Execute commands in a container and return the output
- **file**: Perform file operations (read, write, list) in a container

### Search Functions
- **browser**: Access websites and capture screenshots
- **web_search**: Unified search orchestrator that agents actually call — pass a `query` and a `mode` (`links`, `answer`, `research`, `exploit`) and it auto-selects, retries, and falls back across the engines below, so you never name an engine explicitly
- **google**: Search the web using Google Custom Search
- **duckduckgo**: Search the web using DuckDuckGo
- **tavily**: Search using Tavily AI search engine
- **firecrawl**: Search using Firecrawl with main-content markdown scraping
- **traversaal**: Search using Traversaal AI search engine
- **perplexity**: Search using Perplexity AI
- **sploitus**: Search for security exploits, vulnerabilities (CVEs), and pentesting tools
- **searxng**: Search using SearXNG meta search engine (aggregates results from multiple engines)
- **internal** *(ftester-only debug function, not an agent tool)*: Opt-in browser-analytics fallback engine that discovers links, scrapes each page, and summarizes the result; requires `WEB_SEARCH_INTERNAL_ENABLED=true`, a configured scraper, and at least one available link engine

### Vector Database Functions
- **search_in_memory**: Search for information in vector database
- **search_guide**: Find guidance documents in vector database
- **search_answer**: Find answers to questions in vector database
- **search_code**: Find code examples in vector database

### AI Agent Functions
- **advice**: Get expert advice from an AI agent
- **coder**: Request code generation or modification
- **maintenance**: Run system maintenance tasks
- **memorist**: Store and organize information in vector database
- **pentester**: Perform security tests and vulnerability analysis
- **search**: Complex search across multiple sources

### Utility Functions
- **describe**: Show information about flows, tasks, and subtasks

</details>

<details>
<summary><b>Debugging Flow Context</b> (click to expand)</summary>

The `describe` function provides detailed information about tasks and subtasks within a flow. This is particularly useful for diagnosing issues when ARCANA encounters problems or gets stuck.

```bash
# List all flows in the system
go run cmd/ftester/main.go describe

# Show all tasks and subtasks for a specific flow
go run cmd/ftester/main.go -flow 123 describe

# Show detailed information for a specific task
go run cmd/ftester/main.go -flow 123 -task 456 describe

# Show detailed information for a specific subtask
go run cmd/ftester/main.go -flow 123 -task 456 -subtask 789 describe

# Show verbose output with full descriptions and results
go run cmd/ftester/main.go -flow 123 describe -verbose
```

This function allows you to identify the exact point where a flow might be stuck and resume processing by directly invoking the appropriate agent function.

</details>

<details>
<summary><b>Function Help and Discovery</b> (click to expand)</summary>

Each function has a help mode that shows available parameters:

```bash
# Get help for a specific function
go run cmd/ftester/main.go [function_name] -help

# Examples:
go run cmd/ftester/main.go terminal -help
go run cmd/ftester/main.go browser -help
go run cmd/ftester/main.go describe -help
```

You can also run ftester without arguments to see a list of all available functions:

```bash
go run cmd/ftester/main.go
```

</details>

<details>
<summary><b>Output Format</b> (click to expand)</summary>

The `ftester` utility uses color-coded output to make interpretation easier:

- **Blue headers**: Section titles and key names
- **Cyan [INFO]**: General information messages
- **Green [SUCCESS]**: Successful operations
- **Red [ERROR]**: Error messages
- **Yellow [WARNING]**: Warning messages
- **Yellow [MOCK]**: Indicates mock mode operation
- **Magenta values**: Function arguments and results

JSON and Markdown responses are automatically formatted for readability.

</details>

<details>
<summary><b>Advanced Usage Scenarios</b> (click to expand)</summary>

### Debugging Stuck AI Flows

When ARCANA gets stuck in a flow:

1. Pause the flow through the UI
2. Use `describe` to identify the current task and subtask
3. Directly invoke the agent function with the same task/subtask IDs
4. Examine the detailed output to identify the issue
5. Resume the flow or manually intervene as needed

### Testing Environment Variables

Verify that API keys and external services are configured correctly:

```bash
# Test Google search API configuration
go run cmd/ftester/main.go google -query "pentesting tools"

# Test browser access to external websites
go run cmd/ftester/main.go browser -url "https://example.com"
```

### Developing New AI Agent Behaviors

When developing new prompt templates or agent behaviors:

1. Create a test flow in the UI
2. Use ftester to directly invoke the agent with different prompts
3. Observe responses and adjust prompts accordingly
4. Check Langfuse for detailed traces of all function calls

### Verifying Docker Container Setup

Ensure containers are properly configured:

```bash
go run cmd/ftester/main.go -flow 123 terminal -command "env | grep -i proxy" -message "Check proxy settings"
```

</details>

<details>
<summary><b>Docker Container Usage</b> (click to expand)</summary>

If you have ARCANA running in Docker, you can use ftester from within the container:

```bash
# Run ftester inside the running ARCANA container
docker exec -it pentagi /opt/pentagi/bin/ftester [arguments]

# Examples:
docker exec -it pentagi /opt/pentagi/bin/ftester -flow 123 describe
docker exec -it pentagi /opt/pentagi/bin/ftester -flow 123 terminal -command "ps aux" -message "List processes"
```

This is particularly useful for production deployments where you don't have a local development environment.

</details>

<details>
<summary><b>Integration with Observability Tools</b> (click to expand)</summary>

All function calls made through ftester are logged to:

1. **Langfuse**: Captures the entire AI agent interaction chain, including prompts, responses, and function calls
2. **OpenTelemetry**: Records metrics, traces, and logs for system performance analysis
3. **Terminal Output**: Provides immediate feedback on function execution

To access detailed logs:

- Check Langfuse UI for AI agent traces (typically at `http://localhost:4000`)
- Use Grafana dashboards for system metrics (typically at `http://localhost:3000`)
- Examine terminal output for immediate function results and errors

</details>

### Command-line Options

The main utility accepts several options:

- `-env <path>` - Path to environment file (optional, default: `.env`)
- `-provider <type>` - Provider type to use (default: `custom`, options: `openai`, `anthropic`, `gemini`, `bedrock`, `ollama`, `deepseek`, `glm`, `kimi`, `qwen`, `minimax`, `custom`)
- `-flow <id>` - Flow ID for testing functions that require it (0 means using mocks, default: `0`)
- `-user <id>` - User ID for testing functions that require it (default: `0`; `1` is the default admin user)
- `-task <id>` - Task ID for agent context (optional)
- `-subtask <id>` - Subtask ID for agent context (optional)

Function-specific arguments are passed after the function name using `-name value` format.

### Pentesting Prompt Methodology

When refining prompts for offensive security work, give the agent a clear methodology instead of a flat list of payloads:

1. Start with explicit scope, authorization, and success criteria
2. Map the application first: roles, routes, parameters, uploads, integrations, and trust boundaries
3. Prioritize attack surfaces systematically instead of testing everything at once
4. Validate findings with reproducible evidence before escalating to deeper exploitation
5. Finish with report-ready notes that capture impact, prerequisites, and next steps

For ARCANA-specific prompt guidance, see [`backend/docs/prompt_engineering_arcana.md`](prompt_engineering_arcana.md). For a practical starting point, reuse and adapt [`examples/prompts/base_web_pentest.md`](../../examples/prompts/base_web_pentest.md) to match the target application, technology stack, and engagement scope.
