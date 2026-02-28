// Package ops defines operation IDs exposed to harnesses.
package ops

// Stable operation IDs.
const (
	OpJiraIssueGet           = "jira.issue.get"
	OpJiraIssueSearch        = "jira.issue.search"
	OpJiraProjectList        = "jira.project.list"
	OpJiraIssueComments      = "jira.issue.comments"
	OpJiraIssueTypes         = "jira.issue.types"
	OpJiraMyself             = "jira.myself"
	OpConfluenceSpaceList    = "confluence.space.list"
	OpConfluenceSpaceGet     = "confluence.space.get"
	OpConfluencePageComments = "confluence.page.comments"
	OpConfluencePageGet      = "confluence.page.get"
	OpConfluencePageSearch   = "confluence.page.search"
)

// Definition describes one operation for allowlisting.
type Definition struct {
	Op      string   `json:"op"`
	Mutates bool     `json:"mutates"`
	Auth    []string `json:"auth"`
	Args    Args     `json:"args"`
}

// Args describes positional and flag args for an operation.
type Args struct {
	Positional []string `json:"positional"`
	Flags      []string `json:"flags"`
}

// All returns all machine-visible operations.
func All() []Definition {
	definitions := jiraDefinitions()
	definitions = append(definitions, confluenceDefinitions()...)

	return definitions
}

func jiraDefinitions() []Definition {
	return []Definition{
		{
			Op:      OpJiraIssueGet,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"issueKey"},
				Flags:      []string{"fields", "expand", "fields-by-keys"},
			},
		},
		{
			Op:      OpJiraIssueSearch,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{},
				Flags: []string{
					"jql",
					"fields",
					"expand",
					"fields-by-keys",
					"limit",
					"page-size",
					"page-token",
				},
			},
		},
		{
			Op:      OpJiraProjectList,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args:    Args{},
		},
		{
			Op:      OpJiraIssueComments,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"issueKey"},
			},
		},
		{
			Op:      OpJiraIssueTypes,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args:    Args{},
		},
		{
			Op:      OpJiraMyself,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args:    Args{},
		},
	}
}

func confluenceDefinitions() []Definition {
	return []Definition{
		{
			Op:      OpConfluenceSpaceList,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{},
				Flags: []string{
					"limit",
					"page-size",
					"cursor",
				},
			},
		},
		{
			Op:      OpConfluenceSpaceGet,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"spaceKey"},
			},
		},
		{
			Op:      OpConfluencePageComments,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"pageID"},
				Flags: []string{
					"limit",
					"page-size",
					"cursor",
					"body-format",
				},
			},
		},
		{
			Op:      OpConfluencePageGet,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"pageID"},
				Flags: []string{
					"body-format",
					"include-labels",
					"include-properties",
					"include-operations",
					"include-versions",
				},
			},
		},
		{
			Op:      OpConfluencePageSearch,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{},
				Flags: []string{
					"cql",
					"limit",
					"page-size",
					"cursor",
					"body-format",
					"include-labels",
					"include-properties",
					"include-operations",
					"include-versions",
				},
			},
		},
	}
}
