package k8s

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func HandleConfigmapUpdate(ctx context.Context, rc client.Client, cm *corev1.ConfigMap, waitDeadline time.Duration) error {
	var currentCM corev1.ConfigMap
	err := rc.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, &currentCM)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := rc.Create(ctx, cm); err != nil {
				return fmt.Errorf("cannot create new configmap for app: %s, err: %w", cm.Name, err)
			}
			return nil
		}
		return fmt.Errorf("cannot get configmap for app: %s err: %w", cm.Name, err)
	}

	cm.Annotations = MergeAnnotations(currentCM.Annotations, cm.Annotations)

	if err := rc.Update(ctx, cm); err != nil {
		return fmt.Errorf("cannot update configmap for app: %s, err: %w", cm.Name, err)
	}

	return nil
}
