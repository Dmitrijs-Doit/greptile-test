package model

import "time"

type AsgConfiguration struct {
	AccountID             string               `json:"accountId" firestore:"accountId"`
	AccountName           string               `json:"accountName" firestore:"accountName"`
	AsgName               string               `json:"asgName" firestore:"asgName"`
	CloudFormationStack   string               `json:"cloudFormationStack" firestore:"cloudFormationStack"`
	Config                Config               `json:"config" firestore:"config"`
	Customer              interface{}          `json:"customer" firestore:"customer"`
	Error                 string               `json:"error" firestore:"error"`
	ExecID                string               `json:"execId" firestore:"execId"`
	InstanceTypesDetails  interface{}          `json:"instanceTypesDetails" firestore:"instanceTypesDetails"`
	IsNewASG              bool                 `json:"isNewASG" firestore:"isNewASG"`
	ManagedStatus         string               `json:"managedStatus" firestore:"managedStatus"`
	Mode                  string               `json:"mode" firestore:"mode"`
	Region                string               `json:"region" firestore:"region"`
	Savings               Savings              `json:"savings" firestore:"savings"`
	Spotisize             Spotisize            `json:"spotisize" firestore:"spotisize"`
	SpotisizeError        bool                 `json:"spotisizeError" firestore:"spotisizeError"`
	SpotisizeErrorDesc    string               `json:"spotisizeErrorDesc" firestore:"spotisizeErrorDesc"`
	SpotisizeNotSupported bool                 `json:"spotisizeNotSupported" firestore:"spotisizeNotSupported"`
	SubnetsDetails        interface{}          `json:"subnetsDetails" firestore:"subnetsDetails"`
	TimeCreated           interface{}          `json:"timeCreated" firestore:"timeCreated"`
	TimeModified          interface{}          `json:"timeModified" firestore:"timeModified"`
	TimeStartedUsingSpots time.Time            `json:"timeStartedUsingSpots" firestore:"timeStartedUsingSpots"`
	Usage                 map[string]UsageData `json:"usage" firestore:"usage"`
}

type Config struct {
	CapacityRebalance                   bool     `json:"CapacityRebalance" firestore:"CapacityRebalance"`
	IgnoreFamily                        bool     `json:"IgnoreFamily" firestore:"IgnoreFamily"`
	IgnoreGeneration                    bool     `json:"IgnoreGeneration" firestore:"IgnoreGeneration"`
	MultiplyFactorLower                 int      `json:"MultiplyFactorLower" firestore:"MultiplyFactorLower"`
	MultiplyFactorUpper                 int      `json:"MultiplyFactorUpper" firestore:"MultiplyFactorUpper"`
	OnDemandAllocationStrategy          string   `json:"OnDemandAllocationStrategy" firestore:"OnDemandAllocationStrategy"`
	OnDemandBaseCapacity                int      `json:"OnDemandBaseCapacity" firestore:"OnDemandBaseCapacity"`
	OnDemandPercentageAboveBaseCapacity int      `json:"OnDemandPercentageAboveBaseCapacity" firestore:"OnDemandPercentageAboveBaseCapacity"`
	SpotAllocationStrategy              string   `json:"SpotAllocationStrategy" firestore:"SpotAllocationStrategy"`
	ExcludedSubnets                     []string `json:"excludedSubnets" firestore:"excludedSubnets"`
	ExcludedInstanceTypes               []string `json:"excludedInstanceTypes" firestore:"excludedInstanceTypes"`
}

type SavingCost struct {
	Desired float64 `json:"Desired" firestore:"Desired"`
	Max     float64 `json:"Max" firestore:"Max"`
	Min     float64 `json:"Min" firestore:"Min"`
}

type Savings struct {
	Hourly  SavingCost `json:"hourly" firestore:"hourly"`
	Monthly SavingCost `json:"monthly" firestore:"monthly"`
}

type RecCurCost struct {
	OnDemandHourCost float64 `json:"onDemandHourCost" firestore:"onDemandHourCost"`
	SpotHourCost     float64 `json:"spotHourCost" firestore:"spotHourCost"`
}

type Costs struct {
	AverageDesiredCapacity int        `json:"averageDesiredCapacity" firestore:"averageDesiredCapacity"`
	Cur                    RecCurCost `json:"cur" firestore:"cur"`
	Rec                    RecCurCost `json:"rec" firestore:"rec"`
}

