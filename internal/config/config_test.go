package config

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
)

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
	defer os.Remove(empty.Name())

	_, err = Load(empty.Name())
	if err == nil {
		t.Errorf("got != nil, want nil")
	}

	// test valid config file
	content := `{
        "ServiceURL":"https://myservice.mycompany.com:443",
        "Realm": "MYKERBEROSREALM.COM",
	"KeepAlive": 300,
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
}`
	valid, err := ioutil.TempFile("", "fw-id-agent-config-test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(valid.Name())

	if _, err := valid.Write([]byte(content)); err != nil {
		log.Fatal(err)
	}

	cfg, err := Load(valid.Name())
	if cfg == nil {
		t.Errorf("got nil, want != nil")
	}

	want := &Config{
		ServiceURL: "https://myservice.mycompany.com:443",
		Realm:      "MYKERBEROSREALM.COM",
		KeepAlive:  300,
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
