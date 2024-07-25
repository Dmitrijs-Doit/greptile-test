package invoices

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slice"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/iterator"
)

type OverdueCustomer struct {
	CustomerID           string
	CustomerLegalName    string
	CustomerContactName  string
	CustomerLegalAddress string
	CustomerContactEmail string
	UsersEmail           []string
	DateNow              time.Time
	InvoicesID           []string
	InvoicesDocID        []string
	ProductType          []string
	UnpaidInvoices       []string
	PaymentDays          int
	DEBIT                float64
	TotalDebit           map[string]float64
	SYMBOL               string
	AccountHolderName    string
	SharedDriveFolderID  string
	HolderBankDetails    BankDetails
	AccountManagerRef    *firestore.DocumentRef
	skipRemedyBreach     bool
	domain               string
	PdfUrls              []*ExternalFile
	EntityID             string
}

type BankDetails struct {
	CompanyID           string         `firestore:"companyId"`
	CompanyName         string         `firestore:"companyName"`
	Countries           []string       `firestore:"countries"`
	WireTransferDetails []WireTransfer `firestore:"wireTransfer"`
}

type WireTransfer struct {
	Key   string `firestore:"key"`
	Value string `firestore:"value"`
}

type BankIdentifier struct {
	ID   string `firestore:"id"`
	Type string `firestore:"type"`
}

type CompaniesBank struct {
	Compnies []BankDetails `firestore:"companies"`
}

const (
	pdfMimeType         string = "application/pdf"
	teamDriveID         string = "0APq0GFH5dedFUk9PVA"
	parentFolder        string = "17kfzlhsdla3ZtwaYXla_5F3VZ-kQxwBN"
	folderName          string = "legal"
	folderMimeType      string = "application/vnd.google-apps.folder"
	fileName            string = "Notice of Remedy - Your Account is at Risk of Termination"
	emailSubject        string = "Notice To Remedy Breaches of Contract"
	serviceAccountEmail string = "google-drive@me-doit-intl-com.iam.gserviceaccount.com"
	docTemplateID       string = "10-x9owY0UgjK4qmB7pk1tgkKc8dr3Qj6zfS8GYrB5pU"
	senderName          string = "DoiT Collection Team"
	senderEmail         string = "warren@doit-intl.com"
)

