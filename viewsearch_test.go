package viewsearch

import "testing"

func TestIncrementDecrementSearchIndex(t *testing.T) {
	model := New(0, 0)
	model.searchResults = []searchResult{
		{
			Line:  1,
			Index: 2,
		},
		{
			Line:  1,
			Index: 6,
		},
		{
			Line:  2,
			Index: 0,
		},
		{
			Line:  3,
			Index: 1,
		},
	}
	for i := 0; i < len(model.searchResults); i++ {
		model.currentResultIndex = i
		model.incrementSearchIndex()
		model.decrementSearchIndex()
		if model.currentResultIndex != i {
			t.Errorf("expected currentResultIndex to be %d but was %d", i, model.currentResultIndex)
		}
		_ = model.searchResults[model.currentResultIndex]
	}
}

func TestDecrementIncrementSearchIndex(t *testing.T) {
	model := New(0, 0)
	model.searchResults = []searchResult{
		{
			Line:  1,
			Index: 2,
		},
		{
			Line:  1,
			Index: 6,
		},
		{
			Line:  2,
			Index: 0,
		},
		{
			Line:  3,
			Index: 1,
		},
	}
	for i := 0; i < len(model.searchResults); i++ {
		model.currentResultIndex = i
		model.decrementSearchIndex()
		model.incrementSearchIndex()
		if model.currentResultIndex != i {
			t.Errorf("expected currentResultIndex to be %d but was %d", i, model.currentResultIndex)
		}
		_ = model.searchResults[model.currentResultIndex]
	}
}

func TestFindAndHighlightMatches(t *testing.T) {
	model := New(0, 0)
	model.SetContent(`
# Today’s Menu

## Appetizers

| Name        | Price | Notes                           |
| ---         | ---   | ---                             |
| Tsukemono   | $2    | Just an appetizer               |
| Tomato Soup | $4    | Made with San Marzano tomatoes  |
| Okonomiyaki | $4    | Takes a few minutes to make     |
| Curry       | $3    | We can add squash if you’d like |

## Seasonal Dishes

| Name                 | Price | Notes              |
| ---                  | ---   | ---                |
| Steamed bitter melon | $2    | Not so bitter      |
| Takoyaki             | $3    | Fun to eat         |
| Winter squash        | $3    | Today it's pumpkin |

## Desserts

| Name         | Price | Notes                 |
| ---          | ---   | ---                   |
| Dorayaki     | $4    | Looks good on rabbits |
| Banana Split | $5    | A classic             |
| Cream Puff   | $3    | Pretty creamy!        |

All our dishes are made in-house by Karen, our chef. Most of our ingredients
are from our garden or the fish market down the street.

Some famous people that have eaten here lately:

* [x] René Redzepi
* [x] David Chang
* [ ] Jiro Ono (maybe some day)

Bon appétit!
`)
	model.findAndHighlightMatches("price")
	expectedResultsLen := 3
	if len(model.searchResults) != 3 {
		t.Errorf("expected %d length but got %d", expectedResultsLen, len(model.searchResults))
	}

	expectedSearchResults := []searchResult{
		{
			Line:  5,
			Index: 16,
		},
		{
			Line:  14,
			Index: 25,
		},
		{
			Line:  22,
			Index: 17,
		},
	}

	for i := 0; i < len(model.searchResults); i++ {
		searchres := model.searchResults[i]
		expectedsearchres := expectedSearchResults[i]
		if searchres.Index != expectedsearchres.Index {
			t.Errorf("expected index %d search result to be at pos %d but was at %d", i, expectedsearchres.Index, searchres.Index)
		}
		if searchres.Line != expectedsearchres.Line {
			t.Errorf("expected index %d search result to be on line %d but was on %d", i, expectedsearchres.Line, searchres.Line)
		}
	}
}
