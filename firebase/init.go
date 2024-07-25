package firebase

import (
	"context"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const (
	// OrphanID customer ID
	OrphanID = "8FMavHMpAtbgsNCsCrzs"
)

var (
	// App : Firebase App
	App *firebase.App

	// Orphan customer
	Orphan *firestore.DocumentRef

	DemoApp *firebase.App
)

func init() {
	ctx := context.Background()

	var err error
	App, err = firebase.NewApp(ctx, &firebase.Config{ProjectID: common.ProjectID})
	
	if err != nil {
		log.Fatalln(err)
	}

	fs, err := App.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	Orphan = fs.Collection("customers").Doc(OrphanID)

	InitDemoApp()
}

func InitDemoApp() {
	ctx := context.Background()
	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretFirebaseDemo)

	if err != nil {
		log.Fatalln(err)
	}

	creds := option.WithCredentialsJSON(data)

	DemoApp, err = firebase.NewApp(ctx, nil, creds)
	if err != nil {
		log.Fatalln(err)
	}
}
