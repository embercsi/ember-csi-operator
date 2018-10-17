# Ember CSI Operator Catalog Source
Ember CSI Operatort

## Quick Start

### Prerequisites

- Working Operator Lifecycle Manager 
- OpenShift/k8s cluster admin privileges

### Deployment
```
oc project openshift-operator-lifecycle-manager
oc apply -f pre.yml
oc apply -f configmap.yml
oc apply -f catalog-source.yml
```
