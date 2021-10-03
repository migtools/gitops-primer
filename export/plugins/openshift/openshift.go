package main

import (
	"encoding/json"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	v1 "k8s.io/api/core/v1"
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
	case "Pod":
		patch, err = UpdateDefaultPullSecrets(*u)
	case "Route":
		patch, err = UpdateRoute(*u)
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

func UpdateDefaultPullSecrets(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	pullSecrets := getPullSecrets(u)

	jsonPatch := jsonpatch.Patch{}

	for n, secret := range pullSecrets {
		if isDefault(secret.Name) {

			patchJSON := fmt.Sprintf(`[
{ "op": "remove", "path": "/spec/imagePullSecrets/%v"}
]`, n)

			patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
			if err != nil {
				return nil, err
			}
			jsonPatch = append(jsonPatch, patch...)
		}
	}

	return jsonPatch, nil
}

func UpdateRoute(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	patchJSON := fmt.Sprintf(`[
{ "op": "remove", "path": "/spec/host"}
]`)

	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}

func isDefault(name string) bool {
	for _, d := range defaultPullSecrets {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}

func getPullSecrets(u unstructured.Unstructured) []v1.LocalObjectReference {
	js, err := u.MarshalJSON()
	if err != nil {
		return nil
	}

	pod := &v1.Pod{}

	err = json.Unmarshal(js, pod)
	if err != nil {
		return nil
	}

	return pod.Spec.ImagePullSecrets
}
