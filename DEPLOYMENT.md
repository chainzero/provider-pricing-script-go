# Production Deployment Guide

This guide provides production-ready deployment strategies for the Go-based Akash pricing script. It is intended for DevOps teams deploying to Kubernetes clusters.

## Current Status

✅ **Go pricing script is validated and working**
- Tested in isolation with multiple scenarios
- Validated against production baseline bids
- Deployed and confirmed working on test provider
- Code changes verified with 10x CPU test (conclusive proof)

## Deployment Challenge

The compiled Go binary (1.8MB after UPX compression) is too large for Kubernetes ConfigMaps (1MB limit), requiring alternative deployment strategies.

## Production Deployment Options

### Option 1: Init Container + GitHub Releases (Recommended) 🏆

**Best for:** Multi-node clusters, automated deployments, version control

#### Overview
An init container downloads the binary from GitHub Releases on pod startup. The binary is stored in an `emptyDir` volume shared with the main provider container.

#### Pros
- ✅ Works on any node in the cluster
- ✅ No manual file management
- ✅ Version controlled via GitHub Releases
- ✅ Easy rollback (change release tag)
- ✅ Scales automatically with cluster
- ✅ Cloud-native best practice
- ✅ No custom images or registries needed

#### Cons
- Requires internet access from cluster
- Slight delay on pod startup (~2-5 seconds download)
- Dependency on GitHub availability

#### Implementation

**Step 1: Create GitHub Release**

```bash
# Build production binary
cd provider-pricing-script-go
GOOS=linux GOARCH=amd64 go build -o pricing-tool-linux cmd/pricing-tool/main.go

# Compress (reduces from ~8MB to ~1.8MB)
upx --best --lzma pricing-tool-linux

# Create GitHub release
gh release create v1.0.0 pricing-tool-linux \
  --title "v1.0.0 - Production Release" \
  --notes "Initial production release of Go-based pricing script"
```

**Step 2: Update Helm Chart**

Add to `provider.yaml` or create a separate values file:

```yaml
# provider-pricing-init.yaml
initContainers:
  - name: fetch-pricing-binary
    image: curlimages/curl:8.5.0
    command:
      - sh
      - -c
      - |
        set -e
        echo "Downloading pricing binary from GitHub..."
        curl -fsSL -o /pricing/bidpricescript \
          https://github.com/chainzero/provider-pricing-script-go/releases/download/v1.0.0/pricing-tool-linux
        
        chmod +x /pricing/bidpricescript
        
        echo "Binary downloaded successfully:"
        ls -lh /pricing/bidpricescript
        
        # Optional: Verify it's a valid ELF binary
        file /pricing/bidpricescript
    volumeMounts:
      - name: pricing-binary
        mountPath: /pricing

# Add volume mount to main provider container
extraVolumeMounts:
  - name: pricing-binary
    mountPath: /bidprice
    readOnly: true

# Define the shared volume
extraVolumes:
  - name: pricing-binary
    emptyDir: {}

# Ensure pricing strategy is set
env:
  - name: AKASH_BID_PRICE_STRATEGY
    value: shellScript
  - name: AKASH_BID_PRICE_SCRIPT_PATH
    value: /bidprice/bidpricescript
```

**Step 3: Deploy**

```bash
helm upgrade akash-provider akash/provider -n akash-services \
  -f provider.yaml \
  -f provider-pricing-init.yaml
```

**Step 4: Verify**

```bash
# Check init container ran successfully
kubectl describe pod akash-provider-0 -n akash-services | grep -A 10 "Init Containers"

# Verify binary is accessible
kubectl exec -it akash-provider-0 -n akash-services -- ls -lh /bidprice/bidpricescript

# Test the binary
kubectl exec -it akash-provider-0 -n akash-services -- /bidprice/bidpricescript --help
```

#### Updating to New Version

```bash
# 1. Create new release
gh release create v1.0.1 pricing-tool-linux --title "v1.0.1 - Bug fixes"

# 2. Update the release tag in provider-pricing-init.yaml
# Change: v1.0.0 -> v1.0.1

# 3. Restart pods
kubectl rollout restart statefulset akash-provider -n akash-services
```

---

### Option 2: Custom Provider Docker Image

**Best for:** Air-gapped clusters, fastest startup times, maximum integration

#### Overview
Build a custom provider image with the pricing binary embedded.

#### Pros
- ✅ Binary always available (no download needed)
- ✅ Fastest pod startup time
- ✅ Works in air-gapped environments
- ✅ Most integrated solution
- ✅ Version pinned to image tag

#### Cons
- Requires container registry (Docker Hub, GHCR, etc.)
- Need to rebuild image for updates
- Slightly more complex CI/CD

#### Implementation

**Step 1: Create Dockerfile**

