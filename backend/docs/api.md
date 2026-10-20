# API

ARCANA exposes REST and GraphQL APIs over the same backend. This chapter covers token management, endpoint discovery, client generation, and worked examples for both interfaces.

## Generating API Tokens

API tokens are managed through the ARCANA web interface:

1. Navigate to **Settings** → **API Tokens** in the web UI
2. Click **Create Token** to generate a new API token
3. Configure token properties:
   - **Name** (optional): A descriptive name for the token
   - **Expiration Date**: When the token will expire (minimum 1 minute, maximum 3 years)
4. Click **Create** and **copy the token immediately** - it will only be shown once for security reasons
5. Use the token as a Bearer token in your API requests

Each token is associated with your user account and inherits your role's permissions.

## Using API Tokens

Include the API token in the `Authorization` header of your HTTP requests:

```bash
# GraphQL API example
curl -X POST https://your-arcana-instance:8443/api/v1/graphql \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ flows { id title status } }"}'

# REST API example
curl https://your-arcana-instance:8443/api/v1/flows \
  -H "Authorization: Bearer YOUR_API_TOKEN"
```

## API Exploration and Testing

ARCANA provides interactive documentation for exploring and testing API endpoints:

### GraphQL Playground

Access the GraphQL Playground at `https://your-arcana-instance:8443/api/v1/graphql/playground`

1. Click the **HTTP Headers** tab at the bottom
2. Add your authorization header:
   ```json
   {
     "Authorization": "Bearer YOUR_API_TOKEN"
   }
   ```
3. Explore the schema, run queries, and test mutations interactively

### Swagger UI

Access the REST API documentation at `https://your-arcana-instance:8443/api/v1/swagger/index.html`

1. Click the **Authorize** button
2. Enter your token in the format: `Bearer YOUR_API_TOKEN`
3. Click **Authorize** to apply
4. Test endpoints directly from the Swagger UI

## Generating API Clients

You can generate type-safe API clients for your preferred programming language using the schema files included with ARCANA:

### GraphQL Clients

The GraphQL schema is available at:
- **Web UI**: Navigate to Settings to download `schema.graphqls`
- **Direct file**: `backend/pkg/graph/schema.graphqls` in the repository

