package contextaware

import (
	"context"
	"errors"
	"strconv"
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
	handle framework.Handle // Usa el alias 'framework'
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
	// labels holds the key-value pairs of labels found on the pod (match Kontext.io)
	annotations map[string]string
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

// Costly functions to be executed once per Pod to schedule
func (pl *ContextAware) PreScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*framework.NodeInfo) *framework.Status {
	klog.InfoS("execute PreScore on ContextAware plugin", "pod", klog.KObj(pod))

	kontextLabels := make(map[string]string)

	podAnnotations := pod.ObjectMeta.Annotations

	// Iterate over pod labels to find context constraints
	for key, value := range podAnnotations {
		if strings.HasPrefix(key, "kontext.io") {
			kontextLabels[key] = value
		}
	}

	klog.InfoS("Annotations readed for", "pod", klog.KObj(pod), "annotations", kontextLabels)

	s := &preScoreState{
		annotations: kontextLabels,
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

// Score invoked at the score extension point (for each node)
func (pl *ContextAware) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	klog.InfoS("execute Score on ContextAware plugin", "pod", klog.KObj(pod), "node", nodeName)
	// Get PreScore Data
	data, err := state.Read(preScoreStateKey)
	if err != nil {
		return 0, framework.AsStatus(err) // Should not happen state must be at least empty
	}
	s, ok := data.(*preScoreState)
	klog.InfoS("preScore State loaded", "annotations", s.annotations)
	if !ok {
		err = xerrors.Errorf("fetched pre score state is not *preScoreState, but %T, %w", data, ErrNotExpectedPreScoreState)
		return 0, framework.AsStatus(err)
	}

	// Get NodeInfo from the framework handle
	nodeInfo, err := pl.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	nodeAnnotations := nodeInfo.Node().GetObjectMeta().GetAnnotations()
	klog.InfoS("Node annotations", "annotations", nodeAnnotations)

	/*Set here login for match score*/
	karmaNodeKey := "kontext.io/karma"
	karmaRequirementKey := "kontext.io/required-karma"
	karmaPodRequirement, ok := s.annotations[karmaRequirementKey]
	matchScore := int64(0)
	if ok {
		karmaPodRequirementsFloat, err := strconv.ParseFloat(karmaPodRequirement, 64) // Example conversion
		if err != nil {
			klog.ErrorS(err, "Error parsing karma score", "pod", klog.KObj(pod), "node", nodeName)
			return 0, framework.AsStatus(err)
		}
		karmaNodeScore, ok := nodeAnnotations[karmaNodeKey]
		if ok {
			karmaNodeScoreFloat, err := strconv.ParseFloat(karmaNodeScore, 64) // Example conversion
			if err != nil {
				klog.ErrorS(err, "Error parsing karma node score", "node", klog.KObj(nodeInfo))
			}
			// If node karma >= pod required karma, compute score to priorize closer scores
			if karmaNodeScoreFloat >= karmaPodRequirementsFloat {
				matchScore = int64(karmaNodeScoreFloat * 100)
			}
		}
	}
	return matchScore, nil
}

// ScoreExtensions of the Score plugin.
func (pl *ContextAware) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// New initializes a new plugin and returns it.
func New(ctx context.Context, arg runtime.Object, h framework.Handle) (framework.Plugin, error) {
	typedArg := &ContextAwareArgs{
		LabelPrefix: "", // Set empty by default
	}
	if arg != nil {
		err := frameworkruntime.DecodeInto(arg, &typedArg)
		if err != nil {
			return nil, xerrors.Errorf("decode arg into ContextAwareArgs: %w", err)
		}
	}
	klog.Info("ContextAwareArgs is successfully applied -> ", typedArg.LabelPrefix)
	return &ContextAware{
		handle:      h,
		labelPrefix: typedArg.LabelPrefix,
	}, nil
}

// DeepCopyObject es necesario para cumplir la interfaz runtime.Object.
func (in *ContextAwareArgs) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(ContextAwareArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copia el receptor al argumento de salida.
func (in *ContextAwareArgs) DeepCopyInto(out *ContextAwareArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	// Aquí copias tus campos. Si tienes punteros o slices, debes copiarlos uno a uno.
	// Como 'LabelPrefix' es un string simple, la asignación *out = *in ya lo cubrió,
	// pero es buena práctica ser explícito si la estructura crece.
	out.LabelPrefix = in.LabelPrefix
}

// ContextAwareArgs is arguments for context aware plugin.
type ContextAwareArgs struct {
	metav1.TypeMeta `json:",inline"`
	LabelPrefix     string `json:"labelPrefix"`
}
