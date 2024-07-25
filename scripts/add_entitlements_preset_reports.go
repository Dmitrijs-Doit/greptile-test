package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

const flexsaveAWS = "8pvl6q86WlQebJHWtRyf"
const eks = "MUe4WVmhJJ4NFxHba5QS"
const credits = "credits" // TODO: change to correct entitlement ID

type Params struct {
	ProjectID string `json:"projectId"`
}

func AddEntitlementsToPresetReports(ctx *gin.Context) []error {
	var params Params
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	presetReports := map[string]string{
		"9UzHGTeTucVygnyeK2fb": flexsaveAWS,
		"FHbS1KOkjM6NqGpKo6m1": flexsaveAWS,
		"JS3UcG42Nb3YKsvtsNsJ": flexsaveAWS,
		"LnsuQ2yLT7mmryHBn0y7": flexsaveAWS,
		"pNCzq1jXtl3pu9mWiuSd": flexsaveAWS,
		"yVf5LquMqryrkf6uQjV7": flexsaveAWS,
		"2IzuTl3YOj8fRKW9EoPb": flexsaveAWS,
		"DQXyOIhViH5oa9WOGM9d": flexsaveAWS,
		"S3rQioYbrDL0me4nZYhM": flexsaveAWS,

		"4B26nPpITK7CtAIrVTzT": eks,
		"L7WbiZOi5gOOzD0WAc1t": eks,
		"OEmuiTb5b3Cj0KDBxRip": eks,
		"XjK5I6Eql5dNFgPw4Qh3": eks,
		"v5ffqOd74s915PrK289W": eks,

		// "Anl2FHDAgyxR4GFellrA": credits,
		// "KKi6lAX4N66XYCVAdqRD": credits,
		// "T36M3b1cuCtxhGW6FbfJ": credits,
		// "V0CF0boimYYMogypi0CO": credits,
		// "ViSpueEq1iXNMcL1n6Un": credits,
		// "lfpV2JN1wWaHqf7KWOdB": credits,
		// "mfeCSTBRGjhUoArSx2mg": credits,
		// "mrV8lKa7USq4NFVsPIhd": credits,
		// "pj0ijucXgFTXfdV8QIWY": credits,
		// "ttwTNnZNnKfUb1SZvRY8": credits,
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	batch := fb.NewAutomaticWriteBatch(fs, 500)

	for reportID, entitlement := range presetReports {
		ref := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports").Doc(reportID)

		batch.Set(ref, map[string]interface{}{
			"entitlements": []string{entitlement},
		}, firestore.MergeAll)
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	return nil
}
