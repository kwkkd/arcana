# Installing and Configuring ARCANA

This guide is the bridge between installing ARCANA and using it. It walks you through a first deployment in order: pick an installation method, set the core server variables, configure and test at least one LLM provider, configure the embedding provider, optionally add search and observability, then start the stack and verify it before your first login.

It is intentionally concise and links to the detailed reference documents instead of repeating them. When you finish here, continue with [Using ARCANA](../../README.md#using-arcana).

## Before you start

- Docker and Docker Compose (or Podman), 2+ vCPU, 4+ GB RAM, 20+ GB free disk, and outbound internet access for pulling images and reaching LLM providers.
- At least one LLM provider you can authenticate to (OpenAI, Anthropic, Gemini, AWS Bedrock, or a local/Ollama/OpenAI-compatible backend). ARCANA will not run without one.
- Decide how you want to install: the interactive installer (recommended) or a manual Docker Compose deployment.

## Step 1 - Choose an installation method

### Option A: Interactive installer (recommended)

The installer is a terminal UI that runs system checks, writes a sane `.env`, helps you configure LLM and search providers, hardens credentials, and starts the stack for you. Build it from source and run it, then follow the prompts. See [Installer](../../README.md#installer-recommended) for build steps and the Docker socket permission notes.

If you use the installer, it can take you through most of Steps 2-6 below interactively. You can still use this guide as a checklist of what to confirm.

### Option B: Manual Docker Compose

Create a working directory, copy `.env.example` to `.env`, fill in your keys, and bring up the stack. The full sequence (including the example provider config files and the `docker compose up -d` command) is in [Manual Docker Compose](../../README.md#manual-docker-compose).

## Step 2 - Set the core server variables

Whichever method you used, confirm these in your `.env` before exposing the instance to anything but localhost:

- `PUBLIC_URL` - the URL users and the browser will actually load, for example `https://arcana.example.com` or `https://192.168.1.100:8443`. Use the real hostname or IP, never `0.0.0.0`.
- `CORS_ORIGINS` - every origin that will reach the UI, comma-separated. Include both `https://localhost:8443` and your external URL if you use both.
- `PENTAGI_LISTEN_IP` / `PENTAGI_LISTEN_PORT` - keep the default `127.0.0.1` for localhost-only, or set the IP to `0.0.0.0` to accept external connections.
- `COOKIE_SIGNING_SALT` and the database passwords (`PENTAGI_POSTGRES_PASSWORD`, `NEO4J_PASSWORD`) - change these away from the defaults before real use.

For external access, firewall rules, and certificate notes, see [Accessing ARCANA from External Networks](../../backend/docs/deployment.md#accessing-arcana-from-external-networks).

## Step 3 - Configure and test an LLM provider

Set the credentials for at least one provider in `.env`. Each provider has its own reference section with the exact variables and supported models:

- [OpenAI](../../backend/docs/providers.md#openai-provider-configuration) (`OPEN_AI_KEY`, `OPEN_AI_SERVER_URL`)
- [Anthropic](../../backend/docs/providers.md#anthropic-provider-configuration) (`ANTHROPIC_API_KEY`, `ANTHROPIC_SERVER_URL`)
- [Google AI (Gemini)](../../backend/docs/providers.md#google-ai-gemini-provider-configuration) (`GEMINI_API_KEY`, `GEMINI_SERVER_URL`)
- [AWS Bedrock](../../backend/docs/providers.md#aws-bedrock-provider-configuration) (`BEDROCK_REGION` plus one auth method)
- [Custom OpenAI-compatible / local](../../backend/docs/providers.md#custom-llm-provider-configuration) (`LLM_SERVER_URL`, `LLM_SERVER_KEY`, `LLM_SERVER_MODEL`) and [Ollama](../../backend/docs/providers.md#ollama-provider-configuration)

Before starting a full flow, confirm the provider actually answers and supports the agent behaviors ARCANA needs by running `ctester`:

```bash
# From a local Go environment (run inside the backend directory)
go run ./cmd/ctester -verbose

# Or from the running container
docker exec -it pentagi /opt/pentagi/bin/ctester -verbose
```

Full usage, including `-type`, `-config`, and per-agent testing, is in [Testing LLM Agents](../../backend/docs/tools.md#testing-llm-agents). If `ctester` reports tool-call or function-call problems with a custom, llama.cpp, vLLM, or SGLang backend, that points at the backend's tool-call parser rather than ARCANA itself.

## Step 4 - Configure and test the embedding provider

ARCANA uses embeddings for semantic search, knowledge storage, and memory. The default provider is OpenAI; when `EMBEDDING_URL` and `EMBEDDING_KEY` are empty and `EMBEDDING_PROVIDER=openai`, it falls back to your `OPEN_AI_SERVER_URL` and `OPEN_AI_KEY`. Relevant variables: `EMBEDDING_PROVIDER`, `EMBEDDING_MODEL`, `EMBEDDING_URL`, `EMBEDDING_KEY`, `EMBEDDING_BATCH_SIZE`, `EMBEDDING_MAX_TEXT_BYTES`, `EMBEDDING_STRIP_NEW_LINES`.

Verify embeddings with `etester`:

```bash
# Local Go environment
go run ./cmd/etester test -verbose

# Running container
docker exec -it pentagi /opt/pentagi/bin/etester test -verbose
```

If you later change the embedding provider or model, flush and reindex the knowledge base. Details and the full command set are in [Embedding Configuration and Testing](../../backend/docs/tools.md#embedding-configuration-and-testing).

## Step 5 - (Optional) Configure search providers

Search providers improve research quality but are optional. ARCANA supports DuckDuckGo (`DUCKDUCKGO_ENABLED`), Sploitus (`SPLOITUS_ENABLED`), Google (`GOOGLE_API_KEY`, `GOOGLE_CX_KEY`), Tavily (`TAVILY_API_KEY`), Firecrawl (`FIRECRAWL_API_KEY`), Traversaal (`TRAVERSAAL_API_KEY`), Perplexity (`PERPLEXITY_API_KEY`), a self-hosted Searxng instance (`SEARXNG_URL`), and an optional internal browser-analytics fallback engine (`WEB_SEARCH_INTERNAL_ENABLED`, off by default). Set the keys for the engines you want in `.env`; the full block is in [`.env.example`](../../.env.example) and in [Search Engine Settings](../../backend/docs/config.md#search-engine-settings).

## Step 6 - (Optional) Enable Graphiti, Langfuse, and observability

These are separate, optional stacks brought up with additional compose files:

- Graphiti knowledge graph (`GRAPHITI_ENABLED=true`, `GRAPHITI_URL`, plus the `NEO4J_*` settings) via `docker-compose-graphiti.yml`.
- Langfuse analytics via `docker-compose-langfuse.yml`.
- Monitoring (Grafana / OpenTelemetry) via `docker-compose-observability.yml`.

Run the base `docker-compose.yml` first so the shared Docker networks exist, then bring up the optional stacks.

## Step 7 - Start and verify

1. Start the stack: `docker compose up -d`.
2. Watch the logs until the backend is ready: `docker compose logs -f pentagi`.
3. Re-run `ctester` and `etester` if you changed any provider settings. For deeper checks of individual agent functions and tools, use `ftester` (see [Function Testing with ftester](../../backend/docs/tools.md#function-testing-with-ftester)).
4. Open `https://localhost:8443` (or your `PUBLIC_URL`) and sign in with the default `admin@arcana.local` / `admin`, then change the password immediately.

After the server is running, several areas are managed in the web console under Settings (Providers, Prompts, ARCANA API tokens), while LLM and search credentials, Langfuse and Graphiti remain server-side configuration. See [Current Web Settings Coverage](../../backend/docs/config.md#current-web-settings-coverage) for the exact split.

## First-run configuration checklist

- [ ] Docker/Podman, CPU, RAM, and disk meet the minimums
- [ ] Installation method chosen (installer or manual Compose)
- [ ] `PUBLIC_URL` and `CORS_ORIGINS` set to the real hostname/IP (not `0.0.0.0`)
- [ ] `COOKIE_SIGNING_SALT` and database passwords changed from defaults
- [ ] At least one LLM provider configured and passing `ctester`
- [ ] Embedding provider configured and passing `etester`
- [ ] Search providers configured (optional)
- [ ] Graphiti / Langfuse / observability enabled if needed (optional)
- [ ] Stack started, logs healthy, default admin password changed

## Next step

Continue with [Using ARCANA](../../README.md#using-arcana) to create your first flow, use prompt templates, and review results.

For hardened or fully local deployments, see the [Worker Node Setup](worker_node.md) and [vLLM + Qwen3.5-27B-FP8](vllm-qwen35-27b-fp8.md) guides.
