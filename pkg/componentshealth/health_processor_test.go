package componentshealth

import (
	"strings"
	"testing"
	"time"

	"github.com/openshift/cluster-health-analyzer/pkg/prom"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateComponentsHealth(t *testing.T) {
	tests := []struct {
		name                    string
		testComponentsFile      string
		expectedNameStatusPairs []nameStatusPair
	}{
		{
			name:               "basic",
			testComponentsFile: "test-data/simple-components.yaml",
			expectedNameStatusPairs: []nameStatusPair{
				{
					name:   "control-plane.nodes",
					status: OK,
				},
				{
					name:   "control-plane.capacity.cpu",
					status: Warning,
				},
				{
					name:   "control-plane.capacity.memory",
					status: OK,
				},
				{
					name:   "control-plane.capacity",
					status: Warning,
				},
				{
					name:   "control-plane",
					status: Warning,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAlertLoader := MockAlertLoader{
				alerts: []models.Alert{
					{
						Labels: models.LabelSet{
							"alertname": "KubeCPUOvercommit",
							"part_of":   "foos",
							"severity":  "warning",
						},
					},
					{
						Labels: models.LabelSet{
							"alertname":  "BarAlert",
							"test_label": "foo",
							"severity":   "critical",
						},
					},
				},
			}

			testProcessor := NewHealthProcessor(0*time.Second, mockAlertLoader, nil)
			testConf, err := loadConfig(tt.testComponentsFile)
			assert.NoError(t, err)
			componentsHealths := testProcessor.evaluateComponentsHealth(testConf.Components)
			assert.Equal(t, tt.expectedNameStatusPairs, componentHealthToNameStatusPairs(componentsHealths))
		})
	}
}

func TestEvaluateComponentHealth(t *testing.T) {
	tests := []struct {
		name                    string
		component               *Component
		expectedComponentHealth *ComponentHealth
	}{
		{
			name: "component with one error child",
			component: &Component{
				Name:   "foo",
				Alerts: AlertsConfig{},
				ChildComponents: []Component{
					{
						Name: "bar",
						Alerts: AlertsConfig{
							Selectors: []Selectors{
								{
									MatchLabels: []map[string][]string{
										{
											"part_of": []string{"bars"},
										},
									},
								},
							},
						},
					},
					{
						Name:   "baz",
						Alerts: AlertsConfig{},
					},
				},
			},
			expectedComponentHealth: newComponentHealth("foo", Error).
				AddChild(
					&ComponentHealth{
						name:         "bar",
						healthStatus: Error,
						alerts: []model.LabelSet{
							{
								srcSeverity:  "critical",
								srcAlertname: "BarAlert",
								srcNamespace: "",
								"part_of":    "bars",
							},
						},
					}).
				AddChild(
					&ComponentHealth{
						name:         "baz",
						healthStatus: OK,
					},
				),
		},
		{
			name: "component with one warn child",
			component: &Component{
				Name:   "foo",
				Alerts: AlertsConfig{},
				ChildComponents: []Component{
					{
						Name: "bar",
						Alerts: AlertsConfig{
							Selectors: []Selectors{
								{
									MatchLabels: []map[string][]string{
										{
											"part_of": []string{"foos"},
										},
									},
								},
							},
						},
					},
					{
						Name:   "baz",
						Alerts: AlertsConfig{},
					},
				},
			},
			expectedComponentHealth: newComponentHealth("foo", Warning).
				AddChild(
					&ComponentHealth{
						name:         "bar",
						healthStatus: Warning,
						alerts: []model.LabelSet{
							{
								srcSeverity:  "warning",
								srcAlertname: "FooAlert",
								srcNamespace: "",
								"part_of":    "foos",
							},
						},
					}).
				AddChild(
					&ComponentHealth{
						name:         "baz",
						healthStatus: OK,
					},
				),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAlertLoader := MockAlertLoader{
				alerts: []models.Alert{
					{
						Labels: models.LabelSet{
							"alertname": "FooAlert",
							"part_of":   "foos",
							"severity":  "warning",
						},
					},
					{
						Labels: models.LabelSet{
							"alertname": "BarAlert",
							"part_of":   "bars",
							"severity":  "critical",
						},
					},
				},
			}

			testProcessor := NewHealthProcessor(0*time.Second, mockAlertLoader, nil)
			componentsHealth, err := testProcessor.evaluateComponent(tt.component)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedComponentHealth, componentsHealth)
		})
	}
}

