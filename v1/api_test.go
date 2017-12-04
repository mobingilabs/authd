package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

func TestValidatePerf(t *testing.T) {
	// return
	u := os.Getenv("MOBINGI_USERNAME")
	p := os.Getenv("MOBINGI_PASSWORD")
	if u != "" && p != "" {
		for i := 0; i < 100; i++ {
			start := time.Now()
			// r, err := http.NewRequest(http.MethodPost, "http://54.199.197.6:30100/api/v1/token")
			c := creds{
				Username: u,
				Password: p,
			}

			payload, _ := json.Marshal(c)
			r, _ := http.NewRequest(http.MethodPost, "http://localhost:8080/api/v1/token", bytes.NewBuffer(payload))
			r.Header.Add("Content-Type", "application/json")

			client := http.Client{}
			resp, err := client.Do(r)
			if err != nil {
				t.Error(err)
				return
			}

			end := time.Now()
			debug.Info(resp)
			debug.Info("delta:", end.Sub(start))
		}
	}
}
