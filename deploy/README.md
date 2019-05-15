This directory contains files required to deploy the Ember-CSI-Operator.

`00-pre.yaml` contains all the prerequisite entities to exist for the deployment of the Operator. These include:
 - Namespace
 - RBAC Rules
 - Service Accounts

01-scc.yaml is a deployment file which is required for OpenShift clusters.
02-operator.yaml is the deployment file for the operator itself.

The examples directory contains example YAML files for some of Ember-CSI drivers.
