package coffee

import (
	"tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/feed"
)

func brew(item *feed.FeedItem) *arabica.Brew {
	if item == nil {
		return nil
	}
	v, _ := item.Record.(*arabica.Brew)
	return v
}

func bean(item *feed.FeedItem) *arabica.Bean {
	if item == nil {
		return nil
	}
	v, _ := item.Record.(*arabica.Bean)
	return v
}

func roaster(item *feed.FeedItem) *arabica.Roaster {
	if item == nil {
		return nil
	}
	v, _ := item.Record.(*arabica.Roaster)
	return v
}

func grinder(item *feed.FeedItem) *arabica.Grinder {
	if item == nil {
		return nil
	}
	v, _ := item.Record.(*arabica.Grinder)
	return v
}

func brewer(item *feed.FeedItem) *arabica.Brewer {
	if item == nil {
		return nil
	}
	v, _ := item.Record.(*arabica.Brewer)
	return v
}

func recipe(item *feed.FeedItem) *arabica.Recipe {
	if item == nil {
		return nil
	}
	v, _ := item.Record.(*arabica.Recipe)
	return v
}
