package dm

import "testing"

func TestParseDSNDecodesApplicationName(t *testing.T) {
	connector := new(DmConnector).init()
	props, _, _, err := connector.parseDSN(
		"dm://user:pass@127.0.0.1:5236?appName=dmpython+%26+matrix%2B1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := props.GetString(AppNameKey, ""); got != "dmpython & matrix+1" {
		t.Fatalf("application name = %q", got)
	}
}
