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
	"github.com/grafana/ebpf-autoinstrument-operator/pkg/sidecar"
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
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Instrumenter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *InstrumenterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop", "request", req.String())

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
	logger := log.FromContext(ctx)
	logger.Info("onDeletion", "req", req)

	return ctrl.Result{}, nil
}

func (r *InstrumenterReconciler) onCreateUpdate(ctx context.Context, instr *appo11yv1alpha1.Instrumenter) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("onCreateUpdate", "name", instr.Name, "namespace", instr.Namespace)

	podList := corev1.PodList{}
	if err := r.List(ctx, &podList,
		client.InNamespace(instr.Namespace),
		client.HasLabels{instr.Spec.Selector.PortLabel}); err != nil {
		return ctrl.Result{}, fmt.Errorf("reading pods: %w", err)
	}

	logger.V(lvl.Debug).Info("found pods to instrument", "len", len(podList.Items))

	iq := sidecar.InstrumentQuery{PortLabel: instr.Spec.Selector.PortLabel}
	for i := range podList.Items {
		pod := &podList.Items[i]
		podLog := logger.WithValues("podName", pod.Name, "podNamespace", pod.Namespace)
		podLog.V(lvl.Debug).Info("checking if Pod needs to be instrumented")
		mustUpdate, err := sidecar.AddInstrumenter(iq, pod)
		if err != nil {
			return ctrl.Result{}, err
		}
		if mustUpdate {
			// we can't really update. We delete the Pod and let the Mutator Hook to attach the sidecar
			podLog.V(lvl.Debug).Info("deleting Pod to recreate it with an instrumenter sidecar")

			if err := r.Delete(ctx, pod); err != nil {
				return ctrl.Result{}, fmt.Errorf("deleting Pod %s/%s: %w", pod.Namespace, pod.Name, err)
			}
			// Pods belonging to a Service or ReplicaSet will be recreated automatically. Simple Pods
			// needs to be created again
			if len(pod.OwnerReferences) == 0 {
				pod.ResourceVersion = ""
				pod.UID = ""
				pod.Status = corev1.PodStatus{}
				if err := r.Create(ctx, pod); err != nil {
					return ctrl.Result{}, fmt.Errorf("can't recreate Pod %s/%s: %w", pod.Namespace, pod.Name, err)
				}
			}
		}
	}

	return ctrl.Result{}, nil
}
