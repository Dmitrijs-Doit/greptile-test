package attributiongroupdeletevalidation

import domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"

type AttributionGroupDeleteValidation struct {
	ID        string                                                    `json:"id"`
	Error     string                                                    `json:"error"`
	Resources map[domainResource.ResourceType][]domainResource.Resource `json:"resources,omitempty"`
}
