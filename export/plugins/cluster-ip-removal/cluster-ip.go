package main

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var defaultPullSecrets = []string{"builder-dockercfg-", "default-dockercfg-", "deployer-dockercfg-"}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("OpenShiftPlugin", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	var patch jsonpatch.Patch
	var err error
	switch u.GetKind() {
	case "Service":
		patch, err = UpdateService(*u)
	}
	if err != nil {
		return transform.PluginResponse{}, err
	}
	return transform.PluginResponse{
		Version:    "v1",
		IsWhiteOut: false,
		Patches:    patch,
	}, nil
}

func UpdateService(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	patchJSON := fmt.Sprintf(`[
{ "op": "remove", "path": "/spec/clusterIPs"}, 
{ "op": "remove", "path": "/spec/clusterIP"}
]`)

	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}
