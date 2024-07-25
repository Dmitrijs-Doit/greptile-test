package bqmodels

import "cloud.google.com/go/bigquery"

type ScheduledQueriesMovementResult struct {
	JobID            string  `bigquery:"jobId"`
	Location         string  `bigquery:"location"`
	BillingProjectID string  `bigquery:"billingProjectId"`
	ScheduledTime    string  `bigquery:"scheduledTime"`
	AllJobs          int64   `bigquery:"allJobs"`
	Slots            float64 `bigquery:"slots"`
	SavingsPrice     float64 `bigquery:"savingsPrice"`
}

type StandardScheduledQueriesMovementResult ScheduledQueriesMovementResult
type EnterpriseScheduledQueriesMovementResult ScheduledQueriesMovementResult
type EnterprisePlusScheduledQueriesMovementResult ScheduledQueriesMovementResult

type ScheduledQueriesMovementResultAccessor interface {
	GetJobID() string
	GetLocation() string
	GetBillingProjectID() string
	GetScheduledTime() string
	GetAllJobs() int64
	GetSlots() float64
	GetSavingsPrice() float64
}

func (s StandardScheduledQueriesMovementResult) GetJobID() string {
	return s.JobID
}

func (s StandardScheduledQueriesMovementResult) GetLocation() string {
	return s.Location
}

func (s StandardScheduledQueriesMovementResult) GetBillingProjectID() string {
	return s.BillingProjectID
}

func (s StandardScheduledQueriesMovementResult) GetScheduledTime() string {
	return s.ScheduledTime
}

func (s StandardScheduledQueriesMovementResult) GetAllJobs() int64 {
	return s.AllJobs
}

func (s StandardScheduledQueriesMovementResult) GetSlots() float64 {
	return s.Slots
}

func (s StandardScheduledQueriesMovementResult) GetSavingsPrice() float64 {
	return s.SavingsPrice
}

func (s EnterpriseScheduledQueriesMovementResult) GetJobID() string {
	return s.JobID
}

func (s EnterpriseScheduledQueriesMovementResult) GetLocation() string {
	return s.Location
}

func (s EnterpriseScheduledQueriesMovementResult) GetBillingProjectID() string {
	return s.BillingProjectID
}

func (s EnterpriseScheduledQueriesMovementResult) GetScheduledTime() string {
	return s.ScheduledTime
}

func (s EnterpriseScheduledQueriesMovementResult) GetAllJobs() int64 {
	return s.AllJobs
}

func (s EnterpriseScheduledQueriesMovementResult) GetSlots() float64 {
	return s.Slots
}

func (s EnterpriseScheduledQueriesMovementResult) GetSavingsPrice() float64 {
	return s.SavingsPrice
}

func (s EnterprisePlusScheduledQueriesMovementResult) GetJobID() string {
	return s.JobID
}

func (s EnterprisePlusScheduledQueriesMovementResult) GetLocation() string {
	return s.Location
}

func (s EnterprisePlusScheduledQueriesMovementResult) GetBillingProjectID() string {
	return s.BillingProjectID
}

func (s EnterprisePlusScheduledQueriesMovementResult) GetScheduledTime() string {
	return s.ScheduledTime
}

func (s EnterprisePlusScheduledQueriesMovementResult) GetAllJobs() int64 {
	return s.AllJobs
}

func (s EnterprisePlusScheduledQueriesMovementResult) GetSlots() float64 {
	return s.Slots
}

func (s EnterprisePlusScheduledQueriesMovementResult) GetSavingsPrice() float64 {
	return s.SavingsPrice
}

