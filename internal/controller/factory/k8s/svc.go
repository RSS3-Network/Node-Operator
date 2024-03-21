package k8s

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func HandleSvcUpdate(ctx context.Context, rc client.Client, svc *corev1.Service, waitDeadline time.Duration) error {
	// Check if the service already exists, if not create a new one
	var currentSvc corev1.Service
	err := rc.Get(ctx, client.ObjectKey{Name: currentSvc.Name, Namespace: currentSvc.Namespace}, &currentSvc)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := rc.Create(ctx, svc); err != nil {
				return fmt.Errorf("cannot create new service for app: %s, err: %w", svc.Name, err)
			}
			return nil
		}
		return fmt.Errorf("cannot get service for app: %s err: %w", svc.Name, err)
	}

	if err := rc.Update(ctx, svc); err != nil {
		return fmt.Errorf("cannot update service for app: %s, err: %w", svc.Name, err)
	}

	return nil
}
