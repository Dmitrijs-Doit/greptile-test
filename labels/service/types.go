package service

import labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"

type AssignLabelsRequest struct {
	CustomerID   string
	Objects      []AssignLabelsObject `json:"objects"`
	AddLabels    []string             `json:"addLabels"`
	RemoveLabels []string             `json:"removeLabels"`
}

type AssignLabelsObject struct {
	ObjectID   string            `json:"objectId"`
	ObjectType labels.ObjectType `json:"objectType"`
}

type CreateLabelRequest struct {
	Name       string            `json:"name"`
	Color      labels.LabelColor `json:"color"`
	CustomerID string
	UserEmail  string
}

type UpdateLabelRequest struct {
	LabelID string
	Name    string            `json:"name"`
	Color   labels.LabelColor `json:"color"`
}
