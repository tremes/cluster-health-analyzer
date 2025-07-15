package componentshealth

import (
	"context"
	"fmt"

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

		if len(o.ObjectsSelectors) > 0 {

		}

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

func translateSelectorsToExpression(selectors []Selector) string {
	for _, s := range selectors {
		fmt.Println("=============== SELECTOR IS ", s)
		var expression string
		for lName, lValues := range s.MatchLabels {
			fmt.Println("=============== MATCH LABELS ARE ", s)
			// for lName, lValues := range matchLabels {
			switch {
			case len(lValues) == 0:
				if len(expression) == 0 {
					expression = lName
				}
			case len(lValues) == 1:
			case len(lValues) > 1:
			}
		}
	}
	return ""
}
