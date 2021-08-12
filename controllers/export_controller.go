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
	"log"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/status"
	password "github.com/sethvargo/go-password/password"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	primerv1alpha1 "github.com/cooktheryan/gitops-primer/api/v1alpha1"
)

// ExportReconciler reconciles a Export object
type ExportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=primer.gitops.io,resources=exports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=primer.gitops.io,resources=exports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=primer.gitops.io,resources=exports/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,resources=route,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=*,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Export object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ExportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// Fetch the Export instance
	instance := &primerv1alpha1.Export{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Export resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Export")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Check if the Job already exists, if not create a new one
	found := &batchv1.Job{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, found); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			if instance.Spec.Method == "git" {
				// Define a new job
				job := r.jobGitForExport(instance)
				log.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
				if err = r.Create(ctx, job); err != nil {
					log.Error(err, "Failed to create new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
					updateErrCondition(instance, err)
					return ctrl.Result{}, err
				}
				// Job created successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			} else if instance.Spec.Method == "download" {
				// Define a new job
				job := r.jobDownloadForExport(instance)
				log.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
				if err = r.Create(ctx, job); err != nil {
					log.Error(err, "Failed to create new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
					updateErrCondition(instance, err)
					return ctrl.Result{}, err
				}
				// Job created successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			}
		}
		log.Error(err, "Failed to get Job")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Check if the Service Account already exists, if not create a new one
	foundSA := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundSA); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new Service Account
			serviceAcct := r.saGenerate(instance)
			log.Info("Creating a new Service Account", "serviceAcct.Namespace", serviceAcct.Namespace, "serviceAcct.Name", serviceAcct.Name)
			if err := r.Create(ctx, serviceAcct); err != nil {
				log.Error(err, "Failed to create new Service Account", "serviceAcct.Namespace", serviceAcct.Namespace, "serviceAcct.Name", serviceAcct.Name)

				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			// Service Account created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Service Account")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Check if the Secret already exists, if not create a new one
	foundSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundSecret); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new Secret
			proxySecret := r.secretGenerate(instance)
			log.Info("Creating a new Secret", "proxySecret.Namespace", proxySecret.Namespace, "proxySecret.Name", proxySecret.Name)
			if err := r.Create(ctx, proxySecret); err != nil {
				log.Error(err, "Failed to create new Secret", "proxySecret.Namespace", proxySecret.Namespace, "proxySecret.Name", proxySecret.Name)

				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			// Secret created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Secret")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Check if the Route already exists, if not create a new one
	foundRoute := &routev1.Route{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundRoute); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new Secret
			appRoute := r.routeGenerate(instance)
			log.Info("Creating a new Route", "appRoute.Namespace", appRoute.Namespace, "appRoute.Name", appRoute.Name)
			if err := r.Create(ctx, appRoute); err != nil {
				log.Error(err, "Failed to create new Route", "appRoute.Namespace", appRoute.Namespace, "appRoute.Name", appRoute.Name)

				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			// Route created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Route")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Check if the Role already exists, if not create a new one
	foundRole := &rbacv1.Role{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundRole); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new Role
			role := r.roleGenerate(instance)
			log.Info("Creating a new Role", "role.Namespace", role.Namespace, "role.Name", role.Name)
			if err := r.Create(ctx, role); err != nil {
				log.Error(err, "Failed to create new Role", "role.Namespace", role.Namespace, "role.Name", role.Name)
				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			// Role created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Role")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Check if the RoleBinding already exists, if not create a new one
	foundRoleBinding := &rbacv1.RoleBinding{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundRoleBinding); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new Role Binding
			roleBinding := r.roleBindingGenerate(instance)
			log.Info("Creating a new Role Binding", "roleBinding.Namespace", roleBinding.Namespace, "roleBinding.Name", roleBinding.Name)
			if err := r.Create(ctx, roleBinding); err != nil {
				log.Error(err, "Failed to create new Role Binding", "roleBinding.Namespace", roleBinding.Namespace, "roleBinding.Name", roleBinding.Name)
				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			// Role Binding created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Role Binding")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Check if the PVC already exists, if not create a new one
	foundVolume := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundVolume); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new PVC
			persistentVC := r.pvcGenerate(instance)
			log.Info("Creating a new PVC", "persistentVC.Namespace", persistentVC.Namespace, "persistentVC.Name", persistentVC.Name)
			if err := r.Create(ctx, persistentVC); err != nil {
				log.Error(err, "Failed to create a PVC", "persistentVC.Namespace", persistentVC.Namespace, "persistentVC.Name", persistentVC.Name)

				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			// Persistent Volume created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get PVC")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = status.Conditions{}
	}

	// Check if the service already exists, if not create a new one
	foundService := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundService); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new service
			service := r.svcGenerate(instance)
			log.Info("Creating a new Service", "service.Namespace", service.Namespace, "service.Name", service.Name)
			if err := r.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create a Service", "service.Namespace", service.Namespace, "service.Name", service.Name)

				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			// Service created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Service")
		updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = status.Conditions{}
	}

	// Update status.Nodes if needed
	instance.Status.Completed = isJobComplete(found)
	instance.Status.Service = "primer-export-" + instance.Name + ":8080"
	if instance.Status.Completed {
		log.Info("Job completed")
		log.Info("Cleaning up Primer Resources")
		if err := r.Status().Update(ctx, instance); err != nil {
			log.Error(err, "Failed to update Export status")
			updateErrCondition(instance, err)
			return ctrl.Result{}, err
		}
		r.Delete(ctx, found, client.PropagationPolicy(metav1.DeletePropagationBackground))
		r.Delete(ctx, foundRole)
		r.Delete(ctx, foundRoleBinding)

		if instance.Spec.Method == "download" {
			log.Info("Serving up Export Download")
			foundDeployment := &appsv1.Deployment{}
			if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundDeployment); err != nil {
				if errors.IsNotFound(err) {
					// Define a new Deployment
					deployment := r.deploymentGenerate(instance)
					log.Info("Creating a new Deployment", "deployment.Namespace", deployment.Namespace, "deployment.Name", deployment.Name)
					if err := r.Create(ctx, deployment); err != nil {
						log.Error(err, "Failed to create a Deployment", "deployment.Namespace", deployment.Namespace, "deployment.Name", deployment.Name)

						updateErrCondition(instance, err)
						return ctrl.Result{}, err
					}
					// Service created successfully - return and requeue
					return ctrl.Result{Requeue: true}, nil
				}
				log.Error(err, "Failed to get Deployment")
				updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
		}

		// Set reconcile status condition complete
		instance.Status.Conditions.SetCondition(
			status.Condition{
				Type:    primerv1alpha1.ConditionReconciled,
				Status:  corev1.ConditionTrue,
				Reason:  primerv1alpha1.ReconciledReasonComplete,
				Message: "Reconcile complete",
			})
	}
	return ctrl.Result{}, nil
}

func updateErrCondition(instance *primerv1alpha1.Export, err error) {
	instance.Status.Conditions.SetCondition(
		status.Condition{
			Type:    primerv1alpha1.ConditionReconciled,
			Status:  corev1.ConditionFalse,
			Reason:  primerv1alpha1.ReconciledReasonError,
			Message: err.Error(),
		})
}

// jobGitForExport returns a instance Job object
func (r *ExportReconciler) jobGitForExport(m *primerv1alpha1.Export) *batchv1.Job {
	mode := int32(0644)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					ServiceAccountName: "primer-export-" + m.Name,
					Containers: []corev1.Container{{
						Name:            m.Name,
						ImagePullPolicy: "IfNotPresent",
						Image:           "quay.io/octo-emerging/gitops-primer-export:latest",
						Command:         []string{"/bin/sh", "-c", "/committer.sh"},
						Env: []corev1.EnvVar{
							{Name: "REPO", Value: m.Spec.Repo},
							{Name: "BRANCH", Value: m.Spec.Branch},
							{Name: "EMAIL", Value: m.Spec.Email},
							{Name: "NAMESPACE", Value: m.Namespace},
							{Name: "METHOD", Value: m.Spec.Method},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "sshkeys", MountPath: "/keys"},
							{Name: "output", MountPath: "/output"},
						},
					}},
					Volumes: []corev1.Volume{
						{Name: "output", VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "primer-export-" + m.Name,
							},
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
	ctrl.SetControllerReference(m, job, r.Scheme)
	return job
}

// jobGitForExport returns a instance Job object
func (r *ExportReconciler) jobDownloadForExport(m *primerv1alpha1.Export) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					ServiceAccountName: "primer-export-" + m.Name,
					Containers: []corev1.Container{{
						Name:            m.Name,
						ImagePullPolicy: "IfNotPresent",
						Image:           "quay.io/octo-emerging/gitops-primer-export:latest",
						Command:         []string{"/bin/sh", "-c", "/committer.sh"},
						Env: []corev1.EnvVar{
							{Name: "METHOD", Value: m.Spec.Method},
							{Name: "NAMESPACE", Value: m.Namespace},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "output", MountPath: "/output"},
						},
					}},
					Volumes: []corev1.Volume{
						{Name: "output", VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "primer-export-" + m.Name,
							},
						},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(m, job, r.Scheme)
	return job
}