func NoticeToRemedy(ctx *gin.Context) {
	fs := common.GetFirestoreClient(ctx)

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretGoogleDrive)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	serviceConfig, err := google.JWTConfigFromJSON(data, drive.DriveFileScope, drive.DriveScope, drive.DriveAppdataScope, drive.DriveMetadataScope)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	client := serviceConfig.Client(ctx)

	driveService, err := drive.New(client)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	customers := getOverdueCustomers(ctx)

	totalPerDaySnap, err := fs.Collection("app").Doc("notice-to-remedy").Get(ctx)
	totalPerDay, err := totalPerDaySnap.DataAt("totalPerDay")
	// emails, err := totalPerDaySnap.DataAt("emails")
	// senderEmailFs, err := totalPerDaySnap.DataAt("senderEmail")

	counter := 0

	for _, customerOverdue := range customers {
		if customerOverdue.skipRemedyBreach == false {
			if counter < int(totalPerDay.(int64)) {
				counter++
			} else {
				return
			}

			foldersList, err := driveService.Files.List().Q(fmt.Sprintf("'%s' in parents", customerOverdue.SharedDriveFolderID)).Corpora("drive").SupportsAllDrives(true).IncludeItemsFromAllDrives(true).DriveId(teamDriveID).Do()
			if err != nil {
				fmt.Println(err)
				return
			}

			needToCreateFolder := true

			var folderID string

			for _, myFile := range foldersList.Files {
				if myFile.Name == folderName {
					needToCreateFolder = false
					folderID = myFile.Id
				}
			}

			if needToCreateFolder == true {
				folder, err := driveService.Files.Create(&drive.File{
					MimeType:    folderMimeType,
					Parents:     []string{customerOverdue.SharedDriveFolderID},
					Name:        folderName,
					TeamDriveId: teamDriveID,
				}).SupportsAllDrives(true).Do()
				if err != nil {
					fmt.Println(err)
					return
				}

				folderID = folder.Id
			}

			permission := &drive.Permission{
				EmailAddress: serviceAccountEmail,
				Role:         "writer",
				Type:         "user",
			}

			f, err := driveService.Files.Copy(docTemplateID, &drive.File{
				Parents:     []string{folderID},
				TeamDriveId: teamDriveID,
				Name:        fileName,
			}).SupportsAllDrives(true).Do()
			if err != nil {
				fmt.Println(err)

				return
			}

			driveService.Permissions.Create(f.Id, permission).Do()
			fmt.Println(folderID)

			docsService, err := docs.New(client)
			if err != nil {
				fmt.Println(err)
				return
			}

			docID := f.Id

			doc, err := docsService.Documents.Get(docID).Do()
			if err != nil {
				fmt.Println(err)
				return
			}

			//fmt.Println(doc.DocumentId)
			fmt.Println(customerOverdue.CustomerID)
			fmt.Println(customerOverdue.TotalDebit)
			fmt.Println(customerOverdue.InvoicesID)

			totalString := ""

			for k, v := range customerOverdue.TotalDebit {
				debitNumber := numberToString(int(v), ',')
				totalString += debitNumber + " " + k + " "
			}

			fmt.Println(totalString)
			fmt.Println("=============================")
			batchUpdateRequest := &docs.BatchUpdateDocumentRequest{
				Requests: []*docs.Request{
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{customer_legal_name}}",
							},
							ReplaceText: customerOverdue.CustomerLegalName,
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{customer_contact_name}}",
							},
							ReplaceText: customerOverdue.CustomerContactName,
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{customer_legal_address}}",
							},
							ReplaceText: customerOverdue.CustomerLegalAddress,
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{date}}",
							},
							ReplaceText: customerOverdue.DateNow.Format("02/01/2006"),
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{products_type}}",
							},
							ReplaceText: strings.Join(customerOverdue.ProductType, ", "),
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{unpaid_invoices}}",
							},
							ReplaceText: strings.Join(customerOverdue.InvoicesID, ", "),
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{payment_days}}",
							},
							ReplaceText: fmt.Sprintf("%d", customerOverdue.PaymentDays),
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{DEBIT}}",
							},
							ReplaceText: totalString,
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{account_holder_name}}",
							},
							ReplaceText: customerOverdue.HolderBankDetails.CompanyName,
						},
					},
					&docs.Request{
						ReplaceAllText: &docs.ReplaceAllTextRequest{
							ContainsText: &docs.SubstringMatchCriteria{
								MatchCase: false,
								Text:      "{{bankDetails}}",
							},
							ReplaceText: getBankAccountInformation(customerOverdue.HolderBankDetails.WireTransferDetails),
						},
					},
				},
			}

			_, err = docsService.Documents.BatchUpdate(doc.DocumentId, batchUpdateRequest).Do()
			if err != nil {
				fmt.Println(err)
				return
			}

			newDoc, err := docsService.Documents.Get(doc.DocumentId).Do()
			if err != nil {
				fmt.Println(err)
				return
			}

			pdfFile, err := driveService.Files.Export(newDoc.DocumentId, pdfMimeType).Download()
			if err != nil {
				fmt.Println(err)
				return
			}

			bodyPdfByte, err := io.ReadAll(pdfFile.Body)
			encoded := base64.StdEncoding.EncodeToString([]byte(bodyPdfByte))

			var invoiceAttachments []mailer.InvoiceAttachments

			for _, exFile := range customerOverdue.PdfUrls {
				if exFile.URL != nil {
					client := &http.Client{}
					req, err := http.NewRequest("GET", *exFile.URL, nil)
					req.Header.Set("User-Agent", "Mozilla/5.0")

					resp, err := client.Do(req)
					if err != nil {
						log.Fatalln(err)
					}

					defer resp.Body.Close()
					bodyInvoicePdf, err := io.ReadAll(resp.Body)
					encodedFile := base64.StdEncoding.EncodeToString([]byte(bodyInvoicePdf))
					invoiceAttachments = append(invoiceAttachments, mailer.InvoiceAttachments{
						PdfFile: encodedFile,
						Key:     *exFile.Key,
					})
				}
			}

			var body string

			removeCustomerDetails := 4

			for _, paragraph := range newDoc.Body.Content {
				if paragraph != nil && paragraph.Paragraph != nil {
					for i := 0; i < len(paragraph.Paragraph.Elements); i++ {
						removeCustomerDetails--
						if removeCustomerDetails <= 0 {
							inputText := paragraph.Paragraph.Elements[i].TextRun.Content
							if strings.Contains(paragraph.Paragraph.Elements[i].TextRun.Content, "following invoice numbers:") {
								index := strings.Index(inputText, "numbers:") + 9

								inputTextFmt := inputText[:index]
								for _, invoiceID := range customerOverdue.InvoicesID {
									inputTextFmt += "<a href='https://console.doit.com/customers/" + customerOverdue.CustomerID + "/invoices/" + customerOverdue.EntityID + "/" + invoiceID + "' >" + invoiceID + "</a>, "
								}

								inputText = inputTextFmt[:len(inputTextFmt)-2]
							}

							body += inputText
						}
					}

					if removeCustomerDetails <= 0 {
						body += "<br/>"
					}
				}
			}

			if common.Production {
				sendEmailToCustomer(ctx, body, encoded, customerOverdue, invoiceAttachments)
				markInvoice(ctx, customerOverdue)
			}
		}
	}
}

