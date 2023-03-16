package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/boojack/shortify/api"
	"github.com/boojack/shortify/common"

	"github.com/labstack/echo/v4"
)

func (s *Server) registerWorkspaceUserRoutes(g *echo.Group) {
	g.POST("/workspace/:id/user", func(c echo.Context) error {
		ctx := c.Request().Context()
		userID, ok := c.Get(getUserIDContextKey()).(int)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing auth session")
		}

		workspaceID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted workspace id").SetInternal(err)
		}

		currentWorkspaceUser, err := s.Store.FindWordspaceUser(ctx, &api.WorkspaceUserFind{
			WorkspaceID: &workspaceID,
			UserID:      &userID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find workspace user").SetInternal(err)
		}
		if currentWorkspaceUser.Role != api.RoleAdmin {
			return echo.NewHTTPError(http.StatusForbidden, "Access forbidden to add workspace user").SetInternal(err)
		}

		workspaceUserUpsert := &api.WorkspaceUserUpsert{
			WorkspaceID: workspaceID,
		}
		if err := json.NewDecoder(c.Request().Body).Decode(workspaceUserUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted post workspace user request").SetInternal(err)
		}

		workspaceUser, err := s.Store.UpsertWorkspaceUser(ctx, workspaceUserUpsert)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to upsert workspace user").SetInternal(err)
		}
		if err := s.Store.ComposeWorkspaceUser(ctx, workspaceUser); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose workspace user").SetInternal(err)
		}

		return c.JSON(http.StatusOK, composeResponse(workspaceUser))
	})

	g.GET("/workspace/:id/user", func(c echo.Context) error {
		ctx := c.Request().Context()
		workspaceID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted workspace id").SetInternal(err)
		}

		workspaceUserList, err := s.Store.FindWordspaceUserList(ctx, &api.WorkspaceUserFind{
			WorkspaceID: &workspaceID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find workspace user list").SetInternal(err)
		}

		for _, workspaceUser := range workspaceUserList {
			if err := s.Store.ComposeWorkspaceUser(ctx, workspaceUser); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose workspace user").SetInternal(err)
			}
		}

		return c.JSON(http.StatusOK, composeResponse(workspaceUserList))
	})

	g.GET("/workspace/:workspaceId/user/:userId", func(c echo.Context) error {
		ctx := c.Request().Context()
		workspaceID, err := strconv.Atoi(c.Param("workspaceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted workspace id").SetInternal(err)
		}
		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted user id").SetInternal(err)
		}

		workspaceUser, err := s.Store.FindWordspaceUser(ctx, &api.WorkspaceUserFind{
			WorkspaceID: &workspaceID,
			UserID:      &userID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find workspace user").SetInternal(err)
		}

		if err := s.Store.ComposeWorkspaceUser(ctx, workspaceUser); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose workspace user").SetInternal(err)
		}

		return c.JSON(http.StatusOK, composeResponse(workspaceUser))
	})

	g.DELETE("/workspace/:workspaceId/user/:userId", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentUserID, ok := c.Get(getUserIDContextKey()).(int)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing auth session")
		}

		workspaceID, err := strconv.Atoi(c.Param("workspaceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted workspace id").SetInternal(err)
		}

		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted user id").SetInternal(err)
		}

		currentWorkspaceUser, err := s.Store.FindWordspaceUser(ctx, &api.WorkspaceUserFind{
			WorkspaceID: &workspaceID,
			UserID:      &currentUserID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find workspace user").SetInternal(err)
		}
		if currentUserID != userID && currentWorkspaceUser.Role != api.RoleAdmin {
			return echo.NewHTTPError(http.StatusForbidden, "Access forbidden to delete workspace user").SetInternal(err)
		}

		workspaceUserDelete := &api.WorkspaceUserDelete{
			WorkspaceID: workspaceID,
			UserID:      userID,
		}
		if err := s.Store.DeleteWorkspaceUser(ctx, workspaceUserDelete); err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Workspace user not found with workspace id %d and user id %d", workspaceID, userID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete workspace user").SetInternal(err)
		}

		shortcutDelete := &api.ShortcutDelete{
			CreatorID:   &userID,
			WorkspaceID: &workspaceID,
		}
		if err := s.Store.DeleteShortcut(ctx, shortcutDelete); err != nil {
			if common.ErrorCode(err) != common.NotFound {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete shortcut").SetInternal(err)
			}
		}

		return c.JSON(http.StatusOK, true)
	})
}
