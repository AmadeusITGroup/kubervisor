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


# architecture

Pod anomaly detection supervisor

![architecture diagram][diagram1]

[diagram1]: ./images/diagram1.png
