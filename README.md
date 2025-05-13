# PV Reclaim Controller

The **PV Reclaim Controller** is a custom Kubernetes controller designed to monitor and manage local `PersistentVolumes (PVs)` in the `Released` state. It transitions them back to the `Available` state, making them eligible for binding with new `PersistentVolumeClaims (PVCs)`.

**Note:** This controller does **not** clean up the underlying volume data till the date. You need to handle this manually if you want to clean up the underylying storage resource.

---

## 🚀 Deployment Instructions

Follow these steps to deploy the controller into your Kubernetes cluster:

### 1. Add the Helm Repository

```bash
helm repo add pv-reclaim https://greninja517.github.io/pv-reclaimer-controller/
helm repo update
```

### 2. Install the Helm Chart
```bash
helm install pv-reclaim-release pv-reclaim/pvreclaimer
# Optional: specify a namespace
# helm install pv-reclaim-release pv-reclaim/pvreclaimer --namespace=<your-namespace>
```

## 📄 Example Usage
### 1. Create a PVReclaim Custom Resource

Save the following YAML to a file: pv-reclaim.yml
```yaml
apiVersion: reclaim.pv-reclaimer.io/v1alpha1
kind: PVReclaim
metadata:
  name: pv-reclaimer
spec:
  persistentVolumeName: "myapp-pv"
```

### 2. Apply the CR to the Cluster
```bash
kubectl apply -f pv-reclaim.yml
```

### 3. Verify the Resource
```bash
kubectl get pvr
```
pvr is the short name for the PVReclaim custom resource.

## Notes:
Ensure that the PV specified in the persistentVolumeName exists and is in the Released state.

The controller will only transition PVs to the Available state. Cleaning up underlying volume data must be handled separately.