package domain

import (
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

type ReportTemplate struct {
	ActiveReport  *firestore.DocumentRef `json:"activeReport" firestore:"activeReport"`
	ActiveVersion *firestore.DocumentRef `json:"activeVersion" firestore:"activeVersion"`
	LastVersion   *firestore.DocumentRef `json:"lastVersion" firestore:"lastVersion"`
	TimeCreated   time.Time              `json:"timeCreated" firestore:"timeCreated,serverTimestamp"`
	Hidden        bool                   `json:"-" firestore:"hidden"`
	ID            string                 `json:"id" firestore:"-"`
	Ref           *firestore.DocumentRef `json:"-" firestore:"-"`
}

type ReportTemplateVersion struct {
	Collaborators []collab.Collaborator `json:"collaborators" firestore:"collaborators"`

	Categories      []string               `json:"categories" firestore:"categories"`
	Cloud           []string               `json:"cloud" firestore:"cloud"`
	Visibility      Visibility             `json:"visibility" firestore:"visibility" binding:"required"`
	Active          bool                   `json:"active" firestore:"active"`
	Approval        Approval               `json:"approval" firestore:"approval"`
	Report          *firestore.DocumentRef `json:"report" firestore:"report"`
	CreatedBy       string                 `json:"createdBy" firestore:"createdBy"`
	PreviousVersion *firestore.DocumentRef `json:"previousVersion" firestore:"previousVersion"`
	Template        *firestore.DocumentRef `json:"template" firestore:"template"`
	TimeCreated     time.Time              `json:"timeCreated" firestore:"timeCreated,serverTimestamp"`
	TimeModified    time.Time              `json:"timeModified" firestore:"timeModified,serverTimestamp"`
	ID              string                 `json:"id" firestore:"-"`
	Ref             *firestore.DocumentRef `json:"-" firestore:"-"`
}

type ReportTemplateWithVersion struct {
	Template    *ReportTemplate        `json:"template"`
	LastVersion *ReportTemplateVersion `json:"lastVersion"`
}

func (rt *ReportTemplate) SetPath() {
	rt.ActiveReport = setPath(rt.ActiveReport)
	rt.ActiveVersion = setPath(rt.ActiveVersion)
	rt.LastVersion = setPath(rt.LastVersion)
}

func (v *ReportTemplateVersion) SetPath() {
	v.Report = setPath(v.Report)
	v.PreviousVersion = setPath(v.PreviousVersion)
	v.Template = setPath(v.Template)
}

func setPath(ref *firestore.DocumentRef) *firestore.DocumentRef {
	if ref != nil {
		path := extractShortPath(ref.Path)
		ref = &firestore.DocumentRef{Path: path, ID: ref.ID}
	}

	return ref
}

func extractShortPath(fullPath string) string {
	index := strings.LastIndex(fullPath, "documents/")
	if index >= 0 {
		return fullPath[index+len("documents/"):]
	}

	return ""
}

func (v ReportTemplateVersion) IsOwner(email string) bool {
	for _, collaborator := range v.Collaborators {
		if collaborator.Email == email {
			return collaborator.Role == collab.CollaboratorRoleOwner
		}
	}

	return false
}

func (v ReportTemplateVersion) CanEdit(email string) bool {
	for _, collaborator := range v.Collaborators {
		if collaborator.Email == email {
			return collaborator.Role == collab.CollaboratorRoleEditor || collaborator.Role == collab.CollaboratorRoleOwner
		}
	}

	return false
}

type Visibility string

const (
	VisibilityGlobal   Visibility = "global"
	VisibilityInternal Visibility = "internal"
	VisibilityPrivate  Visibility = "private"
)

type Approval struct {
	ApprovedBy   *string    `json:"approvedBy" firestore:"approvedBy"`
	Changes      []Message  `json:"changes" firestore:"changes"`
	Status       Status     `json:"status" firestore:"status"`
	TimeApproved *time.Time `json:"timeApproved" firestore:"timeApproved"`
}

type Status string

const (
	StatusApproved Status = "approved"
	StatusCanceled Status = "canceled"
	StatusPending  Status = "pending"
	StatusRejected Status = "rejected"
)

type Message struct {
	Email     string    `json:"email" firestore:"email"`
	Text      string    `json:"text" firestore:"text"`
	Timestamp time.Time `json:"timestamp" firestore:"timestamp"`
}

func NewDefaultReportTemplate() *ReportTemplate {
	return &ReportTemplate{}
}
