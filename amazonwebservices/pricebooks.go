package amazonwebservices

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	createdBySuffix                         = "via Hello CMP"
	pricebooksResourceName                  = "price_books"
	pricebookAssignmentsResourceName        = "price_book_assignments"
	pricebookAccountAssignmentsResourceName = "price_book_account_assignments"

	canCreateCHTPricebook = false

	testPricebookID = 1963155723
	testCustomerID  = 15372
)

type PricebooksResponse struct {
	Pricebooks []*Pricebook `json:"price_books"`
}

type PricebookAssignmentsResponse struct {
	PricebookAssignments []*PricebookAssignment `json:"price_book_assignments"`
}

type PricebookAccountAssignmentsResponse struct {
	PricebookAccountAssignments []*PricebookAccountAssignment `json:"price_book_account_assignments"`
}

type CreatePricebookResponse struct {
	Pricebook *Pricebook `json:"price_book"`
	Warnings  []string   `json:"warnings"`
}

type PricebookSpecification struct {
	Specification string `json:"specification"`
}

type Pricebook struct {
	ID             int64                  `json:"id,omitempty" firestore:"id"`
	BookName       string                 `json:"book_name" firestore:"bookName"`
	CustomerID     string                 `json:"customer_id" firestore:"customerId"`
	PayerID        string                 `json:"payer_id" firestore:"payerId"`
	UpdatedBy      string                 `json:"-" firestore:"updatedBy"`
	CreatedAt      string                 `json:"created_at,omitempty" firestore:"createdAt"`
	UpdatedAt      string                 `json:"updated_at,omitempty" firestore:"updatedAt"`
	FileHash       string                 `json:"file_hash,omitempty" firestore:"fileHash"`
	Specification  *CHTBillingRules       `json:"specification,omitempty" firestore:"specification"`
	Editable       bool                   `json:"editable" firestore:"editable"`
	IsCHTPricebook bool                   `json:"-" firestore:"IsCHTPricebook"`
	Assignments    []*PricebookAssignment `json:"-" firestore:"assignments"`
	Status         string                 `json:"status,omitempty" firestore:"status"`
}

type PricebookAssignment struct {
	ID                 int64                         `json:"id,omitempty" firestore:"id"`
	PricebookID        int64                         `json:"price_book_id" firestore:"pricebookId"`
	CustomerID         int64                         `json:"target_client_api_id" firestore:"customerId"`
	CreatedAt          string                        `json:"created_at,omitempty" firestore:"createdAt"`
	UpdatedAt          string                        `json:"updated_at,omitempty" firestore:"updatedAt"`
	CustomerName       string                        `json:"-" firestore:"customerName"`
	CMPCustomerID      string                        `json:"customer_id" firestore:"customerId"`
	PayerID            string                        `json:"payer_id" firestore:"payerId"`
	ContractID         string                        `json:"contract_id" firestore:"contractId"`
	AccountAssignments []*PricebookAccountAssignment `json:"-" firestore:"accountAssignments"`
}

type PricebookFSAssignment struct {
	PricebookID  int64     `json:"price_book_id" firestore:"pricebookId"`
	CustomerID   string    `json:"customer_id" firestore:"customerId"`
	PayerID      string    `json:"payer_id" firestore:"payerId"`
	ContractID   string    `json:"contract_id" firestore:"contractId"`
	AssignedAt   time.Time `json:"-" firestore:"assignedAt"`
	CustomerName string    `json:"-" firestore:"customerName"`
}

type PricebookAccountAssignment struct {
	ID                     int64    `json:"id,omitempty" firestore:"id"`
	PricebookAssignmentID  int64    `json:"price_book_assignment_id" firestore:"pricebookAssignmentId"`
	CustomerID             int64    `json:"target_client_api_id" firestore:"customerId"`
	BillingAccountOwnerID  string   `json:"billing_account_owner_id,omitempty" firestore:"billingAccountOwnerId"`
	BillingAccountOwnerIDs []string `json:"billing_account_owner_ids,omitempty" firestore:"billingAccountOwnerIds"`
}

