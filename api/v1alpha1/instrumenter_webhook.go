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

package v1alpha1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/grafana/ebpf-autoinstrument-operator/pkg/helper/lvl"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var webhookLog = logf.Log.WithName("pod-sidecar-webhook")

type podSidecarWebHook struct {
	client.Client
}

// SetupWebhookWithManager needs to manually register the webhook (not using the kubebuilder/operator-sdk workflow)
// as it needs to be registered towards a core type that is not registerd as type by the controller.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	webhookLog.Info("registering webhook server")
	return builder.WebhookManagedBy(mgr).
		For(&v1.Pod{}).
		WithDefaulter(&podSidecarWebHook{Client: mgr.GetClient()}).
		Complete()
}

var _ admission.CustomDefaulter = (*podSidecarWebHook)(nil)

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=NoneOnDryRun,groups="",resources=pods,verbs=create;update,versions=v1,name=minstrumenter.kb.io,admissionReviewVersions=v1

func (wh *podSidecarWebHook) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		webhookLog.Error(fmt.Errorf("received object is not a *v1.Pod: %T", obj),
			"this must be a bug in the code. Please contact the developers. Ignoring request")
		return nil
	}
	log := webhookLog.WithValues("podName", pod.Name, "podNamespace", pod.Namespace)
	dbg := log.V(lvl.Debug)

	// Check if there is any instrumenter in the given namespace
	// TODO: find a way to cache it in memory?
	instrumenters := InstrumenterList{}
	if err := wh.List(ctx, &instrumenters, client.InNamespace(pod.Namespace)); err != nil {
		log.Error(err, "requesting instrumenters list. Ignoring request")
		return nil
	}

	dbg.Info("queried instrumenters for that namespace", "len", len(instrumenters.Items))
	// It should never happen that two instrumenters match the same Pod,
	// at the moment, we leave it as an undefined behavior.
	for i := range instrumenters.Items {
		instr := &instrumenters.Items[i]
		dbg.Info("checking if the Pod needs to be instrumented", "instrumenter", instr.Name)
		if InstrumentIfRequired(instr, pod) {
			dbg.Info("pod successfully instrumented")
			return nil
		}
	}

	return nil
}
