package hubspot

import (
	"embed"
	"encoding/csv"
)

// Approved GCP services
var gcpServices = map[string]struct{}{
	"Anthos":                                    {},
	"API Gateway":                               {},
	"App Engine":                                {},
	"Artifact Registry":                         {},
	"BigQuery":                                  {},
	"BigQuery BI Engine":                        {},
	"BigQuery Data Transfer Service":            {},
	"BigQuery Reservation API":                  {},
	"BigQuery Storage API":                      {},
	"Cloud AutoML":                              {},
	"Cloud Bigtable":                            {},
	"Cloud Build":                               {},
	"Cloud CDN":                                 {},
	"Cloud Composer":                            {},
	"Cloud Data Analytics":                      {},
	"Cloud Data Fusion":                         {},
	"Cloud Data Loss Prevention":                {},
	"Cloud Dataflow":                            {},
	"Cloud Dialogflow API":                      {},
	"Cloud DNS":                                 {},
	"Cloud Document AI API":                     {},
	"Cloud Domains":                             {},
	"Cloud Filestore":                           {},
	"Cloud Functions":                           {},
	"Cloud Healthcare":                          {},
	"Cloud IoT Core":                            {},
	"Cloud Key Management Service (KMS)":        {},
	"Cloud Machine Learning Engine":             {},
	"Cloud Memorystore for Memcached":           {},
	"Cloud Memorystore for Redis":               {},
	"Cloud Natural Language":                    {},
	"Cloud Natural Language API":                {},
	"Cloud Pub/Sub":                             {},
	"Cloud Run":                                 {},
	"Cloud Scheduler":                           {},
	"Cloud Spanner":                             {},
	"Cloud Speech API":                          {},
	"Cloud SQL":                                 {},
	"Cloud Storage":                             {},
	"Cloud Talent Solution":                     {},
	"Cloud Tasks":                               {},
	"Cloud Test Lab":                            {},
	"Cloud TPU":                                 {},
	"Cloud Video Intelligence API":              {},
	"Cloud Vision API":                          {},
	"Compute Engine":                            {},
	"Confidential Computing":                    {},
	"Container Builder":                         {},
	"Container Engine":                          {},
	"Container Registry Vulnerability Scanning": {},
	"Custom Search":                             {},
	"Data Catalog":                              {},
	"Directions API":                            {},
	"Distance Matrix API":                       {},
	"DLP API":                                   {},
	"Elevation API":                             {},
	"Firebase":                                  {},
	"Firebase Auth":                             {},
	"Firebase Database":                         {},
	"Firebase Hosting":                          {},
	"Firebase Realtime Database":                {},
	"Firebase Test Lab":                         {},
	"Geocoding API":                             {},
	"Geolocation API":                           {},
	"Google Maps Android API":                   {},
	"Google Maps SDK for iOS":                   {},
	"Google Service Control":                    {},
	"Identity Platform":                         {},
	"Kubernetes Engine":                         {},
	"Looker Data Platform SaaS":                 {},
	"Managed Service for Microsoft Active Directory": {},
	"Maps and Street View API":                       {},
	"Maps API":                                       {},
	"Maps API v3":                                    {},
	"Maps Elevation API":                             {},
	"Maps Embed API":                                 {},
	"Maps JavaScript API":                            {},
	"Maps Static API":                                {},
	"Networking":                                     {},
	"Places API":                                     {},
	"Places API for Android":                         {},
	"Places API for iOS":                             {},
	"Prediction":                                     {},
	"Pub/Sub Lite":                                   {},
	"reCAPTCHA Enterprise":                           {},
	"Recommendations AI":                             {},
	"Roads API":                                      {},
	"Secret Manager":                                 {},
	"Security Command Center":                        {},
	"Source Repository":                              {},
	"Stackdriver":                                    {},
	"Stackdriver Logging":                            {},
	"Stackdriver Monitoring":                         {},
	"Stackdriver Trace":                              {},
	"Static Maps API":                                {},
	"Street View API":                                {},
	"Street View Image API":                          {},
	"Street View Static API":                         {},
	"superQuery":                                     {},
	"Support":                                        {},
	"Time Zone API":                                  {},
	"Timezone API":                                   {},
	"Translate":                                      {},
	"VMware Engine":                                  {},
	"Web Risk":                                       {},
	"Zync":                                           {},
}

//go:embed awsServices.csv
var awsServicesCSV embed.FS

// Approved AWS services
func awsServices() map[string]struct{} {
	file, err := awsServicesCSV.Open("awsServices.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	_, err = reader.Read()
	if err != nil {
		panic(err)
	}

	services := make(map[string]struct{})

	for {
		row, err := reader.Read()
		if err != nil {
			break
		}

		services[row[0]] = struct{}{}
	}

	return services
}
