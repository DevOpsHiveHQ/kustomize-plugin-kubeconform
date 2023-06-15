package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"io/ioutil"
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

// write Filter fun to validate resource list items and in case if error,
// read the kc.IO json output and map it to results in framework.Results
func (kcv *KubeconformValidator) Filter(rlItems []*yaml.RNode) ([]*yaml.RNode, error) {
	kc := &Kubeconform{IO: &bytes.Buffer{}}
	kc.loadResourceListItems(rlItems)
	cfg, out := kc.configure(&kcv.Spec)

	type validationError struct {
		Path string `json:"path"`
		Msg  string `json:"msg"`
	}

	type resource struct {
		Filename         string            `json:"filename"`
		Kind             string            `json:"kind"`
		Name             string            `json:"name"`
		Version          string            `json:"version"`
		Status           string            `json:"status"`
		Msg              string            `json:"msg"`
		ValidationErrors []validationError `json:"validationErrors"`
	}

	type ValidationOutput struct {
		Resources []resource `json:"resources"`
	}

	if err := kubeconform.Validate(cfg, out); err != nil {

		// read kc.IO stream and covert it to string
		kcIO, _ := ioutil.ReadAll(kc.IO)
		// unmarshal kcIO json output to ValidationOutput struct
		var vo ValidationOutput
		if err := json.Unmarshal(kcIO, &vo); err != nil {
			return nil, errors.WrapPrefixf(err, "failed to unmarshal kc.IO json output")
		}
		var validationResults framework.Results
		for _, resource := range vo.Resources {
			for errIndex, _ := range resource.ValidationErrors {

				validationResults = append(validationResults, &framework.Result{
					Message:  resource.ValidationErrors[0].Msg,
					Severity: "error",
					ResourceRef: &yaml.ResourceIdentifier{
						TypeMeta: yaml.TypeMeta{
							Kind: resource.Kind,
						},
						NameMeta: yaml.NameMeta{
							Name: resource.Name,
						},
					},
					Field: &framework.Field{
						Path:          resource.ValidationErrors[errIndex].Path,
						ProposedValue: 1,
					},
				})
			}
		}
		return nil, validationResults
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
