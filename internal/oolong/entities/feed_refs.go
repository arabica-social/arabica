package oolong

// RecordLookup returns the raw record for an AT-URI reference. Callers decide
// whether that source is the feed index, witness cache, or a PDS lookup.
type RecordLookup func(refURI string) (map[string]any, bool)

// resolveBrewFeedRefs hydrates tea (with nested vendor), vessel and infuser
// references on an oolong brew. Missing refs are silently skipped.
func resolveBrewFeedRefs(model any, recordData map[string]any, lookup func(string) (map[string]any, bool)) {
	brew, ok := model.(*Brew)
	if !ok || brew == nil {
		return
	}
	HydrateBrewRefs(brew, recordData, lookup)
}

// HydrateBrewRefs hydrates tea (with nested vendor), vessel and infuser
// references on an oolong brew. Missing refs are silently skipped.
func HydrateBrewRefs(brew *Brew, recordData map[string]any, lookup RecordLookup) {
	if brew == nil || recordData == nil || lookup == nil {
		return
	}

	if teaRef, ok := recordData["teaRef"].(string); ok && teaRef != "" {
		if teaData, found := lookup(teaRef); found {
			if tea, err := RecordToTea(teaData, teaRef); err == nil && brew.Tea == nil {
				brew.Tea = tea
			}
			HydrateTeaRefs(brew.Tea, teaData, lookup)
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
	HydrateTeaRefs(tea, recordData, lookup)
}

// HydrateTeaRefs hydrates a tea's vendor reference.
func HydrateTeaRefs(tea *Tea, recordData map[string]any, lookup RecordLookup) {
	if tea == nil || recordData == nil || lookup == nil || tea.Vendor != nil {
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
