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
	SetUpdateStatusTo(ctx context.Context, r client.Client, status nodev1alpha1.UpdateStatus, maybeReason error) error
}

func reconcileWithDiff(ctx context.Context, rc client.Client, obj ObjectWithSpecPatch, cb func() (ctrl.Result, error)) (ctrl.Result, error) {
	specDiff := obj.SpecDiff()
	if specDiff {
		if err := obj.SetUpdateStatusTo(ctx, rc, nodev1alpha1.UpdateStatusExpanding, nil); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update object status: %w", err)
		}
	}
	result, err := cb()

	if err != nil {
		if updateErr := obj.SetUpdateStatusTo(ctx, rc, nodev1alpha1.UpdateStatusFailed, err); updateErr != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update object status: %q, origin err: %w", updateErr, err)
		}
		return result, fmt.Errorf("callback error: %w", err)
	}

	if err = obj.SetUpdateStatusTo(ctx, rc, nodev1alpha1.UpdateStatusOperational, nil); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update object status: %w", err)
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