func TestComponentHealthsToMetrics(t *testing.T) {
	tests := []struct {
		name             string
		componentsHealth []*ComponentHealth
		expectedMetrics  []prom.Metric
	}{
		{
			name: "healthy component doesn't create a new metric",
			componentsHealth: []*ComponentHealth{
				{
					name:         "healthy-component",
					healthStatus: OK,
				},
			},
			expectedMetrics: nil,
		},
		{
			name: "one component with 2 alerts firing creates 2 metrics",
			componentsHealth: []*ComponentHealth{
				{
					name:         "bar",
					healthStatus: Warning,
					alerts: []model.LabelSet{
						{
							srcAlertname: "BarAlert",
							"part_of":    "bars",
						},
						{
							srcAlertname: "AnotherBarAlert",
							"part_of":    "bars",
						},
					},
				},
			},
			expectedMetrics: []prom.Metric{
				{
					Labels: model.LabelSet{
						"component":  "bar",
						srcAlertname: "BarAlert",
						"part_of":    "bars",
						"status":     "warning",
					},
					Value: 1,
				},
				{
					Labels: model.LabelSet{
						"component":  "bar",
						srcAlertname: "AnotherBarAlert",
						"part_of":    "bars",
						"status":     "warning",
					},
					Value: 1,
				},
			},
		},
		{
			name: "component with parent and childs",
			componentsHealth: []*ComponentHealth{
				newComponentHealth("testParent", OK).
					AddChild(
						newComponentHealth("foo", Error).
							AddChild(&ComponentHealth{
								name:         "bar",
								healthStatus: Error,
								parent: &ComponentHealth{
									name: "foo",
									parent: &ComponentHealth{
										name: "testParent",
									},
								},
								alerts: []model.LabelSet{
									{
										srcAlertname: "BarAlert",
										"part_of":    "bars",
									},
								},
							}).
							AddChild(
								&ComponentHealth{
									name:         "baz",
									healthStatus: OK,
								},
							)),
			},
			expectedMetrics: []prom.Metric{
				{
					Labels: model.LabelSet{
						"component":  "testParent.foo.bar",
						srcAlertname: "BarAlert",
						"part_of":    "bars",
						"status":     "error",
					},
					Value: 2,
				},
				{
					Labels: model.LabelSet{
						"component": "testParent.foo",
						"status":    "error",
					},
					Value: 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := createHealthMetrics(tt.componentsHealth)
			assert.ElementsMatch(t, tt.expectedMetrics, metrics)
		})
	}
}

func componentHealthToNameStatusPairs(componentHealths []*ComponentHealth) []nameStatusPair {
	var res []nameStatusPair
	for _, c := range componentHealths {
		childRes := componentHealthToNameStatusPairs(c.childComponents)
		res = append(res, childRes...)
		res = append(res, nameStatusPair{
			name:   fullComponentName(c),
			status: c.healthStatus,
		})
	}
	return res
}

func newComponentHealth(name string, health HealthStatus) *ComponentHealth {
	return &ComponentHealth{
		name:         name,
		healthStatus: health,
	}
}

type nameStatusPair struct {
	name   string
	status HealthStatus
}

type MockAlertLoader struct {
	alerts []models.Alert
	err    error
}

func (m MockAlertLoader) ActiveAlerts() ([]models.Alert, error) {
	return m.alerts, m.err
}

// ActiveAlertsWithLabels returns only the alerts matching all the provided labels
func (m MockAlertLoader) ActiveAlertsWithLabels(labels []string) ([]models.Alert, error) {
	var res []models.Alert
	labelsToMatch := labelSliceToMap(labels)
	for _, a := range m.alerts {
		allMatch := true
		for k, v := range labelsToMatch {
			val, ok := a.Labels[k]

			switch v.operator {
			case "=":
				if !ok || val != v.value {
					allMatch = false
				}
			case "!=":
				if !ok {
					allMatch = false
				}

			}
		}
		if allMatch {
			res = append(res, a)
		}
	}
	return res, m.err
}

type labelWithOperator struct {
	key      string
	operator string
	value    string
}

func labelSliceToMap(labels []string) map[string]labelWithOperator {
	m := make(map[string]labelWithOperator, len(labels))
	for _, l := range labels {
		var pairAsSlice []string
		var operator string
		if strings.Contains(l, "!=") {
			pairAsSlice = strings.Split(l, "!=")
			operator = "!="
		} else {
			pairAsSlice = strings.Split(l, "=")
			operator = "="
		}
		m[pairAsSlice[0]] = labelWithOperator{
			operator: operator,
			value:    pairAsSlice[1],
		}
	}
	return m
}