type CHTBillingRules struct {
	XMLName   xml.Name    `xml:"CHTBillingRules" firestore:"-" json:"-"`
	CreatedBy string      `xml:"createdBy,attr" firestore:"createdBy" json:"-"`
	Date      string      `xml:"date,attr" firestore:"date" json:"-"`
	Comment   string      `xml:"Comment" firestore:"comment" json:"comment"`
	RuleGroup []RuleGroup `xml:"RuleGroup" firestore:"ruleGroups" json:"ruleGroups"`
}

type RuleGroup struct {
	XMLName     xml.Name      `xml:"RuleGroup" firestore:"-" json:"-"`
	StartDate   string        `xml:"startDate,omitempty,attr" firestore:"startDate,omitempty" json:"startDate,omitempty"`
	EndDate     string        `xml:"endDate,omitempty,attr" firestore:"endDate,omitempty" json:"endDate,omitempty"`
	Enabled     bool          `xml:"enabled,omitempty,attr" firestore:"enabled,omitempty" json:"enabled,omitempty"`
	BillingRule []BillingRule `xml:"BillingRule" firestore:"billingRules" json:"billingRules"`
}

type BillingRule struct {
	XMLName             xml.Name         `xml:"BillingRule" firestore:"-" json:"-"`
	Name                string           `xml:"name,attr" firestore:"name" json:"name"`
	IncludeDataTransfer bool             `xml:"includeDataTransfer,omitempty,attr" firestore:"includeDataTransfer,omitempty" json:"includeDataTransfer,omitempty"`
	IncludeRIPurchases  bool             `xml:"includeRIPurchases,omitempty,attr" firestore:"includeRIPurchases,omitempty" json:"includeRIPurchases,omitempty"`
	BasicBillingRule    BasicBillingRule `xml:"BasicBillingRule" firestore:"basicBillingRule" json:"basicBillingRule"`
	Product             Product          `xml:"Product,omitempty" firestore:"product,omitempty" json:"product"`
}

type BasicBillingRule struct {
	XMLName           xml.Name `xml:"BasicBillingRule" firestore:"-" json:"-"`
	BillingAdjustment float64  `xml:"billingAdjustment,attr" firestore:"billingAdjustment" json:"billingAdjustment"`
	BillingRuleType   string   `xml:"billingRuleType,attr" firestore:"billingRuleType" json:"billingRuleType"`
}

type Product struct {
	XMLName             xml.Name              `xml:"Product" firestore:"-" json:"-"`
	ProductName         string                `xml:"productName,attr" firestore:"productName" json:"productName"`
	IncludeDataTransfer bool                  `xml:"includeDataTransfer,omitempty,attr" firestore:"includeDataTransfer,omitempty" json:"includeDataTransfer,omitempty"`
	IncludeRIPurchases  bool                  `xml:"includeRIPurchases,omitempty,attr" firestore:"includeRIPurchases,omitempty" json:"includeRIPurchases,omitempty"`
	UsageType           []UsageType           `xml:"UsageType,omitempty" firestore:"usageType,omitempty" json:"usageType,omitempty"`
	Operation           []Operation           `xml:"Operation,omitempty" firestore:"operation,omitempty" json:"operation,omitempty"`
	Region              []Region              `xml:"Region,omitempty" firestore:"region,omitempty" json:"region,omitempty"`
	InstanceProperties  []InstanceProperties  `xml:"InstanceProperties,omitempty" firestore:"instanceProperties,omitempty" json:"instanceProperties,omitempty"`
	LineItemDescription []LineItemDescription `xml:"LineItemDescription,omitempty" firestore:"lineItemDescription,omitempty" json:"lineItemDescription,omitempty"`
}

type UsageType struct {
	XMLName xml.Name `xml:"UsageType" firestore:"-" json:"-"`
	Name    string   `xml:"name,attr" firestore:"name" json:"name"`
}

