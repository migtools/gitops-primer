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
	"os"
	"strconv"
	"strings"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/status"
	password "github.com/sethvargo/go-password/password"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	primerv1alpha1 "github.com/cooktheryan/gitops-primer/api/v1alpha1"
)

const (
	gitopsPrimerLabel = "openshift.gitops.primer"
	finalizer         = "openshift.gitops.primer"
)

// ExportReconciler reconciles a Export object
type ExportReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	DownloaderImage string
	ExportImage     string
	OauthImage      string
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
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=users;groups,verbs=impersonate
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=*,verbs=get;list
//+kubebuilder:rbac:groups="",resourceNames=gitops-primer-system,resources=namespaces,verbs=get;update

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
		r.updateErrCondition(instance, err)
		return ctrl.Result{}, err
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(instance, finalizer) {
		controllerutil.AddFinalizer(instance, finalizer)
		err := r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if instance.DeletionTimestamp != nil {
		// clean up
		err := r.CleanUpOpjects(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(instance, finalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// get securityContext appropriate to cluster version
	securityContext, err := getSecurityContext()
	if err != nil {
		return ctrl.Result{}, err
	}

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

				r.updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
				if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundVolume); err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					} else {
						return false, err
					}
				}
				return true, nil
			}); err != nil {
				return ctrl.Result{}, err
			}
			// Persistent Volume created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get PVC")
		r.updateErrCondition(instance, err)
	}

	// Check if the export job already exists, if not create a new one
	// based on if its git or download the appropriate func will be called
	found := &batchv1.Job{}
	if !instance.Status.Extracted {
		if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, found); err != nil {
			if errors.IsNotFound(err) {
				if instance.Spec.Method == "git" {
					// Define a new job
					job := r.jobGitForExport(instance, securityContext)
					log.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
					if err = r.Create(ctx, job); err != nil {
						log.Error(err, "Failed to create new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
						r.updateErrCondition(instance, err)
						return ctrl.Result{}, err
					}
					if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
						if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, found); err != nil {
							if errors.IsNotFound(err) {
								return false, nil
							} else {
								return false, err
							}
						}
						return true, nil
					}); err != nil {
						return ctrl.Result{}, err
					}
					// Job created successfully - return and requeue
					return ctrl.Result{Requeue: true}, nil
				} else if instance.Spec.Method == "download" {
					// Define a new job
					job := r.jobDownloadForExport(instance, securityContext)
					log.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
					if err = r.Create(ctx, job); err != nil {
						log.Error(err, "Failed to create new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
						r.updateErrCondition(instance, err)
						return ctrl.Result{}, err
					}
					if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
						if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, found); err != nil {
							if errors.IsNotFound(err) {
								return false, nil
							} else {
								return false, err
							}
						}
						return true, nil
					}); err != nil {
						return ctrl.Result{}, err
					}
					// Job created successfully - return and requeue
					return ctrl.Result{Requeue: true}, nil
				}
			}
			log.Error(err, "Failed to get Job")
			r.updateErrCondition(instance, err)
		}
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

				r.updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
				if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundSA); err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					} else {
						return false, err
					}
				}
				return true, nil
			}); err != nil {
				return ctrl.Result{}, err
			}
			// Service Account created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Service Account")
		r.updateErrCondition(instance, err)
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
			log.Info("Creating a new oauth Secret", "proxySecret.Namespace", proxySecret.Namespace, "proxySecret.Name", proxySecret.Name)
			if err := r.Create(ctx, proxySecret); err != nil {
				log.Error(err, "Failed to create new oauth Secret", "proxySecret.Namespace", proxySecret.Namespace, "proxySecret.Name", proxySecret.Name)

				r.updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
				if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundSecret); err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					} else {
						return false, err
					}
				}
				return true, nil
			}); err != nil {
				return ctrl.Result{}, err
			}
			// Secret created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get oauth Secret")
		r.updateErrCondition(instance, err)
	}

	// Check if the Route already exists, if not create a new one
	foundRoute := &routev1.Route{}
	if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundRoute); err != nil {
		if instance.Status.Completed {
			return ctrl.Result{}, nil
		}
		if errors.IsNotFound(err) {
			// Define a new Route
			appRoute := r.routeGenerate(instance)
			log.Info("Creating a new Route", "appRoute.Namespace", appRoute.Namespace, "appRoute.Name", appRoute.Name)
			if err := r.Create(ctx, appRoute); err != nil {
				log.Error(err, "Failed to create new Route", "appRoute.Namespace", appRoute.Namespace, "appRoute.Name", appRoute.Name)

				r.updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
				if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundRoute); err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					} else {
						return false, err
					}
				}
				return true, nil
			}); err != nil {
				return ctrl.Result{}, err
			}
			// Route created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Route")
		r.updateErrCondition(instance, err)
	}

	// Check if the Cluster Role already exists, if not create a new one
	foundClusterRole := &rbacv1.ClusterRole{}
	if !instance.Status.Extracted {
		if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Namespace + "-" + instance.Name, Namespace: instance.Namespace}, foundClusterRole); err != nil {
			if errors.IsNotFound(err) {
				// Define a new Role
				clusterRole := r.clusterRoleGenerate(instance)
				log.Info("Creating a new Cluster Role", "clusterRole.Namespace", clusterRole.Namespace, "clusterRole.Name", clusterRole.Name)
				if err := r.Create(ctx, clusterRole); err != nil {
					log.Error(err, "Failed to create new Cluster Role", "clusterRole.Namespace", clusterRole.Namespace, "clusterRole.Name", clusterRole.Name)
					r.updateErrCondition(instance, err)
					return ctrl.Result{}, err
				}
				if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
					if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Namespace + "-" + instance.Name, Namespace: instance.Namespace}, foundClusterRole); err != nil {
						if errors.IsNotFound(err) {
							return false, nil
						} else {
							return false, err
						}
					}
					return true, nil
				}); err != nil {
					return ctrl.Result{}, err
				}
				// Cluster Role created successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			}
			log.Error(err, "Failed to get Cluster Role")
			r.updateErrCondition(instance, err)
		}
	}

	// Check if the Cluster Role Binding already exists, if not create a new one
	foundClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	if !instance.Status.Extracted {
		if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Namespace + "-" + instance.Name, Namespace: instance.Namespace}, foundClusterRoleBinding); err != nil {
			if isJobComplete(found) || instance.Status.Extracted {
				return ctrl.Result{}, nil
			}
			if errors.IsNotFound(err) {
				// Define a new Cluster Role Binding
				clusterRoleBinding := r.clusterRoleBindingGenerate(instance)
				log.Info("Creating a new Cluster Role Binding", "clusterRoleBinding.Namespace", clusterRoleBinding.Namespace, "clusterRoleBinding.Name", clusterRoleBinding.Name)
				if err := r.Create(ctx, clusterRoleBinding); err != nil {
					log.Error(err, "Failed to create new Cluster Role Binding", "clusterRoleBinding.Namespace", clusterRoleBinding.Namespace, "clusterRoleBinding.Name", clusterRoleBinding.Name)
					r.updateErrCondition(instance, err)
					return ctrl.Result{}, err
				}
				if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
					if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Namespace + "-" + instance.Name, Namespace: instance.Namespace}, foundClusterRoleBinding); err != nil {
						if errors.IsNotFound(err) {
							return false, nil
						} else {
							return false, err
						}
					}
					return true, nil
				}); err != nil {
					return ctrl.Result{}, err
				}
				// Cluster Role Binding created successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			}
			log.Error(err, "Failed to get Cluster Role Binding")
			r.updateErrCondition(instance, err)
		}
	}

	// Check if method is download then check if network policy exists,
	// if not create a new one
	if instance.Spec.Method == "download" {
		foundNetPol := &networkingv1.NetworkPolicy{}
		if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundNetPol); err != nil {
			if instance.Status.Completed {
				return ctrl.Result{}, nil
			}
			if errors.IsNotFound(err) {
				// Define a new Network Policy
				netPol := r.netPolGenerate(instance)
				log.Info("Creating a new Network Policy", "netPol.Namespace", netPol.Namespace, "netPol.Name", netPol.Name)
				if err := r.Create(ctx, netPol); err != nil {
					log.Error(err, "Failed to create new Network Policy", "netPol.Namespace", netPol.Namespace, "netPol.Name", netPol.Name)
					r.updateErrCondition(instance, err)
					return ctrl.Result{}, err
				}
				if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
					if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundNetPol); err != nil {
						if errors.IsNotFound(err) {
							return false, nil
						} else {
							return false, err
						}
					}
					return true, nil
				}); err != nil {
					return ctrl.Result{}, err
				}
				// NetPol  created successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			}
			log.Error(err, "Failed to get Network Policy")
			r.updateErrCondition(instance, err)
		}
	}

	// Check if the PVC already exists, if not create a new one
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

				r.updateErrCondition(instance, err)
				return ctrl.Result{}, err
			}
			if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
				if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundService); err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					} else {
						return false, err
					}
				}
				return true, nil
			}); err != nil {
				return ctrl.Result{}, err
			}
			// Service created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get Service")
		r.updateErrCondition(instance, err)
	}

	if isJobComplete(found) {
		instance.Status.Extracted = isJobComplete(found)
		if err := r.Status().Update(ctx, instance); err != nil {
			log.Error(err, "Failed to update Export status")
			r.updateErrCondition(instance, err)
			return ctrl.Result{}, err
		}
		log.Info("Cleaning up clusterrole")
		r.Delete(ctx, foundClusterRole)
		log.Info("cleaning clusterrolebinding")
		r.Delete(ctx, foundClusterRoleBinding)
		log.Info("job cleanup")
		r.Delete(ctx, found, client.PropagationPolicy(metav1.DeletePropagationForeground))
	}

	if instance.Status.Extracted {
		log.Info("Items extracted successfully")
		instance.Status.Route = "https://" + defineRoute(foundRoute) + "/" + instance.Namespace + "-" + instance.ObjectMeta.CreationTimestamp.Rfc3339Copy().Format(time.RFC3339) + ".zip"
		if err := r.Status().Update(ctx, instance); err != nil {
			log.Error(err, "Failed to update Export status")
			r.updateErrCondition(instance, err)
		}
	}

	// Check if deployment already exists, if not create one
	// Deployment is created only for download to serve up
	// the zip file that is created during export
	foundDeployment := &appsv1.Deployment{}
	if instance.Spec.Method == "download" && instance.Status.Extracted {
		log.Info("Serving up Export Download")
		if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundDeployment); err != nil {
			if errors.IsNotFound(err) {
				// Define a new Deployment
				deployment := r.deploymentGenerate(instance, securityContext)
				log.Info("Creating a new Deployment", "deployment.Namespace", deployment.Namespace, "deployment.Name", deployment.Name)
				if err := r.Create(ctx, deployment); err != nil {
					log.Error(err, "Failed to create a Deployment", "deployment.Namespace", deployment.Namespace, "deployment.Name", deployment.Name)

					r.updateErrCondition(instance, err)
					return ctrl.Result{}, err
				}
				if err := wait.Poll(time.Second*1, time.Second*15, func() (done bool, err error) {
					if err := r.Get(ctx, types.NamespacedName{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}, foundDeployment); err != nil {
						if errors.IsNotFound(err) {
							return false, nil
						} else {
							return false, err
						}
					}
					return true, nil
				}); err != nil {
					return ctrl.Result{}, err
				}
				// Service created successfully - return and requeue
				return ctrl.Result{Requeue: true}, nil
			}
			log.Error(err, "Failed to get Deployment")
			r.updateErrCondition(instance, err)
		}
	}

	if instance.Spec.Method == "download" && isDeploymentReady(foundDeployment) {
		log.Info("Items extracted successfully")
		instance.Status.Completed = isDeploymentReady(foundDeployment)
		if err := r.Status().Update(ctx, instance); err != nil {
			log.Error(err, "Failed to update Export status")
			r.updateErrCondition(instance, err)
			return ctrl.Result{}, err
		}
	} else {
		log.Info("Deployment is not ready")
	}

	return ctrl.Result{}, nil
}

