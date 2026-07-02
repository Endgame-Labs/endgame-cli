package cmd

import "testing"

func TestParseGraphWhereExpressionBoolean(t *testing.T) {
	filter, err := parseGraphWhereExpression("is_closed=false")
	if err != nil {
		t.Fatalf("parseGraphWhereExpression returned error: %v", err)
	}
	if got := filter["field"]; got != "is_closed" {
		t.Fatalf("field = %v, want is_closed", got)
	}
	if got := filter["op"]; got != "eq" {
		t.Fatalf("op = %v, want eq", got)
	}
	if got := filter["value"]; got != false {
		t.Fatalf("value = %v, want false", got)
	}
	if got := filter["valueType"]; got != "boolean" {
		t.Fatalf("valueType = %v, want boolean", got)
	}
}

func TestParseGraphWhereExpressionNumber(t *testing.T) {
	filter, err := parseGraphWhereExpression("amount>=500000")
	if err != nil {
		t.Fatalf("parseGraphWhereExpression returned error: %v", err)
	}
	if got := filter["field"]; got != "amount" {
		t.Fatalf("field = %v, want amount", got)
	}
	if got := filter["op"]; got != "gte" {
		t.Fatalf("op = %v, want gte", got)
	}
	if got := filter["value"]; got != float64(500000) {
		t.Fatalf("value = %v, want 500000", got)
	}
	if got := filter["valueType"]; got != "number" {
		t.Fatalf("valueType = %v, want number", got)
	}
}

func TestParseGraphWhereExpressionStringNotEqual(t *testing.T) {
	filter, err := parseGraphWhereExpression("stage_name!=Closed Won")
	if err != nil {
		t.Fatalf("parseGraphWhereExpression returned error: %v", err)
	}
	if got := filter["field"]; got != "stage_name" {
		t.Fatalf("field = %v, want stage_name", got)
	}
	if got := filter["op"]; got != "neq" {
		t.Fatalf("op = %v, want neq", got)
	}
	if got := filter["value"]; got != "Closed Won" {
		t.Fatalf("value = %v, want Closed Won", got)
	}
	if got := filter["valueType"]; got != "string" {
		t.Fatalf("valueType = %v, want string", got)
	}
}

func TestParseGraphWhereExpressionRejectsInvalid(t *testing.T) {
	if _, err := parseGraphWhereExpression("amount>500000"); err == nil {
		t.Fatal("parseGraphWhereExpression succeeded, want error")
	}
}

func TestGraphListArgumentsFromFlags(t *testing.T) {
	arguments, err := graphListArguments(graphListOptions{
		where:   []string{"is_closed=false", "amount>=500000"},
		limit:   3,
		orderBy: "updated_at_desc",
	}, []string{"opportunity"})
	if err != nil {
		t.Fatalf("graphListArguments returned error: %v", err)
	}
	if got := arguments["type"]; got != "opportunity" {
		t.Fatalf("type = %v, want opportunity", got)
	}
	if got := arguments["limit"]; got != 3 {
		t.Fatalf("limit = %v, want 3", got)
	}
	if got := arguments["orderBy"]; got != "updated_at_desc" {
		t.Fatalf("orderBy = %v, want updated_at_desc", got)
	}
	where, ok := arguments["where"].([]map[string]any)
	if !ok {
		t.Fatalf("where has type %T, want []map[string]any", arguments["where"])
	}
	if len(where) != 2 {
		t.Fatalf("len(where) = %d, want 2", len(where))
	}
}

func TestGraphListArgumentsJSONFallbackAddsContext(t *testing.T) {
	arguments, err := graphListArguments(graphListOptions{
		jsonArgs: `{"type":"account","limit":2}`,
		limit:    50,
	}, nil)
	if err != nil {
		t.Fatalf("graphListArguments returned error: %v", err)
	}
	if got := arguments["type"]; got != "account" {
		t.Fatalf("type = %v, want account", got)
	}
	if _, ok := arguments["goal"]; !ok {
		t.Fatal("goal was not added")
	}
	if _, ok := arguments["task"]; !ok {
		t.Fatal("task was not added")
	}
	if _, ok := arguments["journey"]; !ok {
		t.Fatal("journey was not added")
	}
}

func TestGraphListArgumentsJSONFallbackUsesTypeArgumentWhenMissing(t *testing.T) {
	arguments, err := graphListArguments(graphListOptions{
		jsonArgs: `{"limit":2}`,
		limit:    50,
	}, []string{"account"})
	if err != nil {
		t.Fatalf("graphListArguments returned error: %v", err)
	}
	if got := arguments["type"]; got != "account" {
		t.Fatalf("type = %v, want account", got)
	}
}

func TestGraphListArgumentsJSONFallbackRejectsConflictingTypeArgument(t *testing.T) {
	_, err := graphListArguments(graphListOptions{
		jsonArgs: `{"type":"opportunity","limit":2}`,
		limit:    50,
	}, []string{"account"})
	if err == nil {
		t.Fatal("graphListArguments succeeded, want error")
	}
}
