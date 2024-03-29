package k8s

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func HandleDeployUpdate(ctx context.Context, rc client.Client, newDeploy *appsv1.Deployment, waitDeadline time.Duration) error {
	var currentDeploy appsv1.Deployment
	err := rc.Get(ctx, types.NamespacedName{Name: newDeploy.Name, Namespace: newDeploy.Namespace}, &currentDeploy)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := rc.Create(ctx, newDeploy); err != nil {
				return fmt.Errorf("cannot create new deployment for app: %s, err: %w", newDeploy.Name, err)
			}
			return waitDeploymentReady(ctx, rc, newDeploy, waitDeadline)
		}
		return fmt.Errorf("cannot get deployment for app: %s err: %w", newDeploy.Name, err)
	}
	newDeploy.Spec.Template.Annotations = MergeAnnotations(currentDeploy.Spec.Template.Annotations, newDeploy.Spec.Template.Annotations)
	//newDeploy.Finalizers = MergeFinalizers(&currentDeploy, nodev1alpha1.NodeFinalizer)
	newDeploy.Status = currentDeploy.Status
	newDeploy.Annotations = MergeAnnotations(currentDeploy.Annotations, newDeploy.Annotations)

	if err := rc.Update(ctx, newDeploy); err != nil {
		return fmt.Errorf("cannot update deployment for app: %s, err: %w", newDeploy.Name, err)
	}

	return waitDeploymentReady(ctx, rc, newDeploy, waitDeadline)
}

func waitDeploymentReady(ctx context.Context, rc client.Client, dep *appsv1.Deployment, deadline time.Duration) error {
	time.Sleep(time.Second * 2)
	return wait.PollUntilContextTimeout(ctx, time.Second*5, deadline, true, func(ctx context.Context) (done bool, err error) {
		var actualDeploy appsv1.Deployment
		if err = rc.Get(ctx, types.NamespacedName{Namespace: dep.Namespace, Name: dep.Name}, &actualDeploy); err != nil {
			return false, fmt.Errorf("cannot fetch actual deployment state: %w", err)
		}
		for _, cond := range actualDeploy.Status.Conditions {
			if cond.Type == appsv1.DeploymentProgressing {
				// https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#complete-deployment
				// Completed status for deployment
				if cond.Reason == "NewReplicaSetAvailable" && cond.Status == "True" {
					return true, nil
				}
				return false, nil
			}
		}
		return false, nil
	})
}
