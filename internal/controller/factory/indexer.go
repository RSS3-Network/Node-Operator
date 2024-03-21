package factory

import (
	"context"
	"fmt"
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	"github.com/rss3-network/node-operator/internal/controller/factory/k8s"
	"github.com/rss3-network/protocol-go/schema/filter"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/rss3-network/node/config"
)

func CreateOrUpdateIndexer(ctx context.Context, log *zap.Logger, cr *nodev1alpha1.Indexer, rc client.Client) error {
	if err := CreateOrUpdateConfig(ctx, log, cr, rc); err != nil {
		return fmt.Errorf("cannot update relabeling asset for vmagent: %w", err)
	}

	// Check if the statefulSet already exists, if not create a new one
	found := &appsv1.StatefulSet{}
	err := rc.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new statefulSet
		sts, err := statefulSetForIndexer(cr, rc)
		if err != nil {
			log.Error("Failed to define new StatefulSet resource for Indexer", zap.Error(err))
			return err
		}

		log.Info("Creating a new StatefulSet",
			zap.String("namespace", sts.Namespace),
			zap.String("name", sts.Name))
		if err = k8s.HandleSTSUpdate(ctx, rc, sts, 0); err != nil {
			log.Error("Failed to create new StatefulSet",
				zap.Error(err),
				zap.String("namespace", sts.Namespace),
				zap.String("name", sts.Name),
			)
			return err
		}
	} else if err != nil {
		log.Error("Failed to get StatefulSet", zap.Error(err))
		// Let's return the error for the reconciliation be re-trigged again
		return err
	}

	return nil
}

func CreateOrUpdateConfig(ctx context.Context, log *zap.Logger, cr *nodev1alpha1.Indexer, rc client.Client) error {
	// Check if the config already exists, if not create a new one
	found := &corev1.ConfigMap{}
	err := rc.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new config
		cm, err := configmapForIndexer(cr, rc)
		if err != nil {
			return err
		}

		log.Info("Creating a new ConfigMap",
			zap.String("namespace", cm.Namespace),
			zap.String("name", cm.Name))
		if err = k8s.HandleConfigmapUpdate(ctx, rc, cm, 0); err != nil {
			log.Error("Failed to create new ConfigMap",
				zap.Error(err),
				zap.String("namespace", cm.Namespace),
				zap.String("name", cm.Name),
			)
			return err
		}
	} else if err != nil {
		log.Error("Failed to get ConfigMap", zap.Error(err))
		// Let's return the error for the reconciliation be re-trigged again
		return err
	}

	return nil

}

func configmapForIndexer(cr *nodev1alpha1.Indexer, rc client.Client) (*corev1.ConfigMap, error) {
	cr = cr.DeepCopy()

	network, err := filter.NetworkString(cr.Spec.Network)
	if err != nil {
		return nil, err
	}
	worker, err := filter.NameString(cr.Spec.Worker)
	if err != nil {
		return nil, err
	}

	configFile := config.File{
		Environment: "production",
		Node: &config.Node{
			Decentralized: []*config.Module{{
				Network:    network,
				Worker:     worker,
				Endpoint:   cr.Spec.Endpoint,
				Parameters: cr.Spec.Params.Options(),
			}},
		},
	}

	data, err := yaml.Marshal(configFile)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		BinaryData: map[string][]byte{
			"config.yaml": data,
		},
	}, nil
}

// statefulSetForIndexer returns a indexer StatefulSet object
func statefulSetForIndexer(cr *nodev1alpha1.Indexer, rc client.Client) (*appsv1.StatefulSet, error) {
	cr = cr.DeepCopy()
	replicas := int32(1)

	podSpec, err := newPodSpecForIndexer(cr)
	if err != nil {
		return nil, err
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Annotations: cr.Annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: cr.SelectorLabels(),
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: cr.PodLabels(),
				},
				Spec: *podSpec,
			},
		},
	}

	// Set Indexer instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, sts, rc.Scheme()); err != nil {
		return nil, err
	}

	return sts, nil

}

func newPodSpecForIndexer(cr *nodev1alpha1.Indexer) (*corev1.PodSpec, error) {
	// Get the Operand image
	image, err := imageForNode()
	if err != nil {
		return nil, err
	}

	podSpec := &corev1.PodSpec{
		Volumes: []corev1.Volume{{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: cr.Name},
					DefaultMode:          func(i int32) *int32 { return &i }(420),
				},
			},
		}},
		Containers: []corev1.Container{{
			Image:           image,
			Name:            "indexer",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env:             cr.Spec.DatabaseRef.EnvVars(),
			Args: []string{
				"--module=indexer",
				fmt.Sprintf("--indexer.network=%s", cr.Spec.Network),
				fmt.Sprintf("--indexer.worker=%s", cr.Spec.Worker),
				fmt.Sprintf("--indexer.parameters=%s", cr.Spec.Params.Options().String()),
			},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "config",
				MountPath: "/etc/config",
			}},
		}},
	}

	return podSpec, nil
}