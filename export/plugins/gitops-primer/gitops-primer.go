package main

import (
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var exportName = []string{"primer-export-"}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("WhiteoutExportPlugin", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	var patch jsonpatch.Patch
	var err error
	var whiteout bool
	switch u.GetKind() {
	case "Export":
		whiteout = true
	case "ServiceAccount":
		whiteout = ExportObjects(*u)
	case "RoleBinding":
		whiteout = ExportObjects(*u)
	case "Role":
		whiteout = ExportObjects(*u)
	case "Job":
		whiteout = ExportObjects(*u)
	case "Secret":
		whiteout = ExportObjects(*u)
	case "PersistentVolumeClaim":
		whiteout = ExportObjects(*u)
	case "NetworkPolicy":
		whiteout = ExportObjects(*u)
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

func ExportObjects(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func isDefault(name string) bool {
	for _, d := range exportName {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}
