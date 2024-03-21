package factory

import (
	"context"
	nodev1alpha1 "github.com/rss3-network/node-operator/api/v1alpha1"
	"github.com/rss3-network/node-operator/internal/controller/factory/k8s"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdateHubService(ctx context.Context, log *zap.Logger, cr *nodev1alpha1.Hub, rc client.Client) (*corev1.Service, error) {
	cr = cr.DeepCopy()

	svc, err := serviceForHub(cr, rc)
	if err != nil {
		log.Error("Failed to define new Service resource for Hub", zap.Error(err))
		return nil, err
	}

	return reconcileService(ctx, rc, svc)
}

func CreateOrUpdateHub(ctx context.Context, log *zap.Logger, cr *nodev1alpha1.Hub, rc client.Client) error {
	// Check if the deployment already exists, if not create a new one

	// Define a new deployment
	dep, err := deploymentForHub(cr, rc)
	if err != nil {
		log.Error("Failed to define new Deployment resource for Hub", zap.Error(err))
		return err
	}

	if err = k8s.HandleDeployUpdate(ctx, rc, dep, nodev1alpha1.WaitReadyTimeout); err != nil {
		log.Error("Failed to handle deployment",
			zap.Error(err),
			zap.String("namespace", dep.Namespace),
			zap.String("name", dep.Name),
		)
		return err
	}

	return nil
}

// deploymentForHub returns a Hub Deployment object
func deploymentForHub(
	cr *nodev1alpha1.Hub, rc client.Client) (*appsv1.Deployment, error) {
	cr = cr.DeepCopy()

	podSpec, err := newPodSpecForHub(cr)
	if err != nil {
		return nil, err
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Annotations: cr.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: cr.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: cr.SelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: cr.PodLabels(),
				},
				Spec: *podSpec,
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(cr, dep, rc.Scheme()); err != nil {
		return nil, err
	}
	return dep, nil
}

func newPodSpecForHub(cr *nodev1alpha1.Hub) (*corev1.PodSpec, error) {
	// Get the Operand image
	image, err := imageForNode()
	if err != nil {
		return nil, err
	}

	return &corev1.PodSpec{
		// TODO(user): Uncomment the following code to configure the nodeAffinity expression
		// according to the platforms which are supported by your solution. It is considered
		// best practice to support multiple architectures. build your manager image using the
		// makefile target docker-buildx. Also, you can use docker manifest inspect <image>
		// to check what are the platforms supported.
		// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity
		//Affinity: &corev1.Affinity{
		//	NodeAffinity: &corev1.NodeAffinity{
		//		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
		//			NodeSelectorTerms: []corev1.NodeSelectorTerm{
		//				{
		//					MatchExpressions: []corev1.NodeSelectorRequirement{
		//						{
		//							Key:      "kubernetes.io/arch",
		//							Operator: "In",
		//							Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
		//						},
		//						{
		//							Key:      "kubernetes.io/os",
		//							Operator: "In",
		//							Values:   []string{"linux"},
		//						},
		//					},
		//				},
		//			},
		//		},
		//	},
		//},
		//SecurityContext: &corev1.PodSecurityContext{
		//	RunAsNonRoot: &[]bool{true}[0],
		//	// IMPORTANT: seccomProfile was introduced with Kubernetes 1.19
		//	// If you are looking for to produce solutions to be supported
		//	// on lower versions you must remove this option.
		//	SeccompProfile: &corev1.SeccompProfile{
		//		Type: corev1.SeccompProfileTypeRuntimeDefault,
		//	},
		//},
		Containers: []corev1.Container{{
			Image:           image,
			Name:            "hub",
			ImagePullPolicy: corev1.PullIfNotPresent,
			// Ensure restrictive context for the container
			// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
			//SecurityContext: &corev1.SecurityContext{
			//	// WARNING: Ensure that the image used defines an UserID in the Dockerfile
			//	// otherwise the Pod will not run and will fail with "container has runAsNonRoot and image has non-numeric user"".
			//	// If you want your workloads admitted in namespaces enforced with the restricted mode in OpenShift/OKD vendors
			//	// then, you MUST ensure that the Dockerfile defines a User ID OR you MUST leave the "RunAsNonRoot" and
			//	// "RunAsUser" fields empty.
			//	RunAsNonRoot: &[]bool{true}[0],
			//	// The hub image does not use a non-zero numeric user as the default user.
			//	// Due to RunAsNonRoot field being set to true, we need to force the user in the
			//	// container to a non-zero numeric user. We do this using the RunAsUser field.
			//	// However, if you are looking to provide solution for K8s vendors like OpenShift
			//	// be aware that you cannot run under its restricted-v2 SCC if you set this value.
			//	RunAsUser:                &[]int64{1001}[0],
			//	AllowPrivilegeEscalation: &[]bool{false}[0],
			//	Capabilities: &corev1.Capabilities{
			//		Add: []corev1.Capability{
			//			"NET_BIND_SERVICE",
			//		},
			//	},
			//},
			Ports: []corev1.ContainerPort{{
				ContainerPort: 80,
				Name:          "http",
			}},
			Env:  cr.Spec.DatabaseRef.EnvVars(),
			Args: []string{"--module=hub"},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 80,
						},
					},
				},
				TimeoutSeconds:   3,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 80,
						},
					},
				},
				TimeoutSeconds:   3,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
		}},
	}, nil
}

// serviceForHub returns a Hub Deployment object
func serviceForHub(
	hub *nodev1alpha1.Hub, rc client.Client) (*corev1.Service, error) {

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        hub.Name,
			Namespace:   hub.Namespace,
			Annotations: hub.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{Name: "http", Protocol: corev1.ProtocolTCP, Port: 80, TargetPort: intstr.FromInt32(80)},
			},
			Selector: hub.SelectorLabels(),
		},
	}

	// Set the ownerRef for the Service
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(hub, svc, rc.Scheme()); err != nil {
		return nil, err
	}
	return svc, nil
}
