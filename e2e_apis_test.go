package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"moksarab/database"
	"moksarab/models"
	"moksarab/routes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

const PORT = "8081"
const BASE_URL = "http://localhost:" + PORT

var errCh chan error

func beforeEach() *fiber.App {
	database.InitilizeDatabase()

	errCh = make(chan error, 1)
	app := InitilizeMocSarabServer()
	go func() {
		if err := app.Listen(":" + PORT); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	time.Sleep(100 * time.Millisecond)

	return app
}

func afterEach(t *testing.T, app *fiber.App) {
	_ = app.Shutdown()
	_ = database.Db.Close()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server error: %v", err)
		}
	default:
	}
}

func sendRequest(client *http.Client, url, method string, body interface{}) (*http.Response, error) {

	var bodyBuf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&bodyBuf).Encode(body); err != nil {
			return nil, fmt.Errorf("failed to encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &bodyBuf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)

}

func createWorkspace(client *http.Client, name, description string) (*http.Response, error) {
	workspace := models.Workspace{
		Name:        name,
		Description: description,
	}
	return sendRequest(client, BASE_URL+"/api/workspaces", "POST", workspace)
}

func TestWorkspace(t *testing.T) {

	app := beforeEach()

	client := &http.Client{}

	workspaceName := "myWorkspace"

	createWorkspaceRes, err := createWorkspace(client, workspaceName, "my work space is for fake work")
	if err != nil {
		t.Fatalf("Creating workspace failed: %v", err)
	}
	defer createWorkspaceRes.Body.Close()

	getWorkspasesRes, err := sendRequest(client, BASE_URL+"/api/workspaces", "GET", nil)
	if err != nil {
		t.Fatalf("Getting workspaces failed: %v", err)
	}
	defer getWorkspasesRes.Body.Close()

	if getWorkspasesRes.StatusCode != http.StatusOK {
		t.Fatalf("Expected while get workspaces 200, got %d", getWorkspasesRes.StatusCode)
	}
	var page models.PageModel[models.Workspace]

	if err := json.NewDecoder(getWorkspasesRes.Body).Decode(&page); err != nil {
		t.Fatalf("Failed to decode get workspaces response: %v", err)
	}

	if len(page.Content) != 1 {
		t.Fatalf("expected 1 workspace, found %d", len(page.Content))
	}

	if !page.First {
		t.Fatal("expected to be fist page, but fist value was false")
	}

	if !page.Last {
		t.Fatal("expected to be last page, but last value was false")
	}

	if page.TotalElements != 1 {
		t.Fatalf("expected 1 total elements, found %d", page.TotalElements)
	}

	if page.TotalPages != 1 {
		t.Fatalf("expected 1 total pages, found %d", page.TotalPages)
	}

	if page.Content[0].Name != workspaceName {
		t.Fatalf("expected workspace name to be %s, found %s", workspaceName, page.Content[0].Name)
	}

	afterEach(t, app)
}

func TestCreatingMock(t *testing.T) {

	app := beforeEach()

	client := &http.Client{}
	createWorkSpaceRes, err := createWorkspace(client, "team1", "the first team")
	if err != nil {
		t.Fatalf("Error creating workspace 'team1': %v", err)
	}
	defer createWorkSpaceRes.Body.Close()

	location, err := createWorkSpaceRes.Location()
	if err != nil {
		t.Fatalf("Error getting workspace location: %v", err)
	}

	locationParts := strings.Split(location.Path, "/")
	workspaceId := locationParts[len(locationParts)-1]

	newMock := routes.CreateNewMockRequest{
		Path:   "/resources/:resourceId",
		Method: "GET",
		Status: 200,
	}
	createNewMockRes, err := sendRequest(client, BASE_URL+"/api/workspaces/"+workspaceId+"/mocks", "POST", newMock)
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer createNewMockRes.Body.Close()

	if createNewMockRes.StatusCode != 201 {
		body, err := io.ReadAll(createNewMockRes.Body)
		if err == nil {
			t.Logf("%v\n", string(body))
		}
		t.Fatalf("expected status to be 201, but found %d", createNewMockRes.StatusCode)
	}

	routesResult, err := database.Db.Query("SELECT id, path, is_param, has_responses, workspace, parent_path, param_name FROM route")
	if err != nil {
		t.Fatalf("Error fetching routes: %v", err)
	}

	var routesList []models.Route

	for routesResult.Next() {
		var route models.Route
		routesResult.Scan(&route.Id, &route.Path, &route.IsParam, &route.HasResponses, &route.Workspace, &route.ParentPath, &route.ParamName)
		routesList = append(routesList, route)
	}

	if len(routesList) != 2 {
		t.Fatalf("expected 2 routes to be created, but found %d", len(routesList))
	}

	assertRoute(routesList[0], "/resources", false, false, workspaceId, sql.NullInt64{Valid: false}, sql.NullString{Valid: false}, t)
	assertRoute(routesList[1], "/<param>", true, true, workspaceId, sql.NullInt64{Int64: routesList[0].Id, Valid: true}, sql.NullString{String: "resourceId", Valid: true}, t)

	routeResponseResult := database.Db.QueryRow("SELECT id, path, path_params, method, status, response from route_response where path = ?", routesList[1].Id)
	var routeResponse models.RouteResponse

	err = routeResponseResult.Scan(&routeResponse.Id, &routeResponse.Path, &routeResponse.PathParams, &routeResponse.Method, &routeResponse.Status, &routeResponse.Response)
	if err != nil {
		t.Fatalf("error extracting route response: %v", err)
	}

	if routeResponse.Path != routesList[1].Id {
		t.Fatalf("expected path id to be %v, but found %v", routesList[1].Id, routeResponse.Path)
	}
	if routeResponse.PathParams.Valid {
		t.Fatalf("expected pathParams to be empty, but found %v", routeResponse.PathParams.String)
	}
	if routeResponse.Method != "GET" {
		t.Fatalf("expected method to be GET, but found %s", routeResponse.Method)
	}
	if routeResponse.Status != 200 {
		t.Fatalf("expected status to be 200, but found %d", routeResponse.Status)
	}
	if routeResponse.Response.Valid {
		t.Fatalf("expected response to be empty, but found %s", routeResponse.Response.String)
	}

	getMocksRes, err := sendRequest(client, BASE_URL+"/api/workspaces/"+workspaceId+"/mocks", "GET", nil)
	if err != nil {
		t.Fatalf("error fetching mocks: %v", err)
	}
	defer getMocksRes.Body.Close()

	if getMocksRes.StatusCode != 200 {
		body, err := io.ReadAll(createNewMockRes.Body)
		if err == nil {
			t.Logf("%v\n", string(body))
		}
		t.Fatalf("expected get mocks response code to be 200, but found %d", getMocksRes.StatusCode)
	}

	var responses []routes.GetMocksResponse
	if err := json.NewDecoder(getMocksRes.Body).Decode(&responses); err != nil {
		t.Fatalf("error decoding get mocks response: %v", err)
	}

	if len(responses) != 1 {
		t.Fatalf("expected to get 1 mock response, but found %d", len(responses))
	}

	if responses[0].FullPath != "/resources/<param>" {
		t.Fatalf("expected full path to be '/resources/<param>', but found %s", responses[0].FullPath)
	}

	if responses[0].ParamNames != "resourceId" {
		t.Fatalf("expected param names to be 'resourceId', but found %s", responses[0].ParamNames)
	}

	if responses[0].Status != 200 {
		t.Fatalf("expected status to be 200, but found %d", responses[0].Status)
	}

	if responses[0].Method != "GET" {
		t.Fatalf("expected method to be GET, but found %s", responses[0].Method)
	}

	if responses[0].ResponseBody.Valid {
		t.Fatalf("expected response body to be empty, but found %s", responses[0].ResponseBody.String)
	}

	if responses[0].DirectPathId != routesList[1].Id {
		t.Fatalf("expected direct path id to be %d, but found %d", routesList[1].Id, responses[0].DirectPathId)
	}

	mockResponse := models.RouteResponse{
		Method:     "GET",
		Status:     400,
		PathParams: sql.NullString{String: "resourceId: 42", Valid: true},
	}

	resOfCreateMockResponse, err := sendRequest(client, BASE_URL+"/api/workspaces/"+workspaceId+"/mocks/"+fmt.Sprintf("%d", responses[0].DirectPathId), "POST", mockResponse)
	if err != nil {
		t.Fatalf("error creating a mocked response with specific param value: %v", err)
	}

	if resOfCreateMockResponse.StatusCode != 201 {
		t.Fatalf("expected creating mocked response status to be 201, but found %d", resOfCreateMockResponse.StatusCode)
	}

	// database.Db.Exec("VACUUM INTO 'test.db'")

	sarabRes, err := sendRequest(client, BASE_URL+"/sarab/"+workspaceId+"/resources/1", "GET", nil)
	if err != nil {
		t.Fatalf("error testing sarab response")
	}

	if sarabRes.StatusCode != newMock.Status {
		body, err := io.ReadAll(sarabRes.Body)
		if err == nil {
			t.Logf("%v\n", string(body))
		}
		t.Fatalf("expected [%s] response to be %d, but found %d", newMock.Path, newMock.Status, sarabRes.StatusCode)
	}

	sarabRes2, err := sendRequest(client, BASE_URL+"/sarab/"+workspaceId+"/resources/42", "GET", nil)
	if err != nil {
		t.Fatalf("error testing sarab2 response")
	}

	if sarabRes2.StatusCode != mockResponse.Status {
		t.Fatalf("expected response to be %d, but found %d", mockResponse.Status, sarabRes2.StatusCode)
	}

	afterEach(t, app)
}

func assertRoute(route models.Route, path string, isParam, hasResponses bool, workspaceId string, parentPath sql.NullInt64, paramName sql.NullString, t *testing.T) {

	if route.Path != path {
		t.Fatalf("expected path to be %s, but found %s", path, route.Path)
	}
	if route.IsParam != isParam {
		t.Fatalf("expected isParam to be %v, but found %v", isParam, route.IsParam)
	}
	if route.HasResponses != hasResponses {
		t.Fatalf("expected hasResponses to be %v, but found %v", hasResponses, route.HasResponses)
	}
	if fmt.Sprint(route.Workspace) != workspaceId {
		t.Fatalf("expected workspaceId to be %s, but found %d", workspaceId, route.Workspace)
	}
	if route.ParentPath.Int64 != parentPath.Int64 {
		t.Fatalf("expected parentPath to be %v, but found %v", parentPath.Int64, route.ParentPath.Int64)
	}
	if route.ParamName.String != paramName.String {
		t.Fatalf("expected paramName to be %v, but found %v", paramName, route.ParamName)
	}
}
