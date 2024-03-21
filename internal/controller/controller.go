package controller

import (
	"context"
	"fmt"
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectWithSpecPatch interface {
	client.Object
	SpecDiff() bool
	PatchApplyAnnotations() (client.Patch, error)
	SetStatusCondition(ctx context.Context, r client.Client, condition nodev1alpha1.UpdateStatus, reason error) error
}

func reconcileWithDiff(ctx context.Context, rc client.Client, obj ObjectWithSpecPatch, cb func() (ctrl.Result, error)) (ctrl.Result, error) {
	specDiff := obj.SpecDiff()
	if specDiff {
		if err := obj.SetStatusCondition(ctx, rc, nodev1alpha1.UpdateStatusPending, nil); err != nil {
			return ctrl.Result{}, fmt.Errorf("cannot update status for cluster: %w", err)
		}
	}
	result, err := cb()

	if err != nil {
		if err := obj.SetStatusCondition(ctx, rc, nodev1alpha1.UpdateStatusFailed, err); err != nil {
			return ctrl.Result{}, fmt.Errorf("cannot update status for cluster: %w", err)
		}
		return result, fmt.Errorf("callback error: %w", err)
	}

	if specDiff {
		patch, err := obj.PatchApplyAnnotations()
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("cannot parse last applied spec for cluster: %w", err)
		}
		if err := rc.Patch(ctx, obj, patch); err != nil {
			return ctrl.Result{}, fmt.Errorf("cannot update cluster with last applied spec: %w", err)
		}
	}

	return result, nil
}
