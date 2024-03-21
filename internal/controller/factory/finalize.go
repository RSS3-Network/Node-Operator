package factory

import (
	"context"
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func AddFinalizer(ctx context.Context, rc client.Client, instance client.Object) error {
	if !controllerutil.ContainsFinalizer(instance, nodev1alpha1.NodeFinalizer) {
		controllerutil.AddFinalizer(instance, nodev1alpha1.NodeFinalizer)
		return rc.Update(ctx, instance)
	}
	return nil
}

func OnHubDelete(ctx context.Context, rc client.Client, instance *nodev1alpha1.Hub) (err error) {
	if controllerutil.ContainsFinalizer(instance, nodev1alpha1.NodeFinalizer) {
		// Let's add here a status "Downgrade" to define that this resource begin its process to be terminated.
		if err = rc.Status().Update(ctx, instance); err != nil {
			return
		}

		if err = rc.Get(ctx, types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Name,
		}, instance); err != nil {
			return
		}

		if err = rc.Status().Update(ctx, instance); err != nil {
			return
		}

		if ok := controllerutil.RemoveFinalizer(instance, nodev1alpha1.NodeFinalizer); !ok {
			return nil
		}

		return rc.Update(ctx, instance)
	}
	return nil
}

func OnIndexerDelete(ctx context.Context, rc client.Client, instance *nodev1alpha1.Indexer) (err error) {
	if controllerutil.ContainsFinalizer(instance, nodev1alpha1.NodeFinalizer) {
		// Let's add here a status "Downgrade" to define that this resource begin its process to be terminated.
		if err = rc.Status().Update(ctx, instance); err != nil {
			return
		}

		if err = rc.Get(ctx, types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Name,
		}, instance); err != nil {
			return
		}

		if err = rc.Status().Update(ctx, instance); err != nil {
			return
		}

		if ok := controllerutil.RemoveFinalizer(instance, nodev1alpha1.NodeFinalizer); !ok {
			return nil
		}

		return rc.Update(ctx, instance)
	}
	return nil
}
