package fetcher_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
)

var _ = Describe("doRequestWithRateLimit", func() {
	var originalClient *http.Client

	BeforeEach(func() {
		originalClient = fetcher.HttpClient
	})

	AfterEach(func() {
		fetcher.HttpClient = originalClient
	})

	It("skal håndtere rate limit og retry riktig", func() {
		callCount := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// Simuler at vi har truffet rate limit
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprint(time.Now().Add(50*time.Millisecond).Unix()))
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintln(w, `{}`)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"message": "ok"}`)
		}))
		defer ts.Close()

		fetcher.HttpClient = ts.Client()

		var result struct{ Message string }
		err := fetcher.DoRequestWithRateLimit("GET", ts.URL, "dummy-token", nil, &result)
		Expect(err).To(BeNil())
		Expect(result.Message).To(Equal("ok"))
		Expect(callCount).To(BeNumerically(">=", 2))
	})

	It("skal sette Content-Type header for POST", func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal("POST"))
			Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"message": "ok"}`)
		}))
		defer ts.Close()

		fetcher.HttpClient = ts.Client()

		var result struct{ Message string }
		err := fetcher.DoRequestWithRateLimit("POST", ts.URL, "token", []byte(`{}`), &result)
		Expect(err).To(BeNil())
		Expect(result.Message).To(Equal("ok"))
	})

	It("skal feile på ugyldig URL (DNS-feil)", func() {
		var result any
		err := fetcher.DoRequestWithRateLimit("GET", "http://invalid-url", "token", nil, &result)
		Expect(err).To(HaveOccurred())
	})

	It("skal feile på ugyldig request-format (syntax)", func() {
		var result any
		err := fetcher.DoRequestWithRateLimit("GET", ":", "token", nil, &result)
		Expect(err).To(HaveOccurred())
	})

	It("skal returnere feil hvis GitHub API svarer med status != 2xx", func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden) // 403
			_, _ = fmt.Fprint(w, `{"message":"access denied"}`)
		}))
		defer ts.Close()

		fetcher.HttpClient = ts.Client()

		var result any
		err := fetcher.DoRequestWithRateLimit("GET", ts.URL, "token", nil, &result)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("GitHub API-feil"))
		Expect(err.Error()).To(ContainSubstring("403"))
		Expect(err.Error()).To(ContainSubstring("access denied"))
	})
})
