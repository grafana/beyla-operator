package sidecar

import (
	"fmt"
	"strconv"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/grafana/ebpf-autoinstrument-operator/pkg/helper"

	v1 "k8s.io/api/core/v1"
)

// TODO: user-overridable
const (
	instrumenterName            = "grafana-ebpf-autoinstrumenter"
	instrumenterImage           = "grafana/ebpf-autoinstrument:latest"
	instrumenterImagePullPolicy = "Always"
)

var log = logf.Log.WithName("sidecar-instrumenter")

type InstrumentQuery struct {
	PortLabel string
}

func AddInstrumenter(iq InstrumentQuery, dst *v1.Pod) (bool, error) {
	// TODO: Add OWNER to Pod
	// TODO: return if it must be updated

	sidecar, err := buildSidecar(&iq, dst)
	if err != nil {
		return false, fmt.Errorf("building Pod sidecar: %w", err)
	}

	// TODO: add it only if it is not yet in the pod or it must change

	dst.Spec.Containers = append(dst.Spec.Containers, *sidecar)
	return true, nil
}

func buildSidecar(iq *InstrumentQuery, dst *v1.Pod) (*v1.Container, error) {
	lbls := dst.ObjectMeta.Labels
	log.Info("labels", "labels", lbls, "query", iq)
	port, err := strconv.Atoi(lbls[iq.PortLabel])
	if err != nil {
		return nil, fmt.Errorf("can't convert %s value %q to integer: %w", iq.PortLabel, port, err)
	}
	// TODO: do not make pod failing if sidecar fails, just report it in the Instrumenter status
	sidecar := v1.Container{
		Name:            instrumenterName,
		Image:           instrumenterImage,
		ImagePullPolicy: instrumenterImagePullPolicy,
		// TODO: capabilities by default, or privileged only if user requests for it
		SecurityContext: &v1.SecurityContext{
			Privileged: helper.Ptr(true),
			RunAsUser:  helper.Ptr(int64(0)),
		},
		Env: []v1.EnvVar{{
			Name:  "SERVICE_NAME",
			Value: dst.Name,
		}, {
			Name:  "PRINT_TRACES",
			Value: "true",
		}, {
			Name:  "OPEN_PORT",
			Value: lbls[iq.PortLabel],
		}},
	}
	// TODO: add prometheus and scrape labels
	return &sidecar, nil
}
