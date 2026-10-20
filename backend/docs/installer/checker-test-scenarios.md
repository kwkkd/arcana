# Checker Test Scenarios

This document outlines test scenarios for the installer's system checking functionality, focusing on failure modes and their detection.

The scenarios cover what the checker observes about this machine — Docker, disk, memory, local connectivity — and the failure mode each check detects.

## Test Scenarios

### 1. Docker Not Installed
**Setup**: Remove Docker from the system
**Expected**:
- `DockerErrorType`: "not_installed"
- `DockerApiAccessible`: false
- `DockerInstalled`: false
- UI shows: "Docker Not Installed" with installation instructions

### 2. Docker Daemon Not Running
**Setup**: Install Docker but stop the daemon (e.g., quit Docker Desktop on macOS)
**Expected**:
- `DockerErrorType`: "not_running"
- `DockerApiAccessible`: false
- `DockerInstalled`: true
- UI shows: "Docker Daemon Not Running" with start instructions

### 3. Docker Permission Denied
**Setup**: Run installer as non-docker user on Linux
**Expected**:
- `DockerErrorType`: "permission"
- `DockerApiAccessible`: false
- `DockerInstalled`: true
- UI shows: "Docker Permission Denied" with usermod instructions

### 4. Remote Docker Connection Failed
**Setup**: Set DOCKER_HOST to invalid address
**Expected**:
- `DockerErrorType`: "api_error"
- `DockerApiAccessible`: false
- UI shows: "Docker API Connection Failed" with DOCKER_HOST troubleshooting

### 5. Write Permissions Denied
**Setup**: Run installer in read-only directory
**Expected**:
- `EnvDirWritable`: false
- UI shows: "Write Permissions Required" with chmod instructions

### 6. Network Issues - DNS Failure
**Setup**: Break name resolution itself — firewall port 53, or point the resolver at a dead server. Adding `127.0.0.1 docker.io` to /etc/hosts does **not** trigger this: the name resolves, and what fails is scenario 7.
**Expected**:
- `SysNetworkFailures`: ["• DNS resolution failed for docker.io"]
- `SysNetworkOK`: false
- UI shows specific DNS failure with resolution steps

The probe is a plain host lookup and it never goes through `PROXY_URL`. On a network where only the proxy can resolve names, this line appears while everything the installer actually does works — treat it as a hint, not a verdict.

### 7. Network Issues - HTTPS Blocked
**Setup**: Block outbound HTTPS (port 443), or point docker.io at a closed port
**Expected**:
- `SysNetworkFailures`: ["• Cannot reach external services via HTTPS"]
- UI shows HTTPS failure with proxy configuration info

The probe is a 5-second GET to https://docker.io through `PROXY_URL` when one is set, and **any answer below 500 counts as reachable** — an interception proxy replying 403 to the probe passes it. It says the path is open, not that the registry is usable.

### 8. Network Issues - Docker Registry Blocked
**Setup**: Block docker.io for the Docker daemon
**Expected**:
- `SysNetworkFailures`: ["• Cannot pull Docker images from registry"]
- UI shows registry access failure

The probe pulls `debian:latest` and succeeds if **either** the main daemon or the worker daemon can pull it, so a broken remote worker alone does not raise this line. It is skipped entirely when either client is absent — an unreachable daemon is scenarios 2-4, not this one.

### 9. Behind Corporate Proxy
**Setup**: Network requires a proxy; `PROXY_URL` unset, or set while the daemon has no proxy of its own
**Expected**:
- Multiple network failures
- UI shows proxy configuration instructions for HTTP_PROXY/HTTPS_PROXY

Only the HTTPS probe uses `PROXY_URL`. The lookup resolves locally, and the pull goes through the Docker daemon with the daemon's own proxy configuration — so a correctly configured `PROXY_URL` can still leave two of the three lines standing.

### 10. Low Memory
**Setup**: System with < 2GB available RAM
**Expected**:
- `SysMemoryOK`: false
- `SysMemoryAvailable`: < 2.0
- UI shows memory requirements with specific numbers

### 11. Low Disk Space
**Setup**: System with < 25GB free space
**Expected**:
- `SysDiskFreeSpaceOK`: false
- `SysDiskAvailable`: < 25.0
- UI shows disk requirements with cleanup suggestions

### 12. Worker Docker Environment Issues
**Setup**: Configure DOCKER_HOST for remote, but remote unavailable
**Expected**:
- `WorkerEnvApiAccessible`: false
- UI shows worker environment troubleshooting

## Environment Variable Tests

### 1. HTTP_PROXY Auto-Detection
**Setup**: Set HTTP_PROXY before running installer
**Expected**: PROXY_URL in .env automatically populated

### 2. DOCKER_HOST Inheritance
**Setup**: Set DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH
**Expected**: 
- Values synchronized to .env on first run via DoSyncNetworkSettings()
- DOCKER_CERT_PATH migrated to PENTAGI_DOCKER_CERT_PATH (host path) + DOCKER_CERT_PATH set to /opt/pentagi/docker/ssl (container path)

## Edge Cases

### 1. Docker Version Too Old
**Setup**: Docker 19.x installed
**Expected**:
- `DockerVersionOK`: false
- UI shows version upgrade instructions

### 2. Docker Compose Missing
**Setup**: Docker installed without Compose
**Expected**:
- `DockerComposeInstalled`: false
- UI shows Compose installation instructions

### 3. Multiple Failures
**Setup**: No Docker + network issues + low resources
**Expected**: All issues shown in priority order:
1. Environment file
2. Write permissions
3. Docker issues
4. Resource issues
5. Network issues

## Testing Commands

```bash
# Simulate Docker not running (macOS)
osascript -e 'quit app "Docker"'

# Simulate permission issues (Linux)
sudo gpasswd -d $USER docker

# Simulate network issues
sudo iptables -A OUTPUT -p tcp --dport 443 -j DROP

# Simulate an unreachable registry — NOT a DNS failure: the name still resolves,
# so this trips the HTTPS probe (scenario 7), not the lookup (scenario 6)
echo "127.0.0.1 docker.io" | sudo tee -a /etc/hosts

# Test with proxy
export HTTP_PROXY=http://proxy:3128
export HTTPS_PROXY=http://proxy:3128

# Test remote Docker
export DOCKER_HOST=tcp://remote:2376
export DOCKER_TLS_VERIFY=1
export DOCKER_CERT_PATH=/path/to/certs  # auto-migrated to PENTAGI_DOCKER_CERT_PATH on startup

go test ./cmd/installer/checker/...
```

## Verification

Each scenario should:
1. Be detected correctly by the checker
2. Show appropriate error message in UI
3. Provide actionable fix instructions
4. Not block other checks unnecessarily
5. Work under both privileged and unprivileged users
