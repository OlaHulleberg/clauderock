package awsutil

// Region represents an AWS region with its identifier and description
type Region struct {
	ID   string
	Name string
}

// GetRegions returns a list of all AWS regions
// Most commonly used regions are listed first for better UX
func GetRegions() []Region {
	return []Region{
		// Most common regions first
		{ID: "us-east-1", Name: "US East (N. Virginia)"},
		{ID: "us-west-2", Name: "US West (Oregon)"},
		{ID: "eu-west-1", Name: "Europe (Ireland)"},
		{ID: "eu-central-1", Name: "Europe (Frankfurt)"},
		{ID: "eu-north-1", Name: "Europe (Stockholm)"},
		{ID: "ap-southeast-1", Name: "Asia Pacific (Singapore)"},
		{ID: "ap-northeast-1", Name: "Asia Pacific (Tokyo)"},

		// US regions
		{ID: "us-east-2", Name: "US East (Ohio)"},
		{ID: "us-west-1", Name: "US West (N. California)"},

		// Europe regions
		{ID: "eu-west-2", Name: "Europe (London)"},
		{ID: "eu-west-3", Name: "Europe (Paris)"},
		{ID: "eu-south-1", Name: "Europe (Milan)"},
		{ID: "eu-south-2", Name: "Europe (Spain)"},
		{ID: "eu-central-2", Name: "Europe (Zurich)"},

		// Asia Pacific regions
		{ID: "ap-south-1", Name: "Asia Pacific (Mumbai)"},
		{ID: "ap-south-2", Name: "Asia Pacific (Hyderabad)"},
		{ID: "ap-northeast-2", Name: "Asia Pacific (Seoul)"},
		{ID: "ap-northeast-3", Name: "Asia Pacific (Osaka)"},
		{ID: "ap-southeast-2", Name: "Asia Pacific (Sydney)"},
		{ID: "ap-southeast-3", Name: "Asia Pacific (Jakarta)"},
		{ID: "ap-southeast-4", Name: "Asia Pacific (Melbourne)"},
		{ID: "ap-east-1", Name: "Asia Pacific (Hong Kong)"},

		// Canada
		{ID: "ca-central-1", Name: "Canada (Central)"},
		{ID: "ca-west-1", Name: "Canada (Calgary)"},

		// South America
		{ID: "sa-east-1", Name: "South America (São Paulo)"},

		// Middle East
		{ID: "me-south-1", Name: "Middle East (Bahrain)"},
		{ID: "me-central-1", Name: "Middle East (UAE)"},

		// Africa
		{ID: "af-south-1", Name: "Africa (Cape Town)"},

		// Israel
		{ID: "il-central-1", Name: "Israel (Tel Aviv)"},
	}
}
