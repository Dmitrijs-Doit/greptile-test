package dal

import (
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
)

// Note: actual fixtures data loaded into the firestore emulator can be found into services/firebase/import-export-firestore/src/testDataDefinition/Assets/export_data/exported_data.json

func fixtureAssetAWSStandalone() *pkg.AWSAsset {
	return &pkg.AWSAsset{
		Properties: &pkg.AWSProperties{
			AccountID:    "023946476650",
			Name:         "doitintl-payer-102",
			FriendlyName: "023946476650",
			OrganizationInfo: &pkg.OrganizationInfo{
				PayerAccount: &domain.PayerAccount{
					AccountID:   "023946476650",
					DisplayName: "standalone-payer-023946476650",
				},
				Status: "ACTIVE",
				Email:  "awsops+102@doit-intl.com",
			},
			SauronRole: false,
			Support:    nil,
		},
	}
}

func fixtureAssetAWSResold() *pkg.AWSAsset {
	return &pkg.AWSAsset{
		Properties: &pkg.AWSProperties{
			AccountID:    "001214633506",
			Name:         "",
			FriendlyName: "001214633506",
			CloudHealth: &pkg.CloudHealthAccountInfo{
				CustomerName: "psquaredpublishing.com",
				CustomerID:   29801,
				AccountID:    13743895348002,
				ExternalID:   "6cda262029ad7b34a64ff537196ab4",
				Status:       "red",
			},
			OrganizationInfo: &pkg.OrganizationInfo{
				PayerAccount: &domain.PayerAccount{
					AccountID:   "279843869311",
					DisplayName: "DoiT Reseller Account #7",
				},
				Status: "ACTIVE",
				Email:  "js@psquaredpublishing.com",
			},
			SauronRole: true,
			Support:    nil,
		},
	}
}
