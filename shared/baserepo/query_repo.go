package baserepo

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type QueryBuilder[T any] struct {
	db *gorm.DB
}

type filterCondition struct {
	Condition string
	Args      []interface{}
	Logic     string // "and" or "or"
}

var opMap = map[string]string{
	"eq":   "%s = ?",
	"gte":  "%s >= ?",
	"lte":  "%s <= ?",
	"lt":   "%s < ?",
	"gt":   "%s > ?",
	"like": "%s LIKE ?",
	"in":   "%s IN ?",
}

func NewQueryBuilder[T any](db *gorm.DB) *QueryBuilder[T] {
	return &QueryBuilder[T]{db: db}
}

func formatOrder(v string) string {
	if v == "" {
		return v
	}

	v = strings.ToLower(v)
	if v == "desc" || v == "asc" {
		return v
	}

	return "desc"
}

func getSortedKeys(m map[int][]filterCondition) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func (q *QueryBuilder[T]) ApplyFilters(filter any) *QueryBuilder[T] {
	if filter == nil {
		return q
	}

	db := q.db

	v := reflect.ValueOf(filter)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	var sorts []string
	groups := make(map[int][]filterCondition)
	groupLogics := make(map[int]string)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("filter")
		sortTag := field.Tag.Get("sort")
		logicTag := strings.ToLower(strings.TrimSpace(field.Tag.Get("logic")))
		groupTag := field.Tag.Get("group")
		groupLogicTag := strings.ToLower(strings.TrimSpace(field.Tag.Get("group_logic")))

		if logicTag == "" {
			logicTag = "and"
		}

		groupID := 0
		if groupTag != "" {
			if g, err := strconv.Atoi(groupTag); err == nil {
				groupID = g
			}
		}

		// Store group-level logic
		if groupLogicTag != "" {
			groupLogics[groupID] = groupLogicTag
		} else if _, exists := groupLogics[groupID]; !exists {
			groupLogics[groupID] = "and" // default group logic
		}

		// Handle sort
		if sortTag != "" {
			val := v.Field(i)
			var order string
			switch field.Type.Kind() {
			case reflect.Ptr:
				if !val.IsNil() && field.Type.Elem().Kind() == reflect.String {
					order = val.Elem().String()
				}
			case reflect.String:
				order = val.String()
				if order == "" {
					order = "desc"
				}
			}
			order = formatOrder(order)
			if order != "" {
				sorts = append(sorts, fmt.Sprintf("%s %s", sortTag, order))
			}
		}

		if tag == "" {
			continue
		}

		value := v.Field(i).Interface()

		isZero := reflect.DeepEqual(value, reflect.Zero(field.Type).Interface())
		if isZero {
			continue
		}

		// Parse filter tag
		parts := strings.Split(tag, ";")
		column := ""
		operator := "eq"
		for _, part := range parts {
			if strings.HasPrefix(part, "column=") {
				column = strings.TrimPrefix(part, "column=")
			} else if strings.HasPrefix(part, "op=") {
				operator = strings.TrimPrefix(part, "op=")
			}
		}
		if column == "" {
			continue
		}

		condition, args := buildCondition(column, operator, value)

		groups[groupID] = append(groups[groupID], filterCondition{
			Condition: condition,
			Args:      args,
			Logic:     logicTag,
		})
	}

	// Combine groups
	var allArgs []interface{}
	var finalClauses []string
	groupIDs := getSortedKeys(groups)

	for idx, groupID := range groupIDs {
		conditions := groups[groupID]
		var subClauses []string
		for i, c := range conditions {
			if i == 0 {
				subClauses = append(subClauses, c.Condition)
			} else {
				subClauses = append(subClauses, strings.ToUpper(c.Logic)+" "+c.Condition)
			}
			allArgs = append(allArgs, c.Args...)
		}
		groupClause := "(" + strings.Join(subClauses, " ") + ")"

		if idx == 0 {
			finalClauses = append(finalClauses, groupClause)
		} else {
			logic := strings.ToUpper(groupLogics[groupID])
			if logic != "OR" {
				logic = "AND"
			}
			finalClauses = append(finalClauses, logic+" "+groupClause)
		}
	}

	if len(finalClauses) > 0 {
		db = db.Where(strings.Join(finalClauses, " "), allArgs...)
	}

	for _, sort := range sorts {
		db = db.Order(sort)
	}

	q.db = db
	return q
}

func buildCondition(column, operator string, value any) (string, []any) {
	args := []any{}

	tmpl, ok := opMap[operator]
	if !ok {
		return "1=0", args // invalid operator
	}

	condition := fmt.Sprintf(tmpl, column)

	switch operator {
	case "like":
		args = append(args, "%"+fmt.Sprint(value)+"%")

	case "in":
		switch v := value.(type) {
		case string:
			args = append(args, strings.Split(v, ","))
		case []any, []string, []uint8, []int, []int64, []int8, []int16, []int32, []float64: // allow slices
			args = append(args, v)
		default:
			return "1=0", args // invalid IN
		}

	default:
		args = append(args, value)
	}

	return condition, args
}
