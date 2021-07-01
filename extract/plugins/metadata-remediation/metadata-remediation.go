package main

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func main() {
	u, err := cli.Unstructured(cli.ObjectReaderOrDie())
	if err != nil {
		cli.WriterErrorAndExit(fmt.Errorf("error getting unstructured object: %#v", err))
	}

	cli.RunAndExit(cli.NewCustomPlugin("MetadataRemediation", Run), u)
}

func Run(u *unstructured.Unstructured) (transform.PluginResponse, error) {
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
{ "op": "remove", "path": "/metadata/managedFields},
{ "op": "remove", "path": "/metadata/creationTimestamp}
]`)

	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}
