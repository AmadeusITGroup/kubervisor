# podkubervisor

The PodKubervisor allow you to control which pods should receive traffic or not based on anomaly detection.It is a new kind of health check system.

Unlike readyness probe, the PodKubervisor can be configured to remove pods from endpoints based on a global view of the health of the pod fleet.
This guarantees that if all pods (or a majority) are under SLA, the system stability is not getting worse because of pod local decisions to "eliminate" itself.


Unlike a service mesh circuit breaker, the PodKubervisor can act as a circuit breaker triggered by servers internal KPIs.
The anomaly detection can be based on analisys done on external data source such as prometheus. It allows to easy build complexe analysis by leveraging external system capabilities such as PromQL in the case of Prometheus.

PodKubervisor comes with its own resource (CRD) to configure the system:
- define the service to monitor
- define the anomaly detection mechanism and configure it
- define the grace period and retry policies
- activation/deactivation/dryrun operation
- display the current health check of the service


## architecture

![architecture diagram][diagram1]

[diagram1]: ./docs/imgs/diagram1.png

- The **BreakerConfig** is the CRD (Kubernetes Custom Resource Definition). It is used to configure the kubervisor for a given Service. It also contains the status for the health of the service.
- The **Controller** reads the **BreakerConfig** to configure **Breaker** and **Activator** workers for the given service. It also monitors the service changes to adapt the configuration of the system. It also monitors the pods to build a cache for all the workers and to compute the health status of each service under control of the Kubervisor. The health of the serivce is persisted inside the status of the associated **BreakConfig*
- The **Breaker** is in charge of invoking the configured **anamaly detection**. Ensuring that it is not going bellow defined threshold or ratio, the **Breaker** will relabel some pods to prevent them to receive traffic.
- The **Activator** is in charge of restablishing traffic on pods after the defined period of inactivity (equivalent to open state in a circuit breaker pattern). Depending on the configured policy and the numbers of retries performed on a pod, the **Activator** can decide to kill the pod or put it in *pause* (out of traffic forever) for further investigation.
- The **Anomaly detector** part (all the blue part in the diagram) is where the data analysis is really performed. Depending on the KPI that you are working on (discrete value, continuous value) or the type of anomaly (ratio, threshold, trend ...) you can select an integrated implementation or delegate the to an external system that would return the list of pods that are out of policy. The proposed internal implementations used data from Prometheus.

more information in the developper [documentation page](./docs/developper_docs.md)

## System Operations

### Admin side

#### CRD

TODO

#### Scope

The Kubervisor is an operator that can run in a dedicated namespace and cover the resource of that namespace only, or as a global operator taking action in all namespace that it had been granted access. The used service account will determine the scope on which the Kubervisor will work.

It requires to be given the following roles:
- get:          pod, service
- list:         pod, service
- update:       pod, service
- watch:        pod, service
- delete:       pod

#### Deployment

TODO (Helm ?)

### User side

To configure the system a user would have to complete the following steps:

- Create the **BreakerConfig** CRD in the namespace of the associated service
- - Select the service (by name)
- - Define the BreakConfiguration to configure the Anomaly Detection mechanism
- - Configure the Activator
- Once the **CRD** status is **Ready** activate the system by adding the following label in the Selector of the service: **kubervisor/traffic=yes**
- - TODO: Alternativelly use the command kubectl .....

To deactivate any effect of the Kubervisor for a given service, simply delete from the Selector the label with key **kubervisor/traffic**