func (r *ExportReconciler) CleanUpOpjects(ctx context.Context, instance *primerv1alpha1.Export) error {
	// Delete all the objects that we created to make sure that things are removed.
	err := r.Delete(ctx, &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}}, client.PropagationPolicy(metav1.DeletePropagationBackground))
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &routev1.Route{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Namespace + "-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}
	err = r.Delete(ctx, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Namespace + "-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	err = r.Delete(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "primer-export-" + instance.Name, Namespace: instance.Namespace}})
	if err != nil && !(errors.IsGone(err) || errors.IsNotFound(err)) {
		return err
	}

	return nil
}

func (r *ExportReconciler) updateErrCondition(instance *primerv1alpha1.Export, err error) {
	instance.Status.Conditions.SetCondition(
		status.Condition{
			Type:    primerv1alpha1.ConditionReconciled,
			Status:  corev1.ConditionFalse,
			Reason:  primerv1alpha1.ReconciledReasonError,
			Message: err.Error(),
		})
	r.Client.Status().Update(context.TODO(), instance)
}

// jobGitForExport returns a instance Job object
func (r *ExportReconciler) jobGitForExport(m *primerv1alpha1.Export, securityContext *corev1.SecurityContext) *batchv1.Job {
	groupString := strings.Join(m.Spec.Group, ";")
	mode := int32(0644)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					ServiceAccountName: "primer-export-" + m.Name,
					Containers: []corev1.Container{{
						Name:            m.Name,
						ImagePullPolicy: "IfNotPresent",
						Image:           r.ExportImage,
						Command:         []string{"/bin/sh", "-c", "/committer.sh"},
						Env: []corev1.EnvVar{
							{Name: "REPO", Value: m.Spec.Repo},
							{Name: "BRANCH", Value: m.Spec.Branch},
							{Name: "EMAIL", Value: m.Spec.Email},
							{Name: "NAMESPACE", Value: m.Namespace},
							{Name: "METHOD", Value: m.Spec.Method},
							{Name: "USER", Value: m.Spec.User},
							{Name: "GROUP", Value: groupString},
							{Name: "LABELS", Value: m.Spec.LabelSelector},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "sshkeys", MountPath: "/keys"},
							{Name: "output", MountPath: "/output"},
						},
						SecurityContext: securityContext,
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
func (r *ExportReconciler) jobDownloadForExport(m *primerv1alpha1.Export, securityContext *corev1.SecurityContext) *batchv1.Job {
	groupString := strings.Join(m.Spec.Group, ";")
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					ServiceAccountName: "primer-export-" + m.Name,
					Containers: []corev1.Container{{
						Name:            m.Name,
						ImagePullPolicy: "IfNotPresent",
						Image:           r.ExportImage,
						Command:         []string{"/bin/sh", "-c", "/committer.sh"},
						Env: []corev1.EnvVar{
							{Name: "METHOD", Value: m.Spec.Method},
							{Name: "NAMESPACE", Value: m.Namespace},
							{Name: "EXPORT_NAME", Value: m.Name},
							{Name: "USER", Value: m.Spec.User},
							{Name: "GROUP", Value: groupString},
							{Name: "TIME", Value: m.ObjectMeta.CreationTimestamp.Rfc3339Copy().Format(time.RFC3339)},
							{Name: "LABELS", Value: m.Spec.LabelSelector},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "output", MountPath: "/output"},
						},
						SecurityContext: securityContext,
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
	routeName := "primer-export-" + m.Name
	oauthRedirectAnnotation := "serviceaccounts.openshift.io/oauth-redirectreference." + routeName
	oauthRedirectValue := `{
  	"kind": "OAuthRedirectReference",
  	"apiVersion": "v1",
  	"reference": {
    "kind": "Route",
    "name": "` + routeName + `"
 	}
	}`
	serviceAcct := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Annotations: map[string]string{
				oauthRedirectAnnotation: oauthRedirectValue,
			},
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
		},
	}
	// Service Account reconcile finished
	ctrl.SetControllerReference(m, serviceAcct, r.Scheme)
	return serviceAcct
}

func (r *ExportReconciler) routeGenerate(m *primerv1alpha1.Export) *routev1.Route {
	// Define a new Route object
	appRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
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

	// Route reconcile finished
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
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"session_secret": []byte(random),
		},
	}
	// Secret reconcile finished
	ctrl.SetControllerReference(m, proxySecret, r.Scheme)
	return proxySecret
}

