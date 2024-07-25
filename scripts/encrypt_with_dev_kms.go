package scripts

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
)

func readFirestoreDocumentField(projectID, collection, documentID string) ([]byte, error) {
	// Create a Firestore client.
	ctx := context.Background()

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %v", err)
	}

	defer client.Close()

	// Reference to the document.
	docRef := client.Collection(collection).Doc(documentID)

	// Get the document snapshot.
	docSnapshot, err := docRef.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %v", err)
	}

	// Check if the document exists.
	if !docSnapshot.Exists() {
		return nil, fmt.Errorf("document does not exist")
	}

	// Get the "key" field value from the document.
	keyFieldValue, exists := docSnapshot.Data()["key"]
	if !exists {
		return nil, fmt.Errorf("field 'key' does not exist in the document")
	}
	// Convert the value to []byte.
	keyFieldBytes, ok := keyFieldValue.([]byte)
	if !ok {
		return nil, fmt.Errorf("unable to convert 'key' field value to []byte")
	}

	return keyFieldBytes, nil
}

func addFieldFirestoreDocument(ctx context.Context, client *firestore.Client, collection string, doc string, dataByte []byte) error {
	_, err := client.Collection(collection).Doc(doc).Update(ctx, []firestore.Update{
		{
			Path:  "keyDev",
			Value: dataByte,
		},
	})
	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred: %s", err)
	}

	return err
}

func addDevKmsCredentials(projectID string, collection string, doc string, dataByte []byte) error {
	ctx := context.Background()

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
		return err
	}

	defer client.Close()

	err = addFieldFirestoreDocument(ctx, client, collection, doc, dataByte)
	if err != nil {
		fmt.Printf("Error adding field to Firestore document: %v\n", err)
		return err
	}

	fmt.Println("Field added to Firestore document successfully.")

	return nil
}

func EncryptWithDevKms(ctx *gin.Context) []error {
	// 1. Read key from DoiT SA Firestore dev
	fsProjectID := "doitintl-cmp-dev"
	fsCollection := "customers/EE8CtpzYiKp0dVAESVrB/cloudConnect"
	fsDocumentID := "google-cloud-114075288177071352357"

	key, err := readFirestoreDocumentField(fsProjectID, fsCollection, fsDocumentID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return []error{err}
	}

	// 2. Decrypt key with prod kms
	decryptedText, err := common.DecryptSymmetric(key)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return []error{err}
	}

	// 3. Encrypt data with dev kms
	encryptedData, err := common.EncryptSymmetric(decryptedText)
	if err != nil {
		fmt.Printf("Error encrypting: %v\n", err)
		return []error{err}
	}

	// 4. Add encrypted data to Firestore prod
	projectIDProd := "me-doit-intl-com"

	err = addDevKmsCredentials(projectIDProd, fsCollection, fsDocumentID, encryptedData)
	if err != nil {
		fmt.Printf("Error adding credentials with dev kms: %v\n", err)
		return []error{err}
	}

	fmt.Printf("Firestore Updated")

	return nil
}