type Operation struct {
	XMLName xml.Name `xml:"Operation" firestore:"-" json:"-"`
	Name    string   `xml:"name,attr" firestore:"name" json:"name"`
}

type Region struct {
	XMLName xml.Name `xml:"Region" firestore:"-" json:"-"`
	Name    string   `xml:"name,attr" firestore:"name" json:"name"`
}

type InstanceProperties struct {
	XMLName      xml.Name `xml:"InstanceProperties" firestore:"-" json:"-"`
	InstanceType string   `xml:"instanceType,omitempty,attr" firestore:"instanceType,omitempty" json:"instanceType,omitempty"`
	InstanceSize string   `xml:"instanceSize,omitempty,attr" firestore:"instanceSize,omitempty" json:"instanceSize,omitempty"`
	Reserved     bool     `xml:"reserved,omitempty,attr" firestore:"reserved,omitempty" json:"reserved,omitempty"`
}

type LineItemDescription struct {
	XMLName      xml.Name `xml:"LineItemDescription" firestore:"-" json:"-"`
	StartsWith   string   `xml:"startsWith,omitempty,attr" firestore:"startsWith,omitempty" json:"startsWith,omitempty"`
	Contains     string   `xml:"contains,omitempty,attr" firestore:"contains,omitempty" json:"contains,omitempty"`
	MatchesRegex string   `xml:"matchesRegex,omitempty,attr" firestore:"matchesRegex,omitempty" json:"matchesRegex,omitempty"`
}

func GetPricebookFSCollection(fs *firestore.Client) *firestore.CollectionRef {
	return fs.Collection("integrations").Doc("cloudhealth").Collection("cloudhealthPricebooks")
}

func GetPricebooksAssignmentsFSCollection(fs *firestore.Client) *firestore.CollectionRef {
	return fs.Collection("integrations").Doc("cloudhealth").Collection("pricebooksAssignments")
}

func GetCustomerPricebooksAssignmentsQuery(fs *firestore.Client, customerID, payerID string) firestore.Query {
	return GetPricebooksAssignmentsFSCollection(fs).Where("customerId", "==", customerID).Where("payerId", "==", payerID)
}

func GetPricebookFSRef(fs *firestore.Client, pricebookID int64) *firestore.DocumentRef {
	return GetPricebookFSCollection(fs).Doc(IntToStr(pricebookID))
}

func IntToStr(number int64) string {
	return strconv.FormatInt(number, 10)
}

func GetCustomerName(ctx context.Context, fs *firestore.Client, customerID string) string {
	dsnap, err := fs.Collection("customers").Doc(customerID).Get(ctx)
	if err != nil {
		return ""
	}

	domain, err := dsnap.DataAt("primaryDomain")
	if err != nil {
		// field does not exist
		return ""
	}

	return domain.(string)
}

func CleanCustomerPayerPricebookAssigment(ctx context.Context, fs *firestore.Client, customerID, payerID string) {
	l := logger.FromContext(ctx)
	query := GetCustomerPricebooksAssignmentsQuery(fs, customerID, payerID)

	iter := query.Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err != nil {
			return
		}

		if _, err := GetPricebooksAssignmentsFSCollection(fs).Doc(doc.Ref.ID).Delete(ctx); err != nil {
			l.Errorf("Failed to delete pricebook assignment %s with error: %s", doc.Ref.ID, err)
		}
	}
}

