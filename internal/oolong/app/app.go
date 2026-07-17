// Package oolongapp constructs the Oolong product configuration.
package oolongapp

import (
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"
)

func New() *domain.App {
	return &domain.App{
		Name:        "oolong",
		NSIDBase:    oolong.NSIDBase,
		Descriptors: entities.AllForApp(oolong.NSIDBase),
		EntityRoutes: []domain.EntityRoute{
			{Type: lexicons.RecordTypeOolongTea, Path: "teas", Noun: "tea"},
			{Type: lexicons.RecordTypeOolongVendor, Path: "vendors", Noun: "vendor"},
			{Type: lexicons.RecordTypeOolongVessel, Path: "vessels", Noun: "vessel"},
			{Type: lexicons.RecordTypeOolongInfuser, Path: "infusers", Noun: "infuser"},
			{Type: lexicons.RecordTypeOolongBrew, Path: "brews", Noun: "brew"},
		},
		Brand: domain.BrandConfig{
			DisplayName:     "Oolong",
			Tagline:         "Your tea, your data",
			SiteDescription: "Oolong is a tea tracking app built on AT Protocol. Your steep logs, teas, and teaware are stored in your own Personal Data Server, giving you full ownership and portability.",
			LightThemeColor: "#b8d5aa",
			DarkThemeColor:  "#162018",
		},
	}
}
