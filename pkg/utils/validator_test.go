package utils

import "testing"

func TestValidationErrorsUseJSONFieldNames(t *testing.T) {
	type req struct {
		EndPoint string `json:"endpoint" validate:"required"`
	}
	errs := FormatValidationErrors(MyValidator.Struct(req{}))
	if len(errs) != 1 || errs[0]["field"] != "endpoint" {
		t.Fatalf("expected field \"endpoint\", got %v", errs)
	}
}