func UpdatePricebookCustomerAssigment(ctx context.Context, fs *firestore.Client, pb PricebookFSAssignment) {
	l := logger.FromContext(ctx)

	query := GetCustomerPricebooksAssignmentsQuery(fs, pb.CustomerID, pb.PayerID)

	iter := query.Documents(ctx)
	defer iter.Stop()

	for {
		doc, _ := iter.Next()
		collection := GetPricebooksAssignmentsFSCollection(fs)

		if doc == nil {
			// This code is less likely to be executed, but it's here just in case
			// Its because for he assigment we first delete all the assigments and then we add the new one, therefore its less likley the a doucment is already exists.
			doc, _, err := collection.Add(ctx, pb)
			if err != nil && doc == nil {
				l.Errorf("Failed to add pricebook assignment with error: %s", err)
			}
		} else {
			if _, err := collection.Doc(doc.Ref.ID).Update(ctx, []firestore.Update{
				{Path: "customerId", Value: pb.CustomerID},
				{Path: "payerId", Value: pb.PayerID},
				{Path: "contractId", Value: pb.ContractID},
				{Path: "customerName", Value: pb.CustomerName},
				{Path: "pricebookID", Value: pb.PricebookID},
				{Path: "assignedAt", Value: pb.AssignedAt},
			}); err != nil {
				l.Errorf("Failed to update pricebook assignment %s with error: %s", doc.Ref.ID, err)
			}
		}

		return
	}
}

func IsPricebookCHTRelated(ctx context.Context, fs *firestore.Client, pricebookID int64) (bool, error) {
	dsnap, err := GetPricebookFSRef(fs, pricebookID).Get(ctx)
	if err != nil {
		return false, err
	}

	v, err := dsnap.DataAt("IsCHTPricebook")
	if err != nil {
		// field does not exist
		return true, nil
	}

	return v.(bool), nil
}

func mapCMPCustomerIDToCHTCustomerID(fs firestore.Client, ctx *gin.Context, CMPCustomerID string) (int64, error) {
	customerRef := fs.Collection("customers").Doc(CMPCustomerID)

	query := fs.Collection("integrations").Doc("cloudhealth").Collection("cloudhealthCustomers").Where("customer", "==", customerRef)

	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		return 0, err
	}

	CHTCustomerID, err := strconv.Atoi(doc.Ref.ID)
	if err != nil {
		return 0, err
	}

	return int64(CHTCustomerID), nil
}

func getCurrentTimeAsStr() string {
	return time.Now().Format("2006-01-02T15:04:05Z")
}

