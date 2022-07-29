package main

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("removeVolumePlugin", "v1", nil, Run))
}

func Run(request transform.PluginRequest) (transform.PluginResponse, error) {
	// plugin writers need to write custom code here.
	u := &request.Unstructured
	var patch jsonpatch.Patch
	var err error
	switch u.GetKind() {
	case "PersistentVolumeClaim":
		patch, err = UpdatePVC(*u)
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

func UpdatePVC(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	patchJSON := fmt.Sprintf(`[
{ "op": "remove", "path": "/spec/volumeName"},
{ "op": "remove", "path": "/metadata/annotations/volume.kubernetes.io~1selected-node"},
{ "op": "remove", "path": "/metadata/annotations/pv.kubernetes.io~1bind-completed"},
{ "op": "remove", "path": "/metadata/annotations/pv.kubernetes.io~1bound-by-controller"}
]`)

	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}
