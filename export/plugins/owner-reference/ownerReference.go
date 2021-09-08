package main

import (
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("OwnerReferenceScrub", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	var patch jsonpatch.Patch
	var err error
	switch u.GetKind() {
	case "ConfigMap":
		patch, err = RemoveOwner(*u)
	case "ReplicaSet":
		patch, err = RemoveOwner(*u)
	case "Revision":
		patch, err = RemoveOwner(*u)
	case "Metric":
		patch, err = RemoveOwner(*u)
	case "PodAutoscaler":
		patch, err = RemoveOwner(*u)
	case "ServerlessService":
		patch, err = RemoveOwner(*u)
	case "Service":
		patch, err = RemoveOwner(*u)
	case "Ingress":
		patch, err = RemoveOwner(*u)
	case "Configuration":
		patch, err = RemoveOwner(*u)
	case "InMemoryChannel":
		patch, err = RemoveOwner(*u)
	case "Route":
		patch, err = RemoveOwner(*u)
	case "Subscription":
		patch, err = RemoveOwner(*u)
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

func RemoveOwner(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	patchJSON := fmt.Sprintf(`[
{ "op": "remove", "path": "/metadata/ownerReferences"}
]`)

	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}
