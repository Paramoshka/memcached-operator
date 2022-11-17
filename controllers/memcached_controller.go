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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	//Need add check for count replicas !!!
	//todo

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

// labelsForApp creates a simple set of labels for Memcached.
func labelsForApp(name string) map[string]string {
	return map[string]string{"cr_name": name, "app": "memcached"}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Complete(r)
}
