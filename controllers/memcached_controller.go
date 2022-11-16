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
	// = log.FromContext(ctx)

	log := r.Log.WithValues("memcached", req.NamespacedName)
	//check memcached
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)
	if err != nil {
		log.Error(err, "failed get memcached!")
	}
	ss := appsv1.StatefulSet{}
	//check memcached
	err = r.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, &ss)
	if err != nil && errors.IsNotFound(err) {
		memcachedStateFullSet := r.StateFullSet(memcached)
		log.Info("Create memcached StateFullSet")
		err = r.Create(ctx, memcachedStateFullSet)
		if err != nil {
			log.Error(err, "Failed create memcached statefull set!")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	} else if err != nil {
		log.Error(err, "Failed get statefullset memcached")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *MemcachedReconciler) StateFullSet(memcached *cachev1alpha1.Memcached) *appsv1.StatefulSet {
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					//todo
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: memcached.Spec.Image,
						Name:  "memcached",
					}},
				},
			},
		},
	}
	return ss
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Complete(r)
}
