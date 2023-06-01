# ebpf-autoinstrument-operator

## TO DO

* What happens if two instrumenters target the same set of pods? Avoid it
* AddSidecar
  * don't add it if it already exists with the actual data
  * get service name from (in order): metadata label, owner name, pod label
* If sidecar fails, do not make pod failing