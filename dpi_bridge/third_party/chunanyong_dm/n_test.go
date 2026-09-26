package dm

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestParseDSNDecodesServiceConfigPath(t *testing.T) {
	connector := new(DmConnector).init()
	props, _, _, err := connector.parseDSN(
		"dm://user:pass@DMPYSVC:5236?svcConfPath=%2Ftmp%2Fdm+config%26test%2Fdm_svc.conf",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := props.GetString("svcConfPath", ""); got != "/tmp/dm config&test/dm_svc.conf" {
		t.Fatalf("service config path = %q", got)
	}
}

func TestServiceConfigLoadsNamedEndpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dm_svc.conf")
	if err := os.WriteFile(path, []byte("DMPYTESTSVC=127.0.0.1:5236\n"), 0600); err != nil {
		t.Fatal(err)
	}
	load(path)
	if _, ok := ServerGroupMap.Load("dmpytestsvc"); !ok {
		t.Fatal("service name was not loaded")
	}
	connector := new(DmConnector).init()
	if err := connector.mergeConfigs("dm://user:pass@DMPYTESTSVC:5236?svcConfPath=" + path); err != nil {
		t.Fatal(err)
	}
	if connector.group == nil || connector.group.name != "DMPYTESTSVC" {
		t.Fatal("service name with the bridge's default port was not resolved")
	}
	load(path + ".missing") // A missing optional file must not panic.
}

func TestConnectorAppliesMppAndReadWriteOptions(t *testing.T) {
	connector := new(DmConnector).init()
	if err := connector.mergeConfigs(
		"dm://user:pass@127.0.0.1:5236?mppLocal=true&rwSeparate=4&rwPercent=80",
	); err != nil {
		t.Fatal(err)
	}
	if !connector.mppLocal || connector.rwSeparate != RW_SEPARATE_DB_APPLY_WAIT || connector.rwPercent != 80 {
		t.Fatalf("MPP or read/write options were not applied")
	}
}
