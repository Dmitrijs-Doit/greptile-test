package common

import (
	"context"
	"fmt"

	cloudkms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

const kmsKeyNameProd = "projects/me-doit-intl-com/locations/global/keyRings/hello-keyring/cryptoKeys/cloud-credentials"
const kmsKeyNameDev = "projects/doitintl-cmp-dev/locations/global/keyRings/hello-keyring-dev/cryptoKeys/cloud-credentials-dev"

// EncryptSymmetric will encrypt the input plaintext with the specified symmetric key.
func EncryptSymmetric(plaintext []byte) ([]byte, error) {
	ctx := context.Background()

	client, err := cloudkms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudkms.NewKeyManagementClient: %v", err)
	}

	defer client.Close()

	kmsKeyName := kmsKeyNameProd
	if ProjectID != productionProject && IsLocalhost {
		kmsKeyName = kmsKeyNameDev
	}

	req := &kmspb.EncryptRequest{
		Name:      kmsKeyName,
		Plaintext: plaintext,
	}

	resp, err := client.Encrypt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Encrypt: %v", err)
	}

	return resp.Ciphertext, nil
}

// DecryptSymmetric will decrypt the input ciphertext bytes using the specified symmetric key.
func DecryptSymmetric(ciphertext []byte) ([]byte, error) {
	ctx := context.Background()

	client, err := cloudkms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudkms.NewKeyManagementClient: %v", err)
	}

	defer client.Close()

	req := &kmspb.DecryptRequest{
		Name:       kmsKeyNameProd,
		Ciphertext: ciphertext,
	}

	resp, err := client.Decrypt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Decrypt: %v", err)
	}

	return resp.Plaintext, nil
}
