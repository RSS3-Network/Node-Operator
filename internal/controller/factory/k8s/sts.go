package k8s

import (
	"context"
	"fmt"
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func HandleSTSUpdate(ctx context.Context, rc client.Client, newSts *appsv1.StatefulSet, waitDeadline time.Duration) error {
	var currentSts appsv1.StatefulSet
	err := rc.Get(ctx, types.NamespacedName{Name: newSts.Name, Namespace: newSts.Namespace}, &currentSts)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := rc.Create(ctx, newSts); err != nil {
				return fmt.Errorf("cannot create new statefulset for app: %s, err: %w", newSts.Name, err)
			}
			return waitStatefulSetReady(ctx, rc, newSts, waitDeadline)
		}
		return fmt.Errorf("cannot get statefulset for app: %s err: %w", newSts.Name, err)
	}

	newSts.Spec.Template.Annotations = MergeAnnotations(currentSts.Spec.Template.Annotations, newSts.Spec.Template.Annotations)
	newSts.Finalizers = MergeFinalizers(&currentSts, nodev1alpha1.NodeFinalizer)
	newSts.Status = currentSts.Status
	newSts.Annotations = MergeAnnotations(currentSts.Annotations, newSts.Annotations)

	if err := rc.Update(ctx, newSts); err != nil {
		return fmt.Errorf("cannot update statefulset for app: %s, err: %w", newSts.Name, err)
	}

	return waitStatefulSetReady(ctx, rc, newSts, waitDeadline)
}

func waitStatefulSetReady(ctx context.Context, rc client.Client, sts *appsv1.StatefulSet, waitDeadline time.Duration) error {
	time.Sleep(2 * time.Second)
	return wait.PollUntilContextTimeout(ctx, time.Second*5, waitDeadline, true, func(ctx context.Context) (bool, error) {
		var actual appsv1.StatefulSet
		if err := rc.Get(ctx, client.ObjectKey{Name: sts.Name, Namespace: sts.Namespace}, &actual); err != nil {
			return false, fmt.Errorf("cannot fetch actual statefulset: %w", err)
		}

		return false, nil
	})
}
