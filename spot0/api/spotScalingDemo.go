package api

import (
	"context"
	"encoding/json"
	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
	"io"
	"os"
	"strings"
)

func unmarshalFile(filePath string, result interface{}) error {
	// Open our jsonFile
	file, err := os.Open(filePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		return err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(buf), &result)

	return err
}

func (s *SpotScalingDemoService) AveragePrices(ctx context.Context, req *model.AveragePricesRequest) (*model.AveragePricesResponse, error) {
	cost := float64(len(req.Subnets)) * float64(len(req.InstanceTypes)) / 100

	return &model.AveragePricesResponse{
		SpotHourCost:     cost,
		OnDemandHourCost: cost,
	}, nil
}

func (s *SpotScalingDemoService) ExecuteSpotScaling(ctx context.Context, req *model.ApplyConfigurationRequest) (*model.Response, error) {
	fs := s.Firestore(ctx)
	asgsCollectionRef := fs.Collection("spot0").Doc("spotApp").Collection("asgs").Where("asgName", "==", req.ASGName)

	docSnaps, err := asgsCollectionRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(docSnaps) == 0 {
		return &model.Response{
			Done: false,
		}, nil
	}

	docData := docSnaps[0].Data()

	var asgMapped model.AsgConfiguration

	if req.ASGName == "Batch Processing ASG" {
		err = unmarshalFile("spot0/api/optimized_asg_batch.json", &asgMapped)
	} else if req.ASGName == "CI/CD ASG" {
		err = unmarshalFile("spot0/api/optimized_asg_ci.json", &asgMapped)
	}

	if err != nil {
		return &model.Response{
			Done: false,
		}, err
	}

	asgMapped.Customer = docData["customer"]
	if req.Configuration != nil {
		updateAsgConfiguration(&asgMapped, req.Configuration)
	}

	updateRef := fs.Collection("spot0").Doc("spotApp").Collection("asgs").Doc(docSnaps[0].Ref.ID)
	_, err = updateRef.Set(ctx, asgMapped)

	return &model.Response{
		Done: err == nil,
	}, err
}

func (s *SpotScalingDemoService) UpdateAsgConfig(ctx context.Context, req *model.UpdateAsgConfigRequest) (*model.Response, error) {
	fs := s.Firestore(ctx)
	asgsCollectionRef := fs.Collection("spot0").Doc("spotApp").Collection("asgs").Where("asgName", "==", req.AsgName)

	docSnaps, err := asgsCollectionRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var asgConfig model.AsgConfiguration

	err = docSnaps[0].DataTo(&asgConfig)
	if err != nil {
		return nil, err
	}

	if len(docSnaps) == 0 {
		return nil, nil
	}

	updateAsgConfiguration(&asgConfig, req.Configuration)

	updateRef := fs.Collection("spot0").Doc("spotApp").Collection("asgs").Doc(docSnaps[0].Ref.ID)
	_, err = updateRef.Set(ctx, &asgConfig)

	return &model.Response{
		Done: err == nil,
	}, err
}

func updateAsgConfiguration(asg *model.AsgConfiguration, configReq *model.Configuration) {
	//Availavilty zones
	if len(configReq.IncludedSubnets) > 0 {
		subnetsWithComa := strings.Join(configReq.IncludedSubnets, ",")
		asg.Spotisize.CurAsg.VPCZoneIdentifier = subnetsWithComa
		asg.Spotisize.RecAsg.VPCZoneIdentifier = subnetsWithComa
		asg.Config.ExcludedSubnets = configReq.ExcludedSubnets
	}

	if configReq.OnDemandBaseCapacity != nil {
		asg.Spotisize.CurAsg.MixedInstancesPolicy.InstancesDistribution.OnDemandBaseCapacity = *configReq.OnDemandBaseCapacity
		asg.Spotisize.CurAsg.MixedInstancesPolicy.InstancesDistribution.OnDemandPercentageAboveBaseCapacity = *configReq.OnDemandPercentageAboveBaseCapacity

		asg.Spotisize.RecAsg.MixedInstancesPolicy.InstancesDistribution.OnDemandBaseCapacity = *configReq.OnDemandBaseCapacity
		asg.Spotisize.RecAsg.MixedInstancesPolicy.InstancesDistribution.OnDemandPercentageAboveBaseCapacity = *configReq.OnDemandPercentageAboveBaseCapacity
	}
	// instanceTypes
	if len(configReq.IncludedInstanceTypes) > 0 {
		var formattedInstances []model.Overrides

		for _, instanceType := range configReq.IncludedInstanceTypes {
			currentInstance := model.Overrides{
				InstanceType: instanceType,
			}
			formattedInstances = append(formattedInstances, currentInstance)
		}

		asg.Spotisize.CurAsg.MixedInstancesPolicy.LaunchTemplate.Overrides = formattedInstances
		asg.Spotisize.RecAsg.MixedInstancesPolicy.LaunchTemplate.Overrides = formattedInstances
		asg.Config.ExcludedInstanceTypes = configReq.ExcludedInstanceTypes
	}
}

func (s *SpotScalingDemoService) UpdateFallbackOnDemandConfig(ctx context.Context, req *model.FallbackOnDemandRequest) (*model.Response, error) {
	return &model.Response{
		Done: true,
	}, nil
}
