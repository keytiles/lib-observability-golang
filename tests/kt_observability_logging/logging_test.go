package kt_observability_logging_test

import (
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability_logging"
)

// Verifies BuildLogLabels maps supported value types (and null) to the expected Label kinds.
func TestBuildLogLabels_SupportedTypes(t *testing.T) {
	// GIVEN
	labels := map[string]any{
		"str":   "hello",
		"flag":  true,
		"num":   42,
		"flt":   3.14,
		"empty": nil,
	}

	// WHEN
	result := kt_observability_logging.BuildLogLabels(labels)
	byKey := labelsByKey(result)

	// THEN
	if len(result) != len(labels) {
		t.Fatalf("expected %d labels, got %d", len(labels), len(result))
	}

	assertLabel(t, byKey, "str", kt_logging.StringType, func(l kt_logging.Label) {
		if l.GetStringValue() != "hello" {
			t.Errorf("str: got %q", l.GetStringValue())
		}
	})
	assertLabel(t, byKey, "flag", kt_logging.BoolType, func(l kt_logging.Label) {
		if !l.GetBoolValue() {
			t.Errorf("flag: expected true")
		}
	})
	assertLabel(t, byKey, "num", kt_logging.FloatType, func(l kt_logging.Label) {
		if l.GetFloatValue() != 42 {
			t.Errorf("num: got %v", l.GetFloatValue())
		}
	})
	assertLabel(t, byKey, "flt", kt_logging.FloatType, func(l kt_logging.Label) {
		if l.GetFloatValue() != 3.14 {
			t.Errorf("flt: got %v", l.GetFloatValue())
		}
	})
	assertLabel(t, byKey, "empty", kt_logging.StringType, func(l kt_logging.Label) {
		if l.GetStringValue() != "<null>" {
			t.Errorf("empty: got %q", l.GetStringValue())
		}
	})
}

// Verifies unsupported value types become a string label with a clear fallback message.
func TestBuildLogLabels_UnsupportedType(t *testing.T) {
	// GIVEN
	labels := map[string]any{
		"arr": []string{"a", "b"},
	}

	// WHEN
	result := kt_observability_logging.BuildLogLabels(labels)
	byKey := labelsByKey(result)

	// THEN
	assertLabel(t, byKey, "arr", kt_logging.StringType, func(l kt_logging.Label) {
		if l.GetStringValue() == "" {
			t.Errorf("arr: expected non-empty fallback string, got empty")
		}
	})
}

// Verifies BuildDefaultGlobalLogLabels returns the standard global label keys.
func TestBuildDefaultGlobalLogLabels_HasStandardKeys(t *testing.T) {
	// GIVEN / WHEN
	result := kt_observability_logging.BuildDefaultGlobalLogLabels()
	byKey := labelsByKey(result)

	// THEN
	for _, key := range []string{"serviceName", "serviceVer", "host", "instId"} {
		if _, ok := byKey[key]; !ok {
			t.Errorf("missing expected global log label key %q", key)
		}
	}
}

func labelsByKey(labels []kt_logging.Label) map[string]kt_logging.Label {
	out := make(map[string]kt_logging.Label, len(labels))
	for _, l := range labels {
		out[l.GetKey()] = l
	}
	return out
}

func assertLabel(t *testing.T, byKey map[string]kt_logging.Label, key string, expectedType kt_logging.LabelType, check func(kt_logging.Label)) {
	t.Helper()
	l, ok := byKey[key]
	if !ok {
		t.Fatalf("missing label key %q", key)
	}
	if l.GetType() != expectedType {
		t.Errorf("%s: expected type %v, got %v", key, expectedType, l.GetType())
	}
	check(l)
}
