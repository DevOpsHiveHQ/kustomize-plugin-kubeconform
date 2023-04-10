# KubeconformValidator

A wrapper for [Kubeconform](https://github.com/yannh/kubeconform)
to work as a Kustomize validator plugin via KRM functions.

Note: This is still a playground.

**Build:**
```sh
go build -o 'dist/kubeconformvalidator' .
```

**Example:**
```yaml
kind: KubeconformValidator
apiVersion: kubeconformvalidator.aabouzaid.com/v1alpha1
metadata:
  name: validate
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: ./dist/kubeconformvalidator
spec:
  # Kubeconform args:
  # https://github.com/yannh/kubeconform#Usage
  args:
  - -verbose
  - -output
  - json
```

**Try:**
```sh
# Make sure to build bin first.
kustomize build --enable-alpha-plugins --enable-exec ./example
```

## TODO
- Add native support for Kubeconform options (instead of passing everything as CLI args).
