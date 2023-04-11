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

//
// Kubeconform.

type Kubeconform struct {
	IO     io.ReadWriter
	Config *config.Config
}

// LoadResourceListItems links between Kustomize KRM input and Kubeconform input.
func (kc *Kubeconform) loadResourceListItems(rlItems []*yaml.RNode) {
	err := (&kio.ByteWriter{Writer: kc.IO}).Write(rlItems)
	if err != nil {
		log.Fatal("failed to load ResourceList items: ", err)
	}
}

func (kc *Kubeconform) configure(kcvSpec *kubeconformValidatorSpec) (config.Config, string) {
	cfg, out, err := config.FromFlags(os.Args[0], kcvSpec.Args)
	if err != nil {
		log.Fatal("failed to parse args: ", err)
	}

	// Override config using KubeconformValidator spec.config.
	// That means "spec.config" has a priority over "spec.args".
	if err := mergo.Merge(&cfg, kcvSpec.Config, mergo.WithOverride); err != nil {
		log.Fatal("failed to merge config: ", err)
	}

	cfg.LoadNGConfig()

	// Configure Kubeconform IO.
	cfg.Stream.Input = kc.IO
	cfg.Stream.Output = kc.IO

	return cfg, out
}

//
// KubeconformValidator.

type KubeconformValidator struct {
	Kind     string `yaml:"kind" json:"kind"`
	Metadata struct {
		Name string `yaml:"name" json:"name"`
	}
	Spec kubeconformValidatorSpec `yaml:"spec" json:"spec"`
}

type kubeconformValidatorSpec struct {
	Args   []string       `yaml:"args" json:"args"`
	Config *config.Config `yaml:"config" json:"config"`
}

//go:embed plugin-schema.yaml
var kubeconformValidatorDefinition string

func (kcv *KubeconformValidator) Schema() (*spec.Schema, error) {
	schema, err := framework.SchemaFromFunctionDefinition(
		resid.NewGvk("validators.kustomize.aabouzaid.com", "v1alpha1", "KubeconformValidator"),
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

func (kcv *KubeconformValidator) Filter(rlItems []*yaml.RNode) ([]*yaml.RNode, error) {
	kc := &Kubeconform{IO: &bytes.Buffer{}}
	kc.loadResourceListItems(rlItems)
	cfg, out := kc.configure(&kcv.Spec)

	// Run Kubeconform validate.
	if err := kubeconform.Validate(cfg, out); err != nil {
		return nil, errors.Wrap(errors.Errorf("Kubeconform validation output: %s", kc.IO))
	}

	return rlItems, nil
}

func main() {
	rlSource := &kio.ByteReadWriter{}
	processor := &framework.VersionedAPIProcessor{FilterProvider: framework.GVKFilterMap{
		"KubeconformValidator": {
			"validators.kustomize.aabouzaid.com/v1alpha1": &KubeconformValidator{},
		}}}

	if err := framework.Execute(processor, rlSource); err != nil {
		log.Fatal(err)
	}
}
