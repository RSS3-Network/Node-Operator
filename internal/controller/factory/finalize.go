package factory

import (
	"context"
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func AddFinalizer(ctx context.Context, rc client.Client, instance client.Object) error {
	if !controllerutil.ContainsFinalizer(instance, nodev1alpha1.NodeFinalizer) {
		controllerutil.AddFinalizer(instance, nodev1alpha1.NodeFinalizer)
		return rc.Update(ctx, instance)
	}
	return nil
}

func RemoveFinalizer(ctx context.Context, rc client.Client, instance client.Object) error {
	if controllerutil.ContainsFinalizer(instance, nodev1alpha1.NodeFinalizer) {
		controllerutil.RemoveFinalizer(instance, nodev1alpha1.NodeFinalizer)
		return rc.Update(ctx, instance)
	}
	return nil
}

func removeFinalizeObjByName(ctx context.Context, rclient client.Client, obj client.Object, name, ns string) error {
	logger := log.FromContext(ctx)
	if err := rclient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if !controllerutil.ContainsFinalizer(obj, nodev1alpha1.NodeFinalizer) {
		return nil
	}
	if ok := controllerutil.RemoveFinalizer(obj, nodev1alpha1.NodeFinalizer); ok {
		logger.Info("removing finalizer",
			"name", name,
			"namespace", ns,
			"kind",
			obj.GetObjectKind().GroupVersionKind().Kind,
		)
	} else {
		logger.Info("finalizer not found",
			"name", name,
			"namespace", ns,
			"kind",
			obj.GetObjectKind().GroupVersionKind().Kind,
		)
		return nil
	}
	return rclient.Update(ctx, obj)
}

func OnHubDelete(ctx context.Context, rc client.Client, instance *nodev1alpha1.Hub) (err error) {
	return removeFinalizeObjByName(ctx, rc, &nodev1alpha1.Hub{}, instance.Name, instance.Namespace)
}

func OnIndexerDelete(ctx context.Context, rc client.Client, instance *nodev1alpha1.Indexer) (err error) {
	return removeFinalizeObjByName(ctx, rc, &nodev1alpha1.Indexer{}, instance.Name, instance.Namespace)
}
