package fullstory

import (
	"context"
	"encoding/json"
	"log"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

type SecretJSON struct {
	UserIdentify string `json:"user_identify_hmac_secret"`
}

type FullstoryService struct {
	*logger.Logging
	*connection.Connection
}

func NewFullstoryService(log *logger.Logging, conn *connection.Connection) *FullstoryService {
	return &FullstoryService{
		log,
		conn,
	}
}

var secret SecretJSON

func init() {
	ctx := context.Background()

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretFullstory)
	if err != nil {
		log.Fatalln(err)
	}

	if err = json.Unmarshal(data, &secret); err != nil {
		log.Fatalln(err)
	}
}

func (s *FullstoryService) GetUserHMAC(ctx context.Context, email string, claims map[string]interface{}) (string, error) {
	if impersonate, prs := claims["impersonate"]; prs {
		if impersonator, ok := (impersonate.(map[string]interface{}))["email"].(string); ok {
			email = impersonator + "::" + email
		}
	}

	hmac, err := common.Sha256HMAC(email, []byte(secret.UserIdentify))
	if err != nil {
		return "", err
	}

	return hmac, nil
}
