# kubervisor

## Build Status

[![Build Status](https://travis-ci.org/AmadeusITGroup/kubervisor.svg?branch=master)](https://travis-ci.org/AmadeusITGroup/kubervisor)
[![Go Report Card](https://goreportcard.com/badge/github.com/amadeusitgroup/kubervisor)](https://goreportcard.com/report/github.com/amadeusitgroup/kubervisor)
[![codecov](https://codecov.io/gh/amadeusitgroup/kubervisor/branch/master/graph/badge.svg)](https://codecov.io/gh/amadeusitgroup/kubervisor)


## Presentation

The Kubervisor allow you to control which pods should receive traffic or not based on anomaly detection.It is a new kind of health check system.

Unlike readiness probe, the Kubervisor can be configured to remove pods from endpoints based on a global view of the health of the pod fleet. This guarantees that if all pods (or a majority) are under SLA, the system stability is not getting worse because of pod local decisions to "eliminate" itself.

Unlike a service mesh circuit breaker, the Kubervisor can act as a circuit breaker triggered by servers internal KPIs. The anomaly detection can be based on analysis done on external data source such as Prometheus. It allows to easily build complex analysis by leveraging external system capabilities such as PromQL in the case of Prometheus.

Kubervisor comes with its own resource (CRD) to configure the system:

- define the service to monitor
- define the anomaly detection mechanism and configure it
- define the grace period and retry policies
- activation/deactivation/dryrun operation
- display the current health check of the service

Presentation done during the KubeCon Europe 2018: https://youtu.be/HIB_haT1z5M


## architecture

![architecture diagram][diagram1]

[diagram1]: ./docs/imgs/diagram1.png

- The **KubervisorService** is the CRD (Kubernetes Custom Resource Definition). It is used to configure the kubervisor for a given Service. It also contains the status for the health of the service.
- The **Controller** reads the **KubervisorService** to configure **Breaker** and **Activator** workers for the given service. It also monitors the service changes to adapt the configuration of the system. It also monitors the pods to build a cache for all the workers and to compute the health status of each service under control of the Kubervisor. The health of the serivce is persisted inside the status of the associated **BreakConfig*
- The **Breaker** is in charge of invoking the configured **anomaly detection**. Ensuring that it is not going bellow defined threshold or ratio, the **Breaker** will relabel some pods to prevent them to receive traffic.
- The **Activator** is in charge of restablishing traffic on pods after the defined period of inactivity (equivalent to open state in a circuit breaker pattern). Depending on the configured policy and the numbers of retries performed on a pod, the **Activator** can decide to kill the pod or put it in *pause* (out of traffic forever) for further investigation.
- The **Anomaly detector** part (all the blue part in the diagram) is where the data analysis is really performed. Depending on the KPI that you are working on (discrete value, continuous value) or the type of anomaly (ratio, threshold, trend ...) you can select an integrated implementation or delegate the to an external system that would return the list of pods that are out of policy. The proposed internal implementations used data from Prometheus.

more information in the developper [documentation page](./docs/developper_docs.md)

## System Operations

### Admin side

#### CRD

When the ```Kubervisor controller``` starts it register automatically the ```kubervisorservices.kubervisor.k8s.io``` CRD.

#### Scope

The Kubervisor is an operator that can run in a dedicated namespace and cover the resource of that namespace only, or as a global operator taking action in all namespace that it had been granted access. The used service account will determine the scope on which the Kubervisor will work.

It requires to be given the following roles:

- get:          pod, service
- list:         pod, service
- update:       pod, service
- watch:        pod, service
- delete:       pod

#### Deployment

An easy way to install the ```Kubervisor``` controller, in your Kubernetes cluster, is to use the ```helm chart``` present in this repository

```console
$ helm intall -n kubervisor --wait charts/kubervisor
NAME:   kubervisor
LAST DEPLOYED: Fri Apr 27 21:35:03 2018
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1beta2/Deployment
NAME        DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
kubervisor  1        1        1           0          3s

==> v1beta1/ClusterRole
NAME        AGE
kubervisor  3s

==> v1beta1/ClusterRoleBinding
NAME        AGE
kubervisor  2s

==> v1/ServiceAccount
NAME        SECRETS  AGE
kubervisor  1        2s
```

#### kubectl plugin

kubervisor provides a kubectl plugin in order to show in a nice way the KubervisorService status information

To install the plugin juste run: ```make plugin```

To run the plugin:

- ```kubectl plugin kubervisor``` will list all the ```KubervisorService``` present in the current namespace.
- ```kubectl plugin kubervisor -k <kubervisorservice-name>``` will display only the ```KubervisorService``` corresponding to the ```-k``` argument.

### User side

To configure the system a user would have to complete the following steps:

- Create the **KubervisorService** CRD in the namespace of the associated service
- - Select the service (by name)
- - Define the BreakConfiguration to configure the Anomaly Detection mechanism
- - Configure the Activator
- Once the **CRD** status is **Ready** activate the system by adding the following label in the Selector of the service: **kubervisor/traffic=yes**
- - TODO: Alternativelly use the command kubectl .....

To deactivate any effect of the Kubervisor for a given service, simply delete from the Selector the label with key **kubervisor/traffic**

If you know that some pods are going to be under control of the Kubervisor, it is advised to directly add a label **kubervisor/traffic=yes** inside the pod template. This label must not be part of template only, not the selector!

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapplication
  labels:
    app: myapplication
spec:
  replicas: 3
  selector:
    matchLabels:                       # <-- kubervisor labels must not appear in the selector!
      app: myapplication
  template:
    metadata:
      labels:
        app: myapplication
        kubervisor/traffic: yes        # <-- add label for kubervisor
    spec:
      containers:
      - name: myapp
        image: myapp:1.7.9
        ports:
        - containerPort: 80
```

This label is automattically added by the controller on the pods if it is missing. But this happens once the resources are synchronized in the controller (every couple of seconds in theory) and of course if the controller is running. Having the label already preset in the pod template prevent so corner case in case the controller is missbehaving or absent.
