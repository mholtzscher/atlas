// Package ops defines operation IDs exposed to harnesses.
package ops

// Stable operation IDs.
const (
	OpJiraIssueDescribe       = "jira.issue.describe"
	OpJiraIssueSearch         = "jira.issue.search"
	OpJiraProjectList         = "jira.project.list"
	OpJiraIssueComments       = "jira.issue.comments"
	OpJiraIssueTypes          = "jira.issue.types"
	OpJiraMyself              = "jira.myself"
	OpConfluenceSpaceList     = "confluence.space.list"
	OpConfluenceSpaceDescribe = "confluence.space.describe"
	OpConfluencePageComments  = "confluence.page.comments"
	OpConfluencePageDescribe  = "confluence.page.describe"
	OpConfluencePageView      = "confluence.page.view"
	OpConfluencePageSearch    = "confluence.page.search"
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
			Op:      OpJiraIssueDescribe,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"issueKey"},
				Flags:      []string{"fields", "expand", "raw"},
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
					"raw",
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
					"raw",
				},
			},
		},
		{
			Op:      OpConfluenceSpaceDescribe,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"spaceKey"},
				Flags:      []string{"raw"},
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
					"raw",
				},
			},
		},
		{
			Op:      OpConfluencePageDescribe,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"pageID"},
				Flags: []string{
					"include-labels",
					"include-properties",
					"include-operations",
					"include-versions",
				},
			},
		},
		{
			Op:      OpConfluencePageView,
			Mutates: false,
			Auth:    []string{"pat", "oauth"},
			Args: Args{
				Positional: []string{"pageID"},
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
					"raw",
				},
			},
		},
	}
}
