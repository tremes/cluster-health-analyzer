package componentshealth

import (
	"context"

	"github.com/inecas/kube-health/pkg/eval"
	"github.com/inecas/kube-health/pkg/khealth"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type HealthChecker interface {
	EvaluateObjects(ctx context.Context, objects []K8sObject) ([]ObjectStatus, error)
}

// kubeHealthChecker is wrapper type
// around kube-health
type kubeHealthChecker struct {
	evaluator *eval.Evaluator
}

// NewKubeHealthChecker creates a new instance of the
// kubeHealthChecker.
func NewKubeHealthChecker() (HealthChecker, error) {
	evaluator, err := khealth.NewHealthEvaluator(nil)
	if err != nil {
		return nil, err
	}
	khChecker := &kubeHealthChecker{
		evaluator: evaluator,
	}

	return khChecker, nil
}

// EvaluateObjects
func (k *kubeHealthChecker) EvaluateObjects(ctx context.Context, objects []K8sObject) ([]ObjectStatus, error) {
	var statuses []ObjectStatus
	for _, o := range objects {
		gr := schema.GroupResource{Group: o.Group, Resource: o.Resource}
		objStatuses, err := k.evaluator.EvalResource(ctx, gr, o.Namespace, o.Name)
		if err != nil {
			return nil, err
		}
		for _, os := range objStatuses {
			objStatus := ObjectStatus{
				Name:         os.Object.Name,
				Namespace:    os.Object.Namespace,
				Resource:     o.Resource,
				HealthStatus: ParseKubeHealthStatus(os.Status().Result),
				Progressing:  os.Status().Progressing,
			}
			statuses = append(statuses, objStatus)
		}
	}
	return statuses, nil
}