Generate clients using tools like:
- **GraphQL Code Generator** (JavaScript/TypeScript): [https://the-guild.dev/graphql/codegen](https://the-guild.dev/graphql/codegen)
- **genqlient** (Go): [https://github.com/Khan/genqlient](https://github.com/Khan/genqlient)
- **Apollo iOS** (Swift): [https://www.apollographql.com/docs/ios](https://www.apollographql.com/docs/ios)

### REST API Clients

The OpenAPI specification is available at:
- **Swagger JSON**: `https://your-arcana-instance:8443/api/v1/swagger/doc.json`
- **Swagger YAML**: Available in `backend/pkg/server/docs/swagger.yaml`

Generate clients using:
- **OpenAPI Generator**: [https://openapi-generator.tech](https://openapi-generator.tech)
  ```bash
  openapi-generator-cli generate \
    -i https://your-arcana-instance:8443/api/v1/swagger/doc.json \
    -g python \
    -o ./pentagi-client
  ```

- **Swagger Codegen**: [https://github.com/swagger-api/swagger-codegen](https://github.com/swagger-api/swagger-codegen)
  ```bash
  swagger-codegen generate \
    -i https://your-arcana-instance:8443/api/v1/swagger/doc.json \
    -l typescript-axios \
    -o ./pentagi-client
  ```

- **swagger-typescript-api** (TypeScript): [https://github.com/acacode/swagger-typescript-api](https://github.com/acacode/swagger-typescript-api)
  ```bash
  npx swagger-typescript-api \
    -p https://your-arcana-instance:8443/api/v1/swagger/doc.json \
    -o ./src/api \
    -n pentagi-api.ts
  ```

## API Usage Examples

<details>
<summary><b>Creating a New Flow (GraphQL)</b></summary>

```graphql
mutation CreateFlow {
  createFlow(
    modelProvider: "openai"
    input: "Test the security of https://example.com"
  ) {
    id
    title
    status
    createdAt
  }
}
```

</details>

<details>
<summary><b>Listing Flows (REST API)</b></summary>

```bash
curl https://your-arcana-instance:8443/api/v1/flows \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  | jq '.flows[] | {id, title, status}'
```

</details>

<details>
<summary><b>Python Client Example</b></summary>

```python
import requests

class ARCANAClient:
    def __init__(self, base_url, api_token):
        self.base_url = base_url
        self.headers = {
            "Authorization": f"Bearer {api_token}",
            "Content-Type": "application/json"
        }
    
    def create_flow(self, provider, target):
        query = """
        mutation CreateFlow($provider: String!, $input: String!) {
          createFlow(modelProvider: $provider, input: $input) {
            id
            title
            status
          }
        }
        """
        response = requests.post(
            f"{self.base_url}/api/v1/graphql",
            json={
                "query": query,
                "variables": {
                    "provider": provider,
                    "input": target
                }
            },
            headers=self.headers
        )
        return response.json()
    
    def get_flows(self):
        response = requests.get(
            f"{self.base_url}/api/v1/flows",
            headers=self.headers
        )
        return response.json()

# Usage
client = ARCANAClient(
    "https://your-arcana-instance:8443",
    "your_api_token_here"
)

# Create a new flow
flow = client.create_flow("openai", "Scan https://example.com for vulnerabilities")
print(f"Created flow: {flow}")

# List all flows
flows = client.get_flows()
print(f"Total flows: {len(flows['flows'])}")
```

</details>

<details>
<summary><b>TypeScript Client Example</b></summary>

```typescript
import axios, { AxiosInstance } from 'axios';

interface Flow {
  id: string;
  title: string;
  status: string;
  createdAt: string;
}

class ARCANAClient {
  private client: AxiosInstance;

  constructor(baseURL: string, apiToken: string) {
    this.client = axios.create({
      baseURL: `${baseURL}/api/v1`,
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json',
      },
    });
  }

  async createFlow(provider: string, input: string): Promise<Flow> {
    const query = `
      mutation CreateFlow($provider: String!, $input: String!) {
        createFlow(modelProvider: $provider, input: $input) {
          id
          title
          status
          createdAt
        }
      }
    `;

    const response = await this.client.post('/graphql', {
      query,
      variables: { provider, input },
    });

    return response.data.data.createFlow;
  }

  async getFlows(): Promise<Flow[]> {
    const response = await this.client.get('/flows');
    return response.data.flows;
  }

  async getFlow(flowId: string): Promise<Flow> {
    const response = await this.client.get(`/flows/${flowId}`);
    return response.data;
  }
}

// Usage
const client = new ARCANAClient(
  'https://your-arcana-instance:8443',
  'your_api_token_here'
);

// Create a new flow
const flow = await client.createFlow(
  'openai',
  'Perform penetration test on https://example.com'
);
console.log('Created flow:', flow);

// List all flows
const flows = await client.getFlows();
console.log(`Total flows: ${flows.length}`);
```

</details>

## Security Best Practices

When working with API tokens:

- **Never commit tokens to version control** - use environment variables or secrets management
- **Rotate tokens regularly** - set appropriate expiration dates and create new tokens periodically
- **Use separate tokens for different applications** - makes it easier to revoke access if needed
- **Monitor token usage** - review API token activity in the Settings page
- **Revoke unused tokens** - disable or delete tokens that are no longer needed
- **Use HTTPS only** - never send API tokens over unencrypted connections

## Token Management

- **View tokens**: See all your active tokens in Settings → API Tokens
- **Edit tokens**: Update token names or revoke tokens
- **Delete tokens**: Permanently remove tokens (this action cannot be undone)
- **Token ID**: Each token has a unique ID that can be copied for reference

The token list shows:
- Token name (if provided)
- Token ID (unique identifier)
- Status (active/revoked/expired)
- Creation date
- Expiration date
