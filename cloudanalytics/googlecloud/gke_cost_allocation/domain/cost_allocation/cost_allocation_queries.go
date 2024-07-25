package domain

const (
	CustomersGKECostAllocationQueryTmpl = `
WITH
all_gke_data AS (
  SELECT
	export_time,
	billing_account_id,
	service,
	cost,
	labels
  FROM
    {table}
  WHERE
  	DATE(export_time) BETWEEN {export_time_from} AND {export_time_to}
    AND (service.description='Compute Engine' OR service.description='Kubernetes Engine')
),
gke_cost_allocation_rows AS (
  SELECT
    billing_account_id,
    labels
  FROM
    all_gke_data
  WHERE
  	cost is not null
    AND 'k8s-namespace' IN UNNEST(labels.key)
),
clusters AS (
  SELECT
    billing_account_id,
    ARRAY_AGG(DISTINCT(value)) AS clusters
  FROM (
    SELECT
      *
    FROM
      gke_cost_allocation_rows
    WHERE
      "goog-k8s-cluster-name" IN UNNEST(labels.key) ),
    UNNEST(labels)
  WHERE
    key = "goog-k8s-cluster-name"
  GROUP BY
    billing_account_id
)
SELECT
    c.billing_account_id,
    c.clusters,
    n.namespaces
  FROM
    clusters AS c
  LEFT JOIN (
    SELECT
      billing_account_id,
      ARRAY_AGG(DISTINCT(value)) AS namespaces
    FROM (
      SELECT
        *
      FROM
        gke_cost_allocation_rows
      WHERE
        "goog-k8s-namespace" IN UNNEST(labels.key) ),
      UNNEST(labels)
    WHERE
      key = "goog-k8s-namespace"
    GROUP BY
      1 ) AS n
  ON
    c.billing_account_id = n.billing_account_id
`

	AllClustersByBillingAccountIDTmpl = `
WITH
all_gke_data AS (
	SELECT
	  export_time,
	  billing_account_id,
	  service,
	  cost,
	  labels
	FROM
	  {table}
	WHERE
		DATE(export_time) BETWEEN {export_time_from} AND {export_time_to}
	  AND (service.description='Compute Engine' OR service.description='Kubernetes Engine')
  ),
gke_rows AS (
	SELECT
	  billing_account_id,
	  labels
	FROM
	  all_gke_data
	WHERE
	  'k8s-namespace' IN UNNEST(labels.key)
)
SELECT
  billing_account_id,
  ARRAY_AGG(DISTINCT(value)) AS clusters
FROM (
  SELECT
	*
  FROM
	gke_rows
  WHERE
	"goog-k8s-cluster-name" IN UNNEST(labels.key) ),
  UNNEST(labels)
WHERE
  key = "goog-k8s-cluster-name"
GROUP BY
  billing_account_id
`

	PresentationCustomerGKECostAllocationQueryTmpl = `
  SELECT
	billing_account_id,
	ARRAY_AGG(DISTINCT kubernetes_cluster_name IGNORE NULLS) AS clusters,
	ARRAY_AGG(DISTINCT kubernetes_namespace IGNORE NULLS) AS namespaces
  FROM
	  {table}
  WHERE
		DATE(export_time) BETWEEN {export_time_from} AND {export_time_to}
	  AND (kubernetes_cluster_name IS NOT NULL OR kubernetes_namespace IS NOT NULL)
	  AND cost IS NOT NULL
  GROUP BY
	billing_account_id
`
)
