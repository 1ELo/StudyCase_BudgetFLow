package querybuilder

import (
	"fmt"
	"strings"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
)

// allowedSortColumns maps user-facing sort keys to safe DB column names.
var allowedSortColumns = map[string]string{
	"created_at":         "projects.created_at",
	"envelope_remaining": "projects.envelope_remaining",
	"title":              "projects.title",
}

// ParseSort parses a sort string (e.g., "-created_at") into a safe column name and direction.
// Prefix "-" means DESC. Returns error if column is not in the allowlist.
func ParseSort(sort string) (column string, desc bool, err error) {
	if sort == "" {
		return "projects.created_at", true, nil // default: newest first
	}

	desc = false
	key := sort
	if strings.HasPrefix(sort, "-") {
		desc = true
		key = sort[1:]
	}

	col, ok := allowedSortColumns[key]
	if !ok {
		return "", false, fmt.Errorf("invalid sort column: %s", key)
	}

	return col, desc, nil
}

// BuildProjectFilter converts a ProjectListQuery into a ProjectFilter
// with safe defaults for pagination.
func BuildProjectFilter(q domain.ProjectListQuery) (domain.ProjectListQuery, error) {
	// Defaults
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	// Validate status if provided
	if q.Status != "" {
		switch domain.ProjectStatus(q.Status) {
		case domain.ProjectOpen, domain.ProjectClosed:
			// valid
		default:
			return q, fmt.Errorf("invalid status filter: %s", q.Status)
		}
	}

	return q, nil
}

// ProjectFilter is used by the repository layer for building DB queries.
type ProjectFilter struct {
	Status     *string
	ManagerID  *int64
	Search     *string
	SortColumn string
	SortDesc   bool
	Page       int
	Limit      int
}
