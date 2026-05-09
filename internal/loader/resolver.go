package loader

import "os"

// WorkflowDirs returns the ordered list of directories to search for a workflow,
// based on the MCP client name detected at initialize time.
// Search order: global dir first, then repo dir. Global takes precedence on same id.
func WorkflowDirs(clientName string) (globalDir string, repoDir string) {
	home, _ := os.UserHomeDir()

	switch clientName {
	case "opencode":
		return home + "/.config/opencode/workflows", ".opencode/workflows"
	case "github-copilot", "copilot":
		return home + "/.copilot/workflows", ".github/workflows-mcp"
	case "claude":
		return home + "/.claude/workflows", ".claude/workflows"
	default:
		return home + "/.config/w7s/workflows", ".w7s/workflows"
	}
}
