package main

import (
	"bytes"
	"io"
	"log"
	"os"

	"github.com/yannh/kubeconform/cmd/kubeconform/validate"
	"github.com/yannh/kubeconform/pkg/config"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KubeconformValidator struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	}
	Spec struct {
		Args []string `yaml:"args"`
	}
}

type Kubeconform struct {
	Input  *io.PipeReader
	Output bytes.Buffer
}

// LoadResourceListItems is a link between Kustomize KRM input and Kubeconform input.
func (kv *Kubeconform) LoadResourceListItems(rlItems []*yaml.RNode) {
	var tmpWriter *io.PipeWriter
	kv.Input, tmpWriter = io.Pipe()
	go func() {
		defer tmpWriter.Close()
		err := (&kio.ByteWriter{Writer: tmpWriter}).Write(rlItems)
		if err != nil {
			log.Fatalf("failed to load ResourceList items: %s\n", err.Error())
		}
	}()
}

func runKubeconform(rlSource *kio.ByteReadWriter) error {
	kcv := &KubeconformValidator{}
	kc := &Kubeconform{}

	fn := func(rlItems []*yaml.RNode) ([]*yaml.RNode, error) {
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
		if exitCode := validate.Run(cfg, out); exitCode != 0 {
			log.Fatalf("validation output: %s", kc.Output.String())
		}

		return rlItems, nil
	}

	process := framework.SimpleProcessor{Config: kcv, Filter: kio.FilterFunc(fn)}
	err := framework.Execute(process, rlSource)
	return err
}

func main() {
	byteReadWriter := &kio.ByteReadWriter{}
	runKubeconform(byteReadWriter)
}
