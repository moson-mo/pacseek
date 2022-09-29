package pacseek

import (
	"sort"
	"strings"

	"github.com/mmcdole/gofeed"
)

// get news from rss feed(s)
func getNews(urls string, limit int) ([]*gofeed.Item, error) {
	p := gofeed.NewParser()
	items := []*gofeed.Item{}
	var retErr error

	for _, url := range strings.Split(urls, ";") {
		feed, err := p.ParseURL(url)
		if err != nil {
			retErr = err
			continue
		}
		items = append(items, feed.Items...)
	}

	if len(items) < limit {
		limit = len(items)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedParsed.Unix() > items[j].PublishedParsed.Unix()
	})

	return items[:limit], retErr
}
