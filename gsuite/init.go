package gsuite

import (
	"context"
	"log"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	reseller "google.golang.org/api/reseller/v1"
)

var (
	Subjects = [...]string{"vadim@doit-g.co.il", "admin@premier.doit-intl.com", "admin@msp.doit-intl.com", "admin@na.doit-intl.com"}

	Resellers = make([]*reseller.Service, 0)
)

func init() {
	ctx := context.Background()

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretGSuiteReseller)
	if err != nil {
		log.Fatalln(err)
	}

	for _, subject := range Subjects {
		resellerConfig, err := google.JWTConfigFromJSON(data, reseller.AppsOrderReadonlyScope)
		if err != nil {
			log.Fatalln(err)
		}

		resellerConfig.Subject = subject
		resellerClient := resellerConfig.Client(oauth2.NoContext)

		resellerService, err := reseller.New(resellerClient)
		if err != nil {
			log.Fatalln(err)
		}

		Resellers = append(Resellers, resellerService)
	}
}
