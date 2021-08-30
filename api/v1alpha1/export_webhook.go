/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"net/http"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	exportlog = logf.Log.WithName("export-resource")
)

func (r *Export) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-primer-gitops-io-v1alpha1-export,mutating=true,failurePolicy=fail,sideEffects=None,groups=primer.gitops.io,resources=exports,verbs=create;update,versions=v1alpha1,name=mexport.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Export{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Export) Default() {
	exportlog.Info("default", "name", r.Name)
	if r.Spec.User == "" {
		r.Spec.User = "bob"
	}
}

//+kubebuilder:webhook:path=/validate-primer-gitops-io-v1alpha1-export,mutating=false,failurePolicy=fail,sideEffects=None,groups=primer.gitops.io,resources=exports,verbs=create;update,versions=v1alpha1,name=vexport.kb.io,admissionReviewVersions={v1,v1beta1}
// exportValidator validates Exports
type exportValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func NewExportValidator(c client.Client) admission.Handler {
	return &exportValidator{Client: c}
}

// exportValidator admits a export if a specific annotation exists.
func (v *exportValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	export := &Export{}

	err := v.decoder.Decode(req, export)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	key := "example-mutating-admission-webhook"
	anno, found := export.Annotations[key]
	if !found {
		return admission.Denied(fmt.Sprintf("missing annotation %s", key))
	}
	if anno != "foo" {
		return admission.Denied(fmt.Sprintf("annotation %s did not have value %q", key, "foo"))
	}

	return admission.Allowed("")
}

// exportValidator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (v *exportValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
