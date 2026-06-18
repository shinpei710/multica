package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Work with projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects in the workspace",
	RunE:  runProjectList,
}

var projectGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get project details",
	Args:  exactArgs(1),
	RunE:  runProjectGet,
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	RunE:  runProjectCreate,
}

var projectUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a project",
	Args:  exactArgs(1),
	RunE:  runProjectUpdate,
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project",
	Args:  exactArgs(1),
	RunE:  runProjectDelete,
}

var projectTrashCmd = &cobra.Command{
	Use:   "trash",
	Short: "List deleted projects that can be restored",
	RunE:  runProjectTrash,
}

var projectRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a deleted project tree",
	Args:  exactArgs(1),
	RunE:  runProjectRestore,
}

var projectStatusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Change project status",
	Args:  exactArgs(2),
	RunE:  runProjectStatus,
}

var projectResourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "Manage resources attached to a project",
}

var projectResourceListCmd = &cobra.Command{
	Use:   "list <project-id>",
	Short: "List resources attached to a project",
	Args:  exactArgs(1),
	RunE:  runProjectResourceList,
}

var projectResourceAddCmd = &cobra.Command{
	Use:   "add <project-id>",
	Short: "Attach a resource to a project (e.g. --type github_repo --url <url>)",
	Args:  exactArgs(1),
	RunE:  runProjectResourceAdd,
}

var projectResourceUpdateCmd = &cobra.Command{
	Use:   "update <project-id> <resource-id>",
	Short: "Edit an attached resource (ref payload, label, or position)",
	Args:  exactArgs(2),
	RunE:  runProjectResourceUpdate,
}

var projectResourceRemoveCmd = &cobra.Command{
	Use:   "remove <project-id> <resource-id>",
	Short: "Detach a resource from a project",
	Args:  exactArgs(2),
	RunE:  runProjectResourceRemove,
}

var validProjectStatuses = []string{
	"planned", "in_progress", "paused", "completed", "cancelled",
}

