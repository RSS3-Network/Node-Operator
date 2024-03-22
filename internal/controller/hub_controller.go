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
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	"github.com/rss3-network/node-operator/internal/controller/factory"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
)

var (
	hubSync sync.Mutex
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
	logger := log.FromContext(ctx)
	hubSync.Lock()
	defer hubSync.Unlock()

	hub := &nodev1alpha1.Hub{}
	if err := r.Get(ctx, req.NamespacedName, hub); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("hub resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get hub")
		return ctrl.Result{}, err
	}

	// Check if the Hub instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if !hub.GetDeletionTimestamp().IsZero() {
		if err := factory.OnHubDelete(ctx, r.Client, hub); err != nil {
			logger.Error(err, "Failed to finalize hub")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, nil
	}

	if err := factory.AddFinalizer(ctx, r.Client, hub); err != nil {
		logger.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	return reconcileWithDiff(ctx, r.Client, hub, func() (ctrl.Result, error) {
		if err := factory.CreateOrUpdateHub(ctx, hub, r.Client); err != nil {
			return ctrl.Result{}, err
		}

		if _, err := factory.CreateOrUpdateHubService(ctx, hub, r.Client); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Status().Update(ctx, hub); err != nil {
			logger.Error(err, "Failed to update hub status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	})
}

//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node.rss3.io,resources=hubs/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *HubReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodev1alpha1.Hub{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
