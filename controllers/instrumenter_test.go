package controllers

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/grafana/ebpf-autoinstrument-operator/pkg/helper"
	"github.com/mariomac/gostream/stream"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"

	"github.com/grafana/ebpf-autoinstrument-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 10
	interval = 50 * time.Millisecond

	defaultNS = "default"
)

var _ = Describe("Instrumenter Controller", Ordered, Serial, func() {
	singleTestPodTemplate := v1.Pod{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      "instrumentable-pod",
			Namespace: defaultNS,
			Labels: map[string]string{
				"grafana.com/instrument-port": "8080",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "my-pod-container",
				Image: "foo-image",
			}},
		},
	}
	instrumenterTemplate := v1alpha1.Instrumenter{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      "my-instrumenter",
			Namespace: defaultNS,
		},
		Spec: v1alpha1.InstrumenterSpec{
			Selector: v1alpha1.Selector{PortLabel: "grafana.com/instrument-port"},
		},
	}
	Context("Instrumeting single Pod", func() {
		singleTestPod, instrumenter := singleTestPodTemplate, instrumenterTemplate
		It("should add an instrumenter sidecar to that Pod", func() {
			By("Creating target Pod")
			Expect(k8sClient.Create(ctx, &singleTestPod)).To(Succeed())

			By("Deploying an instrumenter instance")
			Expect(k8sClient.Create(ctx, &instrumenter)).To(Succeed())

			By("waiting to the Pod to be restarted and regenerated")
			Eventually(func() error {
				pod := v1.Pod{}
				if err := k8sClient.Get(ctx,
					types.NamespacedName{Name: "instrumentable-pod", Namespace: defaultNS},
					&pod); err != nil {
					return err
				}
				return assertPod(&pod)
			}, timeout, interval).Should(Succeed())
		})
		It("should properly remove the created resources", func() {
			Expect(k8sClient.Delete(ctx, &singleTestPod)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, &instrumenter)).Should(Succeed())
			expectNotFound(&instrumenter)
		})
	})

	Context("Uninstrumenting Pod after Instrumenter removal", func() {
		singleTestPod, instrumenter := singleTestPodTemplate, instrumenterTemplate
		It("Prerequisite: running an instrumented Pod, and an instrumenter", func() {
			Expect(k8sClient.Create(ctx, &singleTestPod)).To(Succeed())
			Expect(k8sClient.Create(ctx, &instrumenter)).To(Succeed())
			Eventually(func() error {
				pod := v1.Pod{}
				if err := k8sClient.Get(ctx,
					types.NamespacedName{Name: "instrumentable-pod", Namespace: defaultNS},
					&pod); err != nil {
					return err
				}
				return assertPod(&pod)
			}, timeout, interval).Should(Succeed())
		})

		It("should remove instrumenter sidecar", func() {
			By("Removing instrumenter")
			Expect(k8sClient.Delete(ctx, &instrumenter)).To(Succeed())

			By("waiting to the Pod to be restarted and regenerated")
			Eventually(func() error {
				pod := v1.Pod{}
				if err := k8sClient.Get(ctx,
					types.NamespacedName{Name: "instrumentable-pod", Namespace: defaultNS},
					&pod); err != nil {
					return err
				}
				if len(pod.Spec.Containers) > 1 {
					return fmt.Errorf("expecting Pod to have a single container. Has %d", len(pod.Spec.Containers))
				}
				if len(pod.Labels) > 0 && pod.Labels[v1alpha1.InstrumentedLabel] != "" {
					return fmt.Errorf("unexpected label %s: %q",
						v1alpha1.InstrumentedLabel, pod.Labels[v1alpha1.InstrumentedLabel])
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})
		It("should properly remove the created resources", func() {
			Expect(k8sClient.Delete(ctx, &singleTestPod)).Should(Succeed())
		})
	})

	Context("Ignoring pods that aren't labeled", func() {
		ignorablePod, instrumenter := singleTestPodTemplate, instrumenterTemplate
		ignorablePod.Labels = nil
		It("should NOT add an instrumenter sidecar to that Pod", func() {
			By("Creating ignorable Pod")
			Expect(k8sClient.Create(ctx, &ignorablePod)).To(Succeed())

			By("Deploying an instrumenter instance")
			Expect(k8sClient.Create(ctx, &instrumenter)).To(Succeed())

			Consistently(func() interface{} {
				pod := &v1.Pod{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ignorablePod.Name,
					Namespace: ignorablePod.Namespace,
				}, pod); err != nil {
					return err
				}
				return pod.Spec.Containers
			}).Should(HaveLen(1))
		})
		It("should properly remove the created resources", func() {
			Expect(k8sClient.Delete(ctx, &ignorablePod)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, &instrumenter)).Should(Succeed())
			expectNotFound(&instrumenter)
		})
	})

	Context("Ignoring pods in another namespace", func() {
		ignorablePod, instrumenter := singleTestPodTemplate, instrumenterTemplate
		ignorablePod.Namespace = "chachacha"
		It("should NOT add an instrumenter sidecar to that Pod", func() {
			By("Creating another namespace")
			Expect(k8sClient.Create(ctx, &v1.Namespace{ObjectMeta: controllerruntime.ObjectMeta{
				Name: ignorablePod.Namespace}})).To(Succeed())

			By("Creating ignorable Pod")
			Expect(k8sClient.Create(ctx, &ignorablePod)).To(Succeed())

			By("Deploying an instrumenter instance")
			Expect(k8sClient.Create(ctx, &instrumenter)).To(Succeed())

			Consistently(func() interface{} {
				pod := &v1.Pod{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ignorablePod.Name,
					Namespace: ignorablePod.Namespace,
				}, pod); err != nil {
					return err
				}
				return pod.Spec.Containers
			}).Should(HaveLen(1))
		})
		It("should properly remove the created resources", func() {
			Expect(k8sClient.Delete(ctx, &ignorablePod)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, &instrumenter)).Should(Succeed())
			expectNotFound(&instrumenter)
		})
	})

	Context("Instrumenting ReplicaSets", func() {
		replicaSet := appsv1.ReplicaSet{
			ObjectMeta: controllerruntime.ObjectMeta{
				Name:      "my-rs",
				Namespace: defaultNS,
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: helper.Ptr[int32](3),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
				Template: v1.PodTemplateSpec{
					ObjectMeta: controllerruntime.ObjectMeta{Labels: map[string]string{
						"app":                         "test",
						"grafana.com/instrument-port": "8080",
					}},
					Spec: singleTestPodTemplate.Spec,
				},
			},
		}
		instrumenter := instrumenterTemplate
		It("should add an instrumenter sidecar to the labeled Pods of a given ReplicaSet", func() {

			Skip("TO DO: move this whole Context to an e2e test, as EnvTest create Pods after deploying a ReplicaSet")

			By("creating a ReplicaSet")
			Expect(k8sClient.Create(ctx, &replicaSet)).To(Succeed())

			By("Deploying an instrumenter instance")
			Expect(k8sClient.Create(ctx, &instrumenter)).To(Succeed())

			By("waiting to all ReplicaSet Pods to be restarted and regenerated")
			Eventually(func() error {
				podsList := v1.PodList{}
				if err := k8sClient.List(ctx,
					&podsList,
					client.InNamespace(defaultNS),
				); err != nil {
					return err
				}
				rsPods := stream.OfSlice(podsList.Items).Filter(func(pod v1.Pod) bool {
					return len(pod.OwnerReferences) > 0 && pod.OwnerReferences[0].Name == "my-rs"
				}).ToSlice()
				if len(rsPods) != 3 {
					return fmt.Errorf("expecting 3 matching pods. Got %d", len(rsPods))
				}
				for _, pod := range rsPods {
					if err := assertPod(&pod); err != nil {
						return err
					}
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})
	})
})

func assertPod(pod *v1.Pod) error {
	if len(pod.Spec.Containers) != 2 {
		return fmt.Errorf("expecting to have 2 containers. Got %d", len(pod.Spec.Containers))
	}
	instrum := pod.Spec.Containers[1]
	if instrum.Name != "grafana-ebpf-autoinstrumenter" {
		return fmt.Errorf("invalid name: %s", instrum.Name)
	}
	if instrum.Image != "grafana/ebpf-autoinstrument:latest" {
		return fmt.Errorf("invalid name: %s", instrum.Name)
	}
	if pod.Spec.ShareProcessNamespace == nil || *pod.Spec.ShareProcessNamespace != true {
		return fmt.Errorf("expecting ShareProcessNamespace=true. Got %v", pod.Spec.ShareProcessNamespace)
	}
	return assertEnvContains(instrum.Env, map[string]string{
		"OPEN_PORT":               "8080",
		"SERVICE_NAME":            "instrumentable-pod",
		"SERVICE_NAMESPACE":       "default",
		"PROMETHEUS_SERVICE_NAME": "instrumentable-pod",
		"PROMETHEUS_PORT":         "9102",
		"PROMETHEUS_PATH":         "/metrics",
	})
}

func assertEnvContains(slice []v1.EnvVar, values map[string]string) error {
	env := map[string]string{}
	for _, e := range slice {
		env[e.Name] = e.Value
	}
	for k, v := range values {
		if env[k] != v {
			return fmt.Errorf("expecting env %v to contain %v=%v. Got: %v",
				env, k, v, env[k])
		}
	}
	return nil
}

func expectNotFound(obj client.Object) {
	Eventually(func() error {
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		}, obj)
		if err == nil || !errors.IsNotFound(err) {
			return fmt.Errorf("expecting Not Found error. Got: %w", err)
		}
		return nil
	}, timeout, interval).Should(Succeed())
}
