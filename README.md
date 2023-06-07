# ebpf-autoinstrument-operator

## How to run (development purposes)

Run a fresh Kind cluster:

```
kind create cluster
```

Install prerequisites:

```
make install-cert-manager
make install-prometheus # optional, if not using OTEL exporter
```

Rebuild and install the operator

```
export IMG=myuser/ebpf-autoinstrument-operator:dev
make generate manifests docker-build kind-load deploy
```

To undeploy:
```
make undeploy
```