func CreatePricebook(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	if !ctx.GetBool(common.DoitOwner) {
		ctx.AbortWithError(http.StatusForbidden, errors.New("user not authorized"))
		return
	}

	var r Pricebook

	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if r.BookName == "" || r.Specification == nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	pricebooksCollection := GetPricebookFSCollection(fs)

	date := getCurrentTimeAsStr()
	email := ctx.GetString("email")
	spec := r.Specification
	spec.CreatedBy = fmt.Sprintf("%s %s", email, createdBySuffix)
	spec.Date = date

	var pricebook Pricebook

	if canCreateCHTPricebook {
		specB, err := xml.MarshalIndent(spec, "", "  ")
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		l.Info(string(specB))

		pricebook, err := createPricebook(r.BookName, string(specB))
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		l.Infof("%+v", pricebook)

		newPricebookSpecification, err := pricebook.GetSpecification()
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		l.Infof("%+v", newPricebookSpecification)

		pricebook.Specification = newPricebookSpecification
		pricebook.IsCHTPricebook = true
	} else {
		pricebook = r

		pricebook.ID = int64(uuid.New().ID())
		pricebook.IsCHTPricebook = false
	}

	pricebook.UpdatedBy = email
	pricebook.CreatedAt = date
	pricebook.UpdatedAt = date

	if _, err := pricebooksCollection.Doc(IntToStr(pricebook.ID)).Set(ctx, pricebook); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func UpdatePricebook(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	if !ctx.GetBool(common.DoitOwner) {
		ctx.AbortWithError(http.StatusForbidden, errors.New("user not authorized"))
		return
	}

	var updatedPricebook Pricebook
	if err := ctx.ShouldBindJSON(&updatedPricebook); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	l.Info(updatedPricebook)

	if updatedPricebook.ID == 0 || updatedPricebook.BookName == "" || updatedPricebook.Specification == nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	chtRelated, err := IsPricebookCHTRelated(ctx, fs, updatedPricebook.ID)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Only allow updating the "Test pricebook" (1963155723) when not in production.
	// This is to prevent developers from changing customers billing because any
	// change to a pricebook may affect customer's billing, even if done from dev env.
	if !common.Production && updatedPricebook.ID != testPricebookID && chtRelated {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	pricebooksCollection := GetPricebookFSCollection(fs)

	date := getCurrentTimeAsStr()
	email := ctx.GetString("email")
	spec := updatedPricebook.Specification
	spec.CreatedBy = fmt.Sprintf("%s %s", email, createdBySuffix)
	spec.Date = date

	if chtRelated {
		spec.Date = time.Now().Format("2006-01-02")

		specB, err := xml.MarshalIndent(spec, "", "  ")
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		chtPricebook, err := updatePricebook(updatedPricebook.ID, updatedPricebook.BookName, string(specB), updatedPricebook.Status)
		if err != nil && chtPricebook.ID == 0 {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	updatedPricebook.UpdatedBy = email
	updatedPricebook.UpdatedAt = date

	paths := []firestore.FieldPath{[]string{"id"}, []string{"bookName"}, []string{"customerId"}, []string{"payerId"}, []string{"updatedBy"}, []string{"createdAt"}, []string{"updatedAt"}, []string{"fileHash"}, []string{"specification"}, []string{"status"}}
	if _, err := pricebooksCollection.Doc(IntToStr(updatedPricebook.ID)).Set(ctx, updatedPricebook, firestore.Merge(paths...)); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func AssignPricebook(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	var r PricebookAssignment

	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	l.Infof("%+v", r)

	chtPricebook, _ := IsPricebookCHTRelated(ctx, fs, r.PricebookID)

	// Only allow assigning the "Test pricebook" (1963155723) to budgetao.com (15372) when not in production.
	// This is to prevent developers from changing customers billing because any
	// change to a pricebook may affect customer's billing, even if done from dev env.
	if !common.Production && chtPricebook {
		if r.ID != testPricebookID || r.CustomerID != testCustomerID {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": "CHT pricebooks can only be assigned in Production"})
			return
		}
	}

	pricebookRef := GetPricebookFSRef(fs, r.PricebookID)

	if chtPricebook {
		var CHTCustomerID = r.CustomerID

		if CHTCustomerID == 0 {
			var err error
			CHTCustomerID, err = mapCMPCustomerIDToCHTCustomerID(*fs, ctx, r.CMPCustomerID)

			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err.Error()})
				return
			}
		}

		customer, err := cloudhealth.GetCustomer(CHTCustomerID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err.Error()})
			return
		}

		l.Info(customer)

		assignment, err := createPricebookAssignment(r.PricebookID, customer.ID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err.Error()})
			return
		}

		if assignment.ID == 0 {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": "assignment.ID == 0"})
			return
		}

		accountAssignments, err := createPricebookAccountAssignment(assignment.ID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err.Error()})
			return
		}

		assignment.CustomerName = customer.Name
		assignment.AccountAssignments = accountAssignments

		if _, err := pricebookRef.Update(ctx, []firestore.Update{
			{FieldPath: []string{"assignments"}, Value: firestore.ArrayUnion(assignment)},
		}); err != nil {
			l.Error(err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err.Error()})
		}
	} else {
		CleanCustomerPayerPricebookAssigment(ctx, fs, r.CMPCustomerID, r.PayerID)

		if r.PricebookID > 0 {
			customerName := GetCustomerName(ctx, fs, r.CMPCustomerID)

			var pb PricebookFSAssignment
			pb.PricebookID = r.PricebookID
			pb.CustomerID = r.CMPCustomerID
			pb.PayerID = r.PayerID
			pb.CustomerName = customerName
			pb.ContractID = r.ContractID
			pb.AssignedAt = time.Now()

			UpdatePricebookCustomerAssigment(ctx, fs, pb)
		}
	}
}

