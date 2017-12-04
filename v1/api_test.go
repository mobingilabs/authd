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
	return
	u := os.Getenv("MOBINGI_USERNAME")
	p := os.Getenv("MOBINGI_PASSWORD")
	if u != "" && p != "" {
		for i := 0; i < 100; i++ {
			start := time.Now()
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
			_ = resp
			debug.Info("delta:", end.Sub(start))
		}
	}
}
