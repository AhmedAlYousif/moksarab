package routes

import (
	"database/sql"
	"fmt"
	"moksarab/config"
	"moksarab/database"
	"moksarab/models"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func RegisterAPIRoutes(router fiber.Router) {
	if config.WorkspaceEnabled {
		router.Post("/workspaces", createWorkspace)
		router.Get("/workspaces", getWorkspaces)
		router.Post("/workspaces/:workspaceId/mocks", createNewMock)
		router.Get("/workspaces/:workspaceId/mocks", getMocks)
		router.Post("/workspaces/:workspaceId/mocks/:mockId", createMockResponse)
	} else {
		router.Post("/mocks", createNewMock)
		router.Get("/mocks", getMocks)
		router.Post("/mocks/:mockId", createMockResponse)
	}
}

func createWorkspace(c *fiber.Ctx) error {

	reqBody := new(models.Workspace)
	if parsingError := c.BodyParser(reqBody); parsingError != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": parsingError.Error(),
		})
	}
	if reqBody.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "workspace name cannot be empty.",
		})
	}

	var id int64
	insertError := database.Db.QueryRowContext(c.Context(), "INSERT INTO workspace (name, description) VALUES (?, ?) RETURNING id",
		reqBody.Name,
		reqBody.Description,
	).Scan(&id)
	if insertError != nil {
		return HandleSQLErrors(c, insertError)
	}

	c.Location(fmt.Sprintf("/workspaces/%d", id))
	return c.SendStatus(fiber.StatusCreated)
}

func getWorkspaces(c *fiber.Ctx) error {

	pageNumber := c.QueryInt("page", 0)
	pageSize := c.QueryInt("size", 10)
	if pageNumber < 0 || pageSize < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Page must be grater than 0 and size must be grater than 1",
		})
	}

	rows, selectError := database.Db.QueryContext(c.Context(), "SELECT id, name, description FROM workspace LIMIT ? OFFSET ?",
		pageSize,
		(pageSize * pageNumber),
	)
	if selectError != nil {
		return HandleSQLErrors(c, selectError)
	}
	defer rows.Close()
	var workspaces []models.Workspace
	for rows.Next() {
		var workspace models.Workspace
		extractError := rows.Scan(&workspace.Id, &workspace.Name, &workspace.Description)
		if extractError != nil {
			return HandleSQLErrors(c, extractError)
		}
		workspaces = append(workspaces, workspace)
	}

	countResult := database.Db.QueryRowContext(c.Context(), "SELECT COUNT(*) FROM workspace")
	var totalElements int
	countError := countResult.Scan(&totalElements)
	if countError != nil {
		return HandleSQLErrors(c, countError)
	}

	pageResponse := models.PageOf(workspaces, pageNumber, pageSize, totalElements)

	return c.Status(fiber.StatusOK).JSON(pageResponse)
}

type CreateNewMockRequest struct {
	Path         string  `json:"path"`
	Method       string  `json:"method"`
	Status       int     `json:"status"`
	ResponseBody *string `json:"response_body,omitempty"`
}

func createNewMock(c *fiber.Ctx) error {

	workspaceId := 4269
	if config.WorkspaceEnabled {
		var err error
		if workspaceId, err = c.ParamsInt("workspaceId", -1); err != nil || workspaceId == -1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad Request",
				"message": "workspaceId must be valid integer",
			})
		}
	}
	log.Debugf("Creating a new mock in workspace %d", workspaceId)
	var reqBody *CreateNewMockRequest

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": err.Error(),
		})
	}

	if !isValidPath(reqBody.Path) || !isValidHttpMethod(strings.ToUpper(reqBody.Method)) || !isValidHttpResponseStatus(reqBody.Status) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Path, Method, and Status must be vaild",
		})
	}

	transaction, err := database.Db.BeginTx(c.Context(), nil)
	if err != nil {
		return HandleSQLErrors(c, err)
	}

	pathParts := getPathParts(reqBody.Path)
	numberOfParts := len(pathParts)
	var lastInseretedId *sql.NullInt64

	for i, part := range pathParts {
		lastInseretedId, err = insertPartReturningIdOrGetExistingRouteId(transaction, part, lastInseretedId, workspaceId, (i+1) == numberOfParts)
		if err != nil {
			return HandleSQLErrors(c, err)
		}
	}

	var mockedResponseBody sql.NullString
	if reqBody.ResponseBody != nil {
		mockedResponseBody = sql.NullString{String: *reqBody.ResponseBody, Valid: true}
	} else {
		mockedResponseBody = sql.NullString{Valid: false}
	}

	_, err = transaction.Exec("INSERT INTO route_response (status, path, method, response) VALUES (?, ?, ?, ?)",
		reqBody.Status,
		lastInseretedId.Int64,
		strings.ToUpper(reqBody.Method),
		mockedResponseBody,
	)
	if err != nil {
		return HandleSQLErrors(c, err)
	}
	err = transaction.Commit()
	if err != nil {
		return HandleSQLErrors(c, err)
	}

	return c.SendStatus(fiber.StatusCreated)
}

func getPathParts(path string) []string {
	if path == "" || path == "/" {
		return []string{"/"}
	}
	re := regexp.MustCompile("(/[^/]+)")
	parts := re.FindAllString("/"+path, -1)
	return parts
}

