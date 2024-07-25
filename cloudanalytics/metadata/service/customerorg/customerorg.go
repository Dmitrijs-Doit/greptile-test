package customerorg

import (
	"cloud.google.com/go/firestore"
)

// TODO: Move this to the DAL layer once CMP-12872 is done.
func GetCustomerOrgMetadataCollectionRef(customerRef *firestore.DocumentRef, orgID, id string) *firestore.CollectionRef {
	return customerRef.Collection("customerOrgs").
		Doc(orgID).
		Collection("assetsReportMetadata").
		Doc(id).
		Collection("reportOrgMetadata")
}
