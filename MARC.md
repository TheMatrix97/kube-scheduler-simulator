# Notes - Marc

> Tested with go1.24.5

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

## Debug

Check `.vscode/launch.json` to run the simulator scheduler in debugmode

1. Stop the containers `simulator-server` and `simulator-scheduler`

2. Use `simulator/cmd/scheduler-local/scheduler-local` to set your scheduler preferences. Don't forget to add the Wrapped suffix to your plugin confings.

Example:
```yaml
kind: KubeSchedulerConfiguration
apiVersion: kubescheduler.config.k8s.io/v1
clientConnection:
  kubeconfig: /home/matrix/phd/kube-scheduler-simulator/simulator/cmd/kubeconfig-local.yaml
profiles:
  - schedulerName: default-scheduler
    plugins:
      multiPoint:
        enabled:
          - name: NodeNumber
            weight: 10
    pluginConfig:
      - name: NodeNumberWrapped
        args:
          reverse: true
```

## Cleanup

```bash
make docker_down docker_down_local
```