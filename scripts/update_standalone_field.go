package scripts

import (
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"

	"github.com/doitintl/firestore/pkg"
)

const (
	// entire collection:
	fieldError    = "error"
	fieldAccounts = pkg.AccountsField

	// specific documents:
	fieldTimeActivated   = "timeActivated"
	fieldEnabledFlexsave = "enabledFlexsave"
	fieldCompleted       = "completed"
	// add the field you want to change as a const
)

type UpdateStandaloneFieldReq struct {
	Field        string            `json:"field"`
	Reverse      bool              `json:"reverse"`
	Platform     string            `json:"platform"`
	DocumentsMap map[string]string `json:"documents"` // key: documentID, value: specific value to update (can be empty in some cases)
}

type mappingFunc func(*firestore.DocumentSnapshot) (interface{}, error)
type specificDocumentMappingFunc func(*gin.Context, *firestore.Client, *firestore.DocumentSnapshot, string) error // todo simplify args?

/*
This scripts is helpful when a field (name/content) is updated or added broadly for flexsave-standalone onboarding collection OR required to add on specific documents

	/integrations/flexsave-standalone/fs-onboarding/{PLATFORM}}-{CUSTOMER_ID}

Request params:
** field - the field of which you wish to change. has to be listed as a const & one of the field of StandaloneOnboarding/GCPStandaloneOnboarding/AWSStandaloneOnboarding (required)
** reverse - whether or not to reverse the field to its previous form, relevant only if revers func is implemented (optional, default=false)
** platform - platform to assert changes on - AWS/GCP (optional, default=both)
** documents - specific documents to assert changes on. supported by specificDocumentMappingFunc only. (optional - if exist only specific docs will be touched, default=null)

When new field need to be added to the script:
1. add field name as a const
2. implement mapping function (either specific or entire collection)
3. (optional) implement reverse mapping function
3. add both field name & mapping func to the "field" switch on relevant router (specific or entire collection)
*/
func UpdateStandaloneField(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	var params UpdateStandaloneFieldReq
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.DocumentsMap != nil {
		return updateSpecificDocuments(ctx, fs, &params)
	}

	return updateEntireCollection(ctx, fs, &params)
}

/*
** router function to handle specific documents updates
when want to update specific documents, please implement mapping function by that type-

specificDocumentMappingFunc func(*gin.Context, *firestore.Client, *firestore.DocumentSnapshot, string) error {}
  - note - "documents" on request payload will be structured:
    {
    "[PLATFORM]-[CUSTOMER_ID]": "[ACCOUNT_ID]/[ADDITIONAL_INFORMATION]" // additional information not always needed
    }
  - function itself has the response for all the required updates
  - router only handles invocations and success counter
*/
func updateSpecificDocuments(ctx *gin.Context, fs *firestore.Client, params *UpdateStandaloneFieldReq) []error {
	errors := []error{}
	success := 0
	fail := 0

	field := params.Field
	documentsToUpdate := params.DocumentsMap

	var specificDocumentMappingFunc specificDocumentMappingFunc

	switch field {
	case fieldCompleted:
		specificDocumentMappingFunc = mapStateToCompleted
		break
	case fieldEnabledFlexsave:
		specificDocumentMappingFunc = mapCustomerEnabledFlexsave
		break
	case fieldTimeActivated:
		specificDocumentMappingFunc = mapTimeActivated
		break
	// add the field you want to change as a switch case
	default:
		return []error{fmt.Errorf("unrecognized field [%s] to update specific documents", field)}
	}

	for docID, value := range documentsToUpdate {
		document, err := getStandaloneCollection(fs).Doc(docID).Get(ctx)
		if err == nil {
			err = specificDocumentMappingFunc(ctx, fs, document, value)
		}

		if err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		success++
	}

	fmt.Printf("updated %d documents successfully\n", success)
	fmt.Printf("failed to update %d documents\n", fail)

	return errors
}

