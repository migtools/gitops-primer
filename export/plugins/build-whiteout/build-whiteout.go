package main

import (
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var defaultConfigMapName = []string{"-global-ca", "-ca", "-sys-config"}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("WhiteoutBuildsPlugin", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	var patch jsonpatch.Patch
	var err error
	var whiteout bool
	switch u.GetKind() {
	case "Build":
		whiteout = true
	case "ConfigMap":
		whiteout = DefaultConfigMap(*u)
	}
	if err != nil {
		return transform.PluginResponse{}, err
	}
	return transform.PluginResponse{
		Version:    "v1",
		IsWhiteOut: whiteout,
		Patches:    patch,
	}, nil
}

func DefaultConfigMap(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefaultConfigmap(check)
}

func isDefaultConfigmap(name string) bool {
	for _, d := range defaultConfigMapName {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}
