package dal

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PipelineConfigFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewPipelineConfigWithClient(fun connection.FirestoreFromContextFun) *PipelineConfigFirestore {
	return &PipelineConfigFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *PipelineConfigFirestore) GetDocRef(ctx context.Context) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(consts.IntegrationsCollection).Doc(consts.GCPFlexsaveStandaloneDoc)
}

func (d *PipelineConfigFirestore) GetPipelineConfig(ctx context.Context) (*dataStructures.PipelineConfig, error) {
	snap, err := d.documentsHandler.Get(ctx, d.GetDocRef(ctx))
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var md dataStructures.PipelineConfig

	err = snap.DataTo(&md)
	if err != nil {
		return nil, err
	}

	return &md, nil
}

func (d *PipelineConfigFirestore) SetPipelineConfig(ctx context.Context, config *dataStructures.PipelineConfig) error {
	_, err := d.documentsHandler.Set(ctx, d.GetDocRef(ctx), config)
	if err != nil {
		return err
	}

	return nil
}

func (d *PipelineConfigFirestore) DeletePipelineConfigDoc(ctx context.Context) error {
	_, err := d.GetDocRef(ctx).Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (d *PipelineConfigFirestore) DeleteRegionBucket(ctx context.Context, region string) error {
	config, err := d.GetPipelineConfig(ctx)
	if err != nil {
		if err == doitFirestore.ErrNotFound {
			return nil
		}

		return err
	}

	if config.RegionsBuckets != nil {
		if config.RegionsBuckets[strings.ToLower(region)] != "" {
			config.RegionsBuckets[strings.ToLower(region)] = ""
		}

		if config.RegionsBuckets[strings.ToUpper(region)] != "" {
			config.RegionsBuckets[strings.ToUpper(region)] = ""
		}

		err = d.SetPipelineConfig(ctx, config)
		if err != nil {
			if err == doitFirestore.ErrNotFound {
				return nil
			}

			return err
		}
	}

	return nil
}
