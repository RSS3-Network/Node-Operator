package factory

import (
	"context"
	"fmt"
	"github.com/rss3-network/node-operator/internal/controller/factory/k8s"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileService(ctx context.Context, rc client.Client, newSvc *v1.Service) (*v1.Service, error) {
	handleDelete := func(svc *v1.Service) error {
		if err := RemoveFinalizer(ctx, rc, svc); err != nil {
			return err
		}
		return rc.Delete(ctx, svc)
	}

	currentSvc := &v1.Service{}
	err := rc.Get(ctx, client.ObjectKey{Name: newSvc.Name, Namespace: newSvc.Namespace}, currentSvc)
	if err != nil {
		if errors.IsNotFound(err) {
			err := rc.Create(ctx, newSvc)
			if err != nil {
				return nil, fmt.Errorf("cannot create new service: %w", err)
			}
			return newSvc, nil
		}
	}

	if newSvc.Spec.Type != currentSvc.Spec.Type {
		if err = handleDelete(currentSvc); err != nil {
			return nil, err
		}
		return reconcileService(ctx, rc, newSvc)
	}

	if newSvc.Spec.ClusterIP != "" &&
		newSvc.Spec.ClusterIP != "None" &&
		newSvc.Spec.ClusterIP != currentSvc.Spec.ClusterIP {
		if err = handleDelete(currentSvc); err != nil {
			return nil, err
		}
		return reconcileService(ctx, rc, newSvc)
	}

	if newSvc.Spec.ClusterIP == "None" && currentSvc.Spec.ClusterIP != "None" {
		if err = handleDelete(currentSvc); err != nil {
			return nil, err
		}
		return reconcileService(ctx, rc, newSvc)
	}

	if newSvc.Spec.ClusterIP == "" && currentSvc.Spec.ClusterIP == "None" {
		if err = handleDelete(currentSvc); err != nil {
			return nil, err
		}
		return reconcileService(ctx, rc, newSvc)
	}

	if newSvc.Spec.ClusterIP != "None" {
		newSvc.Spec.ClusterIP = currentSvc.Spec.ClusterIP
	}

	if newSvc.Spec.Type == currentSvc.Spec.Type {
		for i := range currentSvc.Spec.Ports {
			port := currentSvc.Spec.Ports[i]
			for j := range newSvc.Spec.Ports {
				newPort := &newSvc.Spec.Ports[j]
				if port.Name == newPort.Name && newPort.NodePort == 0 {
					newPort.NodePort = port.NodePort
					break
				}
			}
		}
	}

	if currentSvc.ResourceVersion != "" {
		newSvc.ResourceVersion = currentSvc.ResourceVersion
	}
	newSvc.Annotations = k8s.MergeAnnotations(currentSvc.Annotations, newSvc.Annotations)
	//newSvc.Finalizers = k8s.MergeFinalizers(currentSvc, nodev1alpha1.NodeFinalizer)

	err = rc.Update(ctx, newSvc)
	if err != nil {
		return nil, fmt.Errorf("cannot update service: %w", err)
	}
	return newSvc, nil
}
