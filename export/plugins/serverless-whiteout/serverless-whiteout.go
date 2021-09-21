package main

import (
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/konveyor/crane-lib/transform"
	"github.com/konveyor/crane-lib/transform/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var knativeRoute = schema.GroupKind{
	Group: "serving.knative.dev",
	Kind:  "Route",
}

var knativeIngress = schema.GroupKind{
	Group: "networking.internal.knative.dev/v1alpha1",
	Kind:  "Ingress",
}

var knativeRevision = schema.GroupKind{
	Group: "serving.knative.dev/v1",
	Kind:  "Revision",
}

var knativeServerlessServing = schema.GroupKind{
	Group: "networking.internal.knative.dev/v1alpha1",
	Kind:  "ServerlessService",
}

func main() {
	cli.RunAndExit(cli.NewCustomPlugin("Serverlesswhiteout", "v1", nil, Run))
}

func Run(u *unstructured.Unstructured, extras map[string]string) (transform.PluginResponse, error) {
	var patch jsonpatch.Patch
	var err error
	var whiteout bool
	groupKind := u.GroupVersionKind().GroupKind()
	if groupKind == knativeRoute {
		whiteout = true
	}
	if groupKind == knativeIngress {
		whiteout = true
	}
	if groupKind == knativeRevision {
		whiteout = true
	}
	if groupKind == knativeServerlessServing {
		whiteout = true
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
