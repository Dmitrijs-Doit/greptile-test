package service

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
)

func (s *ReportTemplateService) GetTemplateData(ctx context.Context, isDoitEmployee bool) ([]domain.ReportTemplate, []domain.ReportTemplateVersion, error) {
	templates, err := s.reportTemplateDAL.GetTemplates(ctx)
	if err != nil {
		return nil, nil, err
	}

	var versionRefs []*firestore.DocumentRef

	for tIdx, template := range templates {
		if isDoitEmployee && template.LastVersion != nil {
			versionRefs = append(versionRefs, template.LastVersion)
		} else if template.ActiveVersion != nil {
			versionRefs = append(versionRefs, template.ActiveVersion)
		}

		templates[tIdx].SetPath()
	}

	versions, err := s.reportTemplateDAL.GetVersions(ctx, versionRefs)
	if err != nil {
		return nil, nil, err
	}

	for vIdx := range versions {
		versions[vIdx].SetPath()
	}

	return templates, versions, nil
}
