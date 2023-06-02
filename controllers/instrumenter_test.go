package controllers

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/grafana/ebpf-autoinstrument-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	timeout  = time.Second * 10
	interval = 50 * time.Millisecond

	defaultNS = "default"
)

var _ = Describe("Instrumenter Controller", Ordered, Serial, func() {

	Context("Instrumeting single Pod", func() {
		It("should add an instrumenter sidecar to that Pod", func() {
			By("Creating target Pod")
			Expect(k8sClient.Create(ctx, &v1.Pod{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "instrumentable-pod",
					Namespace: defaultNS,
					Labels: map[string]string{
						"autoinstrument.open.port": "8080",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  "my-pod-container",
						Image: "foo-image",
					}},
				},
			})).To(Succeed())

			By("Deploying an instrumenter instance")
			Expect(k8sClient.Create(ctx, &v1alpha1.Instrumenter{
				ObjectMeta: controllerruntime.ObjectMeta{
					Name:      "my-instrumenter",
					Namespace: defaultNS,
				},
				Spec: v1alpha1.InstrumenterSpec{
					Selector: v1alpha1.Selector{PortLabel: "autoinstrument.open.port"},
				},
			})).To(Succeed())

			By("waiting to the Pod to be restarted and regenerated")
			Eventually(func() error {
				pod := v1.Pod{}
				if err := k8sClient.Get(ctx,
					types.NamespacedName{Name: "instrumentable-pod", Namespace: defaultNS},
					&pod); err != nil {
					return err
				}
				if len(pod.Spec.Containers) != 2 {
					return fmt.Errorf("expecting to have 2 containers. Got %d", len(pod.Spec.Containers))
				}
				instrum := pod.Spec.Containers[1]
				Expect(instrum.Name).To(Equal("grafana-ebpf-autoinstrumenter"))
				Expect(instrum.Image).To(Equal("grafana/ebpf-autoinstrument:latest"))
				Expect(instrum.Env).To(ContainElement(v1.EnvVar{Name: "OPEN_PORT", Value: "8080"}))
				return nil
			}, timeout, interval).Should(Succeed())
		})
	})
})