type UserSlotsTopQueriesResult struct {
	UserID                string  `bigquery:"userId"`
	JobID                 string  `bigquery:"jobId"`
	Location              string  `bigquery:"location"`
	BillingProjectID      string  `bigquery:"billingProjectId"`
	ExecutedQueries       int64   `bigquery:"executedQueries"`
	AvgExecutionTimeSec   float64 `bigquery:"avgExecutionTimeSec"`
	TotalExecutionTimeSec float64 `bigquery:"totalExecutionTimeSec"`
	AvgSlots              float64 `bigquery:"avgSlots"`
	AvgScanTB             float64 `bigquery:"avgScanTB"`
	TotalScanTB           float64 `bigquery:"totalScanTB"`
}

type UserSlotsResult struct {
	UserID string  `bigquery:"userId"`
	Slots  float64 `bigquery:"slots"`
}

type RunUserSlotsResult struct {
	UserSlotsTopQueries []UserSlotsTopQueriesResult
	UserSlots           []UserSlotsResult
}

type RunStandardUserSlotsResult struct {
	UserSlotsTopQueries []UserSlotsTopQueriesResult
	UserSlots           []UserSlotsResult
}

type RunEnterpriseUserSlotsResult struct {
	UserSlotsTopQueries []UserSlotsTopQueriesResult
	UserSlots           []UserSlotsResult
}
type RunEnterprisePlusUserSlotsResult struct {
	UserSlotsTopQueries []UserSlotsTopQueriesResult
	UserSlots           []UserSlotsResult
}

type RunUserSlotsResultAccessor interface {
	GetUserSlotsTopQueries() []UserSlotsTopQueriesResult
	GetUserSlots() []UserSlotsResult
}

func (r RunStandardUserSlotsResult) GetUserSlotsTopQueries() []UserSlotsTopQueriesResult {
	return r.UserSlotsTopQueries
}

func (r RunStandardUserSlotsResult) GetUserSlots() []UserSlotsResult {
	return r.UserSlots
}

func (r RunEnterpriseUserSlotsResult) GetUserSlotsTopQueries() []UserSlotsTopQueriesResult {
	return r.UserSlotsTopQueries
}

func (r RunEnterpriseUserSlotsResult) GetUserSlots() []UserSlotsResult {
	return r.UserSlots
}

func (r RunEnterprisePlusUserSlotsResult) GetUserSlotsTopQueries() []UserSlotsTopQueriesResult {
	return r.UserSlotsTopQueries
}

func (r RunEnterprisePlusUserSlotsResult) GetUserSlots() []UserSlotsResult {
	return r.UserSlots
}

type BillingProjectSlotsTopQueriesResult struct {
	BillingProjectID      string  `bigquery:"billingProjectId"`
	UserID                string  `bigquery:"userId"`
	JobID                 string  `bigquery:"jobId"`
	Location              string  `bigquery:"location"`
	ExecutedQueries       int64   `bigquery:"executedQueries"`
	AvgExecutionTimeSec   float64 `bigquery:"avgExecutionTimeSec"`
	TotalExecutionTimeSec float64 `bigquery:"totalExecutionTimeSec"`
	AvgSlots              float64 `bigquery:"avgSlots"`
	AvgScanTB             float64 `bigquery:"avgScanTB"`
	TotalScanTB           float64 `bigquery:"totalScanTB"`
}

type BillingProjectSlotsTopUsersResult struct {
	BillingProjectID string  `bigquery:"billingProjectId"`
	UserEmail        string  `bigquery:"user_email"`
	Slots            float64 `bigquery:"slots"`
}

type BillingProjectSlotsResult struct {
	BillingProjectID string  `bigquery:"billingProjectId"`
	Slots            float64 `bigquery:"slots"`
}

type RunBillingProjectResult struct {
	Slots      []BillingProjectSlotsResult
	TopQueries []BillingProjectSlotsTopQueriesResult
	TopUsers   []BillingProjectSlotsTopUsersResult
}

type RunBillingProjectResultAccessor interface {
	GetSlots() []BillingProjectSlotsResult
	GetTopQueries() []BillingProjectSlotsTopQueriesResult
	GetTopUsers() []BillingProjectSlotsTopUsersResult
}

type RunStandardBillingProjectResult RunBillingProjectResult

