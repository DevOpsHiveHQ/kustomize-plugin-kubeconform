package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/yannh/kubeconform/cmd/kubeconform/validate"
	"github.com/yannh/kubeconform/pkg/config"
	"gopkg.in/yaml.v2"
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

func (kv *KubeconformValidator) Load(manifest string) {
	// Get KubeconformValidator manifest.
	err := yaml.Unmarshal([]byte(manifest), &kv)
	if err != nil {
		log.Fatal("YAML Unmarshal: ", err)
	}

	// Check if the manifest uses the KubeconformValidator plugin uses the correct kind.
	if kv.Kind != "KubeconformValidator" {
		log.Fatal("Resource kind: The manifest should use \"KubeconformValidator\" kind")
	}
}

func getResourceList(input *os.File) *fn.ResourceList {
	// Read stdin.
	stdin, err := io.ReadAll(input)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the stdin input is ResourceList.
	rl, err := fn.ParseResourceList(stdin)
	if err != nil {
		log.Println(err)
		log.Fatal("ParseResourceList: The input should be a single file of kind \"ResourceList\"")
	}

	return rl
}

func main() {
	var output bytes.Buffer
	krmInput := getResourceList(os.Stdin)
	kvManifest := &KubeconformValidator{}

	cfg, out, err := config.FromFlags(os.Args[0], kvManifest.Spec.Args)
	if err != nil {
		log.Fatalf("failed parsing command line: %s\n", err.Error())
	}
	cfg.Stream.Input = strings.NewReader(krmInput.Items.String())
	cfg.Stream.Output = &output
	exitCode := validate.Run(cfg, out)

	log.Fatalf("%d - %s", exitCode, output.String())

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	fmt.Println(krmInput.Items.String())
}
