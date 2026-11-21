package ai

import "strings"

func (c catalog) resolveBeanRoasterName(patch map[string]any) (ambiguous bool) {
	name, ok := patch["new_roaster_name"].(string)
	if !ok {
		return false
	}
	name = strings.Join(strings.Fields(name), " ")
	if key, _ := patch["roaster_rkey"].(string); key != "" {
		delete(patch, "new_roaster_name")
		return false
	}
	if name == "" {
		patch["new_roaster_name"] = nil
		return false
	}
	for _, item := range c {
		if item.Kind != "roaster" || normalized(name) != normalized(item.Label) {
			continue
		}
		delete(patch, "new_roaster_name")
		if !item.uniqueLabel {
			return true
		}
		patch["roaster_rkey"] = item.rkey
		return false
	}
	patch["new_roaster_name"] = name
	return false
}
