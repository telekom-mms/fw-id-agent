package config

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

// TestConfigGetKeepAlive tests GetKeepAlive of Config
func TestConfigGetKeepAlive(t *testing.T) {
	config := &Config{KeepAlive: 5}
	want := 5 * time.Minute
	got := config.GetKeepAlive()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigGetTimeout tests GetTimeout of Config
func TestConfigGetTimeout(t *testing.T) {
	config := &Config{Timeout: 30}
	want := 30 * time.Second
	got := config.GetTimeout()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigGetRetryTimer tests GetRetryTimer of Config
func TestConfigGetRetryTimer(t *testing.T) {
	config := &Config{RetryTimer: 15}
	want := 15 * time.Second
	got := config.GetRetryTimer()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigGetStartDelay tests GetStartDelay of Config
func TestConfigGetStartDelay(t *testing.T) {
	config := &Config{StartDelay: 20}
	want := 20 * time.Second
	got := config.GetStartDelay()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigValid tests Valid of Config
func TestConfigValid(t *testing.T) {
	// invalid
	want := false
	for _, got := range []bool{
		(*Config)(nil).Valid(),
		(&Config{}).Valid(),
		Default().Valid(),
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

// TestDefault tests Default
func TestDefault(t *testing.T) {
	want := &Config{
		KeepAlive:  5,
		Timeout:    30,
		RetryTimer: 15,
		MinUserID:  1000,
		StartDelay: 20,
	}
	got := Default()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoad tests Load
func TestLoad(t *testing.T) {
	// test invalid path
	_, err := Load("does not exist")
	if err == nil {
		t.Errorf("got != nil, want nil")
	}

	// test empty config file
	empty, err := ioutil.TempFile("", "fw-id-agent-config-test")
	if err != nil {
		log.Fatal(err)
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
	"Timeout": 30,
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
                ]
        },
	"Verbose": true,
	"MinUserID": 1000,
	"StartDelay": 20
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

		valid, err := ioutil.TempFile("", "fw-id-agent-config-test")
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			_ = os.Remove(valid.Name())
		}()

		if _, err := valid.Write([]byte(content)); err != nil {
			log.Fatal(err)
		}

		cfg, _ := Load(valid.Name())
		if cfg == nil {
			t.Errorf("got nil, want != nil")
		}

		want := &Config{
			ServiceURL: "https://myservice.mycompany.com:443",
			Realm:      "MYKERBEROSREALM.COM",
			KeepAlive:  5,
			Timeout:    30,
			RetryTimer: 15,
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
			},
			Verbose:    true,
			MinUserID:  1000,
			StartDelay: 20,
		}
		if !reflect.DeepEqual(want, cfg) {
			t.Errorf("got %v, want %v", cfg, want)
		}
	}
}