// validateProjectStatus rejects unknown statuses client-side so a typo fails
// fast with the valid list instead of a server round-trip and a 400. Shared by
// `project create`, `project update`, and `project status`.
func validateProjectStatus(status string) error {
	for _, s := range validProjectStatuses {
		if s == status {
			return nil
		}
	}
	return fmt.Errorf("invalid status %q; valid values: %s", status, strings.Join(validProjectStatuses, ", "))
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectTrashCmd)
	projectCmd.AddCommand(projectRestoreCmd)
	projectCmd.AddCommand(projectStatusCmd)
	projectCmd.AddCommand(projectResourceCmd)

	projectResourceCmd.AddCommand(projectResourceListCmd)
	projectResourceCmd.AddCommand(projectResourceAddCmd)
	projectResourceCmd.AddCommand(projectResourceUpdateCmd)
	projectResourceCmd.AddCommand(projectResourceRemoveCmd)

	// project list
	projectListCmd.Flags().String("output", "table", "Output format: table or json")
	projectListCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")
	projectListCmd.Flags().String("status", "", "Filter by status")

	// project get
	projectGetCmd.Flags().String("output", "json", "Output format: table or json")

	// project create
	projectCreateCmd.Flags().String("title", "", "Project title (required)")
	projectCreateCmd.Flags().String("description", "", "Project description")
	projectCreateCmd.Flags().String("status", "", "Project status")
	projectCreateCmd.Flags().String("icon", "", "Project icon (emoji)")
	projectCreateCmd.Flags().String("lead", "", "Lead name (member or agent)")
	projectCreateCmd.Flags().String("parent", "", "Parent project id or prefix")
	projectCreateCmd.Flags().Float64("position", 0, "Display position within the parent")
	projectCreateCmd.Flags().StringArray("repo", nil, "Attach a github_repo resource by URL (may be repeated)")
	projectCreateCmd.Flags().String("output", "json", "Output format: table or json")

	// project resource list
	projectResourceListCmd.Flags().String("output", "table", "Output format: table or json")
	projectResourceListCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")

	// project resource add — generic shape: any --type with a JSON --ref payload
	// works without further CLI changes. github_repo is supported via the
	// dedicated --url / --default-branch-hint shortcuts as a convenience.
	projectResourceAddCmd.Flags().String("type", "github_repo", "Resource type (e.g. github_repo, local_directory — see docs)")
	projectResourceAddCmd.Flags().String("url", "", "Shortcut: the repo URL (only used when --type github_repo)")
	projectResourceAddCmd.Flags().String("default-branch-hint", "", "Shortcut: optional default branch hint (only used when --type github_repo)")
	projectResourceAddCmd.Flags().String("local-path", "", "Shortcut: absolute path to the working directory (only used when --type local_directory)")
	projectResourceAddCmd.Flags().String("daemon-id", "", "Shortcut: id of the daemon that owns the local path (only used when --type local_directory)")
	projectResourceAddCmd.Flags().String("ref-label", "", "Shortcut: optional label embedded in resource_ref (only used when --type local_directory)")
	projectResourceAddCmd.Flags().String("ref", "", "Generic JSON resource_ref payload — overrides the per-type shortcuts when set")
	projectResourceAddCmd.Flags().String("label", "", "Optional human-readable label")
	projectResourceAddCmd.Flags().String("output", "json", "Output format: table or json")

	// project resource update — mirrors `add` flags, but every field is
	// optional so the caller can edit one thing at a time.
	projectResourceUpdateCmd.Flags().String("url", "", "Shortcut: new repo URL (github_repo)")
	projectResourceUpdateCmd.Flags().String("default-branch-hint", "", "Shortcut: new default branch hint (github_repo)")
	projectResourceUpdateCmd.Flags().String("local-path", "", "Shortcut: new absolute local path (local_directory)")
	projectResourceUpdateCmd.Flags().String("daemon-id", "", "Shortcut: new daemon id (local_directory)")
	projectResourceUpdateCmd.Flags().String("ref-label", "", "Shortcut: new label embedded in resource_ref (local_directory)")
	projectResourceUpdateCmd.Flags().String("ref", "", "Generic JSON resource_ref payload — overrides per-type shortcuts when set")
	projectResourceUpdateCmd.Flags().String("label", "", "New human-readable label; pass an empty string to clear")
	projectResourceUpdateCmd.Flags().Bool("clear-label", false, "Clear the human-readable label")
	projectResourceUpdateCmd.Flags().Int32("position", 0, "New display position")
	projectResourceUpdateCmd.Flags().String("output", "json", "Output format: table or json")

	// project resource remove
	projectResourceRemoveCmd.Flags().String("output", "table", "Output format: table or json")

	// project update
	projectUpdateCmd.Flags().String("title", "", "New title")
	projectUpdateCmd.Flags().String("description", "", "New description")
	projectUpdateCmd.Flags().String("status", "", "New status")
	projectUpdateCmd.Flags().String("icon", "", "New icon (emoji)")
	projectUpdateCmd.Flags().String("lead", "", "New lead name (member or agent)")
	projectUpdateCmd.Flags().String("parent", "", "New parent project id or prefix")
	projectUpdateCmd.Flags().Bool("clear-parent", false, "Move the project to the top level")
	projectUpdateCmd.Flags().Float64("position", 0, "New display position within the parent")
	projectUpdateCmd.Flags().String("output", "json", "Output format: table or json")

	// project delete
	projectDeleteCmd.Flags().String("output", "json", "Output format: table or json")
	projectDeleteCmd.Flags().Bool("confirm", false, "Confirm deleting a non-empty project tree")

	// project trash / restore
	projectTrashCmd.Flags().String("output", "table", "Output format: table or json")
	projectTrashCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")
	projectRestoreCmd.Flags().String("output", "json", "Output format: table or json")

	// project status
	projectStatusCmd.Flags().String("output", "table", "Output format: table or json")
}

