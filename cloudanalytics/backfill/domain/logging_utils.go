package domain

import (
	"fmt"
	"net/url"
)

// Define errors used for Cloud Logging
const (
	E0001 = "failed getting Service Account details"
	E0002 = "failed decrypting Service Account key"
	E0003 = "failed getting dataset location"
	E0004 = "failed creating BigQuery client"
	E0005 = "missing permissions to access customer's billing table"
	E0006 = "failed getting table (partition) size or it doesn't exist"
	E0007 = "failed creating destination dataset"
	E0008 = "failed checking if destination table exists"
	E0009 = "failed creating destination table"
	E0010 = "failed exporting destination bucket in GCS"
	E0011 = "copy job failed"
)

// Define errors messages displayed in the CMP
const (
	M0001 = "Failed to get service account details. Please verify that your service account status is healthy."
	M0002 = "Failed to get service account details. Please verify that your service account key file is formatted correctly."
	M0003 = "Failed to copy the data. Please verify that your project ID and dataset ID are correct."
	M0004 = "The copy job failed. Please verify that your service account has BigQuery Viewer permissions."
	M0005 = "Your service account does not have permission to access the billing table at the provided location. Please verify that you have added the BigQuery Data Viewer role to your service account."
	M0006 = "Failed to copy the data. Please verify the provided table exists."
	M0007 = "We were unable to copy your billing data due to a system error. Our engineering team has been alerted and will investigate the issue. We are sorry for the inconvenience!"
)

const backfillSupportTemplateID = "gcp-backfill-issue"

// DisplayMessageMapping represents a mapping (error : display message in the CMP)
var DisplayMessageMapping = map[string]string{
	E0001: M0001,
	E0002: M0002,
	E0003: M0003,
	E0004: M0004,
	E0005: M0005,
	E0006: M0006,
	E0007: M0007,
	E0008: M0007,
	E0009: M0007,
	E0010: M0007,
	E0011: M0007,
}

func GetActionFromMessage(message string, data *FlowInfo) string {
	switch message {
	case M0007:
		params := url.Values{}
		params.Add("billing-account-id", data.BillingAccountID)
		params.Add("project-id", data.ProjectID)
		params.Add("dataset-id", data.DatasetID)
		params.Add("table-id", data.TableID)

		return "url:/customers/" + data.CustomerID + "/support/new/" + backfillSupportTemplateID + "?" + params.Encode()
	case M0001, M0002:
		return "url:/customers/" + data.CustomerID + "/settings/gcp"
	case M0004, M0005:
		return "url:https://help.doit.com/google-cloud/import-historical-billing-data#step-2-grant-the-required-permissions"
	}

	return "modal"
}

func GetDisplayMessageFromError(err error) string {
	// If no error occurred, don't send any error display message
	if err == nil {
		return ""
	}

	errString := fmt.Sprintf("%s", err)
	if val, ok := DisplayMessageMapping[errString]; ok {
		return val
	}

	// Send default error message
	return M0007
}
