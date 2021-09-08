package main

import (
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var defaultSecretName = []string{"builder-dockercfg-", "builder-token", "default-dockercfg-", "default-token", "deployer-dockercfg-", "deployer-token"}
var defaultRoleBindingName = []string{"system:"}
var defaultConfigMapName = []string{"kube-root-ca.crt", "openshift-service-ca.crt", "config-trusted-cabundle"}
var defaultServiceAccountName = []string{"builder", "deployer", "default"}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("WhiteoutDefaultsPlugin", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	var patch jsonpatch.Patch
	var err error
	var whiteout bool
	switch u.GetKind() {
	case "Secret":
		whiteout = DefaultSecret(*u)
	case "RoleBinding":
		whiteout = DefaultRoleBinding(*u)
	case "ConfigMap":
		whiteout = DefaultConfigMap(*u)
	case "ServiceAccount":
		whiteout = DefaultServiceAccount(*u)
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

func DefaultSecret(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefaultSecret(check)
}

func isDefaultSecret(name string) bool {
	for _, d := range defaultSecretName {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
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

func DefaultServiceAccount(u unstructured.Unstructured) bool {
	check := u.GetName()
	return isDefaultServiceAccount(check)
}

func isDefaultServiceAccount(name string) bool {
	for _, d := range defaultServiceAccountName {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}
