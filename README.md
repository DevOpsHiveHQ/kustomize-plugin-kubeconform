# KubeconformValidator

A wrapper for [Kubeconform](https://github.com/yannh/kubeconform)
to work as a Kustomize validator plugin via KRM functions.

Note: This is still a playground.

## Build

```sh
go build -o 'dist/kubeconformvalidator' .
```

## Example
```yaml
apiVersion: validators.kustomize.aabouzaid.com/v1alpha1
kind: KubeconformValidator
metadata:
  name: validate
  annotations:
    config.kubernetes.io/function: |
      # Exec KRM functions.
      exec:
       path: ../dist/kubeconformvalidator

      # # Containerized KRM functions.
      # container:
      #   image: aabouzaid/kubeconformvalidator
      #   network: true
spec:
  # Configure Kubeconform.
  config:
    output: json
    skip:
    - AlertmanagerConfig
  # Also, direct Kubeconform args could be used but "spec.args" has lower priority over "spec.config".
  # https://github.com/yannh/kubeconform#Usage
  # args:
  # - -output
  # - json
  # - -skip
  # - AlertmanagerConfig
```

## Try
```sh
# Make sure to build bin first.
kustomize build --enable-alpha-plugins --enable-exec ./example
```

The Kustomize output if the validation failed (there are schema errors):
```
Kubeconform validation output: {
  "resources": [
    {
      "filename": "stdin",
      "kind": "Service",
      "name": "validated-core-resource",
      "version": "v1",
      "status": "statusInvalid",
      "msg": "problem validating schema. Check JSON formatting: jsonschema: '/spec/ports/0/port' does not validate with https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/master-standalone/service-v1.json#/properties/spec/properties/ports/items/properties/port/type: expected integer, but got string",
      "validationErrors": [
        {
          "path": "/spec/ports/0/port",
          "msg": "expected integer, but got string"
        }
      ]
    }
  ]
}
Error: couldn't execute function: exit status 1
```

The Kustomize output if the validation succeeded (there are no schema errors):
```
apiVersion: v1
kind: Service
metadata:
  name: validated-core-resource
spec:
  ports:
  - port: 8080
    protocol: TCP
    targetPort: 8080
  type: ClusterIP
---
apiVersion: monitoring.coreos.com/v1alpha1
kind: AlertmanagerConfig
metadata:
  name: skipped-custom-resource
spec:
  receivers:
  - name: webhook
    webhookConfigs:
    - url: http://example.com/
```
