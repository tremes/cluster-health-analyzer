package componentshealth

import (
	"testing"

	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateAlerts(t *testing.T) {
	tests := []struct {
		name                 string
		alerts               AlertsConfig
		expectedActiveAlerts []model.LabelSet
	}{
		{
			name: "One label with just a key",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"part_of": []string{},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					"part_of":    "", // TODO - fix the value?
					srcNamespace: "foo-ns",
				},
				{
					srcAlertname: "FooAlert",
					"part_of":    "",
					srcSeverity:  "warning",
					srcNamespace: "second-foo-ns",
				},
				{
					srcAlertname: "BarAlert",
					"part_of":    "",
					srcSeverity:  "critical",
					srcNamespace: "bar-ns",
				},
			},
		},
		{
			name: "Multiple label values (OR) and one matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"FooAlert", "BazAlert"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					srcNamespace: "foo-ns",
				},
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					srcNamespace: "second-foo-ns",
				},
			},
		},
		{
			name: "Multiple label values (OR) and none matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"BazAlert", "QuxAlert"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: nil,
		},
		{
			name: "Multiple label values (OR) and all matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"FooAlert", "BarAlert"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					srcNamespace: "foo-ns",
				},
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					srcNamespace: "second-foo-ns",
				},
				{
					srcAlertname: "BarAlert",
					srcSeverity:  "critical",
					srcNamespace: "bar-ns",
				},
			},
		},
		{
			name: "Multiple labels (AND) but only one matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"part_of":     []string{"testing"},
								"alertname":   []string{"FooAlert"},
								"nonexisting": []string{"value"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: nil,
		},
		{
			name: "Multiple labels (AND) and all matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"FooAlert"},
								"part_of":   []string{"foos"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					"part_of":    "foos",
					srcSeverity:  "warning",
					srcNamespace: "foo-ns",
				},
				{
					srcAlertname: "FooAlert",
					"part_of":    "foos",
					srcSeverity:  "warning",
					srcNamespace: "second-foo-ns",
				},
			},
		},
		{
			name: "Multiple labels (AND), multiple values but only one matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"Alert", "Blah"},
								"part_of":   []string{"foos"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: nil,
		},
		{
			name: "Multiple labels, multiple values and all matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"Alert", "FooAlert"},
								"part_of":   []string{"foos"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					"part_of":    "foos",
					srcSeverity:  "warning",
					srcNamespace: "foo-ns",
				},
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					srcNamespace: "second-foo-ns",
					"part_of":    "foos",
				},
			},
		},
		{
			name: "Multiple labels (AND) and all matches one alert",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"namespace": []string{"foo-ns"},
								"part_of":   []string{"bars", "shits", "foos"},
								"alertname": []string{"FooAlert"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					"part_of":    "foos",
					srcSeverity:  "warning",
					srcNamespace: "foo-ns",
				},
			},
		},
		{
			name: "Multiple matchlabels attributes and none matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"FooAlert"},
								"part_of":   []string{"testing"},
							},
						},
					},
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"BarAlert"},
								"part_of":   []string{"testing"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: nil,
		},
		{
			name: "Multiple matchlabels attributes and one matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"FooAlert"},
								"part_of":   []string{"foos"},
							},
						},
					},
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"BarAlert"},
								"part_of":   []string{"testing"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					"part_of":    "foos",
					srcSeverity:  "warning",
					srcNamespace: "foo-ns",
				},
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					srcNamespace: "second-foo-ns",
					"part_of":    "foos",
				},
			},
		},
		{
			name: "Multiple matchlabels attributes and all matches",
			alerts: AlertsConfig{
				Selectors: []Selectors{
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"FooAlert"},
								"part_of":   []string{"foos"},
							},
						},
					},
					{
						MatchLabels: []map[string][]string{
							{
								"alertname": []string{"BarAlert"},
								"part_of":   []string{"bars"},
							},
						},
					},
				},
			},
			expectedActiveAlerts: []model.LabelSet{
				{
					srcAlertname: "FooAlert",
					"part_of":    "foos",
					srcSeverity:  "warning",
					srcNamespace: "foo-ns",
				},
				{
					srcAlertname: "FooAlert",
					srcSeverity:  "warning",
					srcNamespace: "second-foo-ns",
					"part_of":    "foos",
				},
				{
					srcAlertname: "BarAlert",
					"part_of":    "bars",
					srcSeverity:  "critical",
					srcNamespace: "bar-ns",
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
							"alertname": "FooAlert",
							"part_of":   "foos",
							"namespace": "foo-ns",
							"severity":  "warning",
						},
					},
					{
						Labels: models.LabelSet{
							"alertname": "BarAlert",
							"part_of":   "bars",
							"namespace": "bar-ns",
							"severity":  "critical",
						},
					},
					{
						Labels: models.LabelSet{
							"alertname": "FooAlert",
							"part_of":   "foos",
							"namespace": "second-foo-ns",
							"severity":  "warning",
						},
					},
				},
			}
			testAlertMatcher := NewAlertMatcher(mockAlertLoader)
			alerts, err := testAlertMatcher.evaluateAlerts(tt.alerts)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedActiveAlerts, alerts)
		})
	}

}
