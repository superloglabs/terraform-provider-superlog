package provider

import "testing"

func TestSameWidgetConfig(t *testing.T) {
	cases := []struct {
		a, b string
		same bool
	}{
		{`{"source":"logs","groupBy":"service.name"}`, `{"groupBy":"service.name","source":"logs"}`, true},
		{`{"markdown":"hi"}`, `{"filter":{},"markdown":"hi"}`, true},
		{`{"source":"logs","limit":10}`, `{"source":"logs","limit":10.0}`, true},
		{`{"source":"logs"}`, `{"source":"traces"}`, false},
		{`{"filter":{"resourceAttrs":[{"key":"a","value":"b"}]}}`, `{"filter":{"resourceAttrs":[]}}`, false},
		{`not json`, `not json`, true},
	}
	for _, c := range cases {
		if got := sameWidgetConfig(c.a, c.b); got != c.same {
			t.Errorf("sameWidgetConfig(%s, %s) = %v, want %v", c.a, c.b, got, c.same)
		}
	}
}

func TestCompactJSONSortsKeys(t *testing.T) {
	if got := compactJSON([]byte(`{ "b": 1, "a": {"d": 2, "c": 3}, "filter": {}, "x": null }`)); got != `{"a":{"c":3,"d":2},"b":1}` {
		t.Errorf("compactJSON = %s", got)
	}
}