func (r *ExportReconciler) pvcGenerate(m *primerv1alpha1.Export) *corev1.PersistentVolumeClaim {
	// Define a new PVC object
	persistentVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
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
	// PVC reconcile finished
	ctrl.SetControllerReference(m, persistentVC, r.Scheme)
	return persistentVC
}

func (r *ExportReconciler) clusterRoleGenerate(m *primerv1alpha1.Export) *rbacv1.ClusterRole {
	// Define a new clusterRole object
	impersonateList := append(m.Spec.Group, m.Spec.User)
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "primer-export-" + m.Namespace + "-" + m.Name,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"users", "groups"},
				Verbs:         []string{"impersonate"},
				ResourceNames: impersonateList,
			},
		},
	}
	// ClusterRole reconcile finished
	return clusterRole
}

func (r *ExportReconciler) clusterRoleBindingGenerate(m *primerv1alpha1.Export) *rbacv1.ClusterRoleBinding {
	// Define a new ClusterRole binding object
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "primer-export-" + m.Namespace + "-" + m.Name,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "primer-export-" + m.Namespace + "-" + m.Name,
			Kind:     "ClusterRole",
		},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: "primer-export-" + m.Name, Namespace: m.Namespace},
		},
	}
	// ClusterRole Binding reconcile finished
	return clusterRoleBinding
}