func (r *ExportReconciler) saGenerate(m *primerv1alpha1.Export) *corev1.ServiceAccount {
	// Define a new Service Account object
	serviceAcct := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primer-export-" + m.Name: "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"primer-export-primer\"}}",
			},
		},
	}
	// Service reconcile finished
	ctrl.SetControllerReference(m, serviceAcct, r.Scheme)
	return serviceAcct
}

func (r *ExportReconciler) routeGenerate(m *primerv1alpha1.Export) *routev1.Route {
	// Define a new Route object
	appRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "primer-export-primer",
			},
			AlternateBackends: []routev1.RouteTargetReference{},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("oauth-proxy"),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   "reencrypt",
				InsecureEdgeTerminationPolicy: "Redirect",
			},
			WildcardPolicy: "",
		},
	}

	// Service reconcile finished
	ctrl.SetControllerReference(m, appRoute, r.Scheme)
	return appRoute
}

func (r *ExportReconciler) secretGenerate(m *primerv1alpha1.Export) *corev1.Secret {
	// Define a new Secret object
	random, err := password.Generate(43, 10, 0, false, false)
	if err != nil {
		log.Fatal(err)
	}
	proxySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"session_secret": []byte(random),
		},
	}
	// Service reconcile finished
	ctrl.SetControllerReference(m, proxySecret, r.Scheme)
	return proxySecret
}

