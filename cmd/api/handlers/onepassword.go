package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	onepasswordSDK "github.com/1Password/connect-sdk-go/onepassword"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/onepassword"
	"github.com/doitintl/onepassword/iface"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/onepassword/pkg"
)

type OnePassword struct {
	logger logger.Provider
	vaults map[string]iface.OnePasswordHandler
}

type UpdateTitleRequest struct {
	Title string `json:"title"`
}

type UpdateUsernameRequest struct {
	Username string `json:"username"`
}

var (
	envHostURL = map[string]string{
		"development": "https://one-password-api-wsqwprteya-uc.a.run.app",
		"production":  "https://one-password-api-alqysnpjoq-uc.a.run.app",
	}

	envMPAVaultID = map[string]string{
		"development": "tjp2jpaps54gxm2pwqnztynqxe",
		"production":  "rahkkon4drtfqry35yfcw3c4fq",
	}

	ErrVaultNotFound  = errors.New("vault not found")
	ErrInvaildRequest = errors.New("invalid request")
)

func New(log logger.Provider, conn *connection.Connection) OnePassword {
	ctx := context.Background()

	hostURL, ok := envHostURL[common.Env]
	if !ok {
		panic(fmt.Errorf("no HostURL config for env %s", common.Env))
	}

	mpaVaultID, ok := envMPAVaultID[common.Env]
	if !ok {
		panic(fmt.Errorf("no MPA VaultID config for env %s", common.Env))
	}

	mpaVaultToken, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretMPAVaultToken)
	if err != nil {
		panic(err)
	}

	return OnePassword{
		logger: log,
		vaults: map[string]iface.OnePasswordHandler{
			mpaVaultID: onepassword.New(pkg.Config{
				HostURL: hostURL,
				Token:   string(mpaVaultToken),
			}),
		},
	}
}

func (h *OnePassword) Create(ctx *gin.Context) error {
	vaultID := ctx.Param("vaultID")

	var item onepasswordSDK.Item
	if err := ctx.ShouldBindJSON(&item); err != nil {
		return web.NewRequestError(ErrInvaildRequest, http.StatusBadRequest)
	}

	vault, ok := h.vaults[vaultID]
	if !ok {
		return web.NewRequestError(fmt.Errorf("vault ID: %s: %w", item.Vault.ID, ErrVaultNotFound), http.StatusBadRequest)
	}

	item.Vault.ID = vaultID

	res, err := vault.CreateIfNotExist(ctx, &item)
	if err != nil {
		return handleServiceErr(err)
	}

	return web.Respond(ctx, res.ID, http.StatusOK)
}

func (h *OnePassword) UpdateTitle(ctx *gin.Context) error {
	vaultID := ctx.Param("vaultID")
	itemID := ctx.Param("itemID")

	var body UpdateTitleRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(ErrInvaildRequest, http.StatusBadRequest)
	}

	vault, ok := h.vaults[vaultID]
	if !ok {
		return web.NewRequestError(fmt.Errorf("vault ID: %s: %w", vaultID, ErrVaultNotFound), http.StatusBadRequest)
	}

	_, err := vault.UpdateTitle(ctx, itemID, vaultID, body.Title)
	if err != nil {
		return handleServiceErr(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *OnePassword) UpdateUsername(ctx *gin.Context) error {
	vaultID := ctx.Param("vaultID")
	itemID := ctx.Param("itemID")

	var body UpdateUsernameRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(ErrInvaildRequest, http.StatusBadRequest)
	}

	vault, ok := h.vaults[vaultID]
	if !ok {
		return web.NewRequestError(fmt.Errorf("vault ID: %s: %w", vaultID, ErrVaultNotFound), http.StatusBadRequest)
	}

	_, err := vault.UpdateUsername(ctx, itemID, vaultID, body.Username)
	if err != nil {
		return handleServiceErr(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *OnePassword) Delete(ctx *gin.Context) error {
	vaultID := ctx.Param("vaultID")
	itemID := ctx.Param("itemID")

	vault, ok := h.vaults[vaultID]
	if !ok {
		return web.NewRequestError(fmt.Errorf("vault ID: %s: %w", vaultID, ErrVaultNotFound), http.StatusBadRequest)
	}

	err := vault.Delete(ctx, itemID, vaultID)
	if err != nil {
		return handleServiceErr(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func handleServiceErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pkg.ErrExists) {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err, ok := err.(*onepasswordSDK.Error); ok {
		return web.NewRequestError(err, err.StatusCode)
	}

	return web.NewRequestError(fmt.Errorf("Something went wrong"), http.StatusInternalServerError)
}
