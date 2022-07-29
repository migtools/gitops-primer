package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	primerv1alpha1 "github.com/cooktheryan/gitops-primer/api/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type App struct {
}

func (app *App) HandleMutate(w http.ResponseWriter, r *http.Request) {
	admissionReview := &admissionv1.AdmissionReview{}

	// read the AdmissionReview from the request json body
	err := readJSON(r, admissionReview)
	if err != nil {
		app.HandleError(w, r, err)
		return
	}

	// unmarshal the export from the AdmissionRequest
	ar := admissionReview.Request.UserInfo
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, &ar); err != nil {
		app.HandleError(w, r, fmt.Errorf("unmarshal to user: %v", err))
		return
	}

	// unmarshal the export from the AdmissionRequest
	export := &primerv1alpha1.Export{}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, export); err != nil {
		app.HandleError(w, r, fmt.Errorf("unmarshal to export: %v", err))
		return
	}

	if export.Spec.User != "" {
		response := &admissionv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AdmissionReview",
				APIVersion: "admission.k8s.io/v1",
			},
			Response: &admissionv1.AdmissionResponse{
				UID:     admissionReview.Request.UID,
				Allowed: true,
			},
		}
		jsonOk(w, &response)
		return

	}

	userName, err := json.Marshal(&ar.Username)
	group, err := json.Marshal(&ar.Groups)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall user: %v", err))
	}

	// build json patch
	patch := []JSONPatchEntry{
		JSONPatchEntry{
			OP:    "add",
			Path:  "/spec/user",
			Value: userName,
		},
		JSONPatchEntry{
			OP:    "add",
			Path:  "/spec/group",
			Value: group,
		},
	}

	patchBytes, err := json.Marshal(&patch)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall jsonpatch: %v", err))
		return
	}

	patchType := admissionv1.PatchTypeJSONPatch

	// build admission response
	admissionResponse := &admissionv1.AdmissionResponse{
		UID:       admissionReview.Request.UID,
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}

	respAdmissionReview := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: admissionResponse,
	}

	jsonOk(w, &respAdmissionReview)
}

type JSONPatchEntry struct {
	OP    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}
