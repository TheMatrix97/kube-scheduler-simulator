# Notes - Marc


## Run the PoC demo:

- Build and run the simulator env

```bash
make docker_build docker_up_local
```

- Deploy the nodes
```bash
kubectl --kubeconfig=simulator/cmd/scheduler/kubeconfig.yaml apply -k poc/nodes
```

- Deploy the Pod, it should be scheduled to node 3
```bash
kubectl --kubeconfig=simulator/cmd/scheduler/kubeconfig.yaml apply -f poc/pod.yaml
```

## Create the scheduler plugin

- Add the implementation in `scheduler/customplugins/{your_plugin}/plugin.go`
> Check nodeNumber current implementation

- Modify the scheduler to include your plugin `simulator/cmd/scheduler.go`

```golang
	command, cancelFn, err := debuggablescheduler.NewSchedulerCommand(
		debuggablescheduler.WithPlugin(nodenumber.Name, nodenumber.New), //Initialize the plugin (Set your nodename)
	)
```

- Modify the `simulator/scheduler/scheduler.yaml` to activate the plugin

```yaml
kind: KubeSchedulerConfiguration
apiVersion: kubescheduler.config.k8s.io/v1
clientConnection:
  kubeconfig: kubeconfig.yaml
profiles:
  - schedulerName: default-scheduler
    plugins:
      multiPoint:
        enabled:
          - name: NodeNumber # Enable the plugin and set weight
            weight: 10
```


## Cleanup

```bash
make docker_down docker_down_local
```