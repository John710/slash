package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/boojack/shortify/api"
	"github.com/boojack/shortify/common"
)

// workspaceRaw is the store model for Workspace.
type workspaceRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdatedTs int64
	RowStatus api.RowStatus

	// Domain specific fields
	Name        string
	Title       string
	Description string
}

func (raw *workspaceRaw) toWorkspace() *api.Workspace {
	return &api.Workspace{
		ID: raw.ID,

		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdatedTs: raw.UpdatedTs,
		RowStatus: raw.RowStatus,

		Name:              raw.Name,
		Title:             raw.Title,
		Description:       raw.Description,
		WorkspaceUserList: []*api.WorkspaceUser{},
	}
}

func (s *Store) CreateWorkspace(ctx context.Context, create *api.WorkspaceCreate) (*api.Workspace, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	workspaceRaw, err := createWorkspace(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.workspaceCache.Store(workspaceRaw.ID, workspaceRaw)
	workspace := workspaceRaw.toWorkspace()
	return workspace, nil
}

func (s *Store) PatchWorkspace(ctx context.Context, patch *api.WorkspacePatch) (*api.Workspace, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	workspaceRaw, err := patchWorkspace(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.workspaceCache.Store(workspaceRaw.ID, workspaceRaw)
	workspace := workspaceRaw.toWorkspace()
	return workspace, nil
}

func (s *Store) FindWordspaceList(ctx context.Context, find *api.WorkspaceFind) ([]*api.Workspace, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	workspaceRawList, err := findWorkspaceList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	list := []*api.Workspace{}
	for _, workspaceRaw := range workspaceRawList {
		s.workspaceCache.Store(workspaceRaw.ID, workspaceRaw)
		list = append(list, workspaceRaw.toWorkspace())
	}

	return list, nil
}

func (s *Store) FindWorkspace(ctx context.Context, find *api.WorkspaceFind) (*api.Workspace, error) {
	if find.ID != nil {
		if cache, ok := s.workspaceCache.Load(*find.ID); ok {
			return cache.(*workspaceRaw).toWorkspace(), nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findWorkspaceList(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("not found workspace with filter %+v", find)}
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d workspaces with filter %+v, expect 1", len(list), find)}
	}

	workspaceRaw := list[0]
	s.workspaceCache.Store(workspaceRaw.ID, workspaceRaw)
	workspace := workspaceRaw.toWorkspace()
	return workspace, nil
}

func (s *Store) DeleteWorkspace(ctx context.Context, delete *api.WorkspaceDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = deleteWorkspace(ctx, tx, delete)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.workspaceCache.Delete(delete.ID)
	return nil
}

func createWorkspace(ctx context.Context, tx *sql.Tx, create *api.WorkspaceCreate) (*workspaceRaw, error) {
	query := `
		INSERT INTO workspace (
			creator_id,
			name,
			title,
			description
		)
		VALUES (?, ?, ?)
		RETURNING id, creator_id, created_ts, updated_ts, row_status, name, title, description
	`
	var workspaceRaw workspaceRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.Name,
		create.Title,
		create.Description,
	).Scan(
		&workspaceRaw.ID,
		&workspaceRaw.CreatorID,
		&workspaceRaw.CreatedTs,
		&workspaceRaw.UpdatedTs,
		&workspaceRaw.RowStatus,
		&workspaceRaw.Name,
		&workspaceRaw.Title,
		&workspaceRaw.Description,
	); err != nil {
		return nil, err
	}

	return &workspaceRaw, nil
}

func patchWorkspace(ctx context.Context, tx *sql.Tx, patch *api.WorkspacePatch) (*workspaceRaw, error) {
	set, args := []string{}, []interface{}{}

	if v := patch.RowStatus; v != nil {
		set, args = append(set, "row_status = ?"), append(args, *v)
	}
	if v := patch.Name; v != nil {
		set, args = append(set, "name = ?"), append(args, *v)
	}
	if v := patch.Title; v != nil {
		set, args = append(set, "title = ?"), append(args, *v)
	}
	if v := patch.Description; v != nil {
		set, args = append(set, "description = ?"), append(args, *v)
	}

	args = append(args, patch.ID)

	query := `
		UPDATE workspace
		SET ` + strings.Join(set, ", ") + `
		WHERE id = ?
		RETURNING id, creator_id, created_ts, updated_ts, row_status, name, title, description
	`
	row, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if row.Next() {
		var workspaceRaw workspaceRaw
		if err := row.Scan(
			&workspaceRaw.ID,
			&workspaceRaw.CreatorID,
			&workspaceRaw.CreatedTs,
			&workspaceRaw.UpdatedTs,
			&workspaceRaw.RowStatus,
			&workspaceRaw.Name,
			&workspaceRaw.Title,
			&workspaceRaw.Description,
		); err != nil {
			return nil, err
		}

		if err := row.Err(); err != nil {
			return nil, err
		}

		return &workspaceRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("workspace ID not found: %d", patch.ID)}
}

func findWorkspaceList(ctx context.Context, tx *sql.Tx, find *api.WorkspaceFind) ([]*workspaceRaw, error) {
	where, args := []string{"1 = 1"}, []interface{}{}

	if v := find.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, "row_status = ?"), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
	}
	if v := find.MemberID; v != nil {
		where, args = append(where, "id IN (SELECT workspace_id FROM workspace_user WHERE user_id = ?)"), append(args, *v)
	}

	query := `
		SELECT 
			id,
			creator_id,
			created_ts,
			updated_ts,
			row_status,
			name,
			title,
			description
		FROM workspace
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_ts DESC, row_status DESC
	`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workspaceRawList := make([]*workspaceRaw, 0)
	for rows.Next() {
		var workspaceRaw workspaceRaw
		if err := rows.Scan(
			&workspaceRaw.ID,
			&workspaceRaw.CreatorID,
			&workspaceRaw.CreatedTs,
			&workspaceRaw.UpdatedTs,
			&workspaceRaw.RowStatus,
			&workspaceRaw.Name,
			&workspaceRaw.Title,
			&workspaceRaw.Description,
		); err != nil {
			return nil, err
		}

		workspaceRawList = append(workspaceRawList, &workspaceRaw)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return workspaceRawList, nil
}

func deleteWorkspace(ctx context.Context, tx *sql.Tx, delete *api.WorkspaceDelete) error {
	result, err := tx.ExecContext(ctx, `
		DELETE FROM workspace WHERE id = ?
	`, delete.ID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return &common.Error{Code: common.NotFound, Err: fmt.Errorf("workspace ID not found: %d", delete.ID)}
	}

	return nil
}