func insertPartReturningIdOrGetExistingRouteId(transaction *sql.Tx, part string, lastInsertedId *sql.NullInt64, workspaceId int, isLastPart bool) (*sql.NullInt64, error) {

	isParam := strings.HasPrefix(part, "/:")
	paramName := sql.NullString{Valid: false}
	if isParam {
		paramName.Valid = true
		paramName.String = strings.Split(part, "/:")[1]
		part = "/<param>"
	}
	var id sql.NullInt64

	existingRouteResult := transaction.QueryRow("SELECT id, has_response FROM route WHERE path = ? and workspace = ? and parent_path = ?",
		part,
		int64(workspaceId),
		lastInsertedId,
	)
	var alreadyHasResponse bool
	err := existingRouteResult.Scan(&id, &alreadyHasResponse)
	if err == nil {
		if isLastPart && !alreadyHasResponse {
			_, err = transaction.Exec("UPDATE route set has_response = 1 where id = ?", id)
			if err != nil {
				return nil, err
			}
		} else if isLastPart && alreadyHasResponse {
			return nil, fmt.Errorf("UNIQUE constraint failed: this route already exist.")
		}
		return &id, nil
	}

	err = transaction.QueryRow("INSERT INTO route (path, parent_path, workspace, has_responses, is_param, param_name) VALUES (?, ?, ?, ?, ?, ?) RETURNING id",
		part,
		lastInsertedId,
		workspaceId,
		isLastPart,
		isParam,
		paramName,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

var validPath = regexp.MustCompile(`^/?([a-zA-Z0-9_\-:]+/?)*$`)

func isValidPath(path string) bool {
	return validPath.MatchString(path)
}

func isValidHttpMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		return true
	}
	return false
}

func isValidHttpResponseStatus(statusCode int) bool {
	return statusCode >= 100 && statusCode <= 599
}

type GetMocksResponse struct {
	ResponseId   int64          `json:"response_id"`
	FullPath     string         `json:"full_path"`
	ParamNames   string         `json:"param_names"`
	Method       string         `json:"method"`
	ResponseBody sql.NullString `json:"response_body"`
	Status       int            `json:"status"`
	DirectPathId int64          `json:"direct_path_id"`
}

func getMocks(c *fiber.Ctx) error {

	workspaceId := 4269
	if config.WorkspaceEnabled {
		var err error
		if workspaceId, err = c.ParamsInt("workspaceId", -1); err != nil || workspaceId == -1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad Request",
				"message": "workspaceId must be valid integer",
			})
		}
	}

	rows, err := database.Db.QueryContext(c.Context(), `	
		WITH RECURSIVE route_path(id, path, parent_path, full_path, param_names, origin_id) AS (
			-- Base case: responses with no path_params, filtered by workspace
			SELECT
				r.id,
				r.path,
				r.parent_path,
				r.path AS full_path,
				r.param_name AS param_names,
				r.id AS origin_id
			FROM route r
			JOIN route_response rr ON rr.path = r.id
			WHERE rr.path_params IS NULL
			  AND r.workspace = ?   -- 👈

			UNION ALL

			-- Recursive step: climb up parent_path, staying within the same workspace
			SELECT
				p.id,
				p.path,
				p.parent_path,
				p.path || rp.full_path,
				COALESCE(p.param_name || ',', '') || rp.param_names,
				rp.origin_id
			FROM route p
			JOIN route_path rp ON rp.parent_path = p.id
			WHERE p.workspace = ?   -- 👈
		)

		-- Final result with pagination
		SELECT
			rr.id AS response_id,
			rp.full_path AS full_path,
			rp.param_names as param_names,
			rr.method,
			rr.response AS response_body,
			rr.status,
			rr.path AS direct_path_id
		FROM route_response rr
		JOIN (
			SELECT origin_id, full_path, param_names
			FROM route_path
			WHERE parent_path IS NULL
		) rp ON rr.path = rp.origin_id
		WHERE rr.path_params IS NULL
		ORDER BY rr.id;
		`,
		workspaceId,
		workspaceId,
	)
	if err != nil {
		return HandleSQLErrors(c, err)
	}
	defer rows.Close()

	var mocks []GetMocksResponse

	for rows.Next() {
		var mock GetMocksResponse
		err := rows.Scan(&mock.ResponseId, &mock.FullPath, &mock.ParamNames, &mock.Method, &mock.ResponseBody, &mock.Status, &mock.DirectPathId)
		if err != nil {
			return HandleSQLErrors(c, err)
		}
		mocks = append(mocks, mock)
	}

	return c.Status(fiber.StatusOK).JSON(mocks)
}

func createMockResponse(c *fiber.Ctx) error {
	workspaceId := 4269
	var mockId int
	var err error
	if config.WorkspaceEnabled {
		if workspaceId, err = c.ParamsInt("workspaceId", -1); err != nil || workspaceId == -1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad Request",
				"message": "workspaceId must be valid integer",
			})
		}
	}
	if mockId, err = c.ParamsInt("mockId", -1); err != nil || mockId == -1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "mockId must be valid integer",
		})
	}

	var reqBody models.RouteResponse
	err = c.BodyParser(&reqBody)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "request body is invalid",
		})
	}

	if !isValidHttpMethod(strings.ToUpper(reqBody.Method)) || !isValidHttpResponseStatus(reqBody.Status) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "HTTP method and response status must be valid",
		})
	}

	if !reqBody.PathParams.Valid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "path params are required",
		})
	}

	_, err = database.Db.ExecContext(c.Context(), "INSERT INTO route_response (path, path_params, method, status, response) VALUES (?, ?, ?, ?, ?)",
		mockId,
		reqBody.PathParams,
		strings.ToUpper(reqBody.Method),
		reqBody.Status,
		reqBody.Response,
	)

	return c.SendStatus(fiber.StatusCreated)
}

func HandleSQLErrors(c *fiber.Ctx, err error) error {
	msg := err.Error()

	log.Debugf("Database Error: %v", err)

	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "Conflict",
			"message": "A record with this value already exists.",
		})
	case strings.Contains(msg, "NOT NULL"):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": msg,
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Database Error",
			"message": msg,
		})
	}
}
