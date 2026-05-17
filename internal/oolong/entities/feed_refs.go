package oolong

// resolveBrewFeedRefs hydrates tea (with nested vendor), vessel and infuser
// references on an oolong brew. Missing refs are silently skipped.
func resolveBrewFeedRefs(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	brew, ok := model.(*Brew)
	if !ok || brew == nil {
		return
	}

	if teaRef, ok := recordData["teaRef"].(string); ok && teaRef != "" {
		if teaData, found := lookup(teaRef); found {
			if tea, err := RecordToTea(teaData, teaRef); err == nil {
				brew.Tea = tea
				if tea != nil {
					if vendorRef, ok := teaData["vendorRef"].(string); ok && vendorRef != "" {
						if vendorData, found := lookup(vendorRef); found {
							if vendor, err := RecordToVendor(vendorData, vendorRef); err == nil {
								brew.Tea.Vendor = vendor
							}
						}
					}
				}
			}
		}
	}

	if vesselRef, ok := recordData["vesselRef"].(string); ok && vesselRef != "" {
		if vesselData, found := lookup(vesselRef); found {
			if vessel, err := RecordToVessel(vesselData, vesselRef); err == nil {
				brew.Vessel = vessel
			}
		}
	}

	if infuserRef, ok := recordData["infuserRef"].(string); ok && infuserRef != "" {
		if infuserData, found := lookup(infuserRef); found {
			if infuser, err := RecordToInfuser(infuserData, infuserRef); err == nil {
				brew.Infuser = infuser
			}
		}
	}
}

// resolveTeaFeedRef hydrates a tea's vendor reference.
func resolveTeaFeedRef(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	tea, ok := model.(*Tea)
	if !ok || tea == nil {
		return
	}
	vendorRef, ok := recordData["vendorRef"].(string)
	if !ok || vendorRef == "" {
		return
	}
	vendorData, found := lookup(vendorRef)
	if !found {
		return
	}
	if vendor, err := RecordToVendor(vendorData, vendorRef); err == nil {
		tea.Vendor = vendor
	}
}
