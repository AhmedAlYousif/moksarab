package config

import "os"

var WorkspaceEnabled = os.Getenv("WORKSPACE_ENABLED") == "true"