type Instances struct {
	AvailabilityZone        string      `json:"AvailabilityZone" firestore:"AvailabilityZone"`
	HealthStatus            string      `json:"HealthStatus" firestore:"HealthStatus"`
	InstanceID              string      `json:"InstanceId" firestore:"InstanceId"`
	InstanceType            string      `json:"InstanceType" firestore:"InstanceType"`
	LaunchConfigurationName interface{} `json:"LaunchConfigurationName" firestore:"LaunchConfigurationName"`
	LaunchTemplate          interface{} `json:"LaunchTemplate" firestore:"LaunchTemplate"`
	LifecycleState          string      `json:"LifecycleState" firestore:"LifecycleState"`
	ProtectedFromScaleIn    bool        `json:"ProtectedFromScaleIn" firestore:"ProtectedFromScaleIn"`
	WeightedCapacity        string      `json:"WeightedCapacity" firestore:"WeightedCapacity"`
}

type InstancesDistribution struct {
	OnDemandAllocationStrategy          string      `json:"OnDemandAllocationStrategy" firestore:"OnDemandAllocationStrategy"`
	OnDemandBaseCapacity                int64       `json:"OnDemandBaseCapacity" firestore:"OnDemandBaseCapacity"`
	OnDemandPercentageAboveBaseCapacity int64       `json:"OnDemandPercentageAboveBaseCapacity" firestore:"OnDemandPercentageAboveBaseCapacity"`
	SpotAllocationStrategy              string      `json:"SpotAllocationStrategy" firestore:"SpotAllocationStrategy"`
	SpotInstancePools                   interface{} `json:"SpotInstancePools" firestore:"SpotInstancePools"`
	SpotMaxPrice                        interface{} `json:"SpotMaxPrice" firestore:"SpotMaxPrice"`
}

type Overrides struct {
	InstanceType                string      `json:"InstanceType" firestore:"InstanceType"`
	LaunchTemplateSpecification interface{} `json:"LaunchTemplateSpecification" firestore:"LaunchTemplateSpecification"`
	WeightedCapacity            string      `json:"WeightedCapacity" firestore:"WeightedCapacity"`
}

type LaunchTemplate struct {
	LaunchTemplateSpecification interface{} `json:"LaunchTemplateSpecification" firestore:"LaunchTemplateSpecification"`
	Overrides                   []Overrides `json:"Overrides" firestore:"Overrides"`
}

type MixedInstancesPolicy struct {
	InstancesDistribution InstancesDistribution `json:"InstancesDistribution" firestore:"InstancesDistribution"`
	LaunchTemplate        LaunchTemplate        `json:"LaunchTemplate" firestore:"LaunchTemplate"`
}

