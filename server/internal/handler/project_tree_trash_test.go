package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
)

func createProjectForTreeTest(t *testing.T, title string, parentID string) ProjectResponse {
	t.Helper()
	body := map[string]any{"title": title}
	if parentID != "" {
		body["parent_project_id"] = parentID
	}
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/projects?workspace_id="+testWorkspaceID, body)
	testHandler.CreateProject(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProject(%s): expected 201, got %d: %s", title, w.Code, w.Body.String())
	}
	var project ProjectResponse
	if err := json.NewDecoder(w.Body).Decode(&project); err != nil {
		t.Fatalf("decode project %s: %v", title, err)
	}
	return project
}

func TestProjectTreeTrashRestoreRestoresWholeBatch(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()
	root := createProjectForTreeTest(t, "trash restore root", "")
	child := createProjectForTreeTest(t, "trash restore child", root.ID)
	grandchild := createProjectForTreeTest(t, "trash restore grandchild", child.ID)
	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM project WHERE id = ANY($1::uuid[])`, []pgtype.UUID{util.MustParseUUID(grandchild.ID), util.MustParseUUID(child.ID), util.MustParseUUID(root.ID)})
	})

	w := httptest.NewRecorder()
	req := newRequest("DELETE", "/api/projects/"+root.ID, nil)
	req = withURLParam(req, "id", root.ID)
	testHandler.DeleteProject(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("DeleteProject without confirm: expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var conflict struct {
		Code       string `json:"code"`
		ChildCount int64  `json:"child_count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&conflict); err != nil {
		t.Fatalf("decode conflict: %v", err)
	}
	if conflict.Code != "project_not_empty" || conflict.ChildCount != 1 {
		t.Fatalf("unexpected conflict body: %+v", conflict)
	}

	w = httptest.NewRecorder()
	req = newRequest("DELETE", "/api/projects/"+root.ID+"?confirm=true", nil)
	req = withURLParam(req, "id", root.ID)
	testHandler.DeleteProject(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteProject confirm: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	var distinctBatchCount int
	if err := testPool.QueryRow(ctx, `
		SELECT count(DISTINCT deleted_batch_id)
		FROM project
		WHERE id = ANY($1::uuid[]) AND deleted_at IS NOT NULL
	`, []pgtype.UUID{util.MustParseUUID(root.ID), util.MustParseUUID(child.ID), util.MustParseUUID(grandchild.ID)}).Scan(&distinctBatchCount); err != nil {
		t.Fatalf("count deleted batches: %v", err)
	}
	if distinctBatchCount != 1 {
		t.Fatalf("expected one shared deleted_batch_id for tree, got %d", distinctBatchCount)
	}

	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/projects/"+root.ID+"/restore", nil)
	req = withURLParam(req, "id", root.ID)
	testHandler.RestoreProject(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("RestoreProject: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var restoreResp struct {
		Projects []ProjectResponse `json:"projects"`
		Total    int               `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&restoreResp); err != nil {
		t.Fatalf("decode restore: %v", err)
	}
	if restoreResp.Total != 3 || len(restoreResp.Projects) != 3 {
		t.Fatalf("expected 3 restored projects, got total=%d projects=%d", restoreResp.Total, len(restoreResp.Projects))
	}

	var activeCount int
	if err := testPool.QueryRow(ctx, `
		SELECT count(*)
		FROM project
		WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL
	`, []pgtype.UUID{util.MustParseUUID(root.ID), util.MustParseUUID(child.ID), util.MustParseUUID(grandchild.ID)}).Scan(&activeCount); err != nil {
		t.Fatalf("count active projects: %v", err)
	}
	if activeCount != 3 {
		t.Fatalf("expected all projects active after restore, got %d", activeCount)
	}

	var restoredParentID string
	if err := testPool.QueryRow(ctx, `SELECT parent_project_id::text FROM project WHERE id = $1`, grandchild.ID).Scan(&restoredParentID); err != nil {
		t.Fatalf("load grandchild parent: %v", err)
	}
	if restoredParentID != child.ID {
		t.Fatalf("expected grandchild parent %s after restore, got %s", child.ID, restoredParentID)
	}
}

func TestUpdateProjectRejectsCycle(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()
	root := createProjectForTreeTest(t, "cycle root", "")
	child := createProjectForTreeTest(t, "cycle child", root.ID)
	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM project WHERE id = ANY($1::uuid[])`, []pgtype.UUID{util.MustParseUUID(child.ID), util.MustParseUUID(root.ID)})
	})

	w := httptest.NewRecorder()
	req := newRequest("PUT", "/api/projects/"+root.ID, map[string]any{"parent_project_id": child.ID})
	req = withURLParam(req, "id", root.ID)
	testHandler.UpdateProject(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateProject cycle: expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, "cycle") {
		t.Fatalf("expected cycle error, got %s", body)
	}
}
