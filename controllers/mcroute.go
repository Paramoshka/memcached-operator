package controllers

import (
	"context"
	cachev1alpha1 "github.com/Paramoshka/memcached-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *MemcachedReconciler) DeploymentMCRoute(m *cachev1alpha1.Memcached, route *mcRoute) *appsv1.Deployment {
	//log := ctrl.Log.WithName("debug")
	jsonCommand, _ := json.Marshal(route)
	stringCommand := "--config-str=" + string(jsonCommand)
	//log.Info("command: ", stringCommand)
	replicas := int32(1)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mcrouter-pool-1",
			Namespace: m.Namespace,
			Labels:    map[string]string{"name": "mcrouter", "app": "mcrouter"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "mcrouter", "app": "mcrouter"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": "mcrouter", "app": "mcrouter"},
				},
				Spec: corev1.PodSpec{
					//InitContainers: []corev1.Container{{
					//	Name:    "wait",
					//	Image:   "busybox",
					//	Command: []string{"/bin/bash", "-c", "--"},
					//	Args:    []string{"while true; do sleep 30; done;"},
					//}},
					Containers: []corev1.Container{{
						Name:  "mcroute",
						Image: "mcrouter/mcrouter",
						Ports: []corev1.ContainerPort{{
							Name:          "mcrouter-port",
							ContainerPort: 11211,
						}},
						Command: []string{"mcrouter", "-p", "11211", stringCommand},
					}},
				},
			},
		},
	}
	err := controllerutil.SetControllerReference(m, dep, r.Scheme)
	if err != nil {
		return nil
	}

	return dep
}

func (r *MemcachedReconciler) McrouterService(m *cachev1alpha1.Memcached) *corev1.Service {
	targetPort := intstr.FromString("mcrouter-port")
	MS := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mcrouter-a",
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name:       "mcrouter-port",
				TargetPort: targetPort,
				Port:       11211,
				Protocol:   corev1.ProtocolTCP,
			}},
			Selector: map[string]string{"app": "mcrouter"},
		},
	}
	return MS
}

// add MCRoute
type mcRoute struct {
	Pools struct {
		A struct {
			Servers []string `json:"servers"`
		} `json:"A"`
	} `json:"pools"`
	Route struct {
		Type              string `json:"type"`
		OperationPolicies struct {
			Delete string `json:"delete"`
			Get    string `json:"get"`
			Gets   string `json:"gets"`
			Set    string `json:"set"`
			Add    string `json:"add"`
		} `json:"operation_policies"`
	} `json:"route"`
}

func (r *MemcachedReconciler) updateCommand(ctx context.Context, m *cachev1alpha1.Memcached) *mcRoute {
	route := &mcRoute{}
	log := ctrl.Log.WithName("debug")
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		client.InNamespace(m.Namespace),
		client.MatchingLabels(map[string]string{"app": "memcached"}),
	}
	err := r.List(ctx, podList, opts...)
	if err != nil {
		log.Info("Error get list pods")
	}
	for _, item := range podList.Items {
		//log.Info("Pod IP:", item.Status.PodIP)
		route.Pools.A.Servers = append(route.Pools.A.Servers, item.Status.PodIP)
	}

	route.Route.Type = "OperationSelectorRoute"
	route.Route.OperationPolicies.Delete = "AllSyncRoute|Pool|A"
	route.Route.OperationPolicies.Gets = "FailoverRoute|Pool|A"
	route.Route.OperationPolicies.Get = "FailoverRoute|Pool|A"
	route.Route.OperationPolicies.Add = "AllInitialRoute|Pool|A"
	route.Route.OperationPolicies.Set = "AllSyncRoute|Pool|A" //need create constructor !!!

	return route
}
