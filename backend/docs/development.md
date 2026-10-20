# Development

Local setup, code generation, database pool sizing, and how to run the backend and frontend.

## Development Requirements

- golang
- nodejs
- docker
- postgres

## Environment Setup

### Backend Setup

Run once `cd backend && go mod download` to install needed packages.

For generating swagger files have to run

```bash
swag init -g ../../pkg/server/router.go -o pkg/server/docs/ --parseDependency --parseInternal --parseDepth 2 -d cmd/pentagi
```

before installing `swag` package via

```bash
go install github.com/swaggo/swag/cmd/swag@v1.8.7
```

For generating graphql resolver files have to run

```bash
go run github.com/99designs/gqlgen --config ./gqlgen/gqlgen.yml
```

after that you can see the generated files in `pkg/graph` folder.

For generating ORM methods (database package) from sqlc configuration

```bash
docker run --rm -v $(pwd):/src -w /src --network pentagi-network -e DATABASE_URL="{URL}" sqlc/sqlc:1.27.0 generate -f sqlc/sqlc.yml
```

For generating Langfuse SDK from OpenAPI specification

```bash
fern generate --local
```

and to install fern-cli

```bash
pnpm add -g fern-api
```

### Testing

For running tests `cd backend && go test -v ./...`

### Frontend Setup

Run once `cd frontend && pnpm install` to install needed packages.

For generating graphql files have to run `pnpm run graphql:generate` which using `graphql-codegen.ts` file.

Be sure that you have `graphql-codegen` installed globally:

```bash
pnpm add -g graphql-codegen
```

After that you can run:
* `pnpm run prettier` to check if your code is formatted correctly
* `pnpm run prettier:fix` to fix it
* `pnpm run lint` to check if your code is linted correctly
* `pnpm run lint:fix` to fix it

For generating SSL certificates you need to run `pnpm run ssl:generate` which using `generate-ssl.ts` file or it will be generated automatically when you run `pnpm run dev`.

### Backend Configuration

