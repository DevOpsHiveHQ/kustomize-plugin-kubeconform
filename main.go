package main

import (
	"bytes"
	_ "embed"
	"io"
	"log"
	"os"

	"github.com/imdario/mergo"
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
	Kind     string `yaml:"kind" json:"kind"`
	Metadata struct {
		Name string `yaml:"name" json:"name"`
	}
	Spec kubeconformValidatorSpec `yaml:"spec" json:"spec"`
}

type kubeconformValidatorSpec struct {
	Args   []string      `yaml:"args" json:"args"`
	Config config.Config `yaml:"config" json:"config"`
}

//go:embed plugin-schema.yaml
var kubeconformValidatorDefinition string

func (kcv *KubeconformValidator) Schema() (*spec.Schema, error) {
	schema, err := framework.SchemaFromFunctionDefinition(
		resid.NewGvk("kubeconformvalidator.aabouzaid.com", "v1alpha1", "KubeconformValidator"),
		kubeconformValidatorDefinition)
	return schema, errors.WrapPrefixf(err, "failed to parse KubeconformValidator schema")
}

func (kcv *KubeconformValidator) Validate() error {
	// At the moment this only validates KubeconformValidator manifest against its openAPIV3Schema.
	return nil
}

// TODO: Implement Default methods if needed.
// func (kcv *KubeconformValidator) Default() error {
// 	return nil
// }

// LoadResourceListItems links between Kustomize KRM input and Kubeconform input.
// TODO: Review the current approach to find out if there is a better solution.
func (kc *Kubeconform) loadResourceListItems(rlItems []*yaml.RNode) {
	var tmpWriter *io.PipeWriter
	kc.Input, tmpWriter = io.Pipe()
	go func() {
		defer tmpWriter.Close()
		err := (&kio.ByteWriter{Writer: tmpWriter}).Write(rlItems)
		if err != nil {
			log.Fatal("failed to load ResourceList items: ", err)
		}
	}()
}

func (kcv *KubeconformValidator) configure() (config.Config, string) {
	cfg, out, err := config.FromFlags(os.Args[0], kcv.Spec.Args)
	if err != nil {
		log.Fatal("failed to parse args: ", err)
	}

	// Override config using KubeconformValidator spec.config.
	// That means "spec.config" has a priority over "spec.args".
	if err := mergo.Merge(&cfg, kcv.Spec.Config, mergo.WithOverride); err != nil {
		log.Fatal("failed to merge config: ", err)
	}

	return cfg, out
}

func (kcv *KubeconformValidator) Filter(rlItems []*yaml.RNode) ([]*yaml.RNode, error) {
	kc := &Kubeconform{}
	kc.loadResourceListItems(rlItems)

	// Configure Kubeconform.
	cfg, out := kcv.configure()
	cfg.Stream = &config.Stream{
		Input:  kc.Input,
		Output: &kc.Output,
	}

	// Run Kubeconform validate.
	if exitCode := kubeconform.Validate(cfg, out); exitCode != 0 {
		return nil, errors.Wrap(errors.Errorf("Kubeconform validation output: %s", kc.Output.String()))
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
