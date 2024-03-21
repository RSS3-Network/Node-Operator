package k8s

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func HandleSvcUpdate(ctx context.Context, rc client.Client, svc *corev1.Service, waitDeadline time.Duration) error {
	// Check if the service already exists, if not create a new one
	var currentSvc corev1.Service
	err := rc.Get(ctx, client.ObjectKey{Name: svc.Name, Namespace: svc.Namespace}, &currentSvc)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := rc.Create(ctx, svc); err != nil {
				return fmt.Errorf("cannot create new service for app: %s, err: %w", svc.Name, err)
			}
			return waitServiceReady(ctx, rc, svc, waitDeadline)
		}
		return fmt.Errorf("cannot get service for app: %s err: %w", svc.Name, err)
	}

	if err := rc.Update(ctx, svc); err != nil {
		return fmt.Errorf("cannot update service for app: %s, err: %w", svc.Name, err)
	}

	return waitServiceReady(ctx, rc, svc, waitDeadline)
}

func waitServiceReady(ctx context.Context, rc client.Client, svc *corev1.Service, waitDeadline time.Duration) error {
	time.Sleep(2 * time.Second)
	return wait.PollUntilContextTimeout(ctx, time.Second, waitDeadline, true, func(ctx context.Context) (bool, error) {
		var actual corev1.Service
		if err := rc.Get(ctx, client.ObjectKey{Name: svc.Name, Namespace: svc.Namespace}, &actual); err != nil {
			return false, fmt.Errorf("cannot fetch actual statefulset: %w", err)
		}

		return false, nil
	})
}
