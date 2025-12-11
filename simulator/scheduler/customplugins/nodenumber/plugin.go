package nodenumber

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/xerrors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"

	// 3. Aquí definimos el alias 'frameworkruntime' para usar 'frameworkruntime.DecodeInto'
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

// NodeNumber is an example plugin that favors nodes that have the number suffix which is the same as the number suffix of the pod name.
// But if a reverse option is true, it favors nodes that have the number suffix which **isn't** the same as the number suffix of pod name.
//
// For example:
// With reverse option false, when schedule a pod named Pod1, a Node named Node1 gets a lower score than a node named Node9.
//
// NOTE: this plugin only handle single digit numbers only.
type NodeNumber struct {
	// if reverse is true, it favors nodes that doesn't have the same number suffix.
	//
	// For example:
	// When schedule a pod named Pod1, a Node named Node1 gets a lower score than a node named Node9.
	handle  framework.Handle // Usa el alias 'framework'
	reverse bool
}

// NodeNumberArgs is arguments for node number plugin.
//
//nolint:revive
type NodeNumberArgs struct {
	metav1.TypeMeta `json:",inline"`

	Reverse bool `json:"reverse"`
}

var (
	_ framework.ScorePlugin    = &NodeNumber{}
	_ framework.PreScorePlugin = &NodeNumber{}
)

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name             = "NodeNumber"
	preScoreStateKey = "PreScore" + Name
)

// Name returns the name of the plugin. It is used in logs, etc.
func (pl *NodeNumber) Name() string {
	return Name
}

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	podSuffixNumber int
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

func (pl *NodeNumber) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*framework.NodeInfo) *framework.Status {
	klog.InfoS("execute PreScore on NodeNumber plugin", "pod", klog.KObj(pod))

	podNameLastChar := pod.Name[len(pod.Name)-1:]
	podnum, err := strconv.Atoi(podNameLastChar)
	if err != nil {
		// return success even if its suffix is non-number.
		return nil
	}

	s := &preScoreState{
		podSuffixNumber: podnum,
	}
	state.Write(preScoreStateKey, s)

	return nil
}

func (pl *NodeNumber) EventsToRegister() []framework.ClusterEvent {
	return []framework.ClusterEvent{
		{Resource: framework.Node, ActionType: framework.Add},
	}
}

var ErrNotExpectedPreScoreState = errors.New("unexpected pre score state")

// Score invoked at the score extension point.
func (pl *NodeNumber) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	klog.InfoS("execute Score on NodeNumber plugin", "pod", klog.KObj(pod))
	data, err := state.Read(preScoreStateKey)
	if err != nil {
		// return success even if there is no value in preScoreStateKey, since the
		// suffix of pod name maybe non-number.
		return 0, nil
	}

	s, ok := data.(*preScoreState)
	if !ok {
		err = xerrors.Errorf("fetched pre score state is not *preScoreState, but %T, %w", data, ErrNotExpectedPreScoreState)
		return 0, framework.AsStatus(err)
	}

	nodeNameLastChar := nodeName[len(nodeName)-1:]

	nodenum, err := strconv.Atoi(nodeNameLastChar)
	if err != nil {
		// return success even if its suffix is non-number.
		return 0, nil
	}

	var matchScore int64 = 10
	var nonMatchScore int64 = 0 //nolint:revive // for better readability.
	if pl.reverse {
		matchScore = 0
		nonMatchScore = 10
	}

	if s.podSuffixNumber == nodenum {
		// if match, node get high score.
		return matchScore, nil
	}

	return nonMatchScore, nil
}

// ScoreExtensions of the Score plugin.
func (pl *NodeNumber) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// New initializes a new plugin and returns it.
func New(ctx context.Context, arg runtime.Object, h framework.Handle) (framework.Plugin, error) {
	// 1. Definir valores por defecto (Defaulting)
	args := &NodeNumberArgs{
		Reverse: false, // Valor por defecto
	}

	// 2. Decodificar la configuración del YAML en tu struct
	if err := frameworkruntime.DecodeInto(arg, args); err != nil {
		return nil, fmt.Errorf("error al decodificar NodeNumberArgs: %w", err)
	}

	// Logging para verificar que funciona
	klog.InfoS("NodeNumberArgs aplicados correctamente", "reverse", args.Reverse)

	// 3. Pasar la configuración a tu plugin
	return &NodeNumber{
		handle:  h,
		reverse: args.Reverse,
	}, nil
}

// DeepCopyObject es necesario para cumplir la interfaz runtime.Object.
func (in *NodeNumberArgs) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(NodeNumberArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copia el receptor al argumento de salida.
func (in *NodeNumberArgs) DeepCopyInto(out *NodeNumberArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	// Aquí copias tus campos. Si tienes punteros o slices, debes copiarlos uno a uno.
	// Como 'Reverse' es un booleano simple, la asignación *out = *in ya lo cubrió,
	// pero es buena práctica ser explícito si la estructura crece.
	out.Reverse = in.Reverse
}
