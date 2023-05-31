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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var webhookLog = logf.Log.WithName("pod-sidecar-webhook")

type PodSidecarWebhook v1.Pod

// SetupWebhookWithManager needs to manually register the webhook (not using the kubebuilder/operator-sdk workflow)
// as it needs to be registered towards a core type that is not registerd as type by the controller.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	webhookLog.Info("registering webhook server")
	return builder.WebhookManagedBy(mgr).
		For(&v1.Pod{}).
		WithDefaulter(&PodSidecarWebhook{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=NoneOnDryRun,groups="",resources=pods,verbs=create;update,versions=v1,name=minstrumenter.kb.io,admissionReviewVersions=v1

func (in *PodSidecarWebhook) Default(ctx context.Context, obj runtime.Object) error {
	webhookLog.Info("Defaulting stuff", "obj", fmt.Sprintf("%#v", obj))
	return nil
}

//var _ webhook.Defaulter = &PodSidecarWebhook{}

//// Default implements webhook.Defaulter so a webhook will be registered for the type
//func (r *PodSidecarWebhook) Default() {
//	webhookLog.Info("default", "name", r.Name)
//
//	// Handle NoneOnDryRun
//	// TODO(user): fill in your defaulting logic.
//}
//
//func (r *PodSidecarWebhook) DeepCopyObject() runtime.Object {
//	return (*v1.Pod)(r).DeepCopyObject()
//}