```dockerfile
# Dockerfile
FROM ghcr.io/akash-network/provider:0.10.1

# Copy the pricing binary
COPY pricing-tool-linux /usr/local/bin/bidpricescript
RUN chmod +x /usr/local/bin/bidpricescript

# Verify binary works
RUN /usr/local/bin/bidpricescript --help || echo "Binary loaded (no --help flag)"

# Add metadata
LABEL org.opencontainers.image.source="https://github.com/chainzero/provider-pricing-script-go"
LABEL org.opencontainers.image.description="Akash Provider with Go pricing script"
LABEL org.opencontainers.image.version="1.0.0"
```

**Step 2: Build and Push**

```bash
# Build for your platform
docker build -t yourregistry/akash-provider-custom:0.10.1-v1.0.0 .

# Push to registry
docker push yourregistry/akash-provider-custom:0.10.1-v1.0.0
```

**Step 3: Update StatefulSet**

```bash
# Update image
kubectl set image statefulset/akash-provider \
  provider=yourregistry/akash-provider-custom:0.10.1-v1.0.0 \
  init=yourregistry/akash-provider-custom:0.10.1-v1.0.0 \
  -n akash-services

# Or edit provider.yaml
# image:
#   repository: yourregistry/akash-provider-custom
#   tag: 0.10.1-v1.0.0
```

**Step 4: Configure Environment**

```yaml
env:
  - name: AKASH_BID_PRICE_STRATEGY
    value: shellScript
  - name: AKASH_BID_PRICE_SCRIPT_PATH
    value: /usr/local/bin/bidpricescript
```

---

### Option 3: DaemonSet Distribution

**Best for:** Multi-node clusters with persistent nodes, HostPath compatibility

#### Overview
A DaemonSet ensures all nodes have the binary in `/opt/akash/`.

#### Pros
- ✅ Automatically deploys to all nodes
- ✅ New nodes get binary automatically
- ✅ Simple provider pod configuration

#### Cons
- Still relies on HostPath
- Extra DaemonSet to manage
- Binary updates require DaemonSet rollout

#### Implementation

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: pricing-binary-distributor
  namespace: akash-services
spec:
  selector:
    matchLabels:
      app: pricing-binary-distributor
  template:
    metadata:
      labels:
        app: pricing-binary-distributor
    spec:
      containers:
      - name: distributor
        image: curlimages/curl:8.5.0
        command:
        - sh
        - -c
        - |
          set -e
          echo "Ensuring pricing binary is present on this node..."
          
          curl -fsSL -o /host-opt/bidpricescript \
            https://github.com/chainzero/provider-pricing-script-go/releases/download/v1.0.0/pricing-tool-linux
          
          chmod +x /host-opt/bidpricescript
          
          echo "Binary ready on node $(hostname)"
          echo "Watching for updates..."
          
          # Sleep forever (DaemonSet keeps binary available)
          while true; do sleep 3600; done
        volumeMounts:
        - name: host-opt
          mountPath: /host-opt
      volumes:
      - name: host-opt
        hostPath:
          path: /opt/akash
          type: DirectoryOrCreate
```

Provider configuration remains using HostPath:
```yaml
volumes:
  - name: bidprice-script
    hostPath:
      path: /opt/akash
      type: Directory
```

---

### Option 4: Shared Persistent Volume

**Best for:** Clusters with shared storage (NFS, Ceph, EFS, etc.)

#### Overview
Store binary on shared storage accessible from all nodes.

#### Pros
- ✅ Works on all nodes
- ✅ Single source of truth
- ✅ Easy updates (update once, all pods get it)
- ✅ No init container overhead

#### Cons
- Requires shared storage infrastructure
- Not all clusters have this capability
- Potential single point of failure

#### Implementation

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pricing-binary-pvc
  namespace: akash-services
spec:
  accessModes:
    - ReadOnlyMany
  storageClassName: nfs-shared  # Your shared storage class
  resources:
    requests:
      storage: 10Mi
```

```yaml
# In provider StatefulSet
volumes:
  - name: pricing-binary
    persistentVolumeClaim:
      claimName: pricing-binary-pvc
      readOnly: true
```

Upload binary to shared volume:
```bash
# Mount PVC to a temporary pod
kubectl run -it --rm uploader --image=alpine --restart=Never \
  --overrides='{"spec":{"volumes":[{"name":"pvc","persistentVolumeClaim":{"claimName":"pricing-binary-pvc"}}],"containers":[{"name":"uploader","image":"alpine","command":["sh"],"volumeMounts":[{"name":"pvc","mountPath":"/data"}]}]}}'

# Inside the pod:
wget https://github.com/.../pricing-tool-linux -O /data/bidpricescript
chmod +x /data/bidpricescript
exit
```

---

## Recommendation Matrix

