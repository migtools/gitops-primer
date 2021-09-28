package main

import (
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var defaultRoleBindingName = []string{"pipelines-scc-rolebinding", "edit", "admin"}
var defaultServiceAccountName = []string{"pipeline-", "pipeline"}
var defaultPipelineCM = []string{"config-service-cabundle", "config-trusted-cabundle"}
var defaultPipelineSecret = []string{"pipeline-token-", "pipeline-dockercfg-"}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("WhiteoutTekton", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	var patch jsonpatch.Patch
	var err error
	var whiteout bool
	switch u.GetKind() {
	case "PipelineRun":
		whiteout = true
	case "TaskRun":
		whiteout = true
	case "ServiceAccount":
		whiteout = DefaultSA(*u)
	case "RoleBinding":
		whiteout = DefaultRoleBinding(*u)
	case "Secret":
		whiteout = DefaultPipelineSecrets(*u)
	case "ConfigMap":
		whiteout = PipelineCM(*u)
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

func DefaultRoleBinding(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefaultBinding(check)
}

func isDefaultBinding(name string) bool {
	for _, d := range defaultRoleBindingName {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}

func DefaultSA(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefaultSA(check)
}

func isDefaultSA(name string) bool {
	for _, d := range defaultServiceAccountName {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}
func PipelineCM(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefaultCM(check)
}

func isDefaultCM(name string) bool {
	for _, d := range defaultPipelineCM {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}

func DefaultPipelineSecrets(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefaultSecret(check)
}

func isDefaultSecret(name string) bool {
	for _, d := range defaultPipelineSecret {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}
