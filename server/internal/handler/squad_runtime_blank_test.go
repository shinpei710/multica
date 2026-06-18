package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createRuntimeBlankSquadTestAgent(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	var agentID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id,
			instructions, custom_env, custom_args, kind
		)
		VALUES ($1, 'squad runtime blank leader', '', 'cloud', '{}'::jsonb,
		        $2, 'workspace', 1, $3, '', '{}'::jsonb, '[]'::jsonb, 'runtime_blank')
		RETURNING id
	`, testWorkspaceID, handlerTestRuntimeID(t), testUserID).Scan(&agentID); err != nil {
		t.Fatalf("create runtime blank agent: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentID)
	})
	return agentID
}

func TestCreateSquadRejectsRuntimeBlankLeader(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	blankID := createRuntimeBlankSquadTestAgent(t)

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPost, "/api/squads?workspace_id="+testWorkspaceID, map[string]any{
		"name":      "Runtime Blank Leader Squad",
		"leader_id": blankID,
	})
	testHandler.CreateSquad(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateSquad: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSquadRejectsRuntimeBlankLeader(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	configuredID := createHandlerTestAgent(t, "squad configured leader", nil)
	blankID := createRuntimeBlankSquadTestAgent(t)

	var squadID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO squad (workspace_id, name, leader_id, creator_id)
		VALUES ($1, 'Runtime Blank Update Squad', $2, $3)
		RETURNING id
	`, testWorkspaceID, configuredID, testUserID).Scan(&squadID); err != nil {
		t.Fatalf("create squad: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM squad WHERE id = $1`, squadID)
	})

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPut, "/api/squads/"+squadID+"?workspace_id="+testWorkspaceID, map[string]any{
		"leader_id": blankID,
	})
	req = withURLParam(req, "id", squadID)
	testHandler.UpdateSquad(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateSquad: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddSquadMemberRejectsRuntimeBlankAgent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	configuredID := createHandlerTestAgent(t, "squad configured member guard leader", nil)
	blankID := createRuntimeBlankSquadTestAgent(t)

	var squadID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO squad (workspace_id, name, leader_id, creator_id)
		VALUES ($1, 'Runtime Blank Member Guard Squad', $2, $3)
		RETURNING id
	`, testWorkspaceID, configuredID, testUserID).Scan(&squadID); err != nil {
		t.Fatalf("create squad: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM squad WHERE id = $1`, squadID)
	})

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPost, "/api/squads/"+squadID+"/members?workspace_id="+testWorkspaceID, map[string]any{
		"member_type": "agent",
		"member_id":   blankID,
	})
	req = withURLParam(req, "id", squadID)
	testHandler.AddSquadMember(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("AddSquadMember: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
