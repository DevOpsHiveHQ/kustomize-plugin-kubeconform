package main

import (
	"bytes"
	"io"
	"log"
	"os"

	"github.com/yannh/kubeconform/cmd/kubeconform"
	"github.com/yannh/kubeconform/pkg/config"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Kubeconform struct {
	Input  *io.PipeReader
	Output bytes.Buffer
}

type KubeconformValidator struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	}
	Spec kubeconformValidatorSpec
}

type kubeconformValidatorSpec struct {
	Args []string `yaml:"args" json:"args"`
	// TODO: Add a native support for Kubeconform args.
}

var kubeconformValidatorDefinition = `
apiVersion: config.kubernetes.io/v1alpha1
kind: KRMFunctionDefinition
metadata:
  name: kubeconformvalidator
spec:
  group: kubeconformvalidator.aabouzaid.com
  names:
    kind: KubeconformValidator
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
            properties:
              name:
                type: string
                minLength: 1
            required:
            - name
          spec:
            properties:
              args:
                type: array
            type: object
        type: object
`

func (kcv *KubeconformValidator) Schema() (*spec.Schema, error) {
	schema, err := framework.SchemaFromFunctionDefinition(
		resid.NewGvk("kubeconformvalidator.aabouzaid.com", "v1alpha1", "KubeconformValidator"),
		kubeconformValidatorDefinition)
	return schema, errors.WrapPrefixf(err, "failed to parse KubeconformValidator schema")
}

// TODO: Implement Validate and Default methods.
// func (kcv *KubeconformValidator) Validate() error {
// 	return nil
// }
// func (kcv *KubeconformValidator) Default() error {
// 	return nil
// }

// LoadResourceListItems links between Kustomize KRM input and Kubeconform input.
// TODO: Review the current approach to find out if there is a better solution.
func (kc *Kubeconform) LoadResourceListItems(rlItems []*yaml.RNode) {
	var tmpWriter *io.PipeWriter
	kc.Input, tmpWriter = io.Pipe()
	go func() {
		defer tmpWriter.Close()
		err := (&kio.ByteWriter{Writer: tmpWriter}).Write(rlItems)
		if err != nil {
			log.Fatalf("failed to load ResourceList items: %s\n", err.Error())
		}
	}()
}

func (kcv *KubeconformValidator) Filter(rlItems []*yaml.RNode) ([]*yaml.RNode, error) {
	kc := &Kubeconform{}
	cfg, out, err := config.FromFlags(os.Args[0], kcv.Spec.Args)
	if err != nil {
		log.Fatalf("failed to parse args: %s\n", err.Error())
	}

	kc.LoadResourceListItems(rlItems)

	// Configure input and output of Kubeconform.
	cfg.Stream = &config.Stream{
		Input:  kc.Input,
		Output: &kc.Output,
	}

	// Run Kubeconform validate.
	if exitCode := kubeconform.Validate(cfg, out); exitCode != 0 {
		log.Fatalf("validation output: %s", kc.Output.String())
	}

	return rlItems, nil
}

func main() {
	rlSource := &kio.ByteReadWriter{}
	processor := &framework.VersionedAPIProcessor{FilterProvider: framework.GVKFilterMap{
		"KubeconformValidator": {
			"kubeconformvalidator.aabouzaid.com/v1alpha1": &KubeconformValidator{},
		}}}

	if err := framework.Execute(processor, rlSource); err != nil {
		log.Fatal(err)
	}
}