func sendEmailToCustomer(ctx *gin.Context, body string, pdfFileEncoded string, customerObj OverdueCustomer, pdfInvoices []mailer.InvoiceAttachments) {
	fs := common.GetFirestoreClient(ctx)

	ccs := []string{"vadim@doit.com", "warren@doit-intl.com", "noam@doit-intl.com", "dina@doit-intl.com", "avi.i@doit-intl.com"}

	if customerObj.AccountManagerRef != nil {
		aManger, err := fs.Collection("accountManagers").Doc(customerObj.AccountManagerRef.ID).Get(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		email, _ := aManger.DataAt("email")
		ccs = append(ccs, email.(string))
	}

	fmt.Println(customerObj.domain)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayString := today.Format("2006-01-02")
	subject := fmt.Sprintf("%s/%s %s ", todayString, customerObj.domain, emailSubject)
	categories := []string{mailer.CatagoryContractsBreach}
	sn := &mailer.SimpleNotification{
		Subject:    subject,
		Body:       body,
		CCs:        ccs, //ccs
		Attachment: pdfFileEncoded,
		Categories: categories,
	}

	tos := []string{customerObj.CustomerContactEmail}
	for _, userEmail := range customerObj.UsersEmail {
		if !slice.Contains(tos, userEmail) {
			tos = append(tos, userEmail)
		}
	}

	if common.Production {
		mailer.SendSimpleEmailWithTemplate(sn, tos, senderName, senderEmail, pdfInvoices, mailer.Config.DynamicTemplates.NoticeToRemedy)
	}
}

func markInvoice(ctx *gin.Context, customerOverdue OverdueCustomer) {
	fs := common.GetFirestoreClient(ctx)

	for _, docID := range customerOverdue.InvoicesDocID {
		fmt.Println(docID)
		fs.Collection("invoices").Doc(docID).Set(ctx, map[string]interface{}{
			"isNoticeToRemedySent": true,
		}, firestore.MergeAll)
	}
}

func getOverdueCustomers(ctx *gin.Context) map[string]OverdueCustomer {
	fs := common.GetFirestoreClient(ctx)

	var snapshotInvoicesID = make(map[string][]string)

	var usersEmail = make(map[string][]string)

	var customersOverdueInovices = make(map[string]OverdueCustomer)

	var totalDebits = make(map[string]map[string]float64)

	requiredPermissions := []string{string(common.PermissionBillingProfiles), string(common.PermissionInvoices)}

	companiesSnap, err := fs.Collection("app").Doc("priority-v2").Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return nil
	}

	var companies CompaniesBank

	if err := companiesSnap.DataTo(&companies); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return nil
	}

	now := time.Now()
	overdueDate := now.AddDate(0, 0, -90)

	iter := fs.Collection("invoices").Where("PAID", "==", false).Where("PAYDATE", "<=", overdueDate).Where("CANCELED", "==", false).Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			fmt.Println(err)
			return nil
		}

		var mainInvoice FullInvoice
		if err := doc.DataTo(&mainInvoice); err != nil {
			fmt.Println(err)
			continue
		}

		if mainInvoice.Total > 5 && mainInvoice.Debit > 0 && mainInvoice.IsNoticeToRemedySent != true && snapshotInvoicesID[mainInvoice.Customer.ID] == nil {
			customerEntitie, err := fs.Collection("entities").Doc(mainInvoice.Entity.ID).Get(ctx)
			customer, err := fs.Collection("customers").Doc(mainInvoice.Customer.ID).Get(ctx)

			invoicesSnaps, err := fs.Collection("invoices").Where("customer", "==", customer.Ref).Where("PAID", "==", false).Where("PAYDATE", "<=", now.AddDate(0, 0, -1)).Where("CANCELED", "==", false).Documents(ctx).GetAll()
			if err != nil {
				fmt.Println(err)
			}

			for _, invoiceSnap := range invoicesSnaps {
				var invoice FullInvoice
				if err := invoiceSnap.DataTo(&invoice); err != nil {
					fmt.Println(err)
					continue
				}

				if invoice.Total > 0 && invoice.Debit > 0 {
					snapshotInvoicesID[invoice.Customer.ID] = append(snapshotInvoicesID[invoice.Customer.ID], invoiceSnap.Ref.ID)

					if err != nil {
						fmt.Println(err)
						return nil
					}

					users, err := common.GetCustomerUsersWithPermissions(ctx, fs, customer.Ref, requiredPermissions)
					if err != nil {
						fmt.Println(err)
						return nil
					}

					for _, user := range users {
						usersEmail[customer.Ref.ID] = append(usersEmail[customer.Ref.ID], user.Email)
					}

					usersEmail[invoice.Customer.ID] = unique(usersEmail[invoice.Customer.ID])

					customerLegalName, _ := customerEntitie.DataAt("name")
					customerContactName, _ := customerEntitie.DataAt("contact.name")
					customerContactEmail, _ := customerEntitie.DataAt("contact.email")

					address := ""
					if country, err := customer.DataAt("enrichment.geo.country"); err == nil && country != nil {
						address += country.(string) + " "
					}

					if city, err := customer.DataAt("enrichment.geo.city"); err == nil && city != nil {
						address += city.(string) + " "
					}

					if streetName, err := customer.DataAt("enrichment.geo.streetName"); err == nil && streetName != nil {
						address += streetName.(string) + " "
					}

					if streetNumber, err := customer.DataAt("enrichment.geo.streetNumber"); err == nil && streetNumber != nil {
						address += streetNumber.(string)
					}

					sharedDriveFolderID, err := customer.DataAt("sharedDriveFolderId")
					if sharedDriveFolderID == nil {
						sharedDriveFolderID = ""
					}

					skipRemedyBreach, err := customer.DataAt("skipRemedyBreach")
					if skipRemedyBreach == nil {
						skipRemedyBreach = false
					}

					var bDetails BankDetails

					for index := 0; index < len(companies.Compnies); index++ {
						if companies.Compnies[index].CompanyID == invoice.Company {
							bDetails = companies.Compnies[index]
						}
					}

					estPaydate := int(invoice.PayDate.Sub(invoice.Date).Hours() / 24)

					var formatProductsName []string

					for _, p := range invoice.Products {
						if p != "other" {
							formatProductsName = append(formatProductsName, common.FormatAssetType(p))
						}
					}

					var accountManagerRef *firestore.DocumentRef

					amID, err := customer.DataAt("accountManagers.doit.account_manager.ref")
					if err != nil {
						fmt.Println(err)

						accountManagerRef = nil
					}

					if amID != nil {
						accountManagerRef = amID.(*firestore.DocumentRef)
					}

					primaryDomain, err := customer.DataAt("primaryDomain")

					if totalDebits[customer.Ref.ID] == nil {
						totalDebits[customer.Ref.ID] = make(map[string]float64)
					}

					totalDebits[customer.Ref.ID][invoice.Symbol] += invoice.Debit
					//fmt.Println(invoice.Debit)
					allPdfUrls := append(customersOverdueInovices[customer.Ref.ID].PdfUrls, invoice.ExternalFilesSubForm...)
					allInvoices := append(customersOverdueInovices[customer.Ref.ID].InvoicesID, invoice.ID)
					productsType := append(customersOverdueInovices[customer.Ref.ID].ProductType, formatProductsName...)

					customerData := OverdueCustomer{
						CustomerID:           customer.Ref.ID,
						CustomerLegalName:    customerLegalName.(string),
						CustomerContactName:  customerContactName.(string),
						CustomerContactEmail: customerContactEmail.(string),
						UsersEmail:           usersEmail[customer.Ref.ID],
						CustomerLegalAddress: address,
						DateNow:              time.Now(),
						InvoicesID:           unique(allInvoices),
						InvoicesDocID:        snapshotInvoicesID[customer.Ref.ID],
						ProductType:          unique(productsType),
						PaymentDays:          estPaydate,
						DEBIT:                customersOverdueInovices[customer.Ref.ID].DEBIT + invoice.Debit,
						TotalDebit:           totalDebits[customer.Ref.ID],
						SYMBOL:               invoice.Symbol,
						AccountHolderName:    invoice.Company,
						HolderBankDetails:    bDetails,
						SharedDriveFolderID:  sharedDriveFolderID.(string),
						AccountManagerRef:    accountManagerRef,
						skipRemedyBreach:     skipRemedyBreach.(bool),
						domain:               primaryDomain.(string),
						PdfUrls:              allPdfUrls,
						EntityID:             invoice.Entity.ID,
					}
					customersOverdueInovices[customer.Ref.ID] = customerData
				}
			}
		}
	}

	return customersOverdueInovices
}

func unique(slice []string) []string {
	uniqMap := make(map[string]struct{})
	for _, v := range slice {
		uniqMap[v] = struct{}{}
	}

	uniqSlice := make([]string, 0, len(uniqMap))
	for v := range uniqMap {
		uniqSlice = append(uniqSlice, v)
	}

	return uniqSlice
}
func numberToString(n int, sep rune) string {
	s := strconv.Itoa(n)
	startOffset := 0

	var buff bytes.Buffer

	if n < 0 {
		startOffset = 1

		buff.WriteByte('-')
	}

	l := len(s)

	commaIndex := 3 - ((l - startOffset) % 3)
	if commaIndex == 3 {
		commaIndex = 0
	}

	for i := startOffset; i < l; i++ {
		if commaIndex == 3 {
			buff.WriteRune(sep)

			commaIndex = 0
		}

		commaIndex++

		buff.WriteByte(s[i])
	}

	return buff.String()
}

func getBankAccountInformation(wireTransfer []WireTransfer) string {
	str := ""
	for i := range wireTransfer {
		str += fmt.Sprintf("%s: %s\n", wireTransfer[i].Key, wireTransfer[i].Value)
	}

	return str
}
