# Graphiti Knowledge Graph

The optional temporal knowledge graph: deployment modes, provider presets, ingestion tuning, verification, and limits.

> [!IMPORTANT]
> Graphiti is an optional **beta** integration and is disabled by default. Review [Limitations and Security](#limitations-and-security) before enabling it in production.

ARCANA integrates with [Graphiti](https://github.com/vxcontrol/pentagi-graphiti), a temporal knowledge graph over Neo4j. It tracks how tools, targets, vulnerabilities and techniques relate over time; the fork carries entity and edge types for pentesting.

## What is Graphiti?

Graphiti asynchronously extracts structured knowledge from agent interactions and builds a graph of entities, relationships, evidence, and temporal context. ARCANA sends agent responses and tool executions to Graphiti and exposes the `graphiti_search` tool to enabled agents. Graphiti complements the primary pgvector memory; it does not replace it.

- **Semantic Memory**: Store and recall relationships between tools, targets, vulnerabilities, and techniques
- **Contextual Understanding**: Track how different pentesting actions relate to each other over time
- **Flow-Scoped Recall**: Reuse knowledge within the active flow without exposing data from other engagements by default
- **Advanced Querying**: Search temporal context, relationships, successful tools, recent episodes, and entities by type

When enabled, ARCANA captures agent responses, tool execution details, and flow/task/subtask context. Ingestion is asynchronous, so newly submitted events can take time to become searchable.

## Deployment Modes and Enabling

Graphiti can run as the bundled Neo4j + Graphiti stack, as an external service, or remain disabled.

For the bundled stack, configure `.env`:

```bash
GRAPHITI_ENABLED=true
GRAPHITI_TIMEOUT=30
GRAPHITI_URL=http://graphiti:8000
GRAPHITI_LLM_CLIENT_TYPE=openai

# Reused by the Graphiti OpenAI preset
OPEN_AI_KEY=your_openai_api_key
OPEN_AI_SERVER_URL=https://api.openai.com/v1

# Bundled Neo4j
NEO4J_USER=neo4j
NEO4J_DATABASE=neo4j
NEO4J_PASSWORD=replace_with_a_strong_password
NEO4J_URI=bolt://neo4j:7687
```

Download the optional compose file and the Neo4j settings with the APOC plugin Graphiti needs when installing manually, then start both stacks:

```bash
curl -O https://raw.githubusercontent.com/kwkkd/arcana/main/docker-compose-graphiti.yml
mkdir -p neo4j/conf neo4j/plugins
for f in conf/neo4j.conf conf/apoc.conf plugins/apoc-5.26.19-core.jar; do
  curl -fsSL -o "neo4j/$f" "https://raw.githubusercontent.com/kwkkd/arcana/main/examples/neo4j/$f"
done
docker compose -f docker-compose.yml -f docker-compose-graphiti.yml up -d
```

The base stack must create the external `pentagi-network` before the Graphiti stack can start. The installer handles stack ordering automatically.

For an external Graphiti deployment, set `GRAPHITI_ENABLED=true` and point `GRAPHITI_URL` to its API. Do not start `docker-compose-graphiti.yml`; configure providers, embeddings, the graph database, and ingest tuning on the external service itself.

ARCANA enables its client only when both `GRAPHITI_ENABLED=true` and `GRAPHITI_URL` is non-empty. At startup it performs three health-check attempts with a two-second backoff. If they all fail, ARCANA logs a warning and continues with Graphiti disabled.

## LLM Provider and Model Presets

`GRAPHITI_LLM_CLIENT_TYPE` selects one deployment-wide preset. Model names and call parameters are not environment variables; they live in `graphiti/<provider>.yaml`.

| Preset | Credentials and endpoint | Shipped main model |
| --- | --- | --- |
| `openai` | `OPEN_AI_KEY`, `OPEN_AI_SERVER_URL` | `openai/gpt-5-mini` |
| `gemini` | `GEMINI_API_KEY`, `GEMINI_SERVER_URL` | `gemini/gemini-3.5-flash-lite` |
| `custom` | `LLM_SERVER_KEY`, `LLM_SERVER_URL` | `Qwen/Qwen3.6-27B-FP8` |
| `litellm` | `GRAPHITI_LITELLM_API_KEY`, `GRAPHITI_LITELLM_BASE_URL` | `openrouter/openai/gpt-oss-20b` |

The Gemini preset uses Graphiti's LiteLLM/OpenAI-compatible client path. Point `GEMINI_SERVER_URL` at a compatible gateway if the native Gemini endpoint does not provide the required OpenAI-compatible API.

Each preset file must contain a matching `provider` plus `MODEL_NAME` and `SMALL_MODEL_NAME` mappings. The small model is used for reranking and lighter calls. Supported call settings include temperature, token limits, sampling and penalty parameters, JSON mode, reasoning effort, verbosity, pricing metadata, and provider-specific `extra_body`.

The installer copies [`examples/graphiti`](../../examples/graphiti) beside the installation as `./graphiti`. The compose mount is controlled by:

```bash
GRAPHITI_CONFIG_PATH=./graphiti
GRAPHITI_CONFIG_DIR=llm_configs
```

`GRAPHITI_CONFIG_PATH` may point directly to `./examples/graphiti` for development. `GRAPHITI_CONFIG_DIR=llm_configs` activates the mounted presets. If an older `.env` omits that variable, a newer compose file mounts an empty host directory at the unused `configs` path instead of hiding the presets built into the image.

> [!NOTE]
> `GRAPHITI_MODEL_NAME` is obsolete and ignored. Edit the active YAML preset instead, then restart the Graphiti container.

## Graphiti Embedding Configuration

By default, Graphiti uses the active LLM preset's credentials and its default OpenAI embedding model. To use ARCANA's shared embedding endpoint explicitly:

```bash
GRAPHITI_SEPARATE_EMBEDDING=true
EMBEDDING_URL=https://embedding.example.com/v1
EMBEDDING_KEY=your_embedding_api_key
EMBEDDING_MODEL=openai/text-embedding-3-large
```

Graphiti's embedder is OpenAI-compatible. `EMBEDDING_PROVIDER` is used by ARCANA but is not passed to Graphiti, so a non-OpenAI-compatible embedding provider cannot be shared directly.

## Ingestion and Extraction Tuning

The supplied defaults prioritize flow isolation and limit expensive extraction to useful events.

Ingest policy actions:

- `REJECT`: do not store the episode.
- `SKIP_LLM`: store the episode for retrieval but do not extract nodes or edges.
- `PROCESS`: store the episode and run full LLM extraction.

| Variable | Default | When to change it |
| --- | --- | --- |
| `GRAPHITI_INGEST_POLICY_RULES` | `{"graphiti_search":"REJECT","tool_execution_terminal":"PROCESS","tool_execution_file":"PROCESS"}` | Add narrow, case-insensitive name/source patterns when specific events need different handling |
| `GRAPHITI_INGEST_POLICY_FIELD` | `both` | Restrict matching to `name` or `source_description` only when event naming is controlled |
| `GRAPHITI_INGEST_POLICY_DEFAULT_ACTION` | `SKIP_LLM` | Use `PROCESS` only when every unmatched event justifies extraction cost |
| `GRAPHITI_INGEST_USE_GROUP_ACTORS` | `true` | Keep enabled to preserve FIFO ordering independently for each flow |
| `GRAPHITI_INGEST_WORKER_COUNT` | `16` | Raise for more concurrent flows when the LLM and database have capacity; lower to control load |
| `GRAPHITI_INGEST_LOCK_BY_GROUP_ID` | `true` | Used only in shared-pool mode; ignored when group actors are enabled |
| `GRAPHITI_INGEST_TASK_MAX_RETRIES` | `1` (`0`-`5`) | Increase for transient LLM/network failures |
| `GRAPHITI_INGEST_TASK_RETRY_DELAY_SEC` | `2.0` (`0.5`-`60`) | Increase when an upstream service needs more recovery time |
| `GRAPHITI_INGEST_TASK_TIMEOUT_SEC` | `0` (`0`-`3600`) | Set a finite value to prevent one stalled request from blocking a flow; `0` disables the timeout |
| `GRAPHITI_INGEST_QUEUE_MAX_SIZE` | `0` | Set a bound to return HTTP 429 instead of allowing an unlimited backlog |
| `GRAPHITI_INGEST_DEAD_LETTER_ENABLED` | `false` | Enable when failed episodes must be retained for operational review |

Extraction uses the following fallback order: full combined extraction (nodes, attributes, summaries, and edges in one call), regular combined extraction (nodes and edges), then separate node/edge extraction. Empty or failed combined results automatically fall back; these log messages are expected during normal operation.

| Variable | Default | Effect |
| --- | --- | --- |
| `GRAPHITI_TAXONOMY_LAYER_PROFILE` | `STRUCTURAL,EVIDENCE,PROGRESS,ATTEMPT` | Controls which edge classes appear in prompts and pass validation; `full`/`all` enables every class and `minimal` selects the core attack graph |
| `GRAPHITI_USE_COMBINED_FULL_EXTRACTION` | `true` | Enables the most compact single-call extraction path |
| `GRAPHITI_USE_COMBINED_EXTRACTION` | `true` | Enables the regular combined fallback |
| `GRAPHITI_COMBINED_FULL_GATING_ENABLED` | `true` | Skips expensive full extraction for low-signal administrative/search events |
| `GRAPHITI_COMBINED_DIAGNOSTIC_SAMPLES` | `false` | Includes content samples in diagnostics; keep disabled because pentest output can contain credentials |
| `GRAPHITI_ANCHOR_NODE_MODE` | `smart` | `smart` loads all key entities plus limited high-volume types; `limit` applies one total cap |
| `GRAPHITI_ANCHOR_NODE_LIMIT` | `25` (`1`-`500`) | Total anchor cap in `limit` mode |
| `GRAPHITI_ANCHOR_MASS_TYPE_LIMIT` | `10` (`1`-`100`) | Per-type cap in `smart` mode; `0` is invalid |
| `GRAPHITI_ANCHOR_QUERY_TIMEOUT` | `10` (`1`-`60`) | Bounds anchor lookup; timeout degrades gracefully to no anchors |

Anchors connect entities across episodes and are used by the separate extraction path. Combined extraction has already produced its edges and does not perform this anchor lookup.

These flags are passed as process environment variables by the bundled compose file. This is important for combined extraction because Graphiti reads those flags when Python modules are imported.

## Runtime, Logging, and Neo4j

The values below are ARCANA's recommended `.env.example`/compose defaults, not the raw Graphiti image fallbacks. Running a freshly pulled image behind an old compose file can instead enable telemetry and global search, use one shared-pool worker with `PROCESS` as the unmatched ingest action, enable the full taxonomy, and disable combined extraction. Keep the image, compose file, `.env`, and presets in sync.

| Variable | Default | Guidance |
| --- | --- | --- |
| `GRAPHITI_CPUS`, `GRAPHITI_MEMORY` | `2.0`, `2G` | Container limits; raise together with concurrency only after observing CPU and memory pressure |
| `GRAPHITI_SEMAPHORE_LIMIT` | `20` | Limits parallel Graphiti coroutines; it is separate from ingest worker concurrency |
| `GRAPHITI_TELEMETRY_ENABLED` | `false` | Enables anonymous Graphiti telemetry when set to `true` |
| `GRAPHITI_LOG_LEVEL` | `INFO` | Use `DEBUG` temporarily; it can produce sensitive and high-volume output |
| `GRAPHITI_LOG_STDOUT` | `events` | `off`, `events`, or `full`; `events` is recommended for containers |
| `GRAPHITI_FLOW_LOGGER_WARN_COUNT` | `256` | Warns about growth of cached per-flow loggers; `0` disables the warning |
| `GRAPHITI_DEBUG_RUNTIME_RESOURCES` | `false` | Enables `/debug/runtime-resources`; expose only to trusted operators |
| `GRAPHITI_SEARCH_SCOPE` | `flowid` | Keep for flow/tenant isolation; `all` enables global searches and can expose other engagements |
| `GRAPHITI_LOG_FORMAT` | `json` | Reserved by the current deployment contract; the Graphiti logger does not yet apply it |
| `NEO4J_CPUS`, `NEO4J_MEMORY` | `4.0`, `4G` | Neo4j container limits; use `neo4j-admin server memory-recommendation --docker` for production sizing |
| `NEO4J_SHM_SIZE` | `4g` | `/dev/shm` limit; actual use counts toward the container memory limit |
| `NEO4J_NOFILE` | `65536` | Open-file soft/hard limit, suitable for many indexes and concurrent connections |
| `NEO4J_HEAP_INITIAL_SIZE`, `NEO4J_HEAP_MAX_SIZE`, `NEO4J_PAGECACHE_SIZE` | `2G`, `2G`, `1G` | JVM heap/page cache sizing; `NEO4J_CPUS`/`NEO4J_MEMORY` only cap the container, the JVM does not reliably size itself to fit inside that cap on its own. Defaults favor heap over page cache — suited to a low write-throughput deployment with occasional wide reads, since query execution/result materialization lives in heap while a small dataset is already comfortably held by 1G of page cache; re-run `neo4j-admin server memory-recommendation --docker` once real data volume is known |
| `NEO4J_TRANSACTION_MAX` | `1G` | Caps a single transaction's memory (`db.memory.transaction.max`) so one runaway/unbounded query (e.g. a Cartesian product or an unbounded variable-length path before a `LIMIT`) fails cleanly with an out-of-memory Cypher error instead of exhausting the whole heap and taking down every other query on the server |
| `NEO4J_BOLT_ADVERTISED_ADDRESS` | empty | Set only when Neo4j Browser and the Bolt connector are reverse-proxied on different public domains (e.g. behind Guarder with CORS/cookie-group support for cross-origin bolt access); format `host:port`. Left empty, Neo4j's discovery endpoint advertises whatever `Host` header the request arrived with, which is correct only when both share one domain |
| `NEO4J_HTTP_ADVERTISED_ADDRESS` | empty | Set only when a reverse proxy in front of Neo4j Browser strips or rewrites the `Host` header, making the dynamic Host-header-based advertised address incorrect; format `host:port`. Leave empty in the common case (proxy forwards `Host` unchanged) |

`NEO4J_USER`, `NEO4J_PASSWORD`, `NEO4J_URI`, and `NEO4J_DATABASE` configure the bundled connection. Neo4j Community Edition supports only its default database; do not configure a separate database name that requires Enterprise multi-database support.

The installer copies [`examples/neo4j`](../../examples/neo4j) beside the installation as `./neo4j`. It contains static, non-`.env`-tunable settings that don't have a `NEO4J_*` variable: `conf/neo4j.conf` and `conf/apoc.conf`, plus a version-pinned `plugins/apoc-*-core.jar`. The compose mount is controlled by:

```bash
NEO4J_DIR=./neo4j
```

`NEO4J_DIR` may point directly to `./examples/neo4j` for development. Both `conf/` and `plugins/` are mounted read-only; the stack still starts on Neo4j's built-in defaults (without APOC) if the directory is absent, since Docker creates an empty one automatically — but Graphiti writes relationships through APOC, so its graph then stays empty while ARCANA logs no error. Do not duplicate any `NEO4J_*` variable from the table above inside `conf/neo4j.conf` — the Neo4j Docker entrypoint always strips a matching line from the mounted file and re-appends the environment variable's value, so a duplicated setting in the file would be silently ignored.

The bundled stack currently wires Neo4j only. The Graphiti image contains FalkorDB support, but using it requires a separately configured deployment because the stock compose file does not expose `GRAPHITI_GRAPH_BACKEND` or `FALKORDB_*`.

## Verification and Troubleshooting

Check service health, queue state, and logs:

```bash
docker compose -f docker-compose.yml -f docker-compose-graphiti.yml ps graphiti neo4j
docker compose -f docker-compose.yml -f docker-compose-graphiti.yml logs -f graphiti
curl -fsS http://localhost:8000/healthcheck
curl -fsS http://localhost:8000/queue-size
```

Neo4j Browser is available at `http://localhost:7474`; the Graphiti OpenAPI UI is at `http://localhost:8000/docs`. Both are bound to localhost by the stock compose file.

Common failures:

- A missing API key or base URL for the selected preset, a missing YAML file, or a YAML `provider` mismatch causes the Graphiti container to fail startup validation.
- `LLM_CLIENT_TYPE=openai` rejects local/custom model prefixes; use the `custom` preset for an OpenAI-compatible local server.
- In `flowid` search mode, requests without a group ID are rejected. ARCANA supplies the flow-derived group ID automatically.
- A bounded full queue returns HTTP 429. `/queue-size` reports waiting, processing, active-group, and dropped counters.
- Invalid retry, timeout, or anchor ranges fail startup rather than being silently normalized.

Update `.env`, `docker-compose-graphiti.yml`, the Graphiti image, and the `graphiti` preset directory together. Pulling only a new image can retain older compose defaults and silently change extraction behavior.

## Limitations and Security

- Graphiti is beta and has no in-app graph explorer.
- One provider preset is active for the entire Graphiti deployment; it is not selected per ARCANA agent or flow.
- Graphiti extraction, reranking, and embeddings incur billing independently of the model used by the main ARCANA flow.
- Search is flow-scoped by default. Cross-flow reuse requires an explicit global-search design and must not be enabled on shared or multi-tenant deployments without additional isolation.
- The Graphiti HTTP API has no authentication layer in the bundled service. The stock compose binds it and Neo4j to `127.0.0.1`; secure external deployments with network controls and authentication at a trusted reverse proxy.
- Agent and tool output may contain credentials and exploitation evidence. Protect Neo4j data, logs, dead letters, diagnostics, and backups accordingly.
- If Graphiti is unavailable, ARCANA continues with its primary memory and vector store after logging the failed startup health check. Set `GRAPHITI_ENABLED=false` to disable the integration explicitly.
