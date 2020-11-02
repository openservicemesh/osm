package injector

import (
	"bytes"
	"net/http"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadTimeout(t *testing.T) {
	assert := assert.New(t)

	expectedResults := map[string]bool{
		"/mutate-pod-creation?timeout=30s":            true,
		"/mutate-pod-creation?timeout=20h":            true,
		"/mutate-pod-creation?timeout=s":              false,
		"/mutate-pod-creation?":                       false,
		"/mutate-pod-creation?timeout=20&timeout=30m": false,
		"randomString":                                false,
	}

	for url, expectedRes := range expectedResults {
		req, err := http.NewRequest(http.MethodPost, url, http.NoBody)
		assert.NoError(err)

		_, err = readTimeout(req)
		if expectedRes {
			assert.NoError(err)
		} else {
			assert.Error(err)
		}
	}
}

func TestDeferredWebhookLogging(t *testing.T) {
	assert := assert.New(t)

	// Redirect zerolog output temporarily to trap the log message
	logsave := log
	var b bytes.Buffer
	log = log.Output(&b)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		timeout, _ := time.ParseDuration("30s")
		defer webhookTimeTrack(time.Now(), timeout)

		time.Sleep(100 * time.Millisecond)
	}()
	wg.Wait()

	log = logsave
	match, _ := regexp.MatchString("Mutate Webhook took .* to execute (.* of it's timeout, 30s)", b.String())
	assert.Equal(true, match)
}
