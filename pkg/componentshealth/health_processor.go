package componentshealth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/openshift/cluster-health-analyzer/pkg/processor"
	"github.com/openshift/cluster-health-analyzer/pkg/prom"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
)

type healthProcessor struct {
	interval                time.Duration
	alertMatcher            alertMatcher
	componentAlertssMetrics prom.MetricSet
}

// NewHealthProcessor
func NewHealthProcessor(interval time.Duration, amAPI AlertLoader, alertsMetrics prom.MetricSet) *healthProcessor {
	alertMatcher := NewAlertMatcher(amAPI)
	return &healthProcessor{
		interval:                interval,
		alertMatcher:            alertMatcher,
		componentAlertssMetrics: alertsMetrics,
	}
}

// Start starts the processor in a goroutine and returns immediately.
func (p *healthProcessor) Start(ctx context.Context) {
	go p.Run(ctx)
}

// Run periodically runs the processor and blocks until the provided context is done.
func (p *healthProcessor) Run(ctx context.Context) {
	conf, err := loadConfig("/etc/config/components.yaml")
	if err != nil {
		slog.Error("Failed to load config ", "error", err)
		return
	}
	healthStatuses := p.evaluateComponentsHealth(conf.Components)
	promMetrics := createHealthMetrics(healthStatuses)
	p.componentAlertssMetrics.Update(promMetrics)

	ticker := time.NewTicker(p.interval)
	for {
		select {
		case <-ticker.C:
			slog.Info("Evaluating health of the components")
			healthStatuses = p.evaluateComponentsHealth(conf.Components)
			promMetrics := createHealthMetrics(healthStatuses)
			p.componentAlertssMetrics.Update(promMetrics)

		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

// loadConfig reads the mounted "/etc/config/components.yaml" file
// and unmarshals the component config.
func loadConfig(filePath string) (*ComponentsConfig, error) {
	conf := &ComponentsConfig{}
	cData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(cData, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (p *healthProcessor) evaluateComponentsHealth(components []Component) []*ComponentHealth {
	var componentHealths []*ComponentHealth
	for _, c := range components {
		slog.Debug("Evaluating health of component", "component", c.Name)
		cHealth, err := p.evaluateComponent(&c)
		if err != nil {
			slog.Error("Failed to evaluate health of component", "name", c.Name, "error", err)
			continue
		}
		componentHealths = append(componentHealths, cHealth)
	}
	return componentHealths
}

// evaluateComponent evaluates the health of the provided component
// and recursively the health of all its child components.
func (p *healthProcessor) evaluateComponent(c *Component) (*ComponentHealth, error) {
	ch := ComponentHealth{name: c.Name}

	worstChildStatus := processor.Healthy
	for _, child := range c.ChildComponents {
		childHealth, err := p.evaluateComponent(&child)
		if err != nil {
			return nil, err
		}
		if healthStatusToHealthValue(childHealth.healthStatus) > worstChildStatus {
			worstChildStatus = healthStatusToHealthValue(childHealth.healthStatus)
		}
		ch.AddChild(childHealth)
	}

	alerts, err := p.alertMatcher.evaluateAlerts(c.Alerts)
	if err != nil {
		return nil, err
	}
	ch.healthStatus = calculateHealthStatus(ParseHealthValue(worstChildStatus), alerts)
	ch.alerts = alerts
	return &ch, nil
}

// createHealthMetrics creates all Prometheus metrics for the slice of
// ComponentHealths
func createHealthMetrics(componentHealth []*ComponentHealth) []prom.Metric {
	var metrics []prom.Metric
	for _, c := range componentHealth {
		compMetrics := componentHealthMetrics(c)
		metrics = append(metrics, compMetrics...)
	}
	return metrics
}

func componentHealthMetrics(cHealth *ComponentHealth) []prom.Metric {
	var metrics []prom.Metric
	for _, child := range cHealth.childComponents {
		childMetrics := componentHealthMetrics(child)
		metrics = append(metrics, childMetrics...)
	}
	componentName := fullComponentName(cHealth)

	for _, a := range cHealth.alerts {
		m := metricWithNameAndStatus(componentName, cHealth.healthStatus)
		m.Labels = mergeLabels(m.Labels, a)
		metrics = append(metrics, m)
	}
	// some childs have some error or warning
	if !cHealth.healthStatus.IsOK() && len(cHealth.alerts) == 0 {
		metrics = append(metrics, metricWithNameAndStatus(componentName, cHealth.healthStatus))
	}
	return metrics
}

func mergeLabels(m model.LabelSet, labelsSet model.LabelSet) model.LabelSet {
	for k, v := range labelsSet {
		m[k] = v
	}
	return m
}

// calculateHealthStatus calculates HealthStatus based on the provided status and alerts
func calculateHealthStatus(hs HealthStatus, alerts []model.LabelSet) HealthStatus {
	if hs.IsError() {
		return Error
	}

	if hs.IsWarning() {
		return Warning
	}

	highestSeverity := processor.Healthy
	for _, alert := range alerts {
		severity := string(alert["src_severity"])
		hv := processor.ParseHealthValue(severity)
		if hv > highestSeverity {
			highestSeverity = hv
		}
	}

	return ParseHealthValue(highestSeverity)
}

// metricWithNameAndStatus is a helper function creating a Prometheus metric
// with provided name and status labels and with the value reflecting the
// health status
func metricWithNameAndStatus(name string, status HealthStatus) prom.Metric {
	return prom.Metric{
		Labels: model.LabelSet{
			"component": model.LabelValue(name),
			"status":    model.LabelValue(status),
		},
		Value: float64(healthStatusToHealthValue(status)),
	}
}

func fullComponentName(c *ComponentHealth) string {
	name := c.name
	if c.parent != nil {
		pName := fullComponentName(c.parent)
		name = fmt.Sprintf("%s.%s", pName, name)
	}
	return name
}

func healthStatusToHealthValue(status HealthStatus) processor.HealthValue {
	switch status {
	case OK:
		return processor.Healthy
	case Warning:
		return processor.Warning
	case Error:
		return processor.Critical
	default:
		return -1
	}
}
