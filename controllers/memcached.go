package controllers

import (
	cachev1alpha1 "github.com/Paramoshka/memcached-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *MemcachedReconciler) StateFullSet(memcached *cachev1alpha1.Memcached) *appsv1.StatefulSet {
	readnessPort := intstr.FromString("memcached-port")
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
			Labels:    map[string]string{"app": "memcached"},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &memcached.Spec.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "memcached"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "memcached"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: memcached.Spec.Image,
						Name:  "memcached",
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: readnessPort,
								},
							},
							InitialDelaySeconds:           10,
							PeriodSeconds:                 10,
							SuccessThreshold:              1,
							FailureThreshold:              3,
							TerminationGracePeriodSeconds: nil,
						},
						Ports: []corev1.ContainerPort{{
							Name:          "memcached-port",
							ContainerPort: 11211,
						}},
						// Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
					}},
				},
			},
		},
	}
	// Garbage Collector
	err := controllerutil.SetControllerReference(memcached, ss, r.Scheme)
	if err != nil {
		return nil
	}
	return ss
}

func (r *MemcachedReconciler) MemcachedService(memcached *cachev1alpha1.Memcached) *corev1.Service {
	targetPort := intstr.FromString("memcached-port")

	ServiceMemcached := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "memcached-service",
			Namespace: memcached.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name:       "memcached-port",
				Protocol:   corev1.ProtocolTCP,
				Port:       11211,
				TargetPort: targetPort,
			}},
			Selector: map[string]string{"app": "memcached"},
		},
	}

	return ServiceMemcached
}
