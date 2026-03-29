package atproto

import (
	"fmt"
	"time"

	"arabica/internal/models"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// toFloat64 extracts a numeric value from an interface{} that may be int or float64.
// JSON decoding produces float64, but in-memory maps may contain int.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

// ========== Recipe Conversions ==========

// RecipeToRecord converts a models.Recipe to an atproto record map
func RecipeToRecord(recipe *models.Recipe, brewerURI string) (map[string]any, error) {
	record := map[string]any{
		"$type":     NSIDRecipe,
		"name":      recipe.Name,
		"createdAt": recipe.CreatedAt.Format(time.RFC3339),
	}

	if brewerURI != "" {
		record["brewerRef"] = brewerURI
	}
	if recipe.BrewerType != "" {
		record["brewerType"] = recipe.BrewerType
	}
	if recipe.CoffeeAmount > 0 {
		record["coffeeAmount"] = int(recipe.CoffeeAmount * 10)
	}
	if recipe.WaterAmount > 0 {
		record["waterAmount"] = int(recipe.WaterAmount * 10)
	}
	if recipe.Notes != "" {
		record["notes"] = recipe.Notes
	}
	if recipe.SourceRef != "" {
		record["sourceRef"] = recipe.SourceRef
	}

	if len(recipe.Pours) > 0 {
		pours := make([]map[string]any, len(recipe.Pours))
		for i, pour := range recipe.Pours {
			pours[i] = map[string]any{
				"waterAmount": pour.WaterAmount,
				"timeSeconds": pour.TimeSeconds,
			}
		}
		record["pours"] = pours
	}

	return record, nil
}

// RecordToRecipe converts an atproto record map to a models.Recipe
func RecordToRecipe(record map[string]any, atURI string) (*models.Recipe, error) {
	recipe := &models.Recipe{}

	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		recipe.RKey = parsedURI.RecordKey().String()
	}

	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}
	recipe.Name = name

	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	recipe.CreatedAt = createdAt

	if brewerType, ok := record["brewerType"].(string); ok {
		recipe.BrewerType = brewerType
	}
	if coffeeAmount, ok := record["coffeeAmount"].(float64); ok {
		recipe.CoffeeAmount = coffeeAmount / 10.0
	}
	if waterAmount, ok := record["waterAmount"].(float64); ok {
		recipe.WaterAmount = waterAmount / 10.0
	}
	if notes, ok := record["notes"].(string); ok {
		recipe.Notes = notes
	}
	if sourceRef, ok := record["sourceRef"].(string); ok {
		recipe.SourceRef = sourceRef
	}

	if poursRaw, ok := record["pours"].([]any); ok {
		recipe.Pours = make([]*models.Pour, len(poursRaw))
		for i, pourRaw := range poursRaw {
			pourMap, ok := pourRaw.(map[string]any)
			if !ok {
				continue
			}
			pour := &models.Pour{}
			if waterAmount, ok := pourMap["waterAmount"].(float64); ok {
				pour.WaterAmount = int(waterAmount)
			}
			if timeSeconds, ok := pourMap["timeSeconds"].(float64); ok {
				pour.TimeSeconds = int(timeSeconds)
			}
			pour.PourNumber = i + 1
			recipe.Pours[i] = pour
		}
	}

	return recipe, nil
}

// ========== Brew Conversions ==========

