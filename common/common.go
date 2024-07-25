package common

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/pubsub"
	"github.com/gin-gonic/gin"
	"golang.org/x/term"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
)

// Bytes!
const (
	KibiByte int64 = 1024
	MebiByte int64 = 1024 * KibiByte
	GibiByte int64 = 1024 * MebiByte
	TebiByte int64 = 1024 * GibiByte
)

var (
	// Assets types
	Assets struct {
		GoogleCloud                  string
		GoogleCloudDirect            string
		GoogleCloudProject           string
		GoogleCloudProjectStandalone string
		GoogleCloudReseller          string
		GoogleCloudStandalone        string
		AmazonWebServices            string
		AmazonWebServicesReseller    string
		AmazonWebServicesStandalone  string
		GSuite                       string
		Office365                    string
		MicrosoftAzure               string
		MicrosoftAzureReseller       string
		MicrosoftAzureStandalone     string
		Zendesk                      string
		BetterCloud                  string
		Looker                       string
		DoiTNavigator                string
		DoiTSolve                    string
		DoiTSolveAccelerator         string
	}

	CtxKeys struct {
		UserID       string
		Email        string
		DoitEmployee string
		DoitOwner    string
		DoitPartner  string
		TenantID     string
		Name         string
		CustomerID   string
		Claims       string
		UID          string
	}

	PubSub *pubsub.Client

	// Cloud task
	ct *cloudtasks.Client

	ProjectID string

	ProjectNumber string

	Domain string

	GAEService string

	GAEVersion string

	Env string

	// Production flag indicating if app is running the production backend on appengine
	Production bool

	// IsLocalhost flag indicating if app is running on localhost
	IsLocalhost bool

	// firestore client singleton
	fs *firestore.Client

	appEngineURLFormat = "https://%s-dot-%s.uc.r.appspot.com"

	location = "us-central1"

	APIGateway string
)

const (
	MicrosoftAzureOfferID = "MS-AZR-0145P"

	MicrosoftAzurePlanOfferIDPrefix = "DZH318Z0BPS6:0001:DZH318Z0B"

	productionProject = "me-doit-intl-com"

	DoitEmployee = "doitEmployee"

	DoitOwner = "doitOwner"

	DoitCustomerID = "EE8CtpzYiKp0dVAESVrB"

	ClaimsTenantID = "tenantId"

	CMP = "cloud-management-platform"

	E2ETestCustomerID = "dyg2p3K5oBxC0jNVh6vo"

	E2ETestBillingAccountID = "E2EE2E-E2EE2E-E2EE2E"

	DoitPartner = "doitPartner"

	TestProjectID = "doitintl-cmp-dev"
)

// Firebase query operators
const (
	ArrayContains    = "array-contains"
	ArrayContainsAny = "array-contains-any"
	NotIn            = "not-in"
	In               = "in"
)

const (
	DayDuration        = 24 * time.Hour
	FinalBillingDayAWS = 10
	FinalBillingDayGCP = 3
)

func initEnvVariables() {
	ProjectID = GetEnv("GOOGLE_CLOUD_PROJECT", "")

	if ProjectID == "" {
		log.Fatalln("environment variable GOOGLE_CLOUD_PROJECT is not set")
	}

	IsLocalhost = gin.Mode() != gin.ReleaseMode
	GAEService = GetEnv("GAE_SERVICE", "scheduled-tasks")
	GAEVersion = GetEnv("GAE_VERSION", "localhost")

	if value := os.Getenv("FIRESTORE_EMULATOR_HOST"); value != "" {
		log.Printf("Using Firestore Emulator: %s", value)
	}

	canConnectToProduction := !IsLocalhost

	if ProjectID == productionProject && !canConnectToProduction && term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println("You are attempting to connect to the production environment. Types 'yes' to confirm.")

		response, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}

		if strings.TrimSpace(response) == "yes" {
			// User intends to connect to production; start a timer to terminate after 30 mins
			fmt.Println("Connecting to production. The application will close after 30 minutes.")

			go func() {
				// Wait for 30 minutes
				time.Sleep(30 * time.Minute)
				fmt.Println("Time's up! Closing the application.")
				os.Exit(0) // This terminates the program
			}()

			canConnectToProduction = true
		} else {
			fmt.Println("Exiting application.")
			os.Exit(0)
		}
	}

	APIGateway = fmt.Sprintf("https://scheduled-tasks-dot-%s.appspot.com", ProjectID)

	switch {
	case ProjectID == productionProject && canConnectToProduction:
		Env = "production"
		ProjectNumber = "135469130251"
		Production = true
		Domain = "console.doit.com"
	case ProjectID == "flexsave":
		appEngineURLFormat = "https://%s-dot-%s.nw.r.appspot.com"
		location = "europe-west2"

		fallthrough
	default:
		Env = "development"
		ProjectNumber = "772991852481"
		Production = false
		Domain = "dev-app.doit.com"
	}
}

