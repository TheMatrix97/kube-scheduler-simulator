package contextaware

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/xerrors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

type ContextAware struct {
	// labelPrefix allows filtering which labels are treated by the scheduler
	labelPrefix string
}

var (
	_ framework.ScorePlugin    = &ContextAware{}
	_ framework.PreScorePlugin = &ContextAware{}
)

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name             = "ContextAware"
	preScoreStateKey = "PreScore" + Name
)

// Name returns the name of the plugin. It is used in logs, etc.
func (pl *ContextAware) Name() string {
	return Name
}

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	// constraints holds the key-value pairs of labels found on the pod
	// that match the configured prefix.
	constraints map[string]string
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

// Reads the labels
func (pl *ContextAware) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*framework.NodeInfo) *framework.Status {
	klog.InfoS("execute PreScore on ContextAware plugin", "pod", klog.KObj(pod))

	constraints := make(map[string]string)

	// Iterate over pod labels to find context constraints
	for key, value := range pod.Labels {
		if pl.labelPrefix == "" || strings.HasPrefix(key, pl.labelPrefix) {
			constraints[key] = value
		}
	}

	klog.InfoS("Constraints readed for", "pod", klog.KObj(pod), constraints)

	s := &preScoreState{
		constraints: constraints,
	}
	state.Write(preScoreStateKey, s)

	return nil
}

func (pl *ContextAware) EventsToRegister() []framework.ClusterEvent {
	return []framework.ClusterEvent{
		{Resource: framework.Node, ActionType: framework.Add},
	}
}

var ErrNotExpectedPreScoreState = errors.New("unexpected pre score state")

// Score invoked at the score extension point.
func (pl *ContextAware) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	klog.InfoS("execute Score on ContextAware plugin", "pod", klog.KObj(pod))
	data, err := state.Read(preScoreStateKey)
	if err != nil {
		// return success even if there is no value in preScoreStateKey, since the
		// suffix of pod name maybe non-number.
		return 0, nil
	}

	s, ok := data.(*preScoreState)
	klog.InfoS("preScore State loaded", s.constraints)
	if !ok {
		err = xerrors.Errorf("fetched pre score state is not *preScoreState, but %T, %w", data, ErrNotExpectedPreScoreState)
		return 0, framework.AsStatus(err)
	}

	/*Set here login for match score*/

	var matchScore int64 = 33 //Static


	return matchScore, nil
}

// ScoreExtensions of the Score plugin.
func (pl *ContextAware) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// New initializes a new plugin and returns it.
func New(ctx context.Context, arg runtime.Object, h framework.Handle) (framework.Plugin, error) {
	typedArg := ContextAwareArgs{LabelPrefix: ""}
	if arg != nil {
		err := frameworkruntime.DecodeInto(arg, &typedArg)
		if err != nil {
			return nil, xerrors.Errorf("decode arg into ContextAwareArgs: %w", err)
		}
		klog.Info("ContextAwareArgs is successfully applied")
	}
	return &ContextAware{labelPrefix: typedArg.LabelPrefix}, nil
}

// ContextAwareArgs is arguments for node number plugin.
//
//nolint:revive
type ContextAwareArgs struct {
	metav1.TypeMeta

	LabelPrefix string `json:"labelPrefix"`
}
