# KubeconformValidator

A wrapper for [Kubeconform](https://github.com/yannh/kubeconform)
to work as a Kustomize validator plugin via KRM functions.

Note: This is still a playground.

**Build:**
```sh
go build  -o "dist/kubeconformvalidator" .
```

**Example:**
```yaml
kind: KubeconformValidator
apiVersion: v1alpha
metadata:
  name: validate
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: ./dist/kubeconformvalidator
spec:
  args:
  - -verbose
  - -n
  - 5
```

**Try:**
```sh
# Make sure to build bin first.
kustomize build --enable-alpha-plugins ./example
```