func createPricebook(bookName, specification string) (*Pricebook, error) {
	path := fmt.Sprintf("/v1/%s", pricebooksResourceName)
	v := map[string]interface{}{
		"book_name":     bookName,
		"specification": specification,
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	body, err := cloudhealth.Client.Post(path, nil, data)
	if err != nil {
		return nil, err
	}

	var response CreatePricebookResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Pricebook, nil
}

func updatePricebook(id int64, bookName, specification string, status string) (*Pricebook, error) {
	path := fmt.Sprintf("/v1/%s/%d", pricebooksResourceName, id)
	v := map[string]interface{}{
		"book_name":     bookName,
		"specification": specification,
		"status":        status,
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	body, err := cloudhealth.Client.Put(path, nil, data)
	if err != nil {
		return nil, err
	}

	var response CreatePricebookResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Pricebook, nil
}

func getPricebook(id int64) (*Pricebook, error) {
	path := fmt.Sprintf("/v1/%s/%d", pricebooksResourceName, id)

	body, err := cloudhealth.Client.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var response Pricebook
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func listPricebooks(page int64, v *[]*Pricebook) error {
	path := fmt.Sprintf("/v1/%s", pricebooksResourceName)
	params := make(map[string][]string)
	params["per_page"] = []string{"100"}
	params["page"] = []string{strconv.FormatInt(page, 10)}

	body, err := cloudhealth.Client.Get(path, params)
	if err != nil {
		return err
	}

	var response PricebooksResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.Pricebooks) > 0 {
		*v = append(*v, response.Pricebooks...)
		return listPricebooks(page+1, v)
	}

	return nil
}

func listPricebookAssignments(page int64, v *[]*PricebookAssignment) error {
	path := fmt.Sprintf("/v1/%s", pricebookAssignmentsResourceName)
	params := make(map[string][]string)
	params["per_page"] = []string{"100"}
	params["page"] = []string{strconv.FormatInt(page, 10)}

	body, err := cloudhealth.Client.Get(path, params)
	if err != nil {
		return err
	}

	var response PricebookAssignmentsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.PricebookAssignments) > 0 {
		*v = append(*v, response.PricebookAssignments...)
		return listPricebookAssignments(page+1, v)
	}

	return nil
}

func listPricebookAccountAssignments(page int64, v *[]*PricebookAccountAssignment) error {
	path := fmt.Sprintf("/v1/%s", pricebookAccountAssignmentsResourceName)
	params := make(map[string][]string)
	params["per_page"] = []string{"100"}
	params["page"] = []string{strconv.FormatInt(page, 10)}

	body, err := cloudhealth.Client.Get(path, params)
	if err != nil {
		return err
	}

	var response PricebookAccountAssignmentsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.PricebookAccountAssignments) > 0 {
		*v = append(*v, response.PricebookAccountAssignments...)
		return listPricebookAccountAssignments(page+1, v)
	}

	return nil
}

func createPricebookAssignment(pricebookID, customerID int64) (*PricebookAssignment, error) {
	path := fmt.Sprintf("/v1/%s", pricebookAssignmentsResourceName)
	v := map[string]interface{}{
		"price_book_id":        pricebookID,
		"target_client_api_id": customerID,
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	body, err := cloudhealth.Client.Post(path, nil, data)
	if err != nil {
		return nil, err
	}

	var response PricebookAssignment
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func createPricebookAccountAssignment(pricebookAssignmentID int64) ([]*PricebookAccountAssignment, error) {
	path := fmt.Sprintf("/v1/%s", pricebookAccountAssignmentsResourceName)
	v := map[string]interface{}{
		"price_book_assignment_id": pricebookAssignmentID,
		"billing_account_owner_id": "ALL",
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	body, err := cloudhealth.Client.Post(path, nil, data)
	if err != nil {
		return nil, err
	}

	var response PricebookAccountAssignmentsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.PricebookAccountAssignments, nil
}

func (pb *Pricebook) GetSpecification() (*CHTBillingRules, error) {
	path := fmt.Sprintf("/v1/%s/%d/specification", pricebooksResourceName, pb.ID)

	body, err := cloudhealth.Client.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var response PricebookSpecification
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var specification CHTBillingRules
	if err := xml.Unmarshal([]byte(response.Specification), &specification); err != nil {
		return nil, err
	}

	return &specification, nil
}
