package service

import "fmt"

type CheckCustomMetricExistsError struct {
	ID       string
	dalError error
}

func (e CheckCustomMetricExistsError) Error() string {
	return fmt.Sprintf("unable to check if custom metric %s exists; %s", e.ID, e.dalError.Error())
}

type CustomMetricNotFoundError struct {
	ID string
}

func (e CustomMetricNotFoundError) Error() string {
	return fmt.Sprintf("custom metric %s not found", e.ID)
}

type PresetMetricsCannotBeDeletedError struct {
	ID string
}

func (e PresetMetricsCannotBeDeletedError) Error() string {
	return fmt.Sprintf("unable to delete %s because it is a preset", e.ID)
}

type MetricIsInUseError struct {
	ID string
}

func (e MetricIsInUseError) Error() string {
	return fmt.Sprintf("metric %s is used by a report", e.ID)
}