/*
** router function to handle entire collection updates
when want to update entire collection, please implement mapping function by that type-

	mappingFunc func(*firestore.DocumentSnapshot) (interface{}, error)

* function only responsible for handling creation/adjustments/modifications of the required filed to update
* router responsible for the actual update on firestore using batch
*/
func updateEntireCollection(ctx *gin.Context, fs *firestore.Client, params *UpdateStandaloneFieldReq) []error {
	errors := []error{}
	success := 0
	fail := 0

	field := params.Field
	platform := params.Platform
	reverse := params.Reverse

	var mappingFunc mappingFunc

	switch field {
	case fieldError:
		field = "errors"

		mappingFunc = mapErrorStateToOnboardingErrors
		if reverse {
			mappingFunc = reverseMapErrorStateToOnboardingErrors
			field = "error"
		}

		break
	case fieldAccounts:
		mappingFunc = mapAWSDocumentToMultipleAccounts

		if reverse {
			// RISKY!
			// implement update if relevant.
			// potential lost of data when customer has multiple accounts.
		}

		break
	// add the field you want to change as a switch case
	default:
		return []error{fmt.Errorf("unrecognized field [%s] to update entire collection", field)}
	}

	var query firestore.Query

	switch platform {
	case string(pkg.AWS):
		query = getStandaloneCollection(fs).Where("type", "==", pkg.AWS)
		break
	case string(pkg.GCP):
		query = getStandaloneCollection(fs).Where("type", "==", pkg.GCP)
		break
	default:
		query = getStandaloneCollection(fs).Query
	}

	batch := doitFirestore.NewBatchProviderWithClient(fs, 10).Provide(ctx)

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	for _, doc := range docSnaps {
		newField, err := mappingFunc(doc)
		if err != nil || newField == nil {
			if err != nil {
				errors = append(errors, err)
				fail++
			}

			continue
		}

		fmt.Printf("Document ID: %s New %s field: %+v\n", doc.Ref.ID, field, newField)

		if err := batch.Set(ctx, doc.Ref, map[string]interface{}{
			field: newField,
		}, firestore.MergeAll); err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		success++
	}

	if err := batch.Commit(ctx); err != nil {
		errors = append(errors, err)
	}

	fmt.Printf("updated %d documents successfully\n", success)
	fmt.Printf("failed to update %d documents\n", fail)

	return errors
}

func getStandaloneCollection(fs *firestore.Client) *firestore.CollectionRef {
	return fs.Collection("integrations").Doc("flexsave-standalone").Collection("fs-onboarding")
}

func getNestedPath(accountID, field string) string {
	if accountID == "" {
		return ""
	}

	return fmt.Sprintf("accounts.%s.%s", accountID, field)
}

// *** MAPPING FUNCTIONS (specific documents) ***:

/*
Add {timeActivated:timestamp} to documents which has {completed:true}

  - payload:
    "field": "timeActivated",
    "documents": {
    "amazon-web-services-CUSTOMER_ID": "ACCOUNT_ID/TIMESTAMP" // ex: "amazon-web-services-Lv86gBf5roruvKGB5poN": "281727049056/2022-04-26",
    }

  - updates (fs-onboarding collection):

    {
    accounts:
    {
    [ACCOUNT_ID]: {
    timeActivated: timestamp
    completed: true (required)
    }
    }
    }
*/
func mapTimeActivated(ctx *gin.Context, fs *firestore.Client, doc *firestore.DocumentSnapshot, value string) error {
	valuesArr := strings.Split(value, "/")

	accountID := valuesArr[0]
	if accountID == "" {
		return fmt.Errorf("empty accountID (failed to parse value for doc %s)", doc.Ref.ID)
	}

	timestampStr := valuesArr[1]
	if timestampStr == "" {
		return fmt.Errorf("empty timestamp for timeActivated (failed to parse value for doc %s)", doc.Ref.ID)
	}

	var document pkg.AWSStandaloneAccounts
	if err := doc.DataTo(&document); err != nil {
		return err
	}

	if document.Accounts != nil && document.Accounts[accountID] != nil && document.Accounts[accountID].Completed {
		timestamp, err := time.Parse("2006-01-02", timestampStr)
		if err != nil {
			return err
		}

		fields := []firestore.Update{
			{Path: getNestedPath(accountID, "timeActivated"), Value: timestamp},
		}
		if _, err := doc.Ref.Update(ctx, fields); err != nil {
			return err
		}

		fmt.Printf("updated document %s with timeActivated:%v\n", doc.Ref.ID, timestamp)
	}

	return nil
}