func (r *ExportReconciler) pvcGenerate(m *primerv1alpha1.Export) *corev1.PersistentVolumeClaim {
	// Define a new PVC object
	persistentVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
	// Service reconcile finished
	ctrl.SetControllerReference(m, persistentVC, r.Scheme)
	return persistentVC
}

func (r *ExportReconciler) roleGenerate(m *primerv1alpha1.Export) *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	// Service reconcile finished
	ctrl.SetControllerReference(m, role, r.Scheme)
	return role
}

func (r *ExportReconciler) roleBindingGenerate(m *primerv1alpha1.Export) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "primer-export-" + m.Name,
			Kind:     "Role",
		},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: "primer-export-" + m.Name},
		},
	}
	// Service reconcile finished
	ctrl.SetControllerReference(m, roleBinding, r.Scheme)
	return roleBinding
}

func (r *ExportReconciler) svcGenerate(m *primerv1alpha1.Export) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Annotations: map[string]string{
				"service.alpha.openshift.io/serving-cert-secret-name": "primer-export-" + m.Name + "-tls",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
					Name: "primer",
				},
				{
					Port: 8888,
					Name: "oauth-proxy",
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name":      "primer-export-" + m.Name,
				"app.kubernetes.io/component": "primer-export-" + m.Name,
				"app.kubernetes.io/part-of":   "primer-export",
			},
		},
	}
	// Service reconcile finished
	ctrl.SetControllerReference(m, service, r.Scheme)
	return service
}

func (r *ExportReconciler) deploymentGenerate(m *primerv1alpha1.Export) *appsv1.Deployment {
	replicas := int32(1)
	secretMode := int32(420)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":      "primer-export-" + m.Name,
					"app.kubernetes.io/component": "primer-export-" + m.Name,
					"app.kubernetes.io/part-of":   "primer-export",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":      "primer-export-" + m.Name,
						"app.kubernetes.io/component": "primer-export-" + m.Name,
						"app.kubernetes.io/part-of":   "primer-export",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "quay.io/octo-emerging/gitops-primer-downloader:latest",
							Name:  "primer-export-" + m.Name,
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8080,
								Name:          "downloader",
							}},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "output", MountPath: "/var/www/html"},
							},
						},
						{
							Image: "quay.io/openshift/origin-oauth-proxy:4.7",
							Name:  "oauth-proxy",
							Args: []string{
								"-provider=openshift",
								"-https-address=:8888",
								"-http-address=",
								"-email-domain=*",
								"-upstream=http://localhost:8080",
								"-tls-cert=/etc/tls/private/tls.crt",
								"-tls-key=/etc/tls/private/tls.key",
								"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
								"-cookie-secret-file=/etc/proxy/secrets/session_secret",
								"-openshift-service-account=primer-export-" + m.Name,
								"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
								"-skip-auth-regex=^/metrics",
								"-openshift-sar={\"resource\":\"namespaces\",\"resourceName\":\"primer-export-primer\",\"namespace\": \"test\",\"verb\":\"get\"}",
							},
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8888,
								Name:          "oath-proxy",
							}},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "primer-oauth-tls", MountPath: "/etc/tls/private"},
								{Name: "secret-primer-proxy", MountPath: "/etc/proxy/secrets"},
							},
						},
					},
					ServiceAccountName: "primer-export-" + m.Name,
					Volumes: []corev1.Volume{
						{Name: "output", VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "primer-export-" + m.Name,
							},
						},
						},
						{Name: "primer-oauth-tls", VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "primer-export-" + m.Name + "-tls",
								DefaultMode: &secretMode,
							},
						},
						},
						{Name: "secret-primer-proxy", VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "primer-export-" + m.Name,
								DefaultMode: &secretMode,
							},
						},
						},
					},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func isJobComplete(job *batchv1.Job) bool {
	return job.Status.Succeeded == 1
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&primerv1alpha1.Export{}).
		Owns(&batchv1.Job{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Secret{}).
		Owns(&routev1.Route{}).
		Complete(r)
}
