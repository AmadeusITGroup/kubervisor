# kubervisor-demo

## Prerequisite

- have a minikube binaries with the kubernetes cluster version >= 1.9
- have ```Helm``` cli binary

## Demo setup

```shell
export GOPATH=$(git rev-parse --show-toplevel)/../../../../
./scripts/init.sh <kubervisor_git_clone_path>
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
