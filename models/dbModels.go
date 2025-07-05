package models

import "database/sql"

type Workspace struct {
	Id          int64  `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

const createWorkspaceTableQuery = `
	CREATE TABLE IF NOT EXISTS workspace (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT
	);
`

type Route struct {
	Id           int64          `json:"id"`
	Path         string         `json:"path"`
	ParentPath   sql.NullInt64  `json:"parent_path"`
	IsParam      bool           `json:"is_param"`
	ParamName    sql.NullString `json:"param_name"`
	HasResponses bool           `json:"has_response"`
	Workspace    int64          `json:"workspace"`
}

const createRouteTableQuery = `
	CREATE TABLE IF NOT EXISTS route (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL,
		parent_path INTEGER,
		is_param BOOLEAN DEFAULT 0,
		param_name TEXT,
		has_responses BOOLEAN DEFAULT 0,
		workspace INTEGER NOT NULL,
		FOREIGN KEY (workspace) REFERENCES workspace(id),
		FOREIGN KEY (parent_path) REFERENCES route(id),
		UNIQUE (path, workspace, parent_path)
	);
`

type RouteResponse struct {
	Id         int64          `json:"id"`
	Path       int64          `json:"path"`
	PathParams sql.NullString `json:"path_params"`
	Method     string         `json:"method"`
	Status     int            `json:"status"`
	Response   sql.NullString `json:"response"`
}

const createRouteResponseTableQuery = `
	CREATE TABLE IF NOT EXISTS route_response (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path INTEGER NOT NULL,
		path_params TEXT,
		method TEXT NOT NULL,
		status INTEGER NOT NULL,
		response TEXT,
		FOREIGN KEY (path) REFERENCES route(id),
		UNIQUE (path_params, path, method)
	);
`

const CreateQueries = "PRAGMA foreign_key = ON; \n " + createWorkspaceTableQuery + " \n " + createRouteTableQuery + " \n " + createRouteResponseTableQuery