func init() {
	var err error

	ctx := context.Background()

	initEnvVariables()

	Assets.GoogleCloud = "google-cloud"
	Assets.GoogleCloudDirect = "google-cloud-direct"
	Assets.GoogleCloudProject = "google-cloud-project"
	Assets.GoogleCloudProjectStandalone = "google-cloud-project-standalone"
	Assets.GoogleCloudReseller = "google-cloud-reseller"
	Assets.GoogleCloudStandalone = "google-cloud-standalone"
	Assets.AmazonWebServices = "amazon-web-services"
	Assets.AmazonWebServicesReseller = "amazon-web-services-reseller"
	Assets.AmazonWebServicesStandalone = "amazon-web-services-standalone"
	Assets.GSuite = "g-suite"
	Assets.Zendesk = "zendesk"
	Assets.Office365 = "office-365"
	Assets.MicrosoftAzure = "microsoft-azure"
	Assets.MicrosoftAzureReseller = "microsoft-azure-reseller"
	Assets.MicrosoftAzureStandalone = "microsoft-azure-standalone"
	Assets.BetterCloud = "bettercloud"
	Assets.Looker = "looker"
	Assets.DoiTNavigator = "navigator"
	Assets.DoiTSolve = "solve"
	Assets.DoiTSolveAccelerator = "solve-accelerator"

	CtxKeys.UserID = "userId"
	CtxKeys.Email = "email"
	CtxKeys.DoitEmployee = "doitEmployee"
	CtxKeys.DoitOwner = "doitOwner"
	CtxKeys.CustomerID = "customerId"
	CtxKeys.DoitPartner = "doitPartner"
	CtxKeys.TenantID = "tenantId"
	CtxKeys.Name = "name"
	CtxKeys.Claims = "claims"
	CtxKeys.UID = "uid"

	PubSub, err = pubsub.NewClient(ctx, ProjectID)
	if err != nil {
		log.Fatalln(err)
	}

	ct, err = cloudtasks.NewClient(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	fs, err = firestore.NewClient(ctx, ProjectID)
	if err != nil {
		log.Fatalln(err)
	}
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func FormatAssetType(assetType string) string {
	switch assetType {
	case Assets.GSuite:
		return "Google Workspace"

	case Assets.GoogleCloud:
		return "Google Cloud"

	case Assets.GoogleCloudProject:
		return "Google Cloud Project"

	case Assets.AmazonWebServices:
		return "Amazon Web Services"

	case Assets.MicrosoftAzure:
		return "Microsoft Azure"

	case Assets.Office365:
		return "Office 365"

	case Assets.BetterCloud:
		return "BetterCloud"

	case Assets.Zendesk:
		return "Zendesk"

	default:
		return assetType
	}
}

func String(v string) *string {
	return &v
}

func Int(v int) *int {
	return &v
}

func Int64(v int64) *int64 {
	return &v
}

func Bool(v bool) *bool {
	return &v
}

// Float
func Float(v float64) *float64 {
	return &v
}

func EnableService(ctx context.Context, projectNumber int64, serviceName string, creds option.ClientOption) error {
	resourceName := fmt.Sprintf("projects/%d/services/%s.googleapis.com", projectNumber, serviceName)

	serviceusageService, err := serviceusage.NewService(ctx, creds)
	if err != nil {
		return err
	}

	getResp, err := serviceusageService.Services.Get(resourceName).Context(ctx).Do()
	if err != nil || getResp == nil {
		return err
	}

	if getResp.State != "ENABLED" {
		inputReq := &serviceusage.EnableServiceRequest{}
		if _, err := serviceusageService.Services.Enable(resourceName, inputReq).Context(ctx).Do(); err != nil {
			return err
		}
	}

	return nil
}

func MsToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	if msInt > 4000000000000 {
		return time.Time{}, errors.New("Time max value is 4000000000000 Milliseconds")
	}

	return EpochMillisecondsToTime(msInt), nil
}

func EpochMillisecondsToTime(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond)).UTC()
}

func ToUnixMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func MakeTimestamp() int64 {
	return ToUnixMillis(time.Now())
}

func GetFirestoreClient(_ context.Context) *firestore.Client {
	return fs
}

var runeSet = []rune("0123456789abcdefghijklmnopqrstuvwxyz")

func RandomSequenceN(n int) string {
	return RandomSequenceNWithRune(n, runeSet)
}

func RandomSequenceNWithRune(n int, set []rune) string {
	suffix := make([]rune, n)
	for i := range suffix {
		suffix[i] = set[rand.Intn(len(set))]
	}

	return string(suffix)
}

// RunConcurrentJobsOnCollection: Loop on firestore collection with 5 cuncurrent goroutines and execute the callback
func RunConcurrentJobsOnCollection(ctx context.Context, collection []*firestore.DocumentSnapshot, maxNbConcurrentGoroutines int, job func(ctx context.Context, doc *firestore.DocumentSnapshot)) {
	concurrentGoroutines := make(chan struct{}, maxNbConcurrentGoroutines)

	var wg sync.WaitGroup

	for _, doc := range collection {
		wg.Add(1)

		go func(ctx context.Context, doc *firestore.DocumentSnapshot) {
			defer wg.Done()

			concurrentGoroutines <- struct{}{}

			job(ctx, doc)

			<-concurrentGoroutines
		}(ctx, doc)
	}

	wg.Wait()
	close(concurrentGoroutines)
}

func IsNil(v interface{}) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
}

const (
	newDoitDomain = "@doit.com"
	oldDoitDomain = "@doit-intl.com"
)

func IsDoitDomain(email string) bool {
	return strings.HasSuffix(email, newDoitDomain) || strings.HasSuffix(email, oldDoitDomain)
}

func IsSameCloudAssetType(a string, b string) bool {
	if a == b {
		return true
	}

	if (a == Assets.GoogleCloud || a == Assets.GoogleCloudProject) &&
		(b == Assets.GoogleCloud || b == Assets.GoogleCloudProject) {
		return true
	}

	return false
}
