// Package ops defines operation IDs exposed to harnesses.
package ops

// Stable operation IDs.
const (
	OpJiraIssueGet         = "jira.issue.get"
	OpJiraIssueSearch      = "jira.issue.search"
	OpConfluencePageGet    = "confluence.page.get"
	OpConfluencePageSearch = "confluence.page.search"
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