type RunEnterpriseBillingProjectResult RunBillingProjectResult

type RunEnterprisePlusBillingProjectResult RunBillingProjectResult

func (r RunStandardBillingProjectResult) GetSlots() []BillingProjectSlotsResult {
	return r.Slots
}

func (r RunEnterpriseBillingProjectResult) GetSlots() []BillingProjectSlotsResult {
	return r.Slots
}

func (r RunEnterprisePlusBillingProjectResult) GetSlots() []BillingProjectSlotsResult {
	return r.Slots
}

func (r RunStandardBillingProjectResult) GetTopQueries() []BillingProjectSlotsTopQueriesResult {
	return r.TopQueries
}

func (r RunEnterpriseBillingProjectResult) GetTopQueries() []BillingProjectSlotsTopQueriesResult {
	return r.TopQueries
}

func (r RunEnterprisePlusBillingProjectResult) GetTopQueries() []BillingProjectSlotsTopQueriesResult {
	return r.TopQueries
}

func (r RunStandardBillingProjectResult) GetTopUsers() []BillingProjectSlotsTopUsersResult {
	return r.TopUsers
}

func (r RunEnterpriseBillingProjectResult) GetTopUsers() []BillingProjectSlotsTopUsersResult {
	return r.TopUsers
}

func (r RunEnterprisePlusBillingProjectResult) GetTopUsers() []BillingProjectSlotsTopUsersResult {
	return r.TopUsers
}

type SlotsExplorer struct {
	Day      bigquery.NullDate `bigquery:"day"`
	Hour     int               `bigquery:"hour"`
	AvgSlots float64           `bigquery:"avgSlots"`
	MaxSlots float64           `bigquery:"maxSlots"`
}

type FlatRateSlotsExplorerResult SlotsExplorer
type StandardSlotsExplorerResult SlotsExplorer
type EnterpriseSlotsExplorerResult SlotsExplorer
type EnterprisePlusSlotsExplorerResult SlotsExplorer

type SlotsExplorerAccessor interface {
	GetDay() bigquery.NullDate
	GetHour() int
	GetAvgSlots() float64
	GetMaxSlots() float64
}

func (f FlatRateSlotsExplorerResult) GetDay() bigquery.NullDate {
	return f.Day
}

func (s StandardSlotsExplorerResult) GetDay() bigquery.NullDate {
	return s.Day
}

func (e EnterpriseSlotsExplorerResult) GetDay() bigquery.NullDate {
	return e.Day
}

func (ep EnterprisePlusSlotsExplorerResult) GetDay() bigquery.NullDate {
	return ep.Day
}

func (f FlatRateSlotsExplorerResult) GetHour() int {
	return f.Hour
}

func (s StandardSlotsExplorerResult) GetHour() int {
	return s.Hour
}

func (e EnterpriseSlotsExplorerResult) GetHour() int {
	return e.Hour
}

func (ep EnterprisePlusSlotsExplorerResult) GetHour() int {
	return ep.Hour
}

func (f FlatRateSlotsExplorerResult) GetAvgSlots() float64 {
	return f.AvgSlots
}

func (s StandardSlotsExplorerResult) GetAvgSlots() float64 {
	return s.AvgSlots
}

func (e EnterpriseSlotsExplorerResult) GetAvgSlots() float64 {
	return e.AvgSlots
}

func (ep EnterprisePlusSlotsExplorerResult) GetAvgSlots() float64 {
	return ep.AvgSlots
}

func (f FlatRateSlotsExplorerResult) GetMaxSlots() float64 {
	return f.MaxSlots
}

func (s StandardSlotsExplorerResult) GetMaxSlots() float64 {
	return s.MaxSlots
}

func (e EnterpriseSlotsExplorerResult) GetMaxSlots() float64 {
	return e.MaxSlots
}

func (ep EnterprisePlusSlotsExplorerResult) GetMaxSlots() float64 {
	return ep.MaxSlots
}
