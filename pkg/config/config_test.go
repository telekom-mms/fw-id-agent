package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/telekom-mms/tnd/pkg/tnd"
)

// TestConfigCopy tests Copy of Config.
func TestConfigCopy(t *testing.T) {
	// test nil
	var want *Config
	if want.Copy() != nil {
		t.Errorf("copy of nil should be nil")
	}

	// test defaults
	want = Default()
	got := want.Copy()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with TND server
	o := Default()
	o.TND.HTTPSServers = []TNDHTTPSConfig{{"example", "hash"}}
	n := o.Copy()
	n.TND.HTTPSServers[0].URL = "example2"
	if reflect.DeepEqual(o, n) {
		t.Errorf("%v and %v should not be equal after change", o, n)
	}
}

// TestConfigGetKeepAlive tests GetKeepAlive of Config.
func TestConfigGetKeepAlive(t *testing.T) {
	config := &Config{KeepAlive: 5}
	want := 5 * time.Minute
	got := config.GetKeepAlive()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigGetLoginTimeout tests GetLoginTimeout of Config.
func TestConfigGetLoginTimeout(t *testing.T) {
	config := &Config{LoginTimeout: 15}
	want := 15 * time.Second
	got := config.GetLoginTimeout()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigGetLogoutTimeout tests GetLoginTimeout of Config.
func TestConfigGetLogoutTimeout(t *testing.T) {
	config := &Config{LogoutTimeout: 5}
	want := 5 * time.Second
	got := config.GetLogoutTimeout()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigGetRetryTimer tests GetRetryTimer of Config.
func TestConfigGetRetryTimer(t *testing.T) {
	config := &Config{RetryTimer: 15}
	want := 15 * time.Second
	got := config.GetRetryTimer()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigGetStartDelay tests GetStartDelay of Config.
func TestConfigGetStartDelay(t *testing.T) {
	config := &Config{StartDelay: 20}
	want := 20 * time.Second
	got := config.GetStartDelay()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigValid tests Valid of Config.
func TestConfigValid(t *testing.T) {
	// invalid
	want := false
	for _, got := range []bool{
		(*Config)(nil).Valid(),
		(&Config{}).Valid(),
		Default().Valid(),
		(&Config{
			ServiceURL: "example.com",
			Realm:      "test",
		}).Valid(),
		(&Config{
			ServiceURL: "example.com",
			Realm:      "test",
			TND:        TNDConfig{HTTPSServers: []TNDHTTPSConfig{{URL: "", Hash: ""}}},
		}).Valid(),
	} {
		if got != want {
			t.Errorf("got %t, want %t", got, want)
		}
	}

	// valid
	valid := Default()
	valid.ServiceURL = "https://testService.com:443"
	valid.Realm = "TESTKERBEROSREALM.COM"
	valid.TND.HTTPSServers = append(valid.TND.HTTPSServers, TNDHTTPSConfig{
		URL:  "https://tnd.testcompany.com:443",
		Hash: "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
	})

	want = true
	got := valid.Valid()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestConfigString tests String of Config.
func TestConfigString(t *testing.T) {
	// default config
	c := Default()
	if c.String() == "" {
		t.Errorf("string should not be empty: %s", c.String())
	}

	// nil
	c = nil
	if c.String() != "null" {
		t.Errorf("string should be null: %s", c.String())
	}

}

// TestDefault tests Default.
func TestDefault(t *testing.T) {
	want := &Config{
		KeepAlive:     5,
		LoginTimeout:  15,
		LogoutTimeout: 5,
		RetryTimer:    15,
		TND:           TNDConfig{Config: tnd.NewConfig()},
		StartDelay:    0,
		Notifications: true,
	}
	got := Default()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewFromJSON tests NewFromJSON.
func TestNewFromJSON(t *testing.T) {
	// test invalid
	if _, err := NewFromJSON([]byte("")); err == nil {
		t.Errorf("invalid json should return error")
	}

	// test valid
	want := Default()
	b, err := want.JSON()
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewFromJSON(b)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoad tests Load.
func TestLoad(t *testing.T) {
	// test invalid path
	_, err := Load("does not exist")
	if err == nil {
		t.Errorf("got != nil, want nil")
	}

	// test empty config file
	empty, err := os.CreateTemp("", "fw-id-agent-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(empty.Name())
	}()

	_, err = Load(empty.Name())
	if err == nil {
		t.Errorf("got != nil, want nil")
	}

	// test valid config file
	// - complete config
	// - partial config with defaults
	for _, content := range []string{
		`{
        "ServiceURL":"https://myservice.mycompany.com:443",
        "Realm": "MYKERBEROSREALM.COM",
	"KeepAlive": 5,
	"LoginTimeout": 15,
	"LogoutTimeout": 5,
	"RetryTimer": 15,
        "TND":{
                "HTTPSServers":[
                        {
                                "URL":"https://tnd1.mycompany.com:443",
                                "Hash":"ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
                        },
                        {
                                "URL":"https://tnd2.mycompany.com:443",
                                "Hash":"ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
                        }
                ],
		"Config":{
                                "WaitCheck": 1000000000,
                                "HTTPSTimeout": 5000000000,
                                "UntrustedTimer": 30000000000,
                                "TrustedTimer": 60000000000
		}
        },
	"Verbose": true,
	"StartDelay": 0,
	"Notifications": true
}`,
		`{
        "ServiceURL":"https://myservice.mycompany.com:443",
        "Realm": "MYKERBEROSREALM.COM",
        "TND":{
                "HTTPSServers":[
                        {
                                "URL":"https://tnd1.mycompany.com:443",
                                "Hash":"ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
                        },
                        {
                                "URL":"https://tnd2.mycompany.com:443",
                                "Hash":"ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
                        }
                ]
        },
	"Verbose": true
}`,
	} {

		valid, err := os.CreateTemp("", "fw-id-agent-config-test")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Remove(valid.Name())
		}()

		if _, err := valid.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}

		cfg, _ := Load(valid.Name())
		if cfg == nil {
			t.Errorf("got nil, want != nil")
			return
		}

		want := &Config{
			ServiceURL:    "https://myservice.mycompany.com:443",
			Realm:         "MYKERBEROSREALM.COM",
			KeepAlive:     5,
			LoginTimeout:  15,
			LogoutTimeout: 5,
			RetryTimer:    15,
			TND: TNDConfig{
				[]TNDHTTPSConfig{
					{
						URL:  "https://tnd1.mycompany.com:443",
						Hash: "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
					},
					{
						URL:  "https://tnd2.mycompany.com:443",
						Hash: "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
					},
				},
				tnd.NewConfig(),
			},
			Verbose:       true,
			StartDelay:    0,
			Notifications: true,
		}
		if !reflect.DeepEqual(want.TND.Config, cfg.TND.Config) {
			t.Errorf("got %v, want %v", cfg.TND.Config, want.TND.Config)
		}
		if !reflect.DeepEqual(want, cfg) {
			t.Errorf("got %v, want %v", cfg, want)
		}
	}
}
