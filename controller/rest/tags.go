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

package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
)

type (
	TagsController struct {
		Controller

		tags services.Tags
	}
)

func NewTagsController(service services.Tags, e *echo.Echo) *TagsController {
	v1 := e.Group("v1")

	controller := TagsController{
		Controller{
			e,
		},
		service,
	}

	v1.POST("/tags", controller.NewTag)
	v1.GET("/tags", controller.GetTags)
	v1.GET("/tags/:tagID", controller.GetTag)
	v1.DELETE("/tags/:tagID", controller.DeleteTag)
	v1.PUT("/tags/:tagID", controller.UpdateTag)
	v1.GET("/tags/:tagID/entries", controller.GetEntriesFromTag)
	v1.PUT("/tags/:tagID/entries", controller.TagEntries)

	return &controller
}

// NewTag creates a new Tag
func (s *TagsController) NewTag(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	tag := models.Tag{}
	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newTag, err := s.tags.New(userID, tag.Name)
	if err == services.ErrTagConflicts {
		return echo.NewHTTPError(http.StatusConflict, "tag with name "+tag.Name+" already exists")
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, newTag)
}

// GetTags returns a list of Tags owned by a user
func (s *TagsController) GetTags(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	params := paginationParams{}
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	tags, next := s.tags.List(userID, models.Page{
		ContinuationID: params.ContinuationID,
		Count:          params.Count,
	})

	return c.JSON(http.StatusOK, map[string]interface{}{
		"tags":           tags,
		"continuationId": next,
	})
}

// DeleteTag with id
func (s *TagsController) DeleteTag(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	err := s.tags.Delete(userID, c.Param("tagID"))
	if err == services.ErrTagNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateTag with id
func (s *TagsController) UpdateTag(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	tag := models.Tag{}

	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	tagID := c.Param("tagID")

	newTag, err := s.tags.Update(userID, tagID, tag.Name)
	switch err {
	case nil:
		return c.JSON(http.StatusOK, newTag)
	case services.ErrTagNotFound:
		return echo.NewHTTPError(http.StatusNotFound, "tag with id "+tagID+" not found")
	case services.ErrTagConflicts:
		return echo.NewHTTPError(http.StatusConflict, "tag with name "+tag.Name+" already exists")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
}

// TagEntries adds a Tag with tagID to a list of entries
func (s *TagsController) TagEntries(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	type EntryIds struct {
		Entries []string `json:"entries"`
	}

	entryIds := new(EntryIds)
	if err := c.Bind(entryIds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	err := s.tags.Apply(userID, c.Param("tagID"), entryIds.Entries)
	if err == services.ErrTagNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetTag with id
func (s *TagsController) GetTag(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	tag, found := s.tags.Tag(userID, c.Param("tagID"))
	if !found {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, tag)
}

// GetEntriesFromTag returns a list of Entries
// that are tagged by a Tag with ID
func (s *TagsController) GetEntriesFromTag(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	params := new(listEntriesParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	page := models.Page{
		ContinuationID: params.ContinuationID,
		Count:          params.Count,
		Newest:         convertOrderByParamToValue(params.OrderBy),
		Marker:         models.MarkerFromString(params.Marker),
	}

	entries, next := s.tags.Entries(userID, page)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries":        entries,
		"continuationID": next,
	})
}
