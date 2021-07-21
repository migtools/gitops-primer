package main

import (
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var extractName = []string{"primer-extract-"}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("WhiteoutExtractPlugin", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	var patch jsonpatch.Patch
	var err error
	var whiteout bool
	switch u.GetKind() {
	case "Extract":
		whiteout = true
	case "ServiceAccount":
		whiteout = ExtractServiceAccount(*u)
	case "RoleBinding":
		whiteout = ExtractRoleBinding(*u)
	case "Role":
		whiteout = ExtractRole(*u)
	case "Job":
		whiteout = ExtractJob(*u)
	case "Secret":
		whiteout = ExtractSecret(*u)
	case "PersistentVolumeClaim":
		whiteout = ExtractPVC(*u)
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

func ExtractServiceAccount(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExtractRoleBinding(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExtractRole(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExtractJob(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExtractSecret(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func ExtractPVC(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefault(check)
}

func isDefault(name string) bool {
	for _, d := range extractName {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}
