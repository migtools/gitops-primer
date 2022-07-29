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

func Run(request transform.PluginRequest) (transform.PluginResponse, error) {
	// plugin writers need to write custom code here.
	u := &request.Unstructured
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
{ "op": "remove", "path": "/metadata/generation"},
{ "op": "remove", "path": "/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration"},
{ "op": "remove", "path": "/metadata/annotations/deployment.kubernetes.io~1revision"}
]`)

	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}
