package routes

import (
	"database/sql"
	"fmt"
	"moksarab/config"
	"moksarab/database"
	"regexp"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

/*
SELECT rr.status, rr.response, rr.path_params,
	COALESCE('/:' || r0.param_name, r0.path) || COALESCE('/:' || r1.param_name, r1.path) AS full_path -- loop over paths in original order
FROM route_response rr
	-- loop over paths in reverse order
	JOIN route r1 ON r1.id = rr.path AND (r1.path = '/1' OR r1.is_param = 1)
	JOIN route r0 ON r0.id = r1.parent_path AND (r0.path = '/test' OR r0.is_param = 1)
WHERE rr.method = 'GET'
	AND r1.workspace = 4269
ORDER BY rr.path_params IS NULL, rr.path_params;
*/

type SarabResponse struct {
	Status    int            `json:"status"`
	Response  sql.NullString `json:"response"`
	PathParam sql.NullString `json:"path_param"`
	FullPath  string         `json:"full_path"`
}

func HandleSarabRequests(c *fiber.Ctx) error {
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
	re := regexp.MustCompile(`^/sarab/\d+`)
	trimmedPath := re.ReplaceAllString(c.Path(), "")
	pathParts := getPathParts(trimmedPath)
	slices.Reverse(pathParts)

	query := fmt.Sprintf(`
			SELECT rr.status, rr.response, rr.path_params,
				%s
			FROM route_response rr
				%s
			WHERE rr.method = ?
				AND r0.workspace = ?
			ORDER BY rr.path_params IS NULL, rr.path_params
			`, getFullPathSelector(len(pathParts)), getJoins(pathParts),
	)

	rows, err := database.Db.QueryContext(
		c.Context(),
		query,
		c.Method(),
		workspaceId,
	)
	if err != nil {
		return HandleSQLErrors(c, err)
	}

	for rows.Next() {
		var response SarabResponse
		rows.Scan(&response.Status, &response.Response, &response.PathParam, &response.FullPath)
		mapPathParamsToFullPath(&response)
		log.Debugf("trying to match [%s] with found response: %+v", trimmedPath, response)
		if trimmedPath == response.FullPath || !response.PathParam.Valid {
			rows.Close()
			if response.Response.Valid {
				return c.Status(response.Status).SendString(response.Response.String)
			}
			return c.SendStatus(response.Status)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error":   "Not Found",
		"message": fmt.Sprintf("path [%s] with http method [%s] is not found", trimmedPath, c.Method()),
	})
}

func getFullPathSelector(partsLength int) string {
	selector := ""
	for i := range partsLength {
		selector = selector + fmt.Sprintf("COALESCE('/:' || r%d.param_name, r%d.path) ", i, i)
		if i+1 != partsLength {
			selector = selector + " || "
		}
	}
	return selector + " AS full_path "
}

func getJoins(pathParts []string) string {
	lastPartIndex := len(pathParts) - 1
	joins := fmt.Sprintf(" JOIN route r%d ON r%d.id = rr.path AND (r%d.path = '%s' OR r%d.is_param = 1)\n",
		lastPartIndex,
		lastPartIndex,
		lastPartIndex,
		pathParts[0],
		lastPartIndex,
	)

	for i := 1; i < len(pathParts); i++ {
		thisPartIndex := lastPartIndex - i
		joins += fmt.Sprintf(" JOIN route r%d ON r%d.id = r%d.parent_path AND (r%d.path = '%s' OR r%d.is_param = 1)\n",
			thisPartIndex,
			thisPartIndex,
			thisPartIndex+1,
			thisPartIndex,
			pathParts[i],
			thisPartIndex,
		)
	}

	return joins
}

func mapPathParamsToFullPath(response *SarabResponse) {

	if response.PathParam.Valid {
		for part := range strings.SplitSeq(response.PathParam.String, ", ") {
			kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
			if len(kv) == 2 {
				response.FullPath = strings.ReplaceAll(response.FullPath, ":"+strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
			}
		}
	}
}
