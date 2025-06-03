package componentshealth

import (
	"github.com/openshift/cluster-health-analyzer/pkg/processor"
	"github.com/prometheus/common/model"
)

type ComponentsConfig struct {
	Components []Component `yaml:"components"`
}

type Component struct {
	Name            string       `yaml:"name"`
	Objects         []K8sObject  `yaml:"objects"`
	ChildComponents []Component  `yaml:"children"`
	Alerts          AlertsConfig `yaml:"alerts"`
}

type K8sObject struct {
	Group    string `yaml:"group"`
	Resource string `yaml:"resource"`
}

type AlertsConfig struct {
	Selectors []Selectors `yaml:"selectors"`
}

type Selectors struct {
	MatchLabels []map[string][]string `yaml:"matchLabels"`
}

type HealthStatus string

func (h HealthStatus) IsOK() bool {
	return h == OK
}

func (h HealthStatus) IsError() bool {
	return h == Error
}

func (h HealthStatus) IsWarning() bool {
	return h == Warning
}

func ParseHealthValue(h processor.HealthValue) HealthStatus {
	switch h {
	case 0:
		return OK
	case 1:
		return Warning
	case 2:
		return Error
	default:
		// We don't recognize the health value, so we'll default to warning
		return Warning
	}
}

var OK HealthStatus = "ok"
var Warning HealthStatus = "warning"
var Error HealthStatus = "error"

type ComponentHealth struct {
	name            string
	parent          *ComponentHealth
	childComponents []*ComponentHealth
	alerts          []model.LabelSet
	healthStatus    HealthStatus
}

func (c *ComponentHealth) AddChild(ch *ComponentHealth) *ComponentHealth {
	ch.parent = c
	c.childComponents = append(c.childComponents, ch)
	return c
}
