package report

import (
	"reflect"
	"testing"
)

func Test_getColsFromInterval(t *testing.T) {
	type args struct {
		timeInterval TimeInterval
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "TimeIntervalDay",
			args: args{
				timeInterval: TimeIntervalDay,
			},
			want: []string{
				"datetime:year",
				"datetime:month",
				"datetime:day",
			},
		},
		{
			name: "TimeIntervalWeek",
			args: args{
				timeInterval: TimeIntervalWeek,
			},
			want: []string{
				"datetime:year",
				"datetime:week",
			},
		},
		{
			name: "TimeIntervalMonth",
			args: args{
				timeInterval: TimeIntervalMonth,
			},
			want: []string{
				"datetime:year",
				"datetime:month",
			},
		},
		{
			name: "TimeIntervalYear",
			args: args{
				timeInterval: TimeIntervalYear,
			},
			want: []string{
				"datetime:year",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetColsFromInterval(tt.args.timeInterval); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetColsFromInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsUsingDimension(t *testing.T) {
	// Test case 1: Config is nil, should return false
	var c *Config

	var expected bool

	actual := c.IsUsingDimension("dim1")

	if actual != expected {
		t.Errorf("Test case 1 failed: expected %v but got %v", expected, actual)
	}
	// Test case 2: Dimension is in Rows, should return true
	c = &Config{
		Rows: []string{"dim1", "dim2"},
		Cols: []string{"dim3", "dim4"},
		Filters: []*ConfigFilter{
			{
				BaseConfigFilter: BaseConfigFilter{
					ID: "filter1",
				},
			},
			{
				BaseConfigFilter: BaseConfigFilter{
					ID: "filter2",
				},
			},
		},
	}

	expected = true
	actual = c.IsUsingDimension("dim1")

	if actual != expected {
		t.Errorf("Test case 2 failed: expected %v but got %v", expected, actual)
	}

	// Test case 3: Dimension is in Cols, should return true
	expected = true
	actual = c.IsUsingDimension("dim3")

	if actual != expected {
		t.Errorf("Test case 3 failed: expected %v but got %v", expected, actual)
	}

	// Test case 4: Dimension is in Filters, should return true
	expected = true
	actual = c.IsUsingDimension("filter2")

	if actual != expected {
		t.Errorf("Test case 4 failed: expected %v but got %v", expected, actual)
	}

	// Test case 5: Dimension is not in any of Rows, Cols or Filters, should return false
	expected = false
	actual = c.IsUsingDimension("dim5")

	if actual != expected {
		t.Errorf("Test case 5 failed: expected %v but got %v", expected, actual)
	}
}

func TestAggregator_Validate(t *testing.T) {
	type args struct {
		aggregator Aggregator
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Invalid",
			args: args{
				aggregator: Aggregator("INVALID"),
			},
		},
		{
			name: "Correct aggregator",
			args: args{
				aggregator: AggregatorPercentTotal,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.aggregator.Validate(); got != tt.want {
				t.Errorf("Aggregator.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeInterval_Validate(t *testing.T) {
	type args struct {
		timeInterval TimeInterval
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Invalid",
			args: args{
				timeInterval: TimeInterval("INVALID"),
			},
		},
		{
			name: "Correct time interval",
			args: args{
				timeInterval: TimeIntervalDay,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.timeInterval.Validate(); got != tt.want {
				t.Errorf("TimeInterval.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeSettings_String(t *testing.T) {
	type args struct {
		timeSettings TimeSettings
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Current month",
			args: args{
				timeSettings: TimeSettings{
					Mode: TimeSettingsModeCurrent,
					Unit: TimeSettingsUnitMonth,
				},
			},
			want: "Current month",
		},
		{
			name: "Last 45 days",
			args: args{
				timeSettings: TimeSettings{
					Mode:   TimeSettingsModeLast,
					Unit:   TimeSettingsUnitDay,
					Amount: 45,
				},
			},
			want: "Last 45 days",
		},
		{
			name: "Last 2 weeks to date",
			args: args{
				timeSettings: TimeSettings{
					Mode:           TimeSettingsModeLast,
					Unit:           TimeSettingsUnitWeek,
					Amount:         2,
					IncludeCurrent: true,
				},
			},
			want: "Last 2 weeks to date",
		},
		{
			name: "Custom range",
			args: args{
				timeSettings: TimeSettings{
					Mode: TimeSettingsModeCustom,
				},
			},
			want: "Custom range",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.timeSettings.String(); got != tt.want {
				t.Errorf("TimeSettings.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeSort_Validate(t *testing.T) {
	type args struct {
		sort Sort
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Invalid",
			args: args{
				sort: Sort("INVALID"),
			},
		},
		{
			name: "Correct time interval",
			args: args{
				sort: SortAsc,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.sort.Validate(); got != tt.want {
				t.Errorf("Sort.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeSettingsUnit_Validate(t *testing.T) {
	type args struct {
		timeSettingsUnit TimeSettingsUnit
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Invalid",
			args: args{
				timeSettingsUnit: TimeSettingsUnit("INVALID"),
			},
		},
		{
			name: "Correct time settings unit",
			args: args{
				timeSettingsUnit: TimeSettingsUnitDay,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.timeSettingsUnit.Validate(); got != tt.want {
				t.Errorf("TimeSettingsUnit.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeSettings_Validate(t *testing.T) {
	type args struct {
		timeSettings TimeSettings
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Invalid mode",
			args: args{
				timeSettings: TimeSettings{
					Mode: TimeSettingsMode("INVALID"),
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid unit",
			args: args{
				timeSettings: TimeSettings{
					Mode: TimeSettingsModeCurrent,
					Unit: TimeSettingsUnit("INVALID"),
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid amount, left side",
			args: args{
				timeSettings: TimeSettings{
					Mode:   TimeSettingsModeLast,
					Unit:   TimeSettingsUnitMonth,
					Amount: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid amount, right side",
			args: args{
				timeSettings: TimeSettings{
					Mode:   TimeSettingsModeLast,
					Unit:   TimeSettingsUnitMonth,
					Amount: 1000000,
				},
			},
			wantErr: true,
		},
		{
			name: "Happy path",
			args: args{
				timeSettings: TimeSettings{
					Mode:   TimeSettingsModeLast,
					Unit:   TimeSettingsUnitMonth,
					Amount: 9,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.timeSettings.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TimeSettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