// ---------------------------------------------------------------------------
// Project commands
// ---------------------------------------------------------------------------

func runProjectList(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	params := url.Values{}
	if client.WorkspaceID != "" {
		params.Set("workspace_id", client.WorkspaceID)
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		params.Set("status", v)
	}

	path := "/api/projects"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result map[string]any
	if err := client.GetJSON(ctx, path, &result); err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	projectsRaw, _ := result["projects"].([]any)

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, projectsRaw)
	}

	fullID, _ := cmd.Flags().GetBool("full-id")
	actors := loadActorDisplayLookup(ctx, client)
	cli.PrintTable(os.Stdout, []string{"ID", "TITLE", "STATUS", "LEAD", "CREATED"}, projectTableRows(projectsRaw, fullID, actors))
	return nil
}

func runProjectGet(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}

	var project map[string]any
	if err := client.GetJSON(ctx, "/api/projects/"+projectRef.ID, &project); err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	// Breadcrumb to the resources sub-collection. Goes to stderr so JSON on
	// stdout stays parseable; the `resource_count` field on the response is
	// the programmatic equivalent. JSON numbers decode as float64.
	if n, _ := project["resource_count"].(float64); n > 0 {
		fmt.Fprintf(os.Stderr, "%d resource(s) attached — run `multica project resource list %s` to view.\n",
			int64(n), strVal(project, "id"))
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		actors := loadActorDisplayLookup(ctx, client)
		lead := formatLead(project, actors)
		headers := []string{"ID", "TITLE", "STATUS", "LEAD", "DESCRIPTION"}
		rows := [][]string{{
			strVal(project, "id"),
			strVal(project, "title"),
			strVal(project, "status"),
			lead,
			strVal(project, "description"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, project)
}

func runProjectCreate(cmd *cobra.Command, _ []string) error {
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		return fmt.Errorf("--title is required")
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	body := map[string]any{"title": title}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		body["description"] = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		if err := validateProjectStatus(v); err != nil {
			return err
		}
		body["status"] = v
	}
	if v, _ := cmd.Flags().GetString("icon"); v != "" {
		body["icon"] = v
	}
	if v, _ := cmd.Flags().GetString("lead"); v != "" {
		aType, aID, resolveErr := resolveAssignee(ctx, client, v, memberOrAgentKinds)
		if resolveErr != nil {
			return fmt.Errorf("resolve lead: %w", resolveErr)
		}
		body["lead_type"] = aType
		body["lead_id"] = aID
	}
	if v, _ := cmd.Flags().GetString("parent"); v != "" {
		parentRef, resolveErr := resolveProjectID(ctx, client, v)
		if resolveErr != nil {
			return fmt.Errorf("resolve parent project: %w", resolveErr)
		}
		body["parent_project_id"] = parentRef.ID
	}
	if cmd.Flags().Changed("position") {
		v, _ := cmd.Flags().GetFloat64("position")
		body["position"] = v
	}

	// Bundle resources into the create payload so the server attaches them in
	// the same transaction; this avoids leaving a half-attached project on
	// failure.
	repos, _ := cmd.Flags().GetStringArray("repo")
	if len(repos) > 0 {
		resources := make([]map[string]any, 0, len(repos))
		for _, repoURL := range repos {
			repoURL = strings.TrimSpace(repoURL)
			if repoURL == "" {
				continue
			}
			resources = append(resources, map[string]any{
				"resource_type": "github_repo",
				"resource_ref":  map[string]any{"url": repoURL},
			})
		}
		if len(resources) > 0 {
			body["resources"] = resources
		}
	}

	var result map[string]any
	if err := client.PostJSON(ctx, "/api/projects", body, &result); err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		headers := []string{"ID", "TITLE", "STATUS"}
		rows := [][]string{{
			strVal(result, "id"),
			strVal(result, "title"),
			strVal(result, "status"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, result)
}

func runProjectUpdate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}

	body := map[string]any{}
	if cmd.Flags().Changed("title") {
		v, _ := cmd.Flags().GetString("title")
		body["title"] = v
	}
	if cmd.Flags().Changed("description") {
		v, _ := cmd.Flags().GetString("description")
		body["description"] = v
	}
	if cmd.Flags().Changed("status") {
		v, _ := cmd.Flags().GetString("status")
		if err := validateProjectStatus(v); err != nil {
			return err
		}
		body["status"] = v
	}
	if cmd.Flags().Changed("icon") {
		v, _ := cmd.Flags().GetString("icon")
		body["icon"] = v
	}
	if cmd.Flags().Changed("lead") {
		v, _ := cmd.Flags().GetString("lead")
		aType, aID, resolveErr := resolveAssignee(ctx, client, v, memberOrAgentKinds)
		if resolveErr != nil {
			return fmt.Errorf("resolve lead: %w", resolveErr)
		}
		body["lead_type"] = aType
		body["lead_id"] = aID
	}
	if cmd.Flags().Changed("parent") && cmd.Flags().Changed("clear-parent") {
		return fmt.Errorf("--parent and --clear-parent are mutually exclusive")
	}
	if cmd.Flags().Changed("parent") {
		v, _ := cmd.Flags().GetString("parent")
		parentRef, resolveErr := resolveProjectID(ctx, client, v)
		if resolveErr != nil {
			return fmt.Errorf("resolve parent project: %w", resolveErr)
		}
		body["parent_project_id"] = parentRef.ID
	}
	if clearParent, _ := cmd.Flags().GetBool("clear-parent"); clearParent {
		body["parent_project_id"] = nil
	}
	if cmd.Flags().Changed("position") {
		v, _ := cmd.Flags().GetFloat64("position")
		body["position"] = v
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update; use flags like --title, --status, --description, --icon, --lead, --parent, --clear-parent")
	}

	var result map[string]any
	if err := client.PutJSON(ctx, "/api/projects/"+projectRef.ID, body, &result); err != nil {
		return fmt.Errorf("update project: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		headers := []string{"ID", "TITLE", "STATUS"}
		rows := [][]string{{
			strVal(result, "id"),
			strVal(result, "title"),
			strVal(result, "status"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, result)
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}

	path := "/api/projects/" + projectRef.ID
	if confirm, _ := cmd.Flags().GetBool("confirm"); confirm {
		path += "?confirm=true"
	}
	if err := client.DeleteJSON(ctx, path); err != nil {
		if payload, ok := projectNotEmptyConflict(err); ok {
			return fmt.Errorf("project is not empty (%d subprojects, %d linked issues); re-run with --confirm to move the whole project tree to trash", payload.ChildCount, payload.IssueCount)
		}
		return fmt.Errorf("delete project: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Project %s moved to trash.\n", projectRef.Display)
	return nil
}

func runProjectTrash(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	var result map[string]any
	if err := client.GetJSON(ctx, "/api/projects/trash", &result); err != nil {
		return fmt.Errorf("list project trash: %w", err)
	}
	projectsRaw, _ := result["projects"].([]any)
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, projectsRaw)
	}

	fullID, _ := cmd.Flags().GetBool("full-id")
	headers := []string{"ID", "TITLE", "DELETED", "EXPIRES"}
	rows := make([][]string, 0, len(projectsRaw))
	for _, raw := range projectsRaw {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, []string{
			displayID(strVal(p, "id"), fullID),
			strVal(p, "title"),
			shortDateString(strVal(p, "deleted_at")),
			shortDateString(strVal(p, "delete_expires_at")),
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func runProjectRestore(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveDeletedProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve deleted project: %w", err)
	}

	var result map[string]any
	if err := client.PostJSON(ctx, "/api/projects/"+projectRef.ID+"/restore", map[string]any{}, &result); err != nil {
		return fmt.Errorf("restore project: %w", err)
	}
	projectsRaw, _ := result["projects"].([]any)
	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		actors := loadActorDisplayLookup(ctx, client)
		cli.PrintTable(os.Stdout, []string{"ID", "TITLE", "STATUS", "LEAD", "CREATED"}, projectTableRows(projectsRaw, true, actors))
		return nil
	}
	return cli.PrintJSON(os.Stdout, result)
}

func runProjectStatus(cmd *cobra.Command, args []string) error {
	id := args[0]
	status := args[1]

	if err := validateProjectStatus(status); err != nil {
		return err
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, id)
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}

	body := map[string]any{"status": status}
	var result map[string]any
	if err := client.PutJSON(ctx, "/api/projects/"+projectRef.ID, body, &result); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Project %s status changed to %s.\n", strVal(result, "title"), status)

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	return nil
}

type projectNotEmptyPayload struct {
	Code       string `json:"code"`
	ChildCount int64  `json:"child_count"`
	IssueCount int64  `json:"issue_count"`
}

func projectNotEmptyConflict(err error) (projectNotEmptyPayload, bool) {
	var httpErr *cli.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusConflict {
		return projectNotEmptyPayload{}, false
	}
	var payload projectNotEmptyPayload
	if json.Unmarshal([]byte(httpErr.Body), &payload) != nil {
		return projectNotEmptyPayload{}, false
	}
	if payload.Code != "project_not_empty" {
		return projectNotEmptyPayload{}, false
	}
	return payload, true
}

// ---------------------------------------------------------------------------
// Project resource commands
// ---------------------------------------------------------------------------

func runProjectResourceList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}

	var result map[string]any
	if err := client.GetJSON(ctx, "/api/projects/"+projectRef.ID+"/resources", &result); err != nil {
		return fmt.Errorf("list project resources: %w", err)
	}
	resourcesRaw, _ := result["resources"].([]any)

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, resourcesRaw)
	}

	fullID, _ := cmd.Flags().GetBool("full-id")
	headers := []string{"ID", "TYPE", "REF", "LABEL"}
	rows := make([][]string, 0, len(resourcesRaw))
	for _, raw := range resourcesRaw {
		r, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, []string{
			displayID(strVal(r, "id"), fullID),
			strVal(r, "resource_type"),
			summarizeResourceRef(r["resource_ref"]),
			strVal(r, "label"),
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func runProjectResourceAdd(cmd *cobra.Command, args []string) error {
	resourceType, _ := cmd.Flags().GetString("type")
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return fmt.Errorf("--type is required")
	}

	body := map[string]any{"resource_type": resourceType}

	// --ref takes precedence: any new resource type works through this path
	// without a CLI change. Per-type shortcuts (--url etc.) only apply when
	// --ref is empty.
	if rawRef, _ := cmd.Flags().GetString("ref"); strings.TrimSpace(rawRef) != "" {
		var ref any
		if err := json.Unmarshal([]byte(rawRef), &ref); err != nil {
			return fmt.Errorf("--ref is not valid JSON: %w", err)
		}
		body["resource_ref"] = ref
	} else {
		switch resourceType {
		case "github_repo":
			urlVal, _ := cmd.Flags().GetString("url")
			urlVal = strings.TrimSpace(urlVal)
			if urlVal == "" {
				return fmt.Errorf("github_repo requires --url (or pass a JSON payload via --ref)")
			}
			ref := map[string]any{"url": urlVal}
			if hint, _ := cmd.Flags().GetString("default-branch-hint"); hint != "" {
				ref["default_branch_hint"] = strings.TrimSpace(hint)
			}
			body["resource_ref"] = ref
		case "local_directory":
			pathVal, _ := cmd.Flags().GetString("local-path")
			pathVal = strings.TrimSpace(pathVal)
			daemonVal, _ := cmd.Flags().GetString("daemon-id")
			daemonVal = strings.TrimSpace(daemonVal)
			if pathVal == "" || daemonVal == "" {
				return fmt.Errorf("local_directory requires --local-path and --daemon-id (or pass a JSON payload via --ref)")
			}
			ref := map[string]any{"local_path": pathVal, "daemon_id": daemonVal}
			if refLabel, _ := cmd.Flags().GetString("ref-label"); strings.TrimSpace(refLabel) != "" {
				ref["label"] = strings.TrimSpace(refLabel)
			}
			body["resource_ref"] = ref
		default:
			return fmt.Errorf("type %q has no built-in CLI shortcut; pass the payload via --ref '<json>'", resourceType)
		}
	}

	if label, _ := cmd.Flags().GetString("label"); label != "" {
		body["label"] = label
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}

	var result map[string]any
	if err := client.PostJSON(ctx, "/api/projects/"+projectRef.ID+"/resources", body, &result); err != nil {
		return fmt.Errorf("add project resource: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		headers := []string{"ID", "TYPE", "REF"}
		rows := [][]string{{
			strVal(result, "id"),
			strVal(result, "resource_type"),
			summarizeResourceRef(result["resource_ref"]),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}
	return cli.PrintJSON(os.Stdout, result)
}

func runProjectResourceUpdate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}
	resourceRef, err := resolveProjectResourceID(ctx, client, projectRef.ID, args[1])
	if err != nil {
		return fmt.Errorf("resolve project resource: %w", err)
	}

	// Fetch the existing row so per-type shortcuts know which schema to
	// emit and which fields to preserve. The server treats resource_ref as
	// opaque-replace, so a partial edit like `--default-branch-hint` has to
	// rebuild the full payload here — otherwise the unmentioned `url` would
	// vanish and the server would 400.
	var existing map[string]any
	if err := client.GetJSON(ctx, "/api/projects/"+projectRef.ID+"/resources", &existing); err != nil {
		return fmt.Errorf("list project resources: %w", err)
	}
	var resourceType string
	var existingRef map[string]any
	if list, ok := existing["resources"].([]any); ok {
		for _, raw := range list {
			row, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if strVal(row, "id") == resourceRef.ID {
				resourceType = strVal(row, "resource_type")
				if ref, ok := row["resource_ref"].(map[string]any); ok {
					existingRef = ref
				}
				break
			}
		}
	}

	body := map[string]any{}

	if rawRef, _ := cmd.Flags().GetString("ref"); strings.TrimSpace(rawRef) != "" {
		var ref any
		if err := json.Unmarshal([]byte(rawRef), &ref); err != nil {
			return fmt.Errorf("--ref is not valid JSON: %w", err)
		}
		body["resource_ref"] = ref
	} else {
		ref, has, err := buildResourceRefFromFlags(cmd, resourceType, existingRef)
		if err != nil {
			return err
		}
		if has {
			body["resource_ref"] = ref
		}
	}

	clearLabel, _ := cmd.Flags().GetBool("clear-label")
	if clearLabel {
		body["label"] = nil
	} else if cmd.Flags().Changed("label") {
		label, _ := cmd.Flags().GetString("label")
		body["label"] = label
	}

	if cmd.Flags().Changed("position") {
		pos, _ := cmd.Flags().GetInt32("position")
		body["position"] = pos
	}

	if len(body) == 0 {
		return fmt.Errorf("nothing to update — pass --ref / --url / --local-path / --label / --position / --clear-label")
	}

	var result map[string]any
	if err := client.PutJSON(ctx, "/api/projects/"+projectRef.ID+"/resources/"+resourceRef.ID, body, &result); err != nil {
		return fmt.Errorf("update project resource: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		headers := []string{"ID", "TYPE", "REF", "LABEL"}
		rows := [][]string{{
			strVal(result, "id"),
			strVal(result, "resource_type"),
			summarizeResourceRef(result["resource_ref"]),
			strVal(result, "label"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}
	return cli.PrintJSON(os.Stdout, result)
}

// buildResourceRefFromFlags collects the per-type shortcut flags into a
// resource_ref payload, seeding from existingRef so partial edits (only
// --default-branch-hint, only --ref-label) preserve the unmentioned fields.
// Returns (ref, true) only when the caller actually set at least one shortcut
// flag — that lets the update command tell "no change requested" apart from
// "change ref to empty object". existingRef may be nil for the `add` path,
// where there is nothing to merge with; in that case partial inputs that miss
// required fields are still rejected.
func buildResourceRefFromFlags(cmd *cobra.Command, resourceType string, existingRef map[string]any) (map[string]any, bool, error) {
	switch resourceType {
	case "github_repo":
		urlSet := cmd.Flags().Changed("url")
		hintSet := cmd.Flags().Changed("default-branch-hint")
		if !urlSet && !hintSet {
			return nil, false, nil
		}
		ref := map[string]any{}
		// Seed from the existing row so a `--default-branch-hint` edit doesn't
		// clobber the `url` (server overwrites resource_ref wholesale).
		if existingRef != nil {
			if u, ok := existingRef["url"].(string); ok && strings.TrimSpace(u) != "" {
				ref["url"] = strings.TrimSpace(u)
			}
			if h, ok := existingRef["default_branch_hint"].(string); ok && strings.TrimSpace(h) != "" {
				ref["default_branch_hint"] = strings.TrimSpace(h)
			}
		}
		if urlSet {
			urlVal, _ := cmd.Flags().GetString("url")
			urlVal = strings.TrimSpace(urlVal)
			if urlVal == "" {
				return nil, false, fmt.Errorf("--url cannot be empty")
			}
			ref["url"] = urlVal
		}
		if hintSet {
			hint := strings.TrimSpace(mustString(cmd, "default-branch-hint"))
			if hint == "" {
				delete(ref, "default_branch_hint")
			} else {
				ref["default_branch_hint"] = hint
			}
		}
		if _, ok := ref["url"]; !ok {
			return nil, false, fmt.Errorf("github_repo: --url is required (no existing url to merge with)")
		}
		return ref, true, nil
	case "local_directory":
		pathSet := cmd.Flags().Changed("local-path")
		daemonSet := cmd.Flags().Changed("daemon-id")
		labelSet := cmd.Flags().Changed("ref-label")
		if !pathSet && !daemonSet && !labelSet {
			return nil, false, nil
		}
		ref := map[string]any{}
		if existingRef != nil {
			if p, ok := existingRef["local_path"].(string); ok && strings.TrimSpace(p) != "" {
				ref["local_path"] = strings.TrimSpace(p)
			}
			if d, ok := existingRef["daemon_id"].(string); ok && strings.TrimSpace(d) != "" {
				ref["daemon_id"] = strings.TrimSpace(d)
			}
			if l, ok := existingRef["label"].(string); ok && strings.TrimSpace(l) != "" {
				ref["label"] = strings.TrimSpace(l)
			}
		}
		if pathSet {
			pathVal := strings.TrimSpace(mustString(cmd, "local-path"))
			if pathVal == "" {
				return nil, false, fmt.Errorf("--local-path cannot be empty")
			}
			ref["local_path"] = pathVal
		}
		if daemonSet {
			daemonVal := strings.TrimSpace(mustString(cmd, "daemon-id"))
			if daemonVal == "" {
				return nil, false, fmt.Errorf("--daemon-id cannot be empty")
			}
			ref["daemon_id"] = daemonVal
		}
		if labelSet {
			refLabel := strings.TrimSpace(mustString(cmd, "ref-label"))
			if refLabel == "" {
				delete(ref, "label")
			} else {
				ref["label"] = refLabel
			}
		}
		if v, ok := ref["local_path"].(string); !ok || v == "" {
			return nil, false, fmt.Errorf("local_directory: --local-path is required (no existing local_path to merge with)")
		}
		if v, ok := ref["daemon_id"].(string); !ok || v == "" {
			return nil, false, fmt.Errorf("local_directory: --daemon-id is required (no existing daemon_id to merge with)")
		}
		return ref, true, nil
	default:
		// Unknown type or empty (resource not found) — caller must use --ref.
		if cmd.Flags().Changed("url") || cmd.Flags().Changed("default-branch-hint") ||
			cmd.Flags().Changed("local-path") || cmd.Flags().Changed("daemon-id") ||
			cmd.Flags().Changed("ref-label") {
			return nil, false, fmt.Errorf("no built-in shortcut for resource type %q; pass the full payload via --ref '<json>'", resourceType)
		}
		return nil, false, nil
	}
}

func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func runProjectResourceRemove(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()

	projectRef, err := resolveProjectID(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve project: %w", err)
	}
	resourceRef, err := resolveProjectResourceID(ctx, client, projectRef.ID, args[1])
	if err != nil {
		return fmt.Errorf("resolve project resource: %w", err)
	}

	if err := client.DeleteJSON(ctx, "/api/projects/"+projectRef.ID+"/resources/"+resourceRef.ID); err != nil {
		return fmt.Errorf("remove project resource: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Resource %s removed from project %s.\n", resourceRef.Display, projectRef.Display)
	return nil
}

// summarizeResourceRef extracts the most useful single string from a
// resource_ref object — for github_repo this is the URL; for
// local_directory it is the local path.
func summarizeResourceRef(raw any) string {
	m, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	if u, ok := m["url"].(string); ok && u != "" {
		return u
	}
	if p, ok := m["local_path"].(string); ok && p != "" {
		return p
	}
	if data, err := json.Marshal(m); err == nil {
		return string(data)
	}
	return ""
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func formatLead(project map[string]any, actors actorDisplayLookup) string {
	lType := strVal(project, "lead_type")
	lID := strVal(project, "lead_id")
	if lType == "" || lID == "" {
		return ""
	}
	return actors.actor(lType, lID)
}

func projectTableRows(projectsRaw []any, fullID bool, actors actorDisplayLookup) [][]string {
	depths := projectDepths(projectsRaw)
	rows := make([][]string, 0, len(projectsRaw))
	for _, raw := range projectsRaw {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := strVal(p, "id")
		title := strVal(p, "title")
		if depth := depths[id]; depth > 0 {
			title = strings.Repeat("  ", depth) + "↳ " + title
		}
		rows = append(rows, []string{
			displayID(id, fullID),
			title,
			strVal(p, "status"),
			formatLead(p, actors),
			shortDateString(strVal(p, "created_at")),
		})
	}
	return rows
}

func projectDepths(projectsRaw []any) map[string]int {
	parentByID := map[string]string{}
	for _, raw := range projectsRaw {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := strVal(p, "id")
		if id == "" {
			continue
		}
		parentByID[id] = strVal(p, "parent_project_id")
	}
	depths := map[string]int{}
	var depthOf func(id string, seen map[string]bool) int
	depthOf = func(id string, seen map[string]bool) int {
		if depth, ok := depths[id]; ok {
			return depth
		}
		parentID := parentByID[id]
		if parentID == "" || seen[id] {
			depths[id] = 0
			return 0
		}
		seen[id] = true
		depths[id] = depthOf(parentID, seen) + 1
		return depths[id]
	}
	for id := range parentByID {
		depthOf(id, map[string]bool{})
	}
	return depths
}

func shortDateString(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func resolveDeletedProjectID(ctx context.Context, client *cli.APIClient, input string) (resolvedID, error) {
	return resolveIDByPrefix(ctx, client, "deleted project", input, fetchDeletedProjectCandidates)
}

func fetchDeletedProjectCandidates(ctx context.Context, client *cli.APIClient) ([]idCandidate, error) {
	var result map[string]any
	if err := client.GetJSON(ctx, "/api/projects/trash", &result); err != nil {
		return nil, err
	}
	projectsRaw, _ := result["projects"].([]any)
	candidates := make([]idCandidate, 0, len(projectsRaw))
	for _, raw := range projectsRaw {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		candidates = append(candidates, idCandidate{
			ID:      strVal(p, "id"),
			Display: strVal(p, "title"),
			Detail:  shortDateString(strVal(p, "deleted_at")),
		})
	}
	return candidates, nil
}
