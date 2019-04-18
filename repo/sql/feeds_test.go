/*
  Copyright (C) 2017 Jorge Martinez Hernandez

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package sql

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type FeedsSuite struct {
	suite.Suite

	repo repo.Feeds
	db   *DB
	user *models.User
	ctg  models.Category
}

func (s *FeedsSuite) TestCreate() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     s.ctg,
	}
	s.repo.Create(s.user, &feed)

	query, found := s.repo.FeedWithID(s.user, feed.APIID)
	s.True(found)

	s.Equal(s.ctg.APIID, query.Category.APIID)
	s.Equal(s.ctg.Name, query.Category.Name)
}

func (s *FeedsSuite) TestList() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			APIID: utils.CreateAPIID(),
			Title: "Test site " + strconv.Itoa(i),
		}
		s.repo.Create(s.user, &feed)
	}

	feeds, next := s.repo.List(s.user, "", 2)
	s.Require().Len(feeds, 2)
	s.NotEmpty(next)
	s.Equal("Test site 0", feeds[0].Title)
	s.Equal("Test site 1", feeds[1].Title)

	feeds, _ = s.repo.List(s.user, next, 3)
	s.Require().Len(feeds, 3)
	s.Equal(feeds[0].APIID, next)
	s.Equal("Test site 2", feeds[0].Title)
	s.Equal("Test site 3", feeds[1].Title)
	s.Equal("Test site 4", feeds[2].Title)

}

func (s *FeedsSuite) TestUpdate() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	s.repo.Create(s.user, &feed)

	feed.Title = "New Name"
	feed.Subscription = "http://example.com/feed"
	err := s.repo.Update(s.user, &feed)
	s.NoError(err)

	updatedFeed, _ := s.repo.FeedWithID(s.user, feed.APIID)
	s.Equal(feed.APIID, updatedFeed.APIID)
	s.Equal("New Name", updatedFeed.Title)
	s.Equal("http://example.com/feed", updatedFeed.Subscription)
}

func (s *FeedsSuite) TestUpdateMissing() {
	err := s.repo.Update(s.user, &models.Feed{})
	s.EqualError(err, repo.ErrModelNotFound.Error())
}

func (s *FeedsSuite) TestDelete() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	s.repo.Create(s.user, &feed)

	err := s.repo.Delete(s.user, feed.APIID)
	s.NoError(err)

	_, found := s.repo.FeedWithID(s.user, feed.APIID)
	s.False(found)
}

func (s *FeedsSuite) TestDeleteMissing() {
	err := s.repo.Delete(s.user, "bogus")
	s.EqualError(err, repo.ErrModelNotFound.Error())
}

func (s *FeedsSuite) TestMark() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	s.repo.Create(s.user, &feed)

	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		s.db.db.Model(s.user).Association("Entries").Append(&entry)
		s.db.db.Model(&feed).Association("Entries").Append(&entry)
	}

	err := s.repo.Mark(s.user, feed.APIID, models.MarkerRead)
	s.NoError(err)

	entries, _ := NewEntries(s.db).ListFromFeed(s.user, feed.APIID, "", 5, false, models.MarkerRead)
	s.Len(entries, 5)
}

func (s *FeedsSuite) TestStats() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	s.repo.Create(s.user, &feed)

	for i := 0; i < 10; i++ {
		var marker models.Marker
		if i < 3 {
			marker = models.MarkerRead
		} else {
			marker = models.MarkerUnread
		}
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      marker,
			Saved:     i < 2,
			Published: time.Now(),
		}

		s.db.db.Model(&feed).Association("Entries").Append(&entry)
	}

	stats, err := s.repo.Stats(s.user, feed.APIID)
	s.NoError(err)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(2, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *FeedsSuite) SetupTest() {
	s.db = NewDB("sqlite3", ":memory:")

	s.user = &models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test_feeds",
	}
	s.db.db.Create(s.user)

	s.ctg = models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "category",
	}
	s.db.db.Create(&s.ctg)

	s.repo = NewFeeds(s.db)
}

func (s *FeedsSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestFeedsSuite(t *testing.T) {
	suite.Run(t, new(FeedsSuite))
}
