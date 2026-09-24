package utils

import "testing"

func TestMatchStatusCode(t *testing.T) {
	for _, bad := range []string{"abc", "99", "600", "6xx", "200,", "2x"} {
		if _, err := MatchStatusCode(bad, 0); err == nil {
			t.Errorf("spec %q should be invalid", bad)
		}
	}
	if ok, _ := MatchStatusCode(" 200 , 3XX ", 302); !ok {
		t.Error("302 should match 3XX with surrounding spaces")
	}
}

func TestValidationErrorsUseJSONFieldNames(t *testing.T) {
	type req struct {
		EndPoint string `json:"endpoint" validate:"required"`
	}
	errs := FormatValidationErrors(MyValidator.Struct(req{}))
	if len(errs) != 1 || errs[0]["field"] != "endpoint" {
		t.Fatalf("expected field \"endpoint\", got %v", errs)
	}
}