/*
Add {enabledFlexsave.AWS:true} on customer for documents which has {completed:true}

  - payload:
    "field": "enabledFlexsave",
    "documents": {
    "amazon-web-services-CUSTOMER_ID": "ACCOUNT_ID" // ex: "amazon-web-services-Lv86gBf5roruvKGB5poN": "281727049056",
    }

  - updates (customers collection):

    {
    enabledFlexsave: {
    AWS: true,
    GCP: boolean
    }
    }
*/
func mapCustomerEnabledFlexsave(ctx *gin.Context, fs *firestore.Client, doc *firestore.DocumentSnapshot, value string) error {
	accountID := value
	if accountID == "" {
		return fmt.Errorf("empty accountID (no value given for doc %s)", doc.Ref.ID)
	}

	var document pkg.AWSStandaloneAccounts
	if err := doc.DataTo(&document); err != nil {
		return err
	}

	if document.Accounts != nil && document.Accounts[accountID] != nil && document.Accounts[accountID].Completed {
		customerDoc, err := document.Accounts[accountID].Customer.Get(ctx)
		if err != nil {
			return err
		}

		enabledFlexsave := common.CustomerEnabledFlexsave{
			AWS: true,
		}

		fields := []firestore.Update{
			{Path: "enabledFlexsave", Value: enabledFlexsave},
		}
		if _, err := customerDoc.Ref.Update(ctx, fields); err != nil {
			return err
		}

		fmt.Printf("updated customer document %s with enabledFlexsave.AWS:true due to completion of %s\n", customerDoc.Ref.ID, doc.Ref.ID)
	}

	return nil
}

/*
Add {completed:true} to documents which has {state:"completed"}

  - payload:
    "field": "completed",
    "documents": {
    "amazon-web-services-CUSTOMER_ID": "ACCOUNT_ID" // ex: "amazon-web-services-Lv86gBf5roruvKGB5poN": "281727049056",
    }

  - updates (fs-onboarding collection):

    {
    accounts:
    {
    [ACCOUNT_ID]: {
    completed: true
    }
    },
    state: "completed" (required)
    }
*/
func mapStateToCompleted(ctx *gin.Context, fs *firestore.Client, doc *firestore.DocumentSnapshot, value string) error {
	accountID := value
	if accountID == "" {
		return fmt.Errorf("empty accountID (no value given for doc %s)", doc.Ref.ID)
	}

	structWithState := struct {
		State string `firestore:"state"`
	}{}

	if err := doc.DataTo(&structWithState); err != nil {
		return err
	}

	var document pkg.AWSStandaloneAccounts
	if err := doc.DataTo(&document); err != nil {
		return err
	}

	if structWithState.State == "completed" &&
		document.Accounts != nil && document.Accounts[accountID] != nil && !document.Accounts[accountID].Completed {
		fields := []firestore.Update{
			{Path: getNestedPath(accountID, "completed"), Value: true},
		}
		if _, err := doc.Ref.Update(ctx, fields); err != nil {
			return err
		}

		fmt.Printf("updated document %s with completed:true\n", doc.Ref.ID)
	}

	return nil
}

// *** MAPPING FUNCTIONS (entire collection) ***:

/*
Change to AWSStandaloneAccounts (root document)
* From	--> AWSStandaloneOnboarding (struct with all standalone information per account)
* To	--> AWSStandaloneAccounts: (map-->  key: aws account id, value: standalone information per account)
{
	type: string
	customer: documentRef
	accounts: map[accountID]*AWSStandaloneOnboarding
}
*/
// note - this is a "semi" mapping. Existing data will persist
func mapAWSDocumentToMultipleAccounts(doc *firestore.DocumentSnapshot) (interface{}, error) {
	var newStruct pkg.AWSStandaloneAccounts
	if newStructErr := doc.DataTo(&newStruct); newStructErr == nil && newStruct.Accounts != nil {
		return nil, fmt.Errorf("document %s is already in the updated form :-)", doc.Ref.ID)
	}

	var document pkg.AWSStandaloneOnboarding
	if err := doc.DataTo(&document); err != nil {
		return nil, err
	}

	if document.AccountID == "" {
		return nil, fmt.Errorf("no account id for document %s thus cannot be used in a multiple accounts structure", doc.Ref.ID)
	}

	accounts := map[string]*pkg.AWSStandaloneOnboarding{
		document.AccountID: &document,
	}

	return accounts, nil
}

