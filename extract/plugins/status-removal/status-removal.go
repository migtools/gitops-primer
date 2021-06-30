package main

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	transformtypes "github.com/konveyor/crane-lib/transform/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func main() {
	u, err := cli.Unstructured(cli.ObjectReaderOrDie())
	if err != nil {
		cli.WriterErrorAndExit(fmt.Errorf("error getting unstructured object: %#v", err))
	}

	cli.RunAndExit(cli.NewCustomPlugin("RemoveStatus", Run), u)
}

func Run(u *unstructured.Unstructured) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	patch, err := RemoveStatus(*u)

	if err != nil {
		return transform.PluginResponse{}, err
	}
	return transform.PluginResponse{
		Version:    "v1",
		IsWhiteOut: false,
		Patches:    patch,
	}, nil
}

func RemoveStatus(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	jsonPatch := jsonpatch.Patch{}
	hasStatus, _ := transformtypes.HasStatusObject(u)
	if hasStatus {
		patchJSON := fmt.Sprintf(`[
				{ "op": "remove", "path": "/status"}
				]`)
		patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
		if err != nil {
			return nil, err
		}
		jsonPatch = append(jsonPatch, patch...)
	}
	return jsonPatch, nil
}
