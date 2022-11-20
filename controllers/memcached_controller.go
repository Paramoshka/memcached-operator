/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	cachev1alpha1 "github.com/Paramoshka/memcached-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MemcachedReconciler reconciles a Memcached object
type MemcachedReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=cache.landomfreedom.ru,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.landomfreedom.ru,resources=memcacheds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.landomfreedom.ru,resources=memcacheds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Memcached object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//logging
	log := ctrl.Log.WithName("debug")
	// Fetch the Memcached instance.
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// Check if the statefullset already exists, if not create a new deployment.
	found := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			// Define and create a new statefullset.
			dep := r.StateFullSet(memcached)
			if err = r.Create(ctx, dep); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, err
		}
	}
	//create service ClusterIP for memcached
	msfound := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: "memcached-service", Namespace: memcached.Namespace}, msfound)
	if err != nil {
		if errors.IsNotFound(err) {
			ms := r.MemcachedService(memcached)
			if err = r.Create(ctx, ms); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, err
		} else {
			return ctrl.Result{}, err
		}
	}
	//check size memcached
	if *found.Spec.Replicas != memcached.Spec.Size {
		found.Spec.Replicas = &memcached.Spec.Size
		err = r.Update(ctx, found)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, err
	}
	// List the pods for this CR's deployment.
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(memcached.Namespace),
		client.MatchingLabels(map[string]string{"app": "memcached"}),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		return ctrl.Result{}, err
	}
	var arrayPodIPs []string
	for i, item := range podList.Items {
		arrayPodIPs = append(arrayPodIPs, item.Status.PodIP+string(":11211"))
		log.Info(string(rune(i)), "pod IP", item.Status.PodIP)
	}
	//Deployment MCRoute
	mcrouteDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: "mcroute-pool-1", Namespace: memcached.Namespace}, mcrouteDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			depMCroute := r.DeploymentMCRoute(memcached, &arrayPodIPs)
			err = r.Create(ctx, depMCroute)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, err
		} else {
			return ctrl.Result{}, err
		}
	}
	//

	return ctrl.Result{}, nil
}

func (r *MemcachedReconciler) StateFullSet(memcached *cachev1alpha1.Memcached) *appsv1.StatefulSet {
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
						Ports: []corev1.ContainerPort{{
							Name:          "memcached-port",
							ContainerPort: 11211,
						}},
						//Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
					}},
				},
			},
		},
	}
	//Garbage Collector
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

func (r *MemcachedReconciler) DeploymentMCRoute(m *cachev1alpha1.Memcached, arrPodIPS *[]string) *appsv1.Deployment {
	replicas := int32(1)
	mc := &mcRoute{}
	mc.Pools.A.Servers = *arrPodIPS
	mc.Route.Type = "OperationSelectorRoute"
	mc.Route.OperationPolicies.Delete = "AllSyncRoute|Pool|A"
	mc.Route.OperationPolicies.Gets = "FailoverRoute|Pool|A"
	mc.Route.OperationPolicies.Get = "FailoverRoute|Pool|A"
	mc.Route.OperationPolicies.Add = "AllInitialRoute|Pool|A"
	mc.Route.OperationPolicies.Set = "AllSyncRoute|Pool|A"

	jsonCommand, _ := json.Marshal(mc)
	stringCommand := "--config-str=" + string(jsonCommand)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mcroute-pool-1",
			Namespace: m.Namespace,
			Labels:    map[string]string{"name": "mcroute"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": "mcroute"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": "mcroute"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "mcroute",
						Image: "mcrouter/mcrouter",
						Ports: []corev1.ContainerPort{{
							Name:          "mcroute-port",
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

//func ServiceMCRoute() *corev1.Service {
//	SMC := &corev1.Service{}
//	return SMC
//}

// SetupWithManager sets up the controller with the Manager.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Complete(r)
}