func (r *ExportReconciler) svcGenerate(m *primerv1alpha1.Export) *corev1.Service {
	// Define a new service and generate secret
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Annotations: map[string]string{
				"service.alpha.openshift.io/serving-cert-secret-name": "primer-export-" + m.Name + "-tls",
			},
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
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

func (r *ExportReconciler) deploymentGenerate(m *primerv1alpha1.Export, securityContext *corev1.SecurityContext) *appsv1.Deployment {
	// Define the deployment for hosting the export
	replicas := int32(1)
	secretMode := int32(420)
	primerName := "primer-export-" + m.Name
	openshiftSar := `{-openshift-sar={"resource": "namespaces","resourceName":` + primerName + `,"namespace": ` + m.Namespace + `,"verb":"get"}"`
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
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
						gitopsPrimerLabel:             "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: r.DownloaderImage,
							Name:  "primer-export-" + m.Name,
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8080,
								Name:          "downloader",
							}},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "output", MountPath: "/var/www/html"},
							},
							SecurityContext: securityContext,
						},
						{
							Image: r.OauthImage,
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
								openshiftSar,
							},
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8888,
								Name:          "oath-proxy",
							}},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "primer-oauth-tls", MountPath: "/etc/tls/private"},
								{Name: "secret-primer-proxy", MountPath: "/etc/proxy/secrets"},
							},
							SecurityContext: securityContext,
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
	// Deployment reconcile finished
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func (r *ExportReconciler) netPolGenerate(m *primerv1alpha1.Export) *networkingv1.NetworkPolicy {
	// Define a new network Policy
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "primer-export-" + m.Name,
			Namespace: m.Namespace,
			Labels: map[string]string{
				gitopsPrimerLabel: "true",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "primer-export-" + m.Name, "app.kubernetes.io/component": "primer-export-" + m.Name, "app.kubernetes.io/part-of": "primer-export-" + m.Name}},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"network.openshift.io/policy-group": "ingress",
						},
					},
				}},
			}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
	// Network Policy reconcile finished
	ctrl.SetControllerReference(m, networkPolicy, r.Scheme)
	return networkPolicy
}

