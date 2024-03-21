package factory

import (
	"context"
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func finalizerWithoutNode(finalizers []string) []string {
	return lo.Filter(finalizers, func(s string, index int) bool {
		return s != nodev1alpha1.NodeFinalizer
	})
}

func AddFinalizer(ctx context.Context, rc client.Client, instance client.Object) error {
	if !controllerutil.ContainsFinalizer(instance, nodev1alpha1.NodeFinalizer) {
		controllerutil.AddFinalizer(instance, nodev1alpha1.NodeFinalizer)
		return rc.Update(ctx, instance)
	}
	return nil
}

func RemoveFinalizer(ctx context.Context, rc client.Client, instance client.Object) error {
	if controllerutil.ContainsFinalizer(instance, nodev1alpha1.NodeFinalizer) {
		instance.SetFinalizers(finalizerWithoutNode(instance.GetFinalizers()))
		return rc.Update(ctx, instance)
	}
	return nil
}

func removeFinalizeObjByName(ctx context.Context, rclient client.Client, obj client.Object, name, ns string) error {
	if err := rclient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, obj); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// fast path
	if !controllerutil.ContainsFinalizer(obj, nodev1alpha1.NodeFinalizer) {
		return nil
	}
	obj.SetFinalizers(finalizerWithoutNode(obj.GetFinalizers()))
	return rclient.Update(ctx, obj)
}

func OnHubDelete(ctx context.Context, rc client.Client, instance *nodev1alpha1.Hub) (err error) {
	if err = removeFinalizeObjByName(ctx, rc, &appsv1.Deployment{}, instance.Name, instance.Namespace); err != nil {
		return err
	}

	if err = removeFinalizeObjByName(ctx, rc, &corev1.Service{}, instance.Name, instance.Namespace); err != nil {
		return err
	}

	//if err = RemoveOrphanedResource(ctx, rc,)
	return removeFinalizeObjByName(ctx, rc, instance, instance.Name, instance.Namespace)
}

func OnIndexerDelete(ctx context.Context, rc client.Client, instance *nodev1alpha1.Indexer) (err error) {
	if err = removeFinalizeObjByName(ctx, rc, &appsv1.StatefulSet{}, instance.Name, instance.Namespace); err != nil {
		return err
	}

	if err = removeFinalizeObjByName(ctx, rc, &corev1.ConfigMap{}, instance.Name, instance.Namespace); err != nil {
		return err
	}

	return RemoveFinalizer(ctx, rc, instance)
}
