# Demonstrate collective sucide

```
> kubectl apply -f pdb.yaml
> kubectl apply -f probesfile.configmap.yaml
> kubectl apply -f deployment.yaml
```

Once the deployment is complete you should have 3 endpoints for the **foo** service

```
> kubectl get endpoints foo
NAME      ENDPOINTS                                   AGE
foo       172.17.0.5:80,172.17.0.6:80,172.17.0.7:80   5m
```

Now edit the configmap to affect the lifeness probe. Change the value **livenessok** to **livenessko**

The change make take couple of seconds to be propagated.

Monitor the endpoints
```
> kubectl get endpoints foo -w
NAME      ENDPOINTS                                   AGE
foo       172.17.0.5:80,172.17.0.6:80,172.17.0.7:80   5m
foo       172.17.0.5:80,172.17.0.7:80   5m
foo       172.17.0.5:80,172.17.0.6:80,172.17.0.7:80   6m
foo       172.17.0.5:80,172.17.0.6:80   6m
foo       172.17.0.5:80,172.17.0.6:80,172.17.0.7:80   6m
foo       172.17.0.6:80,172.17.0.7:80   6m
foo       172.17.0.5:80,172.17.0.6:80,172.17.0.7:80   6m
foo       172.17.0.5:80,172.17.0.7:80   6m
foo       172.17.0.5:80   6m
foo                 7m
```

We can see that if a factor exterior to the pod affect the probe, we can end up with a collective sucide of all our containers.

There is nothing protecting us against a bad probe. Even setting disruption budget will not prevent the kubelet to kill and restart container.

Monitor the pods
```
> k get pods -w
NAME                   READY     STATUS    RESTARTS   AGE
foo-77c6fff44d-97b9h   1/1       Running   0          1m
foo-77c6fff44d-g4g82   1/1       Running   0          1m
foo-77c6fff44d-szb7w   1/1       Running   0          1m
foo-77c6fff44d-97b9h   0/1       Running   1         1m
foo-77c6fff44d-97b9h   1/1       Running   1         1m
foo-77c6fff44d-g4g82   0/1       Running   0         1m
foo-77c6fff44d-g4g82   0/1       Running   1         1m
foo-77c6fff44d-g4g82   1/1       Running   1         1m
foo-77c6fff44d-szb7w   0/1       Running   1         2m
foo-77c6fff44d-szb7w   1/1       Running   1         2m
foo-77c6fff44d-97b9h   0/1       Running   2         2m
foo-77c6fff44d-97b9h   1/1       Running   2         2m
foo-77c6fff44d-g4g82   0/1       Running   2         2m
foo-77c6fff44d-g4g82   1/1       Running   2         2m
foo-77c6fff44d-szb7w   0/1       Running   2         2m
foo-77c6fff44d-szb7w   1/1       Running   2         2m
foo-77c6fff44d-97b9h   0/1       Running   2         2m
foo-77c6fff44d-97b9h   0/1       Running   3         2m
foo-77c6fff44d-97b9h   1/1       Running   3         2m
foo-77c6fff44d-g4g82   0/1       Running   3         2m
foo-77c6fff44d-g4g82   1/1       Running   3         3m
foo-77c6fff44d-szb7w   0/1       Running   3         3m
foo-77c6fff44d-szb7w   1/1       Running   3         3m
foo-77c6fff44d-97b9h   0/1       Running   4         3m
foo-77c6fff44d-97b9h   1/1       Running   4         3m
foo-77c6fff44d-g4g82   0/1       Running   4         3m
foo-77c6fff44d-g4g82   1/1       Running   4         3m
foo-77c6fff44d-szb7w   0/1       Running   4         3m
foo-77c6fff44d-szb7w   1/1       Running   4         3m
foo-77c6fff44d-97b9h   0/1       CrashLoopBackOff   4         3m
foo-77c6fff44d-g4g82   0/1       CrashLoopBackOff   4         4m
foo-77c6fff44d-szb7w   0/1       CrashLoopBackOff   4         4m
foo-77c6fff44d-97b9h   0/1       Running   5         4m
foo-77c6fff44d-97b9h   1/1       Running   5         4m
foo-77c6fff44d-g4g82   0/1       Running   5         4m
foo-77c6fff44d-g4g82   1/1       Running   5         4m
foo-77c6fff44d-szb7w   0/1       Running   5         4m
foo-77c6fff44d-szb7w   1/1       Running   5         4m
foo-77c6fff44d-97b9h   0/1       CrashLoopBackOff   5         5m
foo-77c6fff44d-g4g82   0/1       Running   5         5m
foo-77c6fff44d-g4g82   0/1       CrashLoopBackOff   5         5m
foo-77c6fff44d-szb7w   0/1       CrashLoopBackOff   5         5m
foo-77c6fff44d-97b9h   0/1       Running   6         6m
foo-77c6fff44d-97b9h   1/1       Running   6         6m
foo-77c6fff44d-g4g82   0/1       Running   6         6m
foo-77c6fff44d-g4g82   1/1       Running   6         6m
foo-77c6fff44d-szb7w   0/1       Running   6         6m
foo-77c6fff44d-szb7w   1/1       Running   6         6m
foo-77c6fff44d-97b9h   0/1       CrashLoopBackOff   6         7m
foo-77c6fff44d-g4g82   0/1       Running   6         7m
foo-77c6fff44d-g4g82   0/1       CrashLoopBackOff   6         7m
foo-77c6fff44d-szb7w   0/1       CrashLoopBackOff   6         7m
```

Looking at the events associated to a pod
```
Events:
  Type     Reason                 Age              From               Message
  ----     ------                 ----             ----               -------
  Normal   Scheduled              7m               default-scheduler  Successfully assigned foo-77c6fff44d-szb7w to minikube
  Normal   SuccessfulMountVolume  7m               kubelet, minikube  MountVolume.SetUp succeeded for volume "probes-volume"
  Normal   SuccessfulMountVolume  7m               kubelet, minikube  MountVolume.SetUp succeeded for volume "default-token-zc24m"
  Warning  Unhealthy              4m (x4 over 6m)  kubelet, minikube  Liveness probe failed:
  Normal   Pulled                 4m (x5 over 7m)  kubelet, minikube  Container image "k8s.gcr.io/busybox" already present on machine
  Normal   Created                4m (x5 over 7m)  kubelet, minikube  Created container
  Normal   Started                4m (x5 over 7m)  kubelet, minikube  Started container
  Normal   Killing                4m (x4 over 5m)  kubelet, minikube  Killing container with id docker://buysebox:Container failed liveness probe.. Container will be killed and recreated.
  Warning  BackOff                2m (x6 over 3m)  kubelet, minikube  Back-off restarting failed container
```
