package preferences

type PreferenceItem struct {
	Key            string
	MaxCharacters  int
	IsNumber       bool
	Value          *string
	PossibleValues []string
	Pinned         bool
}

func DefaultPreferences() []PreferenceItem {
	return []PreferenceItem{
		{Key: "Tenancy", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false, PossibleValues: []string{"Host", "Shared", "Dedicated", "NA", ""}},
		{Key: "EBSOptimized", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false, PossibleValues: []string{"Yes", "No", ""}},
		{Key: "OperatingSystem", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false, PossibleValues: []string{"Linux", "Windows", "SUSE", "Ubuntu Pro", "Red Hat Enterprise Linux with HA", "RHEL", "NA", ""}},
		{Key: "LicenseModel", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false, PossibleValues: []string{"Bring your own license", "No License required", "NA", ""}},
		{Key: "Region", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false},
		{Key: "Hypervisor", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false},
		{Key: "CurrentGeneration", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false, PossibleValues: []string{"Yes", "No", ""}},
		{Key: "PhysicalProcessor", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false},
		{Key: "ClockSpeed", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false},
		{Key: "ProcessorArchitecture", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: true, PossibleValues: []string{"64-bit", "32-bit or 64-bit", ""}},
		{Key: "SupportedArchitectures", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false},
		{Key: "ENASupported", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false},
		{Key: "EncryptionInTransitSupported", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false, PossibleValues: []string{"Yes", "No", ""}},
		{Key: "SupportedRootDeviceTypes", MaxCharacters: 30, IsNumber: false, Value: nil, Pinned: false},
		{Key: "Cores", MaxCharacters: 30, IsNumber: true, Value: nil, Pinned: false},
		{Key: "Threads", MaxCharacters: 30, IsNumber: true, Value: nil, Pinned: false},
		{Key: "vCPU", MaxCharacters: 30, IsNumber: true, Value: nil, Pinned: false},
		{Key: "MemoryGB", MaxCharacters: 30, IsNumber: true, Value: nil, Pinned: true},
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
