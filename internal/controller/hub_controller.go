/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
)

const hubFinalizer = "node.rss3.io/finalizer"

const (
	// typeAvailableHub represents the status of the Deployment reconciliation
	typeAvailableHub = "Available"
	// typeDegradedHub represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	typeDegradedHub = "Degraded"
)

// HubReconciler reconciles a Hub object
type HubReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Hub object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *HubReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	hub := &nodev1alpha1.Hub{}
	err := r.Get(ctx, req.NamespacedName, hub)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("hub resource not fount. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get hub")
		return ctrl.Result{}, err
	}

	if hub.Status.Conditions == nil || len(hub.Status.Conditions) == 0 {
		meta.SetStatusCondition(
			&hub.Status.Conditions,
			metav1.Condition{
				Type:    typeAvailableHub,
				Status:  metav1.ConditionUnknown,
				Reason:  "Reconciling",
				Message: "Starting reconciliation",
			})

		if err = r.Status().Update(ctx, hub); err != nil {
			log.Error(err, "Failed to update Hub status")
			return ctrl.Result{}, err
		}

		if err := r.Get(ctx, req.NamespacedName, hub); err != nil {
			log.Error(err, "Failed to re-fetch hub")
			return ctrl.Result{}, err
		}
	}

	if !controllerutil.ContainsFinalizer(hub, hubFinalizer) {
		log.Info("Adding Finalizer for Hub")
		if ok := controllerutil.AddFinalizer(hub, hubFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, hub); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the Hub instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isHubMarkedToBeDeleted := hub.GetDeletionTimestamp() != nil
	if isHubMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(hub, hubFinalizer) {
			log.Info("Performing Finalizer Operations for Hub before delete CR")

			// Let's add here a status "Downgrade" to define that this resource begin its process to be terminated.
			meta.SetStatusCondition(&hub.Status.Conditions, metav1.Condition{Type: typeDegradedHub,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", hub.Name)})

			if err := r.Status().Update(ctx, hub); err != nil {
				log.Error(err, "Failed to update Hub status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before remove the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			r.doFinalizerOperationsForHub(hub)

			// TODO(user): If you add operations to the doFinalizerOperationsForHub method
			// then you need to ensure that all worked fine before deleting and updating the Downgrade status
			// otherwise, you should requeue here.

			// Re-fetch the hub Custom Resource before update the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raise the issue "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, hub); err != nil {
				log.Error(err, "Failed to re-fetch hub")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&hub.Status.Conditions, metav1.Condition{Type: typeDegradedHub,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", hub.Name)})

			if err := r.Status().Update(ctx, hub); err != nil {
				log.Error(err, "Failed to update Hub status")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer for Hub after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(hub, hubFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer for Hub")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, hub); err != nil {
				log.Error(err, "Failed to remove finalizer for Hub")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: hub.Name, Namespace: hub.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForHub(hub)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for Hub")

			// The following implementation will update the status
			meta.SetStatusCondition(&hub.Status.Conditions, metav1.Condition{Type: typeAvailableHub,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", hub.Name, err)})

			if err := r.Status().Update(ctx, hub); err != nil {
				log.Error(err, "Failed to update Hub status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// finalizeHub will perform the required operations before delete the CR.
func (r *HubReconciler) doFinalizerOperationsForHub(cr *nodev1alpha1.Hub) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	// Note: It is not recommended to use finalizers with the purpose of delete resources which are
	// created and managed in the reconciliation. These, such as the Deployment created on this reconcile,
	// are defined as depended on the custom resource. See that we use the method ctrl.SetControllerReference.
	// to set the ownerRef which means that the Deployment will be deleted by the Kubernetes API.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/

	// The following implementation will raise an event
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

// deploymentForHub returns a Hub Deployment object
func (r *HubReconciler) deploymentForHub(
	hub *nodev1alpha1.Hub) (*appsv1.Deployment, error) {
	ls := labelsForHub(hub.Name)
	replicas := hub.Spec.Replicas

	// Get the Operand image
	image, err := imageForNode()
	if err != nil {
		return nil, err
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hub.Name,
			Namespace: hub.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					// TODO(user): Uncomment the following code to configure the nodeAffinity expression
					// according to the platforms which are supported by your solution. It is considered
					// best practice to support multiple architectures. build your manager image using the
					// makefile target docker-buildx. Also, you can use docker manifest inspect <image>
					// to check what are the platforms supported.
					// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity
					//Affinity: &corev1.Affinity{
					//	NodeAffinity: &corev1.NodeAffinity{
					//		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					//			NodeSelectorTerms: []corev1.NodeSelectorTerm{
					//				{
					//					MatchExpressions: []corev1.NodeSelectorRequirement{
					//						{
					//							Key:      "kubernetes.io/arch",
					//							Operator: "In",
					//							Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
					//						},
					//						{
					//							Key:      "kubernetes.io/os",
					//							Operator: "In",
					//							Values:   []string{"linux"},
					//						},
					//					},
					//				},
					//			},
					//		},
					//	},
					//},
					//SecurityContext: &corev1.PodSecurityContext{
					//	RunAsNonRoot: &[]bool{true}[0],
					//	// IMPORTANT: seccomProfile was introduced with Kubernetes 1.19
					//	// If you are looking for to produce solutions to be supported
					//	// on lower versions you must remove this option.
					//	SeccompProfile: &corev1.SeccompProfile{
					//		Type: corev1.SeccompProfileTypeRuntimeDefault,
					//	},
					//},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "hub",
						ImagePullPolicy: corev1.PullIfNotPresent,
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						//SecurityContext: &corev1.SecurityContext{
						//	// WARNING: Ensure that the image used defines an UserID in the Dockerfile
						//	// otherwise the Pod will not run and will fail with "container has runAsNonRoot and image has non-numeric user"".
						//	// If you want your workloads admitted in namespaces enforced with the restricted mode in OpenShift/OKD vendors
						//	// then, you MUST ensure that the Dockerfile defines a User ID OR you MUST leave the "RunAsNonRoot" and
						//	// "RunAsUser" fields empty.
						//	RunAsNonRoot: &[]bool{true}[0],
						//	// The hub image does not use a non-zero numeric user as the default user.
						//	// Due to RunAsNonRoot field being set to true, we need to force the user in the
						//	// container to a non-zero numeric user. We do this using the RunAsUser field.
						//	// However, if you are looking to provide solution for K8s vendors like OpenShift
						//	// be aware that you cannot run under its restricted-v2 SCC if you set this value.
						//	RunAsUser:                &[]int64{1001}[0],
						//	AllowPrivilegeEscalation: &[]bool{false}[0],
						//	Capabilities: &corev1.Capabilities{
						//		Add: []corev1.Capability{
						//			"NET_BIND_SERVICE",
						//		},
						//	},
						//},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "http",
						}},
						Env:  hub.Spec.DatabaseRef.EnvVars(),
						Args: []string{"--module=hub"},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.IntOrString{
										Type:   intstr.Int,
										IntVal: 80,
									},
								},
							},
							TimeoutSeconds:   3,
							PeriodSeconds:    10,
							SuccessThreshold: 1,
							FailureThreshold: 3,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.IntOrString{
										Type:   intstr.Int,
										IntVal: 80,
									},
								},
							},
							TimeoutSeconds:   3,
							PeriodSeconds:    10,
							SuccessThreshold: 1,
							FailureThreshold: 3,
						},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(hub, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

// labelsForHub returns the labels for selecting the resources
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func labelsForHub(name string) map[string]string {
	var imageTag string
	image, err := imageForNode()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{"app.kubernetes.io/name": "Hub",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/part-of":    "node-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *HubReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodev1alpha1.Hub{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
