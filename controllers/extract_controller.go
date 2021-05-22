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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	primerv1alpha1 "github.com/cooktheryan/gitops-primer/api/v1alpha1"
)

type ExtractReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	role        *rbacv1.Role
	roleBinding *rbacv1.RoleBinding
}

//+kubebuilder:rbac:groups=primer.gitops.io,resources=extracts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primer.gitops.io,resources=extracts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primer.gitops.io,resources=extracts/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=*,verbs=get;list

func (r *ExtractReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Req.Namespace", req.Namespace, "Req.Name", req.Name)
	logger.Info("Reconciling Primer")
	extract := &primerv1alpha1.Extract{}
	err := r.Get(ctx, req.NamespacedName, extract)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if the job already exists, if not create a new job.
	found := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: extract.Name, Namespace: extract.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			// Define and create a new job.
			job := r.jobToExtract(extract)
			if err = r.Create(ctx, job); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	// Check if the SA already exists, if not create it
	access := &corev1.ServiceAccount{}
	err = r.Get(ctx, types.NamespacedName{Name: extract.Name, Namespace: extract.Namespace}, access)
	if err != nil {
		if errors.IsNotFound(err) {
			// Define and create a new role.
			sacct := r.saGenerate(extract)
			if err = r.Create(ctx, sacct); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	// Check if the role already exists, if not create it
	r.role = &rbacv1.Role{}
	err = r.Get(ctx, types.NamespacedName{Name: extract.Name, Namespace: extract.Namespace}, r.role)
	if err != nil {
		if errors.IsNotFound(err) {
			// Define and create a new role.
			rolecreate := r.roleGenerate(extract)
			if err = r.Create(ctx, rolecreate); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}
	// Check if the binding already exists, if not create it
	r.roleBinding = &rbacv1.RoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Name: extract.Name, Namespace: extract.Namespace}, r.roleBinding)
	if err != nil {
		if errors.IsNotFound(err) {
			// Define and create a new role.
			bindingcreate := r.bindingGenerate(extract)
			if err = r.Create(ctx, bindingcreate); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, err
}

func (r *ExtractReconciler) roleGenerate(m *primerv1alpha1.Extract) *rbacv1.Role {
	accessrole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
	}
	accessrole.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"get", "list"},
		},
	}

	controllerutil.SetControllerReference(m, accessrole, r.Scheme)
	return accessrole
}

func (r *ExtractReconciler) saGenerate(m *primerv1alpha1.Extract) *corev1.ServiceAccount {
	sacct := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
	}
	controllerutil.SetControllerReference(m, sacct, r.Scheme)
	return sacct
}

func (r *ExtractReconciler) bindingGenerate(m *primerv1alpha1.Extract) *rbacv1.RoleBinding {
	accessbinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
	}
	accessbinding.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Name:     "bogo",
		Kind:     "Role",
	}
	accessbinding.Subjects = []rbacv1.Subject{
		{Kind: "ServiceAccount", Name: m.Name},
	}

	controllerutil.SetControllerReference(m, accessbinding, r.Scheme)
	return accessbinding
}

func (r *ExtractReconciler) jobToExtract(m *primerv1alpha1.Extract) *batchv1.Job {
	mode := int32(0600)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					ServiceAccountName: m.Name,
					Containers: []corev1.Container{{
						Image:   "quay.io/octo-emerging/gitops-primer-extract:latest",
						Name:    "primer-extract",
						Command: []string{"/bin/sh", "-c", "/committer.sh"},
						Env: []corev1.EnvVar{
							{Name: "REPO", Value: m.Spec.Repo},
							{Name: "BRANCH", Value: m.Spec.Branch},
							{Name: "ACTION", Value: m.Spec.Action},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "sshkeys", MountPath: "/keys"},
							{Name: "repo", MountPath: "/repo"},
						},
					}},
					Volumes: []corev1.Volume{
						{Name: "repo", VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
						},
						{Name: "sshkeys", VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  m.Spec.Secret,
								DefaultMode: &mode,
							}},
						},
					},
				},
			},
		},
	}
	controllerutil.SetControllerReference(m, job, r.Scheme)
	return job
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExtractReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primerv1alpha1.Extract{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
