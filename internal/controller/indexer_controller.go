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
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	"github.com/rss3-network/node-operator/internal/controller/factory"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IndexerReconciler reconciles a Indexer object
type IndexerReconciler struct {
	client.Client
	Log    *zap.Logger
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
	log := r.Log.With(zap.String("indexer", req.NamespacedName.String()))

	indexer := &nodev1alpha1.Indexer{}
	err := r.Get(ctx, req.NamespacedName, indexer)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("indexer resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error("failed to get indexer", zap.Error(err))
		return ctrl.Result{}, err
	}

	if err = factory.AddFinalizer(ctx, r.Client, indexer); err != nil {
		log.Error("failed to add finalizer", zap.Error(err))
		return ctrl.Result{}, err
	}

	isIndexerMarkedToBeDeleted := indexer.GetDeletionTimestamp() != nil
	if isIndexerMarkedToBeDeleted {
		if err := factory.OnIndexerDelete(ctx, r.Client, indexer); err != nil {
			log.Error("failed to finalize indexer", zap.Error(err))
			return ctrl.Result{}, err
		}
	}

	return reconcileWithDiff(ctx, r.Client, indexer, func() (ctrl.Result, error) {

		if err := factory.CreateOrUpdateIndexer(ctx, r.Log, indexer, r.Client); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Status().Update(ctx, indexer); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update indexer status: %w", err)
		}
		return ctrl.Result{}, nil
	})

}

//+kubebuilder:rbac:groups=node.rss3.io,resources=indexers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node.rss3.io,resources=indexers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node.rss3.io,resources=indexers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *IndexerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodev1alpha1.Indexer{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
