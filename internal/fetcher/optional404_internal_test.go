package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("doRequestWithRateLimitAndOptional404", func() {
	var originalClient *http.Client
	var originalBackoff func(int) time.Duration

	BeforeEach(func() {
		originalClient = HttpClient
		originalBackoff = RetryBackoff
		RetryBackoff = func(_ int) time.Duration { return time.Millisecond }
	})

	AfterEach(func() {
		HttpClient = originalClient
		RetryBackoff = originalBackoff
	})

	It("skal returnere nil ved 404 (ressurs ikke tilgjengelig)", func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintln(w, `{"message":"not found"}`)
		}))
		defer ts.Close()

		HttpClient = ts.Client()
		ctx := context.Background()
		var result any
		err := doRequestWithRateLimitAndOptional404(ctx, "GET", ts.URL, "token", nil, &result)

		Expect(err).ToNot(HaveOccurred())
	})

	It("skal returnere feil ved andre ikke-2xx statuskoder", func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprintln(w, `{"message":"forbidden"}`)
		}))
		defer ts.Close()

		HttpClient = ts.Client()
		ctx := context.Background()
		var result any
		err := doRequestWithRateLimitAndOptional404(ctx, "GET", ts.URL, "token", nil, &result)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("403"))
	})
})
