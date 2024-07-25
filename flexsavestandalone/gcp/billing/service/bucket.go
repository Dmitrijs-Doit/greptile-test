package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	gcpCommon "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Bucket interface {
	Create(ctx context.Context, location string, test bool) (string, error)
	DeleteEmptyBucket(ctx context.Context, bucketName string) error
	DeleteBucket(ctx context.Context, bucketName string) error
	DeleteFileFromBucket(ctx context.Context, bucketName string, lastBucketWriteTimestamp string, billingAccount string) error
	GrantServiceAccountPermissionsOnBucket(ctx context.Context, bucketname, mail, billingAccount string) error
	DeleteAllBuckets(ctx context.Context) error
	Delete(ctx context.Context, location string) error
	Empty(ctx context.Context, location string) error
	DeleteAllFilesFromBucket(ctx context.Context, bucketName string) error
}
type BucketImpl struct {
	loggerProvider logger.Provider
	*connection.Connection
	pipelineConfig *dal.PipelineConfigFirestore
}

func NewBucket(log logger.Provider, conn *connection.Connection) *BucketImpl {
	return &BucketImpl{
		log,
		conn,
		dal.NewPipelineConfigWithClient(conn.Firestore),
	}
}

func (s *BucketImpl) Create(ctx context.Context, location string, test bool) (string, error) {
	bucketAttrs := &storage.BucketAttrs{
		Location: location,
		Lifecycle: storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action: storage.LifecycleAction{Type: storage.DeleteAction},
					Condition: storage.LifecycleCondition{
						AgeInDays: 10,
					},
				},
			},
		},
	}

	bucketName := fmt.Sprintf("%s_%s_%s", utils.GetProjectName(), consts.BucketPrefix, strings.ToLower(location))
	if test {
		bucketName = fmt.Sprintf("test_%s", bucketName)
	}

	bucket := s.CloudStorage(ctx).Bucket(bucketName)
	if err := bucket.Create(ctx, utils.GetProjectName(), bucketAttrs); err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code != http.StatusConflict {
				return "", err
			}
		} else {
			return "", err
		}
	}

	return bucketName, nil
}
func (s *BucketImpl) DeleteEmptyBucket(ctx context.Context, bucketName string) error {
	bucket := s.CloudStorage(ctx).Bucket(bucketName)
	if err := bucket.Delete(ctx); err != nil {
		return fmt.Errorf("Bucket(%q).Delete: %v", bucketName, err)
	}

	return nil
}

func (s *BucketImpl) DeleteBucket(ctx context.Context, bucketName string) error {
	if err := s.DeleteAllFilesFromBucket(ctx, bucketName); err != nil && err != gcpCommon.ErrBucketEmpy {
		return err
	}

	if err := s.DeleteEmptyBucket(ctx, bucketName); err != nil {
		return err
	}

	return nil
}

func (s *BucketImpl) DeleteFileFromBucket(ctx context.Context, bucketName string, lastBucketWriteTimestamp string, billingAccount string) error {
	//grab the bucket:
	bucket := s.CloudStorage(ctx).Bucket(bucketName)
	query := &storage.Query{Prefix: ""}

	var names []string

	it := bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return fmt.Errorf("error finding files in %v for deletion: %v", bucket, err)
		}
		//get the correct file name using the billingAccountID and the lastBucketWriteTimestamp
		if strings.Contains(attrs.Name, billingAccount+"/"+lastBucketWriteTimestamp) {
			names = append(names, attrs.Name)
		}
	}

	if len(names) > 0 {
		for _, name := range names {
			//delete all the files with billingAccountID and the lastBucketWriteTimestamp
			if err := bucket.Object(name).Delete(ctx); err != nil {
				return fmt.Errorf("error deleting file %v - %v", name, err)
			}

			logger := s.loggerProvider(ctx)
			logger.Infof("file %v deleted.\n", name)
		}

		return nil
	}

	return fmt.Errorf("file not found in bucket")
}

func (s *BucketImpl) GrantServiceAccountPermissionsOnBucket(ctx context.Context, bucketname, mail, billingAccount string) error {
	logger := s.loggerProvider(ctx)
	bucket := s.CloudStorage(ctx).Bucket(bucketname)

	policy, err := bucket.IAM().V3().Policy(ctx)
	if err != nil {
		err = fmt.Errorf("unable to get iam policies. Caused by %s", err)
		logger.Error(err)

		return err
	}

	roleGranted := false
	role := fmt.Sprintf("projects/%s/roles/%s", utils.GetProjectName(), consts.DedicatedRole)

	for _, p := range policy.Bindings {
		logger.Infof("binding %#v found. checking content...", p)

		if p.Role == role {
			for _, member := range p.Members {
				logger.Infof("member %#v found. checking content...", member)

				if member == "serviceAccount:"+mail {
					logger.Infof("member %s found on role %s for BA %s", member, role, billingAccount)

					roleGranted = true

					break
				}
			}

			break
		}
	}

	if !roleGranted {
		binding := &iampb.Binding{
			Role:    role,
			Members: []string{"serviceAccount:" + mail},
		}
		logger.Infof("binding %+v created for role %s for BA %s", binding, role, billingAccount)

		policy.Bindings = append(policy.Bindings, binding)
		if err := bucket.IAM().V3().SetPolicy(ctx, policy); err != nil {
			err = fmt.Errorf("unable to set policy %+v for BA %s. Caused by %s", policy, billingAccount, err)
			logger.Error(err)

			return err
		}
	}

	return nil
}

func (s *BucketImpl) DeleteAllBuckets(ctx context.Context) error {
	cs := s.CloudStorage(ctx)
	it := cs.Buckets(ctx, utils.GetProjectName())

	for {
		battrs, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		if strings.Contains(battrs.Name, consts.BucketPrefix) {
			err = s.DeleteBucket(ctx, battrs.Name)
			if err != nil {
				return err
			}

			err = s.pipelineConfig.DeleteRegionBucket(ctx, battrs.Location)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *BucketImpl) Delete(ctx context.Context, location string) error {
	return nil
}

func (s *BucketImpl) Empty(ctx context.Context, location string) error {
	return nil
}

// deleteFileFromBucket
func (s *BucketImpl) DeleteAllFilesFromBucket(ctx context.Context, bucketName string) error {
	//grab the bucket:
	bucket := s.CloudStorage(ctx).Bucket(bucketName)
	query := &storage.Query{Prefix: ""}

	var names []string

	it := bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return fmt.Errorf("error finding files in %v for deletion: %v", bucket, err)
		}

		names = append(names, attrs.Name)
	}

	if len(names) > 0 {
		for _, name := range names {
			//delete all the files with billingAccountID and the lastBucketWriteTimestamp
			go func(name string) {
				if err := bucket.Object(name).Delete(ctx); err != nil {
					logger := s.loggerProvider(ctx)
					logger.Errorf("error deleting file %v - %v", name, err)
				}
			}(name)
		}

		return nil
	}

	return gcpCommon.ErrBucketEmpy
}