// not in use unless wiping out old data
func reverseMapAWSDocumentToMultipleAccounts(doc *firestore.DocumentSnapshot) (interface{}, error) {
	var oldStruct pkg.AWSStandaloneOnboarding
	if oldStructErr := doc.DataTo(&oldStruct); oldStructErr == nil && !oldStruct.LastUpdated.IsZero() {
		return nil, fmt.Errorf("document %s is already in the old form :-)", doc.Ref.ID)
	}

	var document pkg.AWSStandaloneAccounts
	if err := doc.DataTo(&document); err != nil {
		return nil, err
	}

	if len(document.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts found in document %s thus cannot be used in the old structure", doc.Ref.ID)
	}

	var newDoc *pkg.AWSStandaloneOnboarding
	for _, account := range document.Accounts {
		newDoc = account
		break
	}

	// todo implement fs.Update for entire struct if relevant...

	return newDoc, nil
}

/*
Change to AWSStandaloneOnboarding.Error field
* From 	--> StandaloneOnboardingErrorState (string of a pre known type)
* To 	--> *StandaloneOnboardingError:

	{
		message: string
		state: string
	}
*/
func mapErrorStateToOnboardingErrors(doc *firestore.DocumentSnapshot) (interface{}, error) {
	oldToNewErrorStates := map[string]string{ // StandaloneOnboardingErrorState old and new
		"recommendations-permissions": "savings",
		"savings-calculation":         "savings",
		"wrong-aws-params":            "savings",
		"config-error":                "savings",
		"contract-agreement":          "contractAgreement",
		"access-permissions":          "activation",
	}

	document := struct {
		Error       string    `firestore:"error"`
		LastUpdated time.Time `firestore:"lastUpdated"`
	}{}

	var newStruct pkg.BaseStandaloneOnboarding
	if newStructErr := doc.DataTo(&newStruct); newStructErr == nil && newStruct.Errors != nil {
		return nil, fmt.Errorf("document %s is already in the updated form :-)", doc.Ref.ID)
	}

	if err := doc.DataTo(&document); err != nil {
		return nil, err
	}

	newError := oldToNewErrorStates[document.Error]
	if newError == "" {
		return nil, nil
	}

	onboardingErrors := map[string]*pkg.StandaloneOnboardingError{
		newError: &pkg.StandaloneOnboardingError{
			Message:         document.Error,
			TimeLastUpdated: document.LastUpdated,
		},
	}

	return onboardingErrors, nil
}

/*
REVERS Changes to AWSStandaloneOnboarding.Error field (revers it to the old form)
* From --> *StandaloneOnboardingError:
{
	message: string
	state: string
}
* To --> StandaloneOnboardingErrorState (string of a pre known type)
*/
// todo support new struct of multiple accounts
func reverseMapErrorStateToOnboardingErrors(doc *firestore.DocumentSnapshot) (interface{}, error) {
	newToOldErrorStates := map[string]string{ // StandaloneOnboardingErrorState new to old
		"initOnboarding":    "config-error",
		"billingProfile":    "savings-calculation",
		"savings":           "savings-calculation",
		"contractAgreement": "contract-agreement",
		"activation":        "access-permissions",
	}

	var document pkg.BaseStandaloneOnboarding
	if err := doc.DataTo(&document); err != nil {
		oldStruct := struct {
			Error string `firestore:"error"`
		}{}
		if oldStructErr := doc.DataTo(&oldStruct); oldStructErr == nil {
			return nil, fmt.Errorf("document %s is already in the old form :-)", doc.Ref.ID)
		}

		return nil, err
	}

	var latestStep string

	var latestUpdate time.Time

	var latestError *pkg.StandaloneOnboardingError

	for step, onboardingError := range document.Errors {
		if onboardingError.TimeLastUpdated.After(latestUpdate) {
			latestUpdate = onboardingError.TimeLastUpdated
			latestStep = step
			latestError = onboardingError
		}
	}

	oldError := newToOldErrorStates[latestStep]

	if latestError != nil && latestError.Message != "" {
		fmt.Printf("error message that has been deleted: %s\n", latestError.Message)
	}

	return oldError, nil
}
