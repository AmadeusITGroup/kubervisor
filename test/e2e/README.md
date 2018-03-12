# End to end Tests

```shell
$go test -c
```

which will compile the `e2e.test` executable in your current directory. Then

```shell
./e2e.test --kubeconfig=$HOME/.kube/config

```

will start the e2e test....