// Check to see if job is completed
func isJobComplete(job *batchv1.Job) bool {
	return job.Status.Succeeded == 1
}

// Identify route to be used for status
func defineRoute(route *routev1.Route) string {
	return route.Spec.Host
}

func isDeploymentReady(deployment *appsv1.Deployment) bool {
	return deployment.Status.ReadyReplicas == 1
}

func getSecurityContext() (*corev1.SecurityContext, error) {
	falseBool := bool(false)
	trueBool := bool(true)
	kubeconf := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(kubeconf)
	if err != nil {
		return nil, err
	}

	version, err := clientset.ServerVersion()
	if err != nil {
		return nil, err
	}

	minor, err := strconv.Atoi(version.Minor)
	if err != nil {
		return nil, err
	}

	if minor < 24 {
		return &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			AllowPrivilegeEscalation: &falseBool,
			RunAsNonRoot:             &trueBool,
		}, nil
	}
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		AllowPrivilegeEscalation: &falseBool,
		RunAsNonRoot:             &trueBool,
		SeccompProfile: &corev1.SeccompProfile{
			Type: "RuntimeDefault",
		},
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	DownloaderImage := os.Getenv("DownloaderImageName")
	if DownloaderImage == "" {
		DownloaderImage = "quay.io/migtools/gitops-primer:v0.0.11"
	}
	r.DownloaderImage = DownloaderImage

	ExportImage := os.Getenv("ExportImageName")
	if ExportImage == "" {
		ExportImage = "quay.io/migtools/gitops-primer-export:v0.0.11"
	}
	r.ExportImage = ExportImage

	OauthImage := os.Getenv("OauthImageName")
	if OauthImage == "" {
		OauthImage = "quay.io/openshift/origin-oauth-proxy:4.10"
	}
	r.OauthImage = OauthImage
	return ctrl.NewControllerManagedBy(mgr).
		For(&primerv1alpha1.Export{}).
		Owns(&batchv1.Job{}, builder.OnlyMetadata).
		Owns(&appsv1.Deployment{}, builder.OnlyMetadata).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 5,
		}).
		Complete(r)
}