| Criteria | Option 1: Init Container | Option 2: Custom Image | Option 3: DaemonSet | Option 4: Shared PV |
|----------|-------------------------|------------------------|---------------------|---------------------|
| **Ease of Setup** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **Multi-node Support** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Version Control** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Air-gap Compatible** | ⭐ | ⭐⭐⭐⭐⭐ | ⭐ | ⭐⭐⭐⭐⭐ |
| **Startup Speed** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Update Ease** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Infrastructure Req** | None | Registry | None | Shared Storage |

## Testing After Deployment

### Verify Binary is Working

```bash
# 1. Check binary exists
kubectl exec -it akash-provider-0 -n akash-services -- ls -lh /bidprice/bidpricescript

# 2. Check environment variables
kubectl exec -it akash-provider-0 -n akash-services -- env | grep AKASH_BID_PRICE

# 3. Test manual execution
kubectl exec -it akash-provider-0 -n akash-services -- sh -c '
echo "0.5668" > /tmp/aktprice.cache
echo '\''{"resources":[{"memory":268435456,"cpu":100,"storage":[{"class":"default","size":268435456}],"count":1,"endpoint_quantity":1,"ip_lease_quantity":0}],"price":{"denom":"uakt","amount":"100000"},"price_precision":18}'\'' | /bidprice/bidpricescript
'

# 4. Monitor real bids
kubectl logs -f akash-provider-0 -n akash-services | grep "submitting fulfillment"
```

### Compare Against Known Values

Create a test deployment and verify the bid price matches expected calculations based on your `PRICE_TARGET_*` configuration.

## Rollback Procedures

### Option 1 (Init Container)
```bash
# Revert to previous release version
# Edit provider-pricing-init.yaml: v1.0.1 -> v1.0.0
kubectl rollout restart statefulset akash-provider -n akash-services
```

### Option 2 (Custom Image)
```bash
# Rollback to previous image
kubectl rollout undo statefulset akash-provider -n akash-services

# Or set specific version
kubectl set image statefulset/akash-provider \
  provider=yourregistry/akash-provider-custom:0.10.1-v1.0.0 \
  -n akash-services
```

### Emergency: Revert to Built-in Pricing
```bash
# Set strategy back to scale
kubectl set env statefulset/akash-provider \
  AKASH_BID_PRICE_STRATEGY=scale \
  -n akash-services

kubectl rollout restart statefulset akash-provider -n akash-services
```

## Monitoring and Observability

### Key Metrics to Monitor

1. **Bid success rate** - Are you winning bids?
2. **Bid prices** - Are they reasonable for your costs?
3. **Script execution time** - Should be < 1 second
4. **Error rates** - Check logs for script failures

### Logging

Enable debug logging if needed:
```yaml
env:
  - name: DEBUG_BID_SCRIPT
    value: "1"
```

Check logs:
```bash
kubectl logs akash-provider-0 -n akash-services | grep DEBUG
```

## Security Considerations

1. **Binary Verification**: Consider adding checksum validation in init container
2. **GitHub Access**: Use private repos if binary contains proprietary logic
3. **Registry Security**: Use private registries with authentication for Option 2
4. **Least Privilege**: Mount volumes as read-only where possible

## Cost Optimization

- **Option 1**: Free (uses GitHub Releases)
- **Option 2**: Registry storage costs (~$0.10-1/month)
- **Option 3**: Minimal (DaemonSet overhead)
- **Option 4**: Shared storage costs (varies)

## Support and Troubleshooting

### Common Issues

**Problem**: Init container fails to download binary
- **Solution**: Check cluster internet access, verify GitHub Release URL

**Problem**: Binary not executable
- **Solution**: Verify `chmod +x` in init container or image build

**Problem**: "No such file or directory" error
- **Solution**: Check volume mounts and `AKASH_BID_PRICE_SCRIPT_PATH`

**Problem**: Bids not using script
- **Solution**: Verify `AKASH_BID_PRICE_STRATEGY=shellScript` is set

### Debug Commands

```bash
# Check all environment variables
kubectl exec akash-provider-0 -n akash-services -- env | sort

# Check all mounts
kubectl exec akash-provider-0 -n akash-services -- mount | grep bidprice

# Test binary manually
kubectl exec akash-provider-0 -n akash-services -- /bidprice/bidpricescript --help
```

## Conclusion

**Recommended Approach for Production: Option 1 (Init Container + GitHub Releases)**

This provides the best balance of:
- Ease of deployment
- Multi-node compatibility  
- Version control
- Update simplicity
- Cost effectiveness

For air-gapped or security-sensitive environments, **Option 2 (Custom Image)** is recommended.

The DevOps team should choose based on:
1. Cluster topology (single node vs. multi-node)
2. Internet access availability
3. Existing infrastructure (registries, shared storage)
4. Update frequency requirements
5. Security and compliance needs

