package config

import "os"

var WorkspaceEnabled = os.Getenv("WORKSPACE_ENABLED") == "true"

var DbPath = os.Getenv("SQLITE_DB_PATH")
var Port = func() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}()
