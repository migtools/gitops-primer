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
		whiteout = ExportServiceAccount(*u)
	case "RoleBinding":
		whiteout = ExportRoleBinding(*u)
	case "Role":
		whiteout = ExportRole(*u)
	case "Job":
		whiteout = ExportJob(*u)
	case "Secret":
		whiteout = ExportSecret(*u)
	case "PersistentVolumeClaim":
		whiteout = ExportPVC(*u)
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

func ExportServiceAccount(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExportRoleBinding(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExportRole(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExportJob(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExportSecret(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExportPVC(u unstructured.Unstructured) bool {
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