// BrewToRecord converts a models.Brew to an atproto record map
// Note: References (beanRef, grinderRef, brewerRef, recipeRef) must be AT-URIs
func BrewToRecord(brew *models.Brew, beanURI, grinderURI, brewerURI, recipeURI string) (map[string]any, error) {
	if beanURI == "" {
		return nil, fmt.Errorf("beanRef (AT-URI) is required")
	}

	record := map[string]any{
		"$type":     NSIDBrew,
		"beanRef":   beanURI,
		"createdAt": brew.CreatedAt.Format(time.RFC3339),
	}

	// Optional fields
	if brew.Method != "" {
		record["method"] = brew.Method
	}
	if brew.Temperature > 0 {
		// Convert float to tenths (93.5 -> 935)
		record["temperature"] = int(brew.Temperature * 10)
	}
	if brew.WaterAmount > 0 {
		record["waterAmount"] = brew.WaterAmount
	}
	if brew.CoffeeAmount > 0 {
		record["coffeeAmount"] = brew.CoffeeAmount
	}
	if brew.TimeSeconds > 0 {
		record["timeSeconds"] = brew.TimeSeconds
	}
	if brew.GrindSize != "" {
		record["grindSize"] = brew.GrindSize
	}
	if grinderURI != "" {
		record["grinderRef"] = grinderURI
	}
	if brewerURI != "" {
		record["brewerRef"] = brewerURI
	}
	if recipeURI != "" {
		record["recipeRef"] = recipeURI
	}
	if brew.TastingNotes != "" {
		record["tastingNotes"] = brew.TastingNotes
	}
	if brew.Rating > 0 {
		record["rating"] = brew.Rating
	}

	// Convert pours to embedded array
	if len(brew.Pours) > 0 {
		pours := make([]map[string]any, len(brew.Pours))
		for i, pour := range brew.Pours {
			pours[i] = map[string]any{
				"waterAmount": pour.WaterAmount,
				"timeSeconds": pour.TimeSeconds,
			}
		}
		record["pours"] = pours
	}

	// Espresso-specific params
	if brew.EspressoParams != nil {
		ep := map[string]any{}
		if brew.EspressoParams.YieldWeight > 0 {
			ep["yieldWeight"] = int(brew.EspressoParams.YieldWeight * 10) // tenths of a gram
		}
		if brew.EspressoParams.Pressure > 0 {
			ep["pressure"] = int(brew.EspressoParams.Pressure * 10) // tenths of a bar
		}
		if brew.EspressoParams.PreInfusionSeconds > 0 {
			ep["preInfusionSeconds"] = brew.EspressoParams.PreInfusionSeconds
		}
		if len(ep) > 0 {
			record["espressoParams"] = ep
		}
	}

	// Pour-over-specific params
	if brew.PouroverParams != nil {
		pp := map[string]any{}
		if brew.PouroverParams.BloomWater > 0 {
			pp["bloomWater"] = brew.PouroverParams.BloomWater
		}
		if brew.PouroverParams.BloomSeconds > 0 {
			pp["bloomSeconds"] = brew.PouroverParams.BloomSeconds
		}
		if brew.PouroverParams.DrawdownSeconds > 0 {
			pp["drawdownSeconds"] = brew.PouroverParams.DrawdownSeconds
		}
		if brew.PouroverParams.BypassWater > 0 {
			pp["bypassWater"] = brew.PouroverParams.BypassWater
		}
		if len(pp) > 0 {
			record["pouroverParams"] = pp
		}
	}

	return record, nil
}