Edit the configuration for `backend` in `.vscode/launch.json` file:
- `DATABASE_URL` - PostgreSQL database URL (eg. `postgres://postgres:postgres@localhost:5432/arcanadb?sslmode=disable`)
- `DOCKER_HOST` - Docker SDK API (eg. for macOS `DOCKER_HOST=unix:///Users/<my-user>/Library/Containers/com.docker.docker/Data/docker.raw.sock`) [more info](https://stackoverflow.com/a/62757128/5922857)

Optional:
- `SERVER_PORT` - Port to run the server (default: `8443`)
- `SERVER_USE_SSL` - Enable SSL for the server (default: `false`)

#### PostgreSQL / pgvector connection pool sizing

ARCANA opens two independent connection pools to the same Postgres instance:

| Pool | Env var | Default | Used by |
|---|---|---|---|
| Shared `sql.DB` | `DATABASE_MAX_OPEN_CONNS` | `25` | All sqlc queries and GORM handlers share a single `*sql.DB` |
| Shared `pgxpool` | `DATABASE_VECTOR_MAX_CONNS` | `10` | All pgvector stores (agent memory + knowledge API) share a single pool |

Additional tuning knob:
- `DATABASE_MAX_IDLE_CONNS` — maximum idle connections kept open in the `sql.DB` pool between requests (default: `5`).

**Budget for the stock `vxcontrol/pgvector` image** (`max_connections = 100`, `superuser_reserved_connections = 3`):

```
Available for client connections  = 97
  pentagi sql.DB  (DATABASE_MAX_OPEN_CONNS)   = 25
  pentagi pgxpool (DATABASE_VECTOR_MAX_CONNS) = 10
  pgexporter                                  =  3
  autovacuum workers                          =  3
  ─────────────────────────────────────────
  Total consumed                              = 41
  Free buffer                                 = 56  (≈ 58 %)
```

The defaults are sized for **10 parallel flows** with concurrent API requests. If you run more flows or deploy multiple ARCANA instances against the same Postgres, raise `max_connections` via the `command` override in `docker-compose.yml` and increase the pool sizes proportionally:

```yaml
pgvector:
  image: vxcontrol/pgvector:latest
  command: postgres -c max_connections=200
```

To inspect the live connection budget on a running deployment:

```bash
# Postgres limits
docker exec pgvector sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
  "SELECT name, setting FROM pg_settings
   WHERE name IN ('"'"'max_connections'"'"', '"'"'superuser_reserved_connections'"'"');"'

# Current usage vs. available
docker exec pgvector sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
  "SELECT max_conn, used, max_conn - used AS available
   FROM (SELECT current_setting('"'"'max_connections'"'"')::int AS max_conn,
                count(*) AS used FROM pg_stat_activity) t;"'

# Breakdown by client
docker exec pgvector sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
  "SELECT application_name, client_addr, state, count(*)
   FROM pg_stat_activity
   WHERE pid <> pg_backend_pid()
   GROUP BY 1, 2, 3 ORDER BY count DESC;"'
```

#### External PostgreSQL and schema handling

`DATABASE_URL` may point at any PostgreSQL instance, not only the bundled `pgvector` container. Two extra knobs apply when — and only when — `TENANT_ID` is set, because that is when ARCANA creates its own schema and rewrites the connection's `search_path`:

| Env var | Default | Purpose |
|---|---|---|
| `DATABASE_EXTENSIONS_SCHEMA` | `public` | Schema holding the shared `vector` and `pg_trgm` extensions that every tenant's `search_path` must reach |
| `DATABASE_SEARCH_PATH_VIA_OPTIONS` | `false` | Send the tenant `search_path` inside the `options` startup parameter instead of as a bare connection parameter |

**Supabase (cloud or self-hosted)** needs both of them considered, and is the reason they exist:

- Supabase installs its bundled extensions into an `extensions` schema instead of `public`, so set `DATABASE_EXTENSIONS_SCHEMA=extensions`. Without it, startup aborts with an error naming the schema where `vector` was actually found — no need to move a provider-managed extension with `ALTER EXTENSION`.
- Supabase's pooler (Supavisor) does not reliably forward a bare `search_path` connection parameter. Prefer a **direct** PostgreSQL connection: self-hosted, expose the `db` service port and bypass the `supavisor` service; cloud, use the "Direct connection" string (or the IPv4 add-on on IPv4-only networks). If the pooler cannot be bypassed, use its session mode and try `DATABASE_SEARCH_PATH_VIA_OPTIONS=true` — ARCANA verifies the effective schema on boot and refuses to start if it did not take effect, so a silent cross-tenant data mix-up is not possible.

Both settings are managed by the installer under *Server Settings*, next to `TENANT_ID` — see [Multi-Instance Deployment](config.md#multi-instance-deployment-tenant_id) for that scenario, and [Multi-Instance Deployment](config.md#multi-instance-deployment-tenant_id) for the full matrix, including the PgBouncer recipe (`pool_mode = session`, `ignore_startup_parameters`, per-tenant `connect_query`).

### Frontend Configuration

Edit the configuration for `frontend` in `.vscode/launch.json` file:
- `VITE_API_URL` - Backend API URL. *Omit* the URL scheme (e.g., `localhost:8080` *NOT* `http://localhost:8080`)
- `VITE_USE_HTTPS` - Enable SSL for the server (default: `false`)
- `VITE_PORT` - Port to run the server (default: `8000`)
- `VITE_HOST` - Host to run the server (default: `0.0.0.0`)

## Running the Application

### Backend

Run the command(s) in `backend` folder:
- Use `.env` file to set environment variables like a `source .env`
- Run `go run cmd/pentagi/main.go` to start the server

> [!NOTE]
> The first run can take a while as dependencies and docker images need to be downloaded to setup the backend environment.

### Frontend

Run the command(s) in `frontend` folder:
- Run `pnpm install` to install the dependencies
- Run `pnpm run dev` to run the web app
- Run `pnpm run build` to build the web app

Open your browser and visit the web app URL.