type Asg struct {
	AutoScalingGroupARN              string                `json:"AutoScalingGroupARN" firestore:"AutoScalingGroupARN"`
	AutoScalingGroupName             string                `json:"AutoScalingGroupName" firestore:"AutoScalingGroupName"`
	AvailabilityZones                []string              `json:"AvailabilityZones" firestore:"AvailabilityZones"`
	CapacityRebalance                bool                  `json:"CapacityRebalance" firestore:"CapacityRebalance"`
	CreatedTime                      interface{}           `json:"CreatedTime" firestore:"CreatedTime"`
	DefaultCooldown                  int                   `json:"DefaultCooldown" firestore:"DefaultCooldown"`
	DesiredCapacity                  int                   `json:"DesiredCapacity" firestore:"DesiredCapacity"`
	EnabledMetrics                   interface{}           `json:"EnabledMetrics" firestore:"EnabledMetrics"`
	HealthCheckGracePeriod           int                   `json:"HealthCheckGracePeriod" firestore:"HealthCheckGracePeriod"`
	HealthCheckType                  string                `json:"HealthCheckType" firestore:"HealthCheckType"`
	Instances                        []Instances           `json:"Instances" firestore:"Instances"`
	LaunchConfigurationName          interface{}           `json:"LaunchConfigurationName" firestore:"LaunchConfigurationName"`
	LaunchTemplate                   interface{}           `json:"LaunchTemplate" firestore:"LaunchTemplate"`
	LoadBalancerNames                interface{}           `json:"LoadBalancerNames" firestore:"LoadBalancerNames"`
	MaxInstanceLifetime              interface{}           `json:"MaxInstanceLifetime" firestore:"MaxInstanceLifetime"`
	MaxSize                          int                   `json:"MaxSize" firestore:"MaxSize"`
	MinSize                          int                   `json:"MinSize" firestore:"MinSize"`
	MixedInstancesPolicy             *MixedInstancesPolicy `json:"MixedInstancesPolicy" firestore:"MixedInstancesPolicy"`
	NewInstancesProtectedFromScaleIn bool                  `json:"NewInstancesProtectedFromScaleIn" firestore:"NewInstancesProtectedFromScaleIn"`
	PlacementGroup                   interface{}           `json:"PlacementGroup" firestore:"PlacementGroup"`
	ServiceLinkedRoleARN             string                `json:"ServiceLinkedRoleARN" firestore:"ServiceLinkedRoleARN"`
	Status                           interface{}           `json:"Status" firestore:"Status"`
	SuspendedProcesses               interface{}           `json:"SuspendedProcesses" firestore:"SuspendedProcesses"`
	Tags                             interface{}           `json:"Tags" firestore:"Tags"`
	TargetGroupARNs                  interface{}           `json:"TargetGroupARNs" firestore:"TargetGroupARNs"`
	TerminationPolicies              []string              `json:"TerminationPolicies" firestore:"TerminationPolicies"`
	VPCZoneIdentifier                string                `json:"VPCZoneIdentifier" firestore:"VPCZoneIdentifier"`
}

type SpotPrices struct {
	InstanceType string  `json:"InstanceType" firestore:"InstanceType"`
	Price        float64 `json:"Price" firestore:"Price"`
}

type CostDetails struct {
	OnDemandNum int          `json:"OnDemandNum" firestore:"OnDemandNum"`
	SpotInstNum int          `json:"SpotInstNum" firestore:"SpotInstNum"`
	SpotPrices  []SpotPrices `json:"SpotPrices" firestore:"SpotPrices"`
	TotalPrice  float64      `json:"TotalPrice" firestore:"TotalPrice"`
}

type AsgCost struct {
	Desired       CostDetails  `json:"Desired" firestore:"Desired"`
	Max           CostDetails  `json:"Max" firestore:"Max"`
	Min           CostDetails  `json:"Min" firestore:"Min"`
	OnDemandPrice float64      `json:"OnDemandPrice" firestore:"OnDemandPrice"`
	SpotAvgPrice  float64      `json:"SpotAvgPrice" firestore:"SpotAvgPrice"`
	SpotPrices    []SpotPrices `json:"SpotPrices" firestore:"SpotPrices"`
}

type CurAsgDetails struct {
	AllOnDemand bool     `json:"allOnDemand" firestore:"allOnDemand"`
	Instances   []string `json:"instances" firestore:"instances"`
}

type Spotisize struct {
	Costs                    Costs         `json:"costs" firestore:"costs"`
	CurAsg                   Asg           `json:"curAsg" firestore:"curAsg"`
	CurAsgCost               AsgCost       `json:"curAsgCost" firestore:"curAsgCost"`
	CurAsgDetails            CurAsgDetails `json:"curAsgDetails" firestore:"curAsgDetails"`
	CurExcludedInstanceTypes []interface{} `json:"curExcludedInstanceTypes" firestore:"curExcludedInstanceTypes"`
	CurExcludedSubnets       interface{}   `json:"curExcludedSubnets" firestore:"curExcludedSubnets"`
	RecAsg                   Asg           `json:"recAsg" firestore:"recAsg"`
	RecAsgCost               AsgCost       `json:"recAsgCost" firestore:"recAsgCost"`
}

type UsageData struct {
	OnDemandInstances InstanceDataCost `json:"onDemandInstances" firestore:"onDemandInstances"`
	SpotInstances     InstanceDataCost `json:"spotInstances" firestore:"spotInstances"`
	TotalSavings      float64          `json:"totalSavings" firestore:"totalSavings"`
}

type InstanceDataCost struct {
	TotalCost float64 `json:"totalCost" firestore:"totalCost"`
}
