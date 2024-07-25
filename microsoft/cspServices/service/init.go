package service

import (
	"log"

	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

type CSPServices map[microsoft.CSPDomain]*CSPService

var CspServices = CSPServices{
	microsoft.CSPDomainIL:     nil,
	microsoft.CSPDomainUS:     nil,
	microsoft.CSPDomainEU:     nil,
	microsoft.CSPDomainUK:     nil,
	microsoft.CSPDomainEurope: nil,
}

func init() {
	for _, accessToken := range microsoft.MPCAccessTokens {
		service, err := NewCSPService(accessToken)
		if err != nil {
			log.Fatalln(err)
		}

		CspServices[accessToken.GetDomain()] = service
	}
}
