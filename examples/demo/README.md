# kubervisor-demo

## Prerequisite

- have a minikube binaries with the kubernetes cluster version >= 1.9
- have ```kubectl``` cli binary
- have ```Helm``` cli binary
- have ```jq``` installed on the host (commandline JSON processor)
- have ```pv``` installed on the host (Concatenate FILE(s), or standard input, to standard output, with monitoring.)
- have ```dnsmasq``` installed

## Demo setup

```shell
export GOPATH=$(git rev-parse --show-toplevel)/../../../../
cd $GOPATH/src/github.com/amadeusitgroup/kubervisor
make plugin
eval $(minikube docker-env)
git tag latest
make container
cd $GOPATH/src/github.com/amadeusitgroup/kubervisor/examples/demo
./scripts/init.sh
```

## Setup dnsmasq

Service are expose via ingress. We are going to use domain to target our ```demo.mk``` for our minikube vm.

```shell
sudo "IP=$(minikube ip)" bash -c 'echo "address=/.demo.mk/$IP" > /etc/dnsmasq.d/minikube'

sudo service dsnmasq restart
```

## Demo

Run the following script that executes the differents commands.

```shell
./scripts/demo.sh
```

## Clear the Demo

If you want to re-run the demo, first run the following command, then you will be able to execute the demo without doint the ```demo setup```.

```shell
./scripts/clear.sh
```
