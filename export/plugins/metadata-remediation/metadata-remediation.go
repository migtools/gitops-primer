package main

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("MetadataRemovalPlugin", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	patch, err := RemoveFields(*u)

	if err != nil {
		return transform.PluginResponse{}, err
	}
	return transform.PluginResponse{
		Version:    "v1",
		IsWhiteOut: false,
		Patches:    patch,
	}, nil
}

func RemoveFields(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	patchJSON := fmt.Sprintf(`[
{ "op": "remove", "path": "/metadata/managedFields"}, 
{ "op": "remove", "path": "/metadata/uid"}, 
{ "op": "remove", "path": "/metadata/creationTimestamp"}, 
{ "op": "remove", "path": "/metadata/resourceVersion"},
{ "op": "remove", "path": "/metadata/selfLink"},
{ "op": "remove", "path": "/metadata/generation"}
]`)

	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}
