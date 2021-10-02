package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Trigger []struct {
	From      Inner  `json:"from"`
	FieldPath string `json:"fieldPath"`
	Pause     string `json:"pause"`
}

type Inner struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("NamespaceRemovalPlugin", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	// plugin writers need to write custome code here.
	var patch jsonpatch.Patch
	var err error
	switch u.GetKind() {
	case "Deployment":
		patch, err = RemoveFields(*u)
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

func RemoveFields(u unstructured.Unstructured) (jsonpatch.Patch, error) {
	val := u.GetAnnotations()["image.openshift.io/triggers"]
	var scrub Trigger
	json.Unmarshal([]byte(val), &scrub)
	d, err := json.Marshal(scrub)
	if err != nil {
		log.Fatal(err)
	}
	escaped := strconv.Quote(string(d))

	patchJSON := fmt.Sprintf(`[
		{"op": "replace", "path": "/metadata/annotations/image.openshift.io~1triggers", "value": %s}
		]`, escaped)
	patch, err := jsonpatch.DecodePatch([]byte(patchJSON))
	if err != nil {
		return nil, err
	}
	return patch, nil
}