// RecordToBrew converts an atproto record map to a models.Brew
// The atURI parameter should be the full AT-URI of this brew record
func RecordToBrew(record map[string]any, atURI string) (*models.Brew, error) {
	brew := &models.Brew{}

	// Extract rkey from AT-URI
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		brew.RKey = parsedURI.RecordKey().String()
	}

	// Required field: beanRef
	beanRef, ok := record["beanRef"].(string)
	if !ok || beanRef == "" {
		return nil, fmt.Errorf("beanRef is required")
	}
	// Store the beanRef for later resolution
	// For now, we'll just note it exists but won't resolve it here

	// Required field: createdAt
	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	brew.CreatedAt = createdAt

	// Optional fields
	if method, ok := record["method"].(string); ok {
		brew.Method = method
	}
	if temp, ok := record["temperature"].(float64); ok {
		// Convert from tenths to float (935 -> 93.5)
		brew.Temperature = temp / 10.0
	}
	if waterAmount, ok := record["waterAmount"].(float64); ok {
		brew.WaterAmount = int(waterAmount)
	}
	if coffeeAmount, ok := record["coffeeAmount"].(float64); ok {
		brew.CoffeeAmount = int(coffeeAmount)
	}
	if timeSeconds, ok := record["timeSeconds"].(float64); ok {
		brew.TimeSeconds = int(timeSeconds)
	}
	if grindSize, ok := record["grindSize"].(string); ok {
		brew.GrindSize = grindSize
	}
	if tastingNotes, ok := record["tastingNotes"].(string); ok {
		brew.TastingNotes = tastingNotes
	}
	if rating, ok := record["rating"].(float64); ok {
		brew.Rating = int(rating)
	}

	// Convert pours from embedded array
	if poursRaw, ok := record["pours"].([]any); ok {
		brew.Pours = make([]*models.Pour, len(poursRaw))
		for i, pourRaw := range poursRaw {
			pourMap, ok := pourRaw.(map[string]any)
			if !ok {
				continue
			}
			pour := &models.Pour{}
			if waterAmount, ok := pourMap["waterAmount"].(float64); ok {
				pour.WaterAmount = int(waterAmount)
			}
			if timeSeconds, ok := pourMap["timeSeconds"].(float64); ok {
				pour.TimeSeconds = int(timeSeconds)
			}
			pour.PourNumber = i + 1 // Sequential numbering
			brew.Pours[i] = pour
		}
	}

	// Espresso params
	if epRaw, ok := record["espressoParams"].(map[string]any); ok {
		ep := &models.EspressoParams{}
		if v, ok := toFloat64(epRaw["yieldWeight"]); ok {
			ep.YieldWeight = v / 10.0
		}
		if v, ok := toFloat64(epRaw["pressure"]); ok {
			ep.Pressure = v / 10.0
		}
		if v, ok := toFloat64(epRaw["preInfusionSeconds"]); ok {
			ep.PreInfusionSeconds = int(v)
		}
		brew.EspressoParams = ep
	}

	// Pour-over params
	if ppRaw, ok := record["pouroverParams"].(map[string]any); ok {
		pp := &models.PouroverParams{}
		if v, ok := toFloat64(ppRaw["bloomWater"]); ok {
			pp.BloomWater = int(v)
		}
		if v, ok := toFloat64(ppRaw["bloomSeconds"]); ok {
			pp.BloomSeconds = int(v)
		}
		if v, ok := toFloat64(ppRaw["drawdownSeconds"]); ok {
			pp.DrawdownSeconds = int(v)
		}
		if v, ok := toFloat64(ppRaw["bypassWater"]); ok {
			pp.BypassWater = int(v)
		}
		brew.PouroverParams = pp
	}

	return brew, nil
}

// ========== Bean Conversions ==========

// BeanToRecord converts a models.Bean to an atproto record map
func BeanToRecord(bean *models.Bean, roasterURI string) (map[string]any, error) {
	record := map[string]any{
		"$type":     NSIDBean,
		"name":      bean.Name,
		"createdAt": bean.CreatedAt.Format(time.RFC3339),
	}

	// Optional fields
	if bean.Origin != "" {
		record["origin"] = bean.Origin
	}
	if bean.Variety != "" {
		record["variety"] = bean.Variety
	}
	if bean.RoastLevel != "" {
		record["roastLevel"] = bean.RoastLevel
	}
	if bean.Process != "" {
		record["process"] = bean.Process
	}
	if bean.Description != "" {
		record["description"] = bean.Description
	}
	if roasterURI != "" {
		record["roasterRef"] = roasterURI
	}
	if bean.Rating != nil {
		record["rating"] = *bean.Rating
	}
	// Always include closed field (defaults to false)
	record["closed"] = bean.Closed
	if bean.SourceRef != "" {
		record["sourceRef"] = bean.SourceRef
	}

	return record, nil
}

// RecordToBean converts an atproto record map to a models.Bean
func RecordToBean(record map[string]any, atURI string) (*models.Bean, error) {
	bean := &models.Bean{}

	// Extract rkey from AT-URI
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		bean.RKey = parsedURI.RecordKey().String()
	}

	// Required field: name
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}
	bean.Name = name

	// Required field: createdAt
	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	bean.CreatedAt = createdAt

	// Optional fields
	if origin, ok := record["origin"].(string); ok {
		bean.Origin = origin
	}
	if variety, ok := record["variety"].(string); ok {
		bean.Variety = variety
	}
	if roastLevel, ok := record["roastLevel"].(string); ok {
		bean.RoastLevel = roastLevel
	}
	if process, ok := record["process"].(string); ok {
		bean.Process = process
	}
	if description, ok := record["description"].(string); ok {
		bean.Description = description
	}
	if rating, ok := record["rating"].(float64); ok {
		r := int(rating)
		bean.Rating = &r
	}
	if closed, ok := record["closed"].(bool); ok {
		bean.Closed = closed
	}
	if sourceRef, ok := record["sourceRef"].(string); ok {
		bean.SourceRef = sourceRef
	}

	return bean, nil
}

// ========== Roaster Conversions ==========

