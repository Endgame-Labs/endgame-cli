package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const graphDefaultJourney = "User invoked the Endgame CLI graph command."

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Read Endgame context graph data",
	Long: `Read Endgame context graph data with CLI-friendly flags.

Use "endgame graph list <type>" for common list reads. Use the lower-level
"endgame tools ..." commands when you need a tool not wrapped here.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args
		return cmd.Help()
	},
}

type graphListOptions struct {
	jsonArgs string
	where    []string
	limit    int
	offset   int
	orderBy  string
}

func newGraphListCommand() *cobra.Command {
	options := graphListOptions{
		limit: 50,
	}

	cmd := &cobra.Command{
		Use:   "list <type>",
		Short: "List context graph entities",
		Long: `List context graph entities by type.

Examples:
  endgame graph list account --limit 5
  endgame graph list opportunity --where is_closed=false --where 'amount>=500000' --order-by updated_at_desc
  endgame graph list opportunity --json '{"type":"opportunity","where":[{"field":"is_closed","op":"eq","value":false,"valueType":"boolean"}],"limit":3}'

The --where flag accepts field=value, field!=value, field>=value, and field<=value.
Values are inferred as booleans, numbers, or strings. Repeat --where to AND filters.
Quote expressions that contain > or < so your shell does not treat them as redirects.
Use --json as a fallback when you need the full list_graph_entities input shape.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(options.jsonArgs) != "" {
				if len(args) > 1 {
					return fmt.Errorf("accepts at most one type argument when --json is set")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("requires exactly one entity type")
			}
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("entity type must be non-empty")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			arguments, err := graphListArguments(options, args)
			if err != nil {
				return err
			}
			return executeMCPTool(cmd, "list_graph_entities", arguments)
		},
	}

	cmd.Flags().StringVar(&options.jsonArgs, "json", "", "raw JSON object for list_graph_entities; overrides shorthand flags")
	cmd.Flags().StringArrayVar(&options.where, "where", nil, "filter expression field=value, field!=value, field>=value, or field<=value; may be repeated")
	cmd.Flags().IntVar(&options.limit, "limit", 50, "maximum entities to return")
	cmd.Flags().IntVar(&options.offset, "offset", 0, "zero-based result offset")
	cmd.Flags().StringVar(&options.orderBy, "order-by", "", "row ordering: display_name_asc, updated_at_desc, or created_at_desc")
	return cmd
}

func init() {
	rootCmd.AddCommand(graphCmd)
	graphCmd.AddCommand(newGraphListCommand())
}

func graphListArguments(options graphListOptions, args []string) (map[string]any, error) {
	if strings.TrimSpace(options.jsonArgs) != "" {
		arguments, err := decodeToolArguments(options.jsonArgs)
		if err != nil {
			return nil, err
		}
		if len(args) == 1 {
			argType := strings.TrimSpace(args[0])
			jsonType, hasJSONType := arguments["type"]
			if !hasJSONType {
				arguments["type"] = argType
			} else {
				jsonTypeString, ok := jsonType.(string)
				if !ok {
					return nil, fmt.Errorf("--json type must be a string when a type argument is provided")
				}
				if jsonTypeString != argType {
					return nil, fmt.Errorf("type argument %q does not match --json type %q", argType, jsonTypeString)
				}
			}
		}
		return addGraphDefaultContext(arguments), nil
	}

	where, err := parseGraphWhere(options.where)
	if err != nil {
		return nil, err
	}
	if options.limit < 1 || options.limit > 200 {
		return nil, fmt.Errorf("--limit must be between 1 and 200")
	}
	if options.offset < 0 || options.offset > 10000 {
		return nil, fmt.Errorf("--offset must be between 0 and 10000")
	}

	arguments := map[string]any{
		"type":    strings.TrimSpace(args[0]),
		"limit":   options.limit,
		"offset":  options.offset,
		"goal":    "Read context graph entities from the Endgame CLI.",
		"task":    fmt.Sprintf("List %s entities using CLI shorthand flags.", strings.TrimSpace(args[0])),
		"journey": graphDefaultJourney,
	}
	if len(where) > 0 {
		arguments["where"] = where
	}
	if orderBy := strings.TrimSpace(options.orderBy); orderBy != "" {
		switch orderBy {
		case "display_name_asc", "updated_at_desc", "created_at_desc":
			arguments["orderBy"] = orderBy
		default:
			return nil, fmt.Errorf("invalid --order-by %q", orderBy)
		}
	}

	return arguments, nil
}

func addGraphDefaultContext(arguments map[string]any) map[string]any {
	if _, ok := arguments["goal"]; !ok {
		arguments["goal"] = "Read context graph entities from the Endgame CLI."
	}
	if _, ok := arguments["task"]; !ok {
		arguments["task"] = "List context graph entities using raw JSON input."
	}
	if _, ok := arguments["journey"]; !ok {
		arguments["journey"] = graphDefaultJourney
	}
	return arguments
}

func parseGraphWhere(expressions []string) ([]map[string]any, error) {
	if len(expressions) == 0 {
		return nil, nil
	}
	if len(expressions) > 10 {
		return nil, fmt.Errorf("at most 10 --where filters are allowed")
	}

	filters := make([]map[string]any, 0, len(expressions))
	for _, expression := range expressions {
		filter, err := parseGraphWhereExpression(expression)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func parseGraphWhereExpression(expression string) (map[string]any, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, fmt.Errorf("--where expression must be non-empty")
	}

	for _, candidate := range []struct {
		token string
		op    string
	}{
		{token: "!=", op: "neq"},
		{token: ">=", op: "gte"},
		{token: "<=", op: "lte"},
		{token: "=", op: "eq"},
	} {
		if field, rawValue, ok := strings.Cut(expression, candidate.token); ok {
			field = strings.TrimSpace(field)
			rawValue = strings.TrimSpace(rawValue)
			if field == "" {
				return nil, fmt.Errorf("invalid --where %q: field must be non-empty", expression)
			}
			if rawValue == "" {
				return nil, fmt.Errorf("invalid --where %q: value must be non-empty", expression)
			}
			value, valueType := parseGraphWhereValue(rawValue)
			return map[string]any{
				"field":     field,
				"op":        candidate.op,
				"value":     value,
				"valueType": valueType,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid --where %q: expected field=value, field!=value, field>=value, or field<=value", expression)
}

func parseGraphWhereValue(rawValue string) (any, string) {
	value := strings.Trim(rawValue, `"'`)
	switch strings.ToLower(value) {
	case "true":
		return true, "boolean"
	case "false":
		return false, "boolean"
	}
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		return number, "number"
	}
	return value, "string"
}
