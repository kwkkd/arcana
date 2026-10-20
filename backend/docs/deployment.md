# Deployment

Production notes that sit outside the quick start: exposing the instance, running it under Podman, choosing sandbox images, OAuth sign-in, and building images.

## Contents

- [Accessing ARCANA from external networks](#accessing-arcana-from-external-networks)
- [Running ARCANA with Podman](#running-arcana-with-podman)
- [GitHub and Google OAuth](#github-and-google-oauth-integration)
- [Docker image configuration](#docker-image-configuration)
- [Building the image](#building)

## Accessing ARCANA from External Networks

By default, the ARCANA web interface and the other compose services bind to `127.0.0.1` (localhost only) for security. To access ARCANA from other machines on your network, you need to configure external access.

> [!IMPORTANT]
> Flow sandboxes are not bound to localhost. Each flow publishes two TCP ports for out-of-band callbacks such as reverse shells, taken from the 2000-port window that starts at `DOCKER_PORTS_BASE` (`28000`–`29999` by default), on `DOCKER_PUBLIC_IP` — `0.0.0.0` by default, that is every interface of the Docker host. Docker routes published ports around `ufw` rules and lets them through `firewalld` with its own `docker` zone, so restrict this range with a firewall in front of the host or, with Docker's default iptables backend, with rules in the `DOCKER-USER` chain. With `DOCKER_NETWORK=host`, sandboxes share the host's network stack, so their listeners are exposed without any port mapping.

### Configuration Steps

1. **Update `.env` file** with your server's IP address:

```bash
# Network binding - allow external connections
PENTAGI_LISTEN_IP=0.0.0.0
PENTAGI_LISTEN_PORT=8443

# Public URL - use your actual server IP or hostname
# Replace 192.168.1.100 with your server's IP address
PUBLIC_URL=https://192.168.1.100:8443

# CORS origins - list all URLs that will access ARCANA
# Include localhost for local access AND your server IP for external access
CORS_ORIGINS=https://localhost:8443,https://192.168.1.100:8443
```

> [!IMPORTANT]
> - Replace `192.168.1.100` with your actual server's IP address
> - Do NOT use `0.0.0.0` in `PUBLIC_URL` or `CORS_ORIGINS` - use the actual IP address
> - Include both localhost and your server IP in `CORS_ORIGINS` for flexibility

2. **Recreate containers** to apply the changes:

```bash
docker compose down
docker compose up -d --force-recreate
```

3. **Verify port binding:**

```bash
docker ps | grep pentagi
```

You should see `0.0.0.0:8443->8443/tcp` or `:::8443->8443/tcp`.

If you see `127.0.0.1:8443->8443/tcp`, the environment variable wasn't picked up. In this case, directly edit `docker-compose.yml` line 31:

```yaml
ports:
  - "0.0.0.0:8443:8443"
```

Then recreate containers again.

4. **Configure firewall** to allow incoming connections on port 8443:

```bash
# Ubuntu/Debian with UFW
sudo ufw allow 8443/tcp
sudo ufw reload

# CentOS/RHEL with firewalld
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --reload
```

5. **Access ARCANA:**

- **Local access:** `https://localhost:8443`
- **Network access:** `https://your-server-ip:8443`

> [!NOTE]
> You'll need to accept the self-signed SSL certificate warning in your browser when accessing via IP address.

## Running ARCANA with Podman

ARCANA fully supports Podman as a Docker alternative. However, when using **Podman in rootless mode**, the scraper service requires special configuration because rootless containers cannot bind privileged ports (ports below 1024).

### Podman Rootless Configuration

The default scraper configuration uses port 443 (HTTPS), which is a privileged port. For Podman rootless, reconfigure the scraper to use a non-privileged port:

**1. Edit `docker-compose.yml`** - modify the `scraper` service (around line 199):

```yaml
scraper:
  image: vxcontrol/scraper:latest
  restart: unless-stopped
  container_name: scraper
  hostname: scraper
  expose:
    - 3000/tcp  # Changed from 443 to 3000
  ports:
    - "${SCRAPER_LISTEN_IP:-127.0.0.1}:${SCRAPER_LISTEN_PORT:-9443}:3000"  # Map to port 3000
  environment:
    - MAX_CONCURRENT_SESSIONS=${LOCAL_SCRAPER_MAX_CONCURRENT_SESSIONS:-10}
    - USERNAME=${LOCAL_SCRAPER_USERNAME:-someuser}
    - PASSWORD=${LOCAL_SCRAPER_PASSWORD:-somepass}
  logging:
    options:
      max-size: 50m
      max-file: "7"
  volumes:
    - scraper-ssl:/usr/src/app/ssl
  networks:
    - pentagi-network
  shm_size: 2g
```

**2. Update `.env` file** - change the scraper URL to use HTTP and port 3000:

```bash
# Scraper configuration for Podman rootless
SCRAPER_PRIVATE_URL=http://someuser:somepass@scraper:3000/
LOCAL_SCRAPER_USERNAME=someuser
LOCAL_SCRAPER_PASSWORD=somepass
```

> [!IMPORTANT]
> Key changes for Podman:
> - Use **HTTP** instead of HTTPS for `SCRAPER_PRIVATE_URL`
> - Use port **3000** instead of 443
> - Change internal `expose` to `3000/tcp`
> - Update port mapping to target `3000` instead of `443`

**3. Recreate containers:**

```bash
podman-compose down
podman-compose up -d --force-recreate
```

**4. Test scraper connectivity:**

```bash
# Test from within the pentagi container
podman exec -it pentagi wget -O- "http://someuser:somepass@scraper:3000/html?url=http://example.com"
```

If you see HTML output, the scraper is working correctly.

### Podman Rootful Mode

If you're running Podman in rootful mode (with sudo), you can use the default configuration without modifications. The scraper will work on port 443 as intended.

### Docker Compatibility

All Podman configurations remain fully compatible with Docker. The non-privileged port approach works identically on both container runtimes.

## GitHub and Google OAuth Integration

OAuth integration with GitHub and Google allows users to authenticate using their existing accounts on these platforms. This provides several benefits:

- Simplified login process without need to create separate credentials
- Enhanced security through trusted identity providers
- Access to user profile information from GitHub/Google accounts
- Seamless integration with existing development workflows

ARCANA uses `PUBLIC_URL` as the public origin/base URL for OAuth redirects. In the default deployment, both GitHub and Google callbacks are handled by:

```text
${PUBLIC_URL}/api/v1/auth/login-callback
```

For GitHub OAuth:

1. Create a new OAuth App in your GitHub account.
2. Set **Homepage URL** to your `PUBLIC_URL`.
3. Set **Authorization callback URL** to `${PUBLIC_URL}/api/v1/auth/login-callback`.
4. Add the client credentials to your `.env` file:

```bash
PUBLIC_URL=https://arcana.example.com
OAUTH_GITHUB_CLIENT_ID=your_github_client_id
OAUTH_GITHUB_CLIENT_SECRET=your_github_client_secret
```

For Google OAuth:

1. Create OAuth credentials in your Google Cloud project.
2. Use the same callback endpoint: `${PUBLIC_URL}/api/v1/auth/login-callback`.
3. Add the client credentials to your `.env` file:

```bash
PUBLIC_URL=https://arcana.example.com
OAUTH_GOOGLE_CLIENT_ID=your_google_client_id
OAUTH_GOOGLE_CLIENT_SECRET=your_google_client_secret
```

Make sure `PUBLIC_URL` matches the externally accessible HTTPS address of your ARCANA instance and does not include the callback path itself. If the URL configured in the OAuth provider does not exactly match the callback generated by ARCANA, the provider will reject the login attempt with a redirect URI mismatch error.

## Docker Image Configuration

ARCANA allows you to configure Docker image selection for executing various tasks. The system automatically chooses the most appropriate image based on the task type, but you can constrain this selection by specifying your preferred images:

| Variable                           | Default                | Description                                                 |
| ---------------------------------- | ---------------------- | ----------------------------------------------------------- |
| `PENTAGI_IMAGE`                    | `vxcontrol/pentagi:latest` | Docker image used for the main ARCANA application service |
| `DOCKER_DEFAULT_IMAGE`             | `debian:latest`        | Default Docker image for general tasks and ambiguous cases  |
| `DOCKER_DEFAULT_IMAGE_FOR_PENTEST` | `vxcontrol/kali-linux` | Default Docker image for security/penetration testing tasks |
| `DOCKER_ALLOWED_IMAGES`            | empty                  | Comma-separated allow-list the selected image must belong to |
| `DOCKER_IMAGE_SELECTION_MODE`      | `llm`                  | `llm` lets the model pick the image, `fixed` always uses the pentest image |

`PENTAGI_IMAGE` changes the image used by the main `pentagi` service in `docker-compose.yml`. The `DOCKER_DEFAULT_IMAGE` and `DOCKER_DEFAULT_IMAGE_FOR_PENTEST` variables only affect automatic worker image selection for task execution inside ARCANA. They do not rewrite the rest of the Compose stack, so services such as `pgvector`, `scraper`, and the optional `graphiti` stack still use the image references defined in the compose files.

`DOCKER_DEFAULT_IMAGE` and `DOCKER_DEFAULT_IMAGE_FOR_PENTEST` are suggestions the model receives in the image selection prompt, not limits: on their own they do not stop a model from answering with some other image. Two settings turn them into limits:

- **`DOCKER_ALLOWED_IMAGES`** restricts the choice. An answer outside the list is discarded and the pentest image is used instead. An entry without a tag admits any tag of that repository, so `vxcontrol/kali-linux` also admits `vxcontrol/kali-linux:2026.1`. While the list is empty the model may pick any image, which is the default behaviour.
- **`DOCKER_IMAGE_SELECTION_MODE=fixed`** removes the choice altogether: the flow always runs in `DOCKER_DEFAULT_IMAGE_FOR_PENTEST` and no model call is made. Use it when weaker or local models answer with the wrong environment.

Regardless of these settings, an answer that is not a usable image reference never reaches Docker: the flow falls back to the pentest image and the rejected answer is logged.

Restricting the choice is useful for:

- **Security Enforcement**: Restricting usage to only verified and trusted images
- **Environment Standardization**: Using corporate or customized images across all operations
- **Performance Optimization**: Utilizing pre-built images with necessary tools already installed

Configuration examples:

```bash
# Using a custom ARCANA application image
PENTAGI_IMAGE=registry.example.com/security/pentagi:latest

# Using a custom image for general tasks
DOCKER_DEFAULT_IMAGE=mycompany/custom-debian:latest

# Using a specialized image for penetration testing
DOCKER_DEFAULT_IMAGE_FOR_PENTEST=mycompany/pentest-tools:v2.0
```

> [!NOTE]
> If a user explicitly specifies a particular Docker image in their task, the system will try to use that exact image, ignoring these settings. These variables only affect the system's automatic image selection process.

For an advanced OpenVAS/GVM experiment that uses a custom pentest image, see [OpenVAS via a Custom Pentest Image](../../examples/guides/openvas-custom-image.md).

### Restricted Networks, Docker Mirrors, and Proxies

If your environment cannot reach Docker Hub (`docker.io`) directly, changing ARCANA environment variables is usually not enough to fix image download failures. ARCANA still relies on Docker's own registry access for Compose-managed services, and the installer network checks also validate Docker Hub reachability.

For restricted networks:

1. Confirm that the host can resolve and reach `docker.io`.
2. If your environment requires an outbound proxy for ARCANA or installer HTTP traffic, set the `PROXY_URL` environment variable. To route Docker image pulls through a proxy, configure the Docker daemon or Docker Desktop proxy separately — Docker does not use ARCANA's `PROXY_URL` for registry access.
3. If Docker Hub is blocked or heavily rate-limited, configure an organization-approved registry mirror or registry proxy before running the installer or `docker compose up`.
4. Restart Docker after changing the daemon configuration, then rerun the installer checks or Compose startup.

Example Docker daemon mirror configuration:

```json
{
  "registry-mirrors": ["https://mirror.example.com"]
}
```

On Linux, this is typically configured in `/etc/docker/daemon.json`. On Docker Desktop, use the equivalent Docker Engine or proxy settings. A Docker Hub mirror covers Docker Hub-hosted images such as `vxcontrol/*`, but the main Compose stack already includes `quay.io/prometheuscommunity/postgres-exporter`, the optional observability stack includes `gcr.io/cadvisor/cadvisor`, and the optional Langfuse stack includes `cgr.dev/chainguard/minio`. Those registries still need direct access or individually approved proxy/mirror paths.

See the official Docker documentation for [registry mirrors](https://docs.docker.com/docker-hub/image-library/mirror/) and [daemon proxy configuration](https://docs.docker.com/engine/daemon/proxy/).

### Troubleshooting: "failed to select primary docker image via llm call"

A flow that fails immediately with `failed to select primary docker image via llm call` usually indicates a problem with the configured LLM backend, not with Docker or the image registry. Older ARCANA versions reported the same failure as `failed to get primary docker image`, which led users to debug Docker even though the registry was healthy.

When a flow starts, ARCANA makes its first LLM call to choose the primary Docker image for the task. This image-selection call runs through the `simple` agent type, so a failure here points at the model assigned to that agent type rather than at Docker. A message such as `API returned unexpected status code: 502` or `404` in this context is returned by the LLM backend, not by Docker Hub.

This is distinct from the registry reachability problems described above: if Docker pulls succeed and the Compose stack starts, but flow creation still fails at image selection, investigate the LLM backend rather than Docker.

To diagnose:

1. Check ARCANA logs first: `docker logs pentagi`.
2. Check the logs of your configured LLM backend (the server behind your provider or `LLM_SERVER_URL`).
3. Verify that the base URL, API key, and model name in [Custom LLM Provider Configuration](providers.md#custom-llm-provider-configuration) are correct and reachable from the container. If you assign different models per agent type, check the model used by the `simple` agent type, since image selection runs through it.
4. For custom, OpenAI-compatible, vLLM, or SGLang backends, confirm that the model supports tool calling (function calling) and that the matching tool-call parser is enabled. A missing or mismatched tool-call parser is a known cause of this failure.

## Building

### Building Docker Image

The Docker build process automatically embeds version information from git tags. To properly version your build, use the provided scripts:

#### Linux/macOS

```bash
# Load version variables
source ./scripts/version.sh

# Standard build
docker build \
  --build-arg PACKAGE_VER=$PACKAGE_VER \
  --build-arg PACKAGE_REV=$PACKAGE_REV \
  -t pentagi:$PACKAGE_VERSION_FULL .

# Multi-platform build
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg PACKAGE_VER=$PACKAGE_VER \
  --build-arg PACKAGE_REV=$PACKAGE_REV \
  -t pentagi:$PACKAGE_VERSION_FULL .

# Build and push
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg PACKAGE_VER=$PACKAGE_VER \
  --build-arg PACKAGE_REV=$PACKAGE_REV \
  -t myregistry/pentagi:$PACKAGE_VERSION_FULL \
  --push .
```

#### Windows (PowerShell)

```powershell
# Load version variables
. .\scripts\version.ps1

# Standard build
docker build `
  --build-arg PACKAGE_VER=$env:PACKAGE_VER `
  --build-arg PACKAGE_REV=$env:PACKAGE_REV `
  -t pentagi:$env:PACKAGE_VERSION_FULL .

# Multi-platform build
docker buildx build `
  --platform linux/amd64,linux/arm64 `
  --build-arg PACKAGE_VER=$env:PACKAGE_VER `
  --build-arg PACKAGE_REV=$env:PACKAGE_REV `
  -t pentagi:$env:PACKAGE_VERSION_FULL .
```

#### Quick build without version

For development builds without version tracking:

```bash
docker build -t pentagi:dev .
```

> [!NOTE]
> - The build scripts automatically determine version from git tags
> - Release builds (on tag commit) have no revision suffix
> - Development builds (after tag) include commit hash as revision (e.g., `1.1.0-ce.hbc6e800`); the edition is `ce` for Community and `ee` for Enterprise, and a release build is `1.1.0-ce`
> - To use the built image locally, update the image name in `docker-compose.yml` or use the `build` option
