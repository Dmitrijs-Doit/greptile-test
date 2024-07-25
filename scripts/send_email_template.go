// csv format should be:
// email,cc
// email1;email2,cc1;cc2
// email3;email4,cc3;cc4

package scripts

import (
	"encoding/csv"
	"os"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/gin-gonic/gin"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Request struct {
	FilePath    string `json:"filePath"`
	TemplateID  string `json:"templateID"`
	FromName    string `json:"fromName"`
	FromAddress string `json:"fromAddress"`
}

func EmailFromCSV(ctx *gin.Context) []error {
	dryRun := ctx.Query("dryRun") == "true"

	var req Request
	if err := ctx.BindJSON(&req); err != nil {
		return []error{err}
	}

	from := mail.NewEmail(req.FromName, req.FromAddress)

	var errors []error

	sendgridClient := sendgrid.NewSendClient(mailer.Config.APIKey)

	l := logger.FromContext(ctx)
	l.Debugf("Dry run: %t", dryRun)

	file, err := os.Open(req.FilePath)
	if err != nil {
		return []error{err}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()

	if err != nil {
		return []error{err}
	}

	var recipients, ccAddresses []string
	for i, record := range records {

		if i == 0 {
			// Skip the header row
			continue
		}

		recipients = append(recipients, record[0])
		ccAddresses = append(ccAddresses, record[1])
	}

	for i, r := range recipients {
		p := mail.NewPersonalization()

		// split on ,
		tos := strings.Split(r, ";")
		ccs := strings.Split(ccAddresses[i], ";")

		for _, r := range tos {
			to := mail.Email{
				Address: r,
				Name:    r,
			}
			p.AddTos(&to)
		}

		for _, cc := range ccs {
			cc := mail.Email{
				Address: cc,
				Name:    cc,
			}
			p.AddCCs(&cc)
		}

		m := mail.NewV3Mail()
		m.SetFrom(from)
		m.SetTemplateID(req.TemplateID)
		m.AddPersonalizations(p)

		if dryRun {
			l.Infof("Would send email %v", m)
			continue
		}

		_, err := sendgridClient.Send(m)
		if err != nil {
			errors = append(errors, err)
			continue
		}

	}

	return errors

}
