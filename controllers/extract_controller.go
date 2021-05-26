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
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=primer.gitops.io,resources=extracts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primer.gitops.io,resources=extracts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primer.gitops.io,resources=extracts/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=*,verbs=get;list

func (r *ExtractReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Namespace", req.Namespace, "Name", req.Name)
	log.Info("Reconciling Primer")
	instance := &primerv1alpha1.Extract{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Reconcile Job object
	result, err := r.jobToExtract(instance, log)
	if err != nil {
		return result, err
	}
	// Reconcile Service Account object
	result, err = r.saGenerate(instance, log)
	if err != nil {
		return result, err
	}
	// Reconcile Role object
	result, err = r.roleGenerate(instance, log)
	if err != nil {
		return result, err
	}
	// Reconcile Role Binding object
	result, err = r.roleBindingGenerate(instance, log)
	if err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *ExtractReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primerv1alpha1.Extract{}).
		Owns(&batchv1.Job{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}

func (r *ExtractReconciler) jobToExtract(cr *primerv1alpha1.Extract, log logr.Logger) (ctrl.Result, error) {
	// Define a new Job object
	job := newJobForCR(cr)

	if err := ctrl.SetControllerReference(cr, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if this Job already exists
	jobFound := &batchv1.Job{}
	err := r.Get(context.Background(), types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, jobFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Job", "Namespace", cr.Namespace, "Job Name", cr.Name)
		err = r.Create(context.Background(), job)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Requeue the object to update its status
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		// Job already exists
		log.Info("Job exists", "Namespace", jobFound.Namespace, "Job name", jobFound.Name)
	}

	return ctrl.Result{}, nil
}

func (r *ExtractReconciler) saGenerate(cr *primerv1alpha1.Extract, log logr.Logger) (ctrl.Result, error) {
	// Define a new Service object
	serviceAcct := newServiceAccountForCR(cr)

	if err := controllerutil.SetControllerReference(cr, serviceAcct, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if this Service Account already exists
	saFound := &corev1.ServiceAccount{}
	err := r.Get(context.Background(), types.NamespacedName{Name: serviceAcct.Name, Namespace: serviceAcct.Namespace}, saFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Service Account", "Namespace", serviceAcct.Namespace, "Service Account Name", serviceAcct.Name)
		err = r.Create(context.Background(), serviceAcct)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Service created successfully - don't requeue
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		// Service already exists
		log.Info("Service Account exists", "Namespace", saFound.Namespace, "Service Account Name", saFound.Name)
	}
	// Service reconcile finished
	return ctrl.Result{}, nil
}

func (r *ExtractReconciler) roleGenerate(cr *primerv1alpha1.Extract, log logr.Logger) (ctrl.Result, error) {
	// Define a new Service object
	accessRole := newRoleForCR(cr)

	if err := controllerutil.SetControllerReference(cr, accessRole, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if this Role already exists
	roleFound := &rbacv1.Role{}
	err := r.Get(context.Background(), types.NamespacedName{Name: accessRole.Name, Namespace: accessRole.Namespace}, roleFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Role", "Namespace", accessRole.Namespace, "Role", accessRole.Name)
		err = r.Create(context.Background(), accessRole)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Service created successfully - don't requeue
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		// Service already exists
		log.Info("Role exists", "Namespace", roleFound.Namespace, "Role", roleFound.Name)
	}
	// Service reconcile finished
	return ctrl.Result{}, nil
}

func (r *ExtractReconciler) roleBindingGenerate(cr *primerv1alpha1.Extract, log logr.Logger) (ctrl.Result, error) {
	// Define a new Service object
	accessRoleBinding := newRoleBindingForCR(cr)

	if err := controllerutil.SetControllerReference(cr, accessRoleBinding, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if this RoleBind already exists
	bindingFound := &rbacv1.RoleBinding{}
	err := r.Get(context.Background(), types.NamespacedName{Name: accessRoleBinding.Name, Namespace: accessRoleBinding.Namespace}, bindingFound)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Role Binding", "Namespace", accessRoleBinding.Namespace, "Role", accessRoleBinding.Name)
		err = r.Create(context.Background(), accessRoleBinding)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Service created successfully - don't requeue
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		// Service already exists
		log.Info("Role exists", "Namespace", bindingFound.Namespace, "Role", bindingFound.Name)
	}
	// Service reconcile finished
	return ctrl.Result{}, nil
}

func newJobForCR(cr *primerv1alpha1.Extract) *batchv1.Job {
	mode := int32(0600)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					ServiceAccountName: cr.Name,
					Containers: []corev1.Container{{
						Image:   "quay.io/octo-emerging/gitops-primer-extract:latest",
						Name:    "primer-extract",
						Command: []string{"/bin/sh", "-c", "/committer.sh"},
						Env: []corev1.EnvVar{
							{Name: "REPO", Value: cr.Spec.Repo},
							{Name: "BRANCH", Value: cr.Spec.Branch},
							{Name: "ACTION", Value: cr.Spec.Action},
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
								SecretName:  cr.Spec.Secret,
								DefaultMode: &mode,
							}},
						},
					},
				},
			},
		},
	}
}

// Returns a new Service account
func newServiceAccountForCR(cr *primerv1alpha1.Extract) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
	}
}

// Returns a new Service account
func newRoleForCR(cr *primerv1alpha1.Extract) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
}

func newRoleBindingForCR(cr *primerv1alpha1.Extract) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Name:     cr.Name,
			Kind:     "Role",
		},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: cr.Name},
		},
	}
}
