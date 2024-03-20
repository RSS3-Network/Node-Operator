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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
)

// IndexerReconciler reconciles a Indexer object
type IndexerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=node.rss3.io,resources=indexers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node.rss3.io,resources=indexers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node.rss3.io,resources=indexers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Indexer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *IndexerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	indexer := &nodev1alpha1.Indexer{}
	err := r.Get(ctx, req.NamespacedName, indexer)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("indexer resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get indexer")
		return ctrl.Result{}, err
	}

	if indexer.Status.Conditions == nil || len(indexer.Status.Conditions) == 0 {
		meta.SetStatusCondition(
			&indexer.Status.Conditions,
			metav1.Condition{
				Type:    typeAvailable,
				Status:  metav1.ConditionUnknown,
				Reason:  "Reconciling",
				Message: "Starting reconciliation",
			})

		if err = r.Status().Update(ctx, indexer); err != nil {
			log.Error(err, "failed to update indexer status")
			return ctrl.Result{}, err
		}

		if err = r.Get(ctx, req.NamespacedName, indexer); err != nil {
			log.Error(err, "failed to get indexer")
			return ctrl.Result{}, err
		}
	}

	if !controllerutil.ContainsFinalizer(indexer, nodeFinalizer) {
		log.Info("Adding finalizer to indexer")
		if ok := controllerutil.AddFinalizer(indexer, nodeFinalizer); !ok {
			log.Error(err, "failed to add finalizer to indexer")
			return ctrl.Result{Requeue: true}, err
		}

		if err = r.Update(ctx, indexer); err != nil {
			log.Error(err, "failed to update indexer")
			return ctrl.Result{}, err
		}
	}

	isIndexerMarkedToBeDeleted := indexer.GetDeletionTimestamp() != nil
	if isIndexerMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(indexer, nodeFinalizer) {
			log.Info("Performing finalizer operations for indexer before delete")

			meta.SetStatusCondition(&indexer.Status.Conditions, metav1.Condition{
				Type:    typeDegraded,
				Status:  metav1.ConditionUnknown,
				Reason:  "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", indexer.Name)})
		}

		if err = r.Status().Update(ctx, indexer); err != nil {
			log.Error(err, "failed to update indexer status")
			return ctrl.Result{}, err
		}

		r.doFinalizerOperations(indexer)

		if err := r.Get(ctx, req.NamespacedName, indexer); err != nil {
			log.Error(err, "failed to re-fetch indexer")
			return ctrl.Result{}, err
		}

		meta.SetStatusCondition(&indexer.Status.Conditions,
			metav1.Condition{
				Type:    typeDegraded,
				Status:  metav1.ConditionTrue,
				Reason:  "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", indexer.Name),
			})

		if err = r.Status().Update(ctx, indexer); err != nil {
			log.Error(err, "failed to update indexer status")
			return ctrl.Result{}, err
		}

		log.Info("Removing finalizer from indexer after successfully perform the operations")
		if ok := controllerutil.RemoveFinalizer(indexer, nodeFinalizer); !ok {
			log.Error(err, "failed to remove finalizer from indexer")
			return ctrl.Result{}, err
		}

		if err = r.Update(ctx, indexer); err != nil {
			log.Error(err, "failed to remove finalizer from indexer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	found := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: indexer.Name, Namespace: indexer.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new StatefulSet
		sts, err := r.statefulSetForIndexer(indexer)
		if err != nil {
			log.Error(err, fmt.Sprintf("failed to create statefulset for indexer %s", indexer.Name))

			meta.SetStatusCondition(
				&indexer.Status.Conditions,
				metav1.Condition{
					Type:    typeAvailable,
					Status:  metav1.ConditionFalse,
					Reason:  "Reconciling",
					Message: fmt.Sprintf("Failed to create statefulset for the custom resource (%s): (%s)", indexer.Name, err),
				},
			)

			if err = r.Status().Update(ctx, indexer); err != nil {
				log.Error(err, "failed to update indexer status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
		if err = r.Create(ctx, sts); err != nil {
			log.Error(err, "failed to create new Statefulset")
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "failed to get statefulset")
		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(
		&indexer.Status.Conditions,
		metav1.Condition{
			Type:    typeAvailable,
			Status:  metav1.ConditionTrue,
			Reason:  "Reconciling",
			Message: fmt.Sprintf("Statefulset for custom resource (%s) created successfully", indexer.Name),
		},
	)

	if err = r.Status().Update(ctx, indexer); err != nil {
		log.Error(err, "failed to update indexer status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

func (r *IndexerReconciler) doFinalizerOperations(cr *nodev1alpha1.Indexer) {
	// Add finalizer logic here
}

// statefulSetForIndexer returns a indexer StatefulSet object
func (r *IndexerReconciler) statefulSetForIndexer(indexer *nodev1alpha1.Indexer) (*appsv1.StatefulSet, error) {
	ls := labelsForIndexer(indexer.Name)

	replicas := int32(1)

	image, err := imageForNode()
	if err != nil {
		return nil, err
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        indexer.Name,
			Namespace:   indexer.Namespace,
			Annotations: indexer.Annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{
								Medium: corev1.StorageMediumDefault,
							},
						},
					}, {
						Name: "template",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "indexer-config-template",
								},
							},
						},
					}},
					InitContainers: []corev1.Container{{
						Image:   "alpine:3.18",
						Name:    "generate-config",
						Command: []string{"/bin/sh", "-c"},
						Args: []string{
							"apk add --no-cache envsubst && envsubst < /etc/template/indexer.yaml.tmpl > /etc/config/indexer.yaml",
						},
						Env: []corev1.EnvVar{{
							Name:  "NODE_INDEXER_NETWORK",
							Value: indexer.Spec.Network,
						}, {
							Name:  "NODE_INDEXER_WORKER",
							Value: indexer.Spec.Worker,
						}, {
							Name:  "NODE_INDEXER_PARAMS",
							Value: indexer.Spec.Params.String(),
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/etc/config",
						}, {
							Name:      "template",
							MountPath: "/etc/template",
						}},
					}},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "indexer",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env:             indexer.Spec.DatabaseRef.EnvVars(),
						Args:            []string{"--module=indexer"},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/etc/config",
						}},
					}},
				},
			},
		},
	}

	// Set Indexer instance as the owner and controller
	if err := controllerutil.SetControllerReference(indexer, sts, r.Scheme); err != nil {
		return nil, err
	}

	return sts, nil

}

// labelsForIndexer returns the labels for selecting the resources
func labelsForIndexer(name string) map[string]string {
	var imageTag string
	image, err := imageForNode()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{
		"app.kubernetes.io/name":      "node",
		"app.kubernetes.io/instance":  name,
		"app.kubernetes.io/version":   imageTag,
		"app.kubernetes.io/component": "indexer",

		"app.kubernetes.io/part-of":    "node-operator",
		"app.kubernetes.io/managed-by": "node-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodev1alpha1.Indexer{}).
		Complete(r)
}