// RoasterToRecord converts a models.Roaster to an atproto record map
func RoasterToRecord(roaster *models.Roaster) (map[string]any, error) {
	record := map[string]any{
		"$type":     NSIDRoaster,
		"name":      roaster.Name,
		"createdAt": roaster.CreatedAt.Format(time.RFC3339),
	}

	// Optional fields
	if roaster.Location != "" {
		record["location"] = roaster.Location
	}
	if roaster.Website != "" {
		record["website"] = roaster.Website
	}
	if roaster.SourceRef != "" {
		record["sourceRef"] = roaster.SourceRef
	}

	return record, nil
}

// RecordToRoaster converts an atproto record map to a models.Roaster
func RecordToRoaster(record map[string]any, atURI string) (*models.Roaster, error) {
	roaster := &models.Roaster{}

	// Extract rkey from AT-URI
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		roaster.RKey = parsedURI.RecordKey().String()
	}

	// Required field: name
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}
	roaster.Name = name

	// Required field: createdAt
	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	roaster.CreatedAt = createdAt

	// Optional fields
	if location, ok := record["location"].(string); ok {
		roaster.Location = location
	}
	if website, ok := record["website"].(string); ok {
		roaster.Website = website
	}
	if sourceRef, ok := record["sourceRef"].(string); ok {
		roaster.SourceRef = sourceRef
	}

	return roaster, nil
}

// ========== Grinder Conversions ==========

// GrinderToRecord converts a models.Grinder to an atproto record map
func GrinderToRecord(grinder *models.Grinder) (map[string]any, error) {
	record := map[string]any{
		"$type":     NSIDGrinder,
		"name":      grinder.Name,
		"createdAt": grinder.CreatedAt.Format(time.RFC3339),
	}

	// Optional fields
	if grinder.GrinderType != "" {
		record["grinderType"] = grinder.GrinderType
	}
	if grinder.BurrType != "" {
		record["burrType"] = grinder.BurrType
	}
	if grinder.Notes != "" {
		record["notes"] = grinder.Notes
	}
	if grinder.SourceRef != "" {
		record["sourceRef"] = grinder.SourceRef
	}

	return record, nil
}

// RecordToGrinder converts an atproto record map to a models.Grinder
func RecordToGrinder(record map[string]any, atURI string) (*models.Grinder, error) {
	grinder := &models.Grinder{}

	// Extract rkey from AT-URI
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		grinder.RKey = parsedURI.RecordKey().String()
	}

	// Required field: name
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}
	grinder.Name = name

	// Required field: createdAt
	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	grinder.CreatedAt = createdAt

	// Optional fields
	if grinderType, ok := record["grinderType"].(string); ok {
		grinder.GrinderType = grinderType
	}
	if burrType, ok := record["burrType"].(string); ok {
		grinder.BurrType = burrType
	}
	if notes, ok := record["notes"].(string); ok {
		grinder.Notes = notes
	}
	if sourceRef, ok := record["sourceRef"].(string); ok {
		grinder.SourceRef = sourceRef
	}

	return grinder, nil
}

// ========== Brewer Conversions ==========

// BrewerToRecord converts a models.Brewer to an atproto record map
func BrewerToRecord(brewer *models.Brewer) (map[string]any, error) {
	record := map[string]any{
		"$type":     NSIDBrewer,
		"name":      brewer.Name,
		"createdAt": brewer.CreatedAt.Format(time.RFC3339),
	}

	// Optional fields
	if brewer.Description != "" {
		record["description"] = brewer.Description
	}
	if brewer.BrewerType != "" {
		record["brewerType"] = brewer.BrewerType
	}
	if brewer.SourceRef != "" {
		record["sourceRef"] = brewer.SourceRef
	}

	return record, nil
}

// RecordToBrewer converts an atproto record map to a models.Brewer
func RecordToBrewer(record map[string]any, atURI string) (*models.Brewer, error) {
	brewer := &models.Brewer{}

	// Extract rkey from AT-URI
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		brewer.RKey = parsedURI.RecordKey().String()
	}

	// Required field: name
	name, ok := record["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}
	brewer.Name = name

	// Required field: createdAt
	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	brewer.CreatedAt = createdAt

	// Optional fields
	if description, ok := record["description"].(string); ok {
		brewer.Description = description
	}
	if brewerType, ok := record["brewerType"].(string); ok {
		brewer.BrewerType = brewerType
	}
	if sourceRef, ok := record["sourceRef"].(string); ok {
		brewer.SourceRef = sourceRef
	}

	return brewer, nil
}

