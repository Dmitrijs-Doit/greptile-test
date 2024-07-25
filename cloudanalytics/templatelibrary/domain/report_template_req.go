package domain

import (
	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type ReportTemplateReq struct {
	Categories  []string             `json:"categories"  binding:"required"`
	Cloud       []string             `json:"cloud"       binding:"required"`
	Visibility  Visibility           `json:"visibility"  binding:"required"`
	Name        string               `json:"name"        binding:"required"`
	Description string               `json:"description" binding:"required"`
	Config      *reportDomain.Config `json:"config"      binding:"required"`
}
