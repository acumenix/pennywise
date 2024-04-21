package preferences

import "github.com/aws/aws-sdk-go-v2/aws"

type PreferenceItem struct {
	Key            string
	MaxCharacters  int
	IsNumber       bool
	Value          *string
	PossibleValues []string
	Pinned         bool
	CanBePinned    bool
	Unit           string
}

func DefaultPreferences() []PreferenceItem {
	return []PreferenceItem{
		{Key: "Tenancy", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false, PossibleValues: []string{"Host", "Shared", "Dedicated", "NA", ""}},
		{Key: "EBSOptimized", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false, PossibleValues: []string{"Yes", "No", ""}},
		//{Key: "OperatingSystem", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false, PossibleValues: []string{"Linux", "Windows", "SUSE", "Ubuntu Pro", "Red Hat Enterprise Linux with HA", "RHEL", "NA", ""}},
		{Key: "LicenseModel", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false, PossibleValues: []string{"Bring your own license", "No License required", "NA", ""}},
		{Key: "Region", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false},
		//{Key: "Hypervisor", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false},
		{Key: "CurrentGeneration", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false, PossibleValues: []string{"Yes", "No", ""}},
		{Key: "PhysicalProcessor", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false},
		{Key: "ClockSpeed", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false},
		{Key: "ProcessorArchitecture", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: true, PossibleValues: []string{"64-bit", "32-bit or 64-bit", ""}},
		//{Key: "SupportedArchitectures", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false},
		{Key: "ENASupported", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false},
		//{Key: "EncryptionInTransitSupported", MaxCharacters: 30, IsNumber: false, Value: nil, CanBePinned: true, Pinned: false, PossibleValues: []string{"Yes", "No", ""}},
		{Key: "SupportedRootDeviceTypes", MaxCharacters: 30, IsNumber: false, Value: aws.String("EBSOnly"), CanBePinned: false, Pinned: false, PossibleValues: []string{"EBSOnly"}},
		//{Key: "Cores", MaxCharacters: 30, IsNumber: true, Value: nil, CanBePinned: true, Pinned: false},
		//{Key: "Threads", MaxCharacters: 30, IsNumber: true, Value: nil, CanBePinned: true, Pinned: false},
		{Key: "vCPU", MaxCharacters: 30, IsNumber: true, Value: nil, CanBePinned: true, Pinned: false},
		{Key: "MemoryGB", MaxCharacters: 30, IsNumber: true, Value: nil, CanBePinned: true, Pinned: true},
		{Key: "CPUBreathingRoom", MaxCharacters: 30, IsNumber: true, Value: aws.String("10"), CanBePinned: false, Pinned: false, Unit: "%"},
		{Key: "MemoryBreathingRoom", MaxCharacters: 30, IsNumber: true, Value: aws.String("10"), CanBePinned: false, Pinned: false, Unit: "%"},
		{Key: "NetworkBreathingRoom", MaxCharacters: 30, IsNumber: true, Value: aws.String("10"), CanBePinned: false, Pinned: false, Unit: "%"},
		{Key: "ObservabilityTimePeriod", MaxCharacters: 30, IsNumber: false, Value: aws.String("7"), CanBePinned: false, Pinned: false, Unit: "days", PossibleValues: []string{"7"}},
	}
}

func Export(pref []PreferenceItem) map[string]*string {
	ex := map[string]*string{}
	for _, p := range pref {
		if p.Pinned {
			ex[p.Key] = nil
		} else {
			if p.Value != nil {
				ex[p.Key] = p.Value
			}
		}
	}
	return ex
}
