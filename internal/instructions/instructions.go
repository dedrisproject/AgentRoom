package instructions

import (
	"fmt"
	"strings"

	"github.com/dedrisproject/agentroom/internal/store"
)

// Generate produces the agent-room.md content for the given agent.
func Generate(agent *store.Agent, room *store.Room, baseURL, adminName string) string {
	// Strip trailing slash from baseURL.
	base := strings.TrimRight(baseURL, "/")

	role := agent.Role
	if role == "" {
		role = "(none)"
	}
	repo := agent.Repo
	if repo == "" {
		repo = "(none)"
	}

	return fmt.Sprintf(`# agent-room.md

You are agent `+"`%s`"+` (role: %s, repo: %s) in room "%s".
You are part of a team of AI agents, each isolated on its own machine with only its own repo.
Never touch another repo: if you need a change outside yours, open it here as a request.
Use AgentRoom to ask for changes, reply, and close resolved blockers.

API: `+"`%s/api/agent`"+` — token: `+"`%s`"+`

`+"```bash"+`
# Read your inbox
curl "%s/api/agent/messages?token=%s"

# Open a request
curl -X POST "%s/api/agent/messages?token=%s" \
  -d to_agent=backend-agent -d subject="Need API change" -d priority=blocker \
  --data-urlencode message="What you need and why it blocks you"

# Reply
curl -X POST "%s/api/agent/messages/123/reply?token=%s" \
  --data-urlencode message="Done, see commit abc123"

# Close a resolved request
curl -X POST "%s/api/agent/messages/123/close?token=%s"
`+"```"+`

Recipients: `+"`%s`"+`, `+"`all`"+`, or the exact name of an agent in the room.
Use `+"`priority=blocker`"+` only when you are actually blocked; otherwise `+"`normal`"+`.
`,
		agent.Name, role, repo, room.Name,
		base, agent.AccessToken,
		base, agent.AccessToken,
		base, agent.AccessToken,
		base, agent.AccessToken,
		base, agent.AccessToken,
		adminName,
	)
}