// ========== Like Conversions ==========

// LikeToRecord converts a models.Like to an atproto record map
// Uses com.atproto.repo.strongRef format for the subject
func LikeToRecord(like *models.Like) (map[string]any, error) {
	if like.SubjectURI == "" {
		return nil, fmt.Errorf("subject URI is required")
	}
	if like.SubjectCID == "" {
		return nil, fmt.Errorf("subject CID is required")
	}

	record := map[string]any{
		"$type": NSIDLike,
		"subject": map[string]any{
			"uri": like.SubjectURI,
			"cid": like.SubjectCID,
		},
		"createdAt": like.CreatedAt.Format(time.RFC3339),
	}

	return record, nil
}

// RecordToLike converts an atproto record map to a models.Like
func RecordToLike(record map[string]any, atURI string) (*models.Like, error) {
	like := &models.Like{}

	// Extract rkey from AT-URI
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		like.RKey = parsedURI.RecordKey().String()
	}

	// Required field: subject (strongRef)
	subject, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("subject is required")
	}
	subjectURI, ok := subject["uri"].(string)
	if !ok || subjectURI == "" {
		return nil, fmt.Errorf("subject.uri is required")
	}
	like.SubjectURI = subjectURI

	subjectCID, ok := subject["cid"].(string)
	if !ok || subjectCID == "" {
		return nil, fmt.Errorf("subject.cid is required")
	}
	like.SubjectCID = subjectCID

	// Required field: createdAt
	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	like.CreatedAt = createdAt

	return like, nil
}

// ========== Comment Conversions ==========

// CommentToRecord converts a models.Comment to an atproto record map
// Uses com.atproto.repo.strongRef format for the subject
func CommentToRecord(comment *models.Comment) (map[string]any, error) {
	if comment.SubjectURI == "" {
		return nil, fmt.Errorf("subject URI is required")
	}
	if comment.SubjectCID == "" {
		return nil, fmt.Errorf("subject CID is required")
	}
	if comment.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	record := map[string]any{
		"$type": NSIDComment,
		"subject": map[string]any{
			"uri": comment.SubjectURI,
			"cid": comment.SubjectCID,
		},
		"text":      comment.Text,
		"createdAt": comment.CreatedAt.Format(time.RFC3339),
	}

	// Add optional parent reference for replies
	if comment.ParentURI != "" && comment.ParentCID != "" {
		record["parent"] = map[string]any{
			"uri": comment.ParentURI,
			"cid": comment.ParentCID,
		}
	}

	return record, nil
}

// RecordToComment converts an atproto record map to a models.Comment
func RecordToComment(record map[string]any, atURI string) (*models.Comment, error) {
	comment := &models.Comment{}

	// Extract rkey from AT-URI
	if atURI != "" {
		parsedURI, err := syntax.ParseATURI(atURI)
		if err != nil {
			return nil, fmt.Errorf("invalid AT-URI: %w", err)
		}
		comment.RKey = parsedURI.RecordKey().String()
	}

	// Required field: subject (strongRef)
	subject, ok := record["subject"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("subject is required")
	}
	subjectURI, ok := subject["uri"].(string)
	if !ok || subjectURI == "" {
		return nil, fmt.Errorf("subject.uri is required")
	}
	comment.SubjectURI = subjectURI

	subjectCID, ok := subject["cid"].(string)
	if !ok || subjectCID == "" {
		return nil, fmt.Errorf("subject.cid is required")
	}
	comment.SubjectCID = subjectCID

	// Required field: text
	text, ok := record["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required")
	}
	comment.Text = text

	// Required field: createdAt
	createdAtStr, ok := record["createdAt"].(string)
	if !ok {
		return nil, fmt.Errorf("createdAt is required")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid createdAt format: %w", err)
	}
	comment.CreatedAt = createdAt

	// Optional field: parent (strongRef for replies)
	if parent, ok := record["parent"].(map[string]any); ok {
		if parentURI, ok := parent["uri"].(string); ok && parentURI != "" {
			comment.ParentURI = parentURI
		}
		if parentCID, ok := parent["cid"].(string); ok && parentCID != "" {
			comment.ParentCID = parentCID
		}
	}

	return comment, nil
}
