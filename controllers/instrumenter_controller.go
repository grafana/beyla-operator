/*
Copyright 2023 Grafana Labs <hello@grafana.com>.

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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appo11yv1alpha1 "github.com/grafana/ebpf-autoinstrument-operator/api/v1alpha1"
	"github.com/grafana/ebpf-autoinstrument-operator/pkg/helper/lvl"
)

// InstrumenterReconciler reconciles a Instrumenter object
type InstrumenterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=appo11y.grafana.com,resources=instrumenters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appo11y.grafana.com,resources=instrumenters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=appo11y.grafana.com,resources=instrumenters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *InstrumenterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile loop", "request", req)

	instr := appo11yv1alpha1.Instrumenter{}
	if err := r.Get(ctx, req.NamespacedName, &instr); err != nil {
		if errors.IsNotFound(err) {
			return r.onDeletion(ctx, req)
		}
		return ctrl.Result{}, fmt.Errorf("reading instrumenter: %w", err)
	}

	if !instr.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.onDeletion(ctx, req)
	}

	return r.onCreateUpdate(ctx, &instr)
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstrumenterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appo11yv1alpha1.Instrumenter{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func (r *InstrumenterReconciler) onDeletion(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "name", req.Name, "namespace", req.Namespace)
	logger.Info("deleted instrumenter")
	dbg := logger.V(lvl.Debug)
	// Look for all the pods in the NS that are instrumented by the removed Instrumenter
	podList := corev1.PodList{}
	if err := r.List(ctx, &podList,
		client.InNamespace(req.Namespace),
		client.HasLabels{appo11yv1alpha1.InstrumentedLabel}); err != nil {
		return ctrl.Result{Requeue: true}, fmt.Errorf("reading pods: %w", err)
	}
	dbg.Info("going to remove all the pods whose "+appo11yv1alpha1.InstrumentedLabel+" points to the deleted instrumenter",
		"candidatePods", len(podList.Items))
	for i := range podList.Items {
		p := &podList.Items[i]
		if instrumenterName := p.Labels[appo11yv1alpha1.InstrumentedLabel]; instrumenterName == req.Name {
			dbg := dbg.WithValues("podName", p.Name, "podNamespace", p.Namespace)
			dbg.Info("removing Pod")
			if err := r.Delete(ctx, p); err != nil {
				return ctrl.Result{Requeue: true}, fmt.Errorf("deleting pod: %w", err)
			}
			// Pods belonging to a Service or ReplicaSet will be recreated automatically. Simple Pods
			// need to be explicitly recreated
			if len(p.OwnerReferences) == 0 {
				dbg.Info("Recreating pod")
				// Pods belonging to a Service or ReplicaSet will be recreated automatically. Simple Pods
				// need to be created again
				if len(p.OwnerReferences) == 0 {
					dbg.Info("Recreating pod")
					appo11yv1alpha1.RemoveInstrumenter(p)
					p.ResourceVersion = ""
					p.UID = ""
					p.Status = corev1.PodStatus{}
					if err := r.Create(ctx, p); err != nil {
						return ctrl.Result{}, fmt.Errorf("can't recreate Pod %s/%s: %w", p.Namespace, p.Name, err)
					}
				}
			}
		} else {
			dbg.Info("this Pod is instumented by another instrumenter. Skipping",
				"instrumentedBy", instrumenterName, "podName", p.Name, "podNamespace", p.Namespace)
		}
	}

	return ctrl.Result{}, nil
}

func (r *InstrumenterReconciler) onCreateUpdate(ctx context.Context, instr *appo11yv1alpha1.Instrumenter) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "name", instr.Name, "namespace", instr.Namespace)
	dbg := logger.V(lvl.Debug)
	dbg.Info("onCreateUpdate", "spec", instr.Spec)

	podList := corev1.PodList{}
	if err := r.List(ctx, &podList,
		client.InNamespace(instr.Namespace),
		client.HasLabels{instr.Spec.Selector.PortLabel}); err != nil {
		return ctrl.Result{}, fmt.Errorf("reading pods: %w", err)
	}

	dbg.Info("list of pods to instrument", "len", len(podList.Items))

	for i := range podList.Items {
		pod := &podList.Items[i]
		podLog := dbg.WithValues("podName", pod.Name, "podNamespace", pod.Namespace)
		podLog.Info("checking if Pod needs to be instrumented")
		if sidec, ok := appo11yv1alpha1.NeedsInstrumentation(instr, pod); ok {
			podLog.Info("Destroying Pod to recreate it with an instrumenter sidecar")
			if err := r.Delete(ctx, pod); err != nil {
				return ctrl.Result{}, fmt.Errorf("deleting Pod %s/%s: %w", pod.Namespace, pod.Name, err)
			}
			// Pods belonging to a Service or ReplicaSet will be recreated automatically. Simple Pods
			// need to be explicitly recreated
			if len(pod.OwnerReferences) == 0 {
				podLog.Info("Recreating pod")
				pod.ResourceVersion = ""
				pod.UID = ""
				pod.Status = corev1.PodStatus{}
				appo11yv1alpha1.AddInstrumenter(instr.Name, sidec, pod)
				if err := r.Create(ctx, pod); err != nil {
					return ctrl.Result{}, fmt.Errorf("can't recreate Pod %s/%s: %w", pod.Namespace, pod.Name, err)
				}
			}
		}
	}

	return ctrl.Result{}, nil
}
