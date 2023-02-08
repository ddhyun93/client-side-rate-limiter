package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type DoerFunc func(*http.Request) (*http.Response, error)

func (f DoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func DecorateCustomHeader(client HTTPClient, value string) DoerFunc {
	return func(req *http.Request) (*http.Response, error) {
		req.Header.Set("X-MY-CUSTOM-HEADER", value)
		return client.Do(req)
	}
}

func DecorateRateLimit(client HTTPClient, ch chan struct{}) DoerFunc {
	return func(req *http.Request) (*http.Response, error) {
		select {
		case <-ch:
		case <-req.Context().Done():
			log.Println("done")
			return nil, req.Context().Err()
		}
		return client.Do(req)
	}
}

func rateLimiter(t time.Duration, reqPerTime int) chan struct{} {
	ticker := time.NewTicker(t)
	ch := make(chan struct{}, reqPerTime)

	go func() {
		for i := 0; i < reqPerTime; i++ {
			ch <- struct{}{}
		}

		for range ticker.C {
			for i := 0; i < 2; i++ {
				select {
				case ch <- struct{}{}:
				default:
					break
				}
			}
		}
	}()
	return ch
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	runServer(ctx, ":18512")
	defer cancel()

	t := time.Second * 1
	reqPerTime := 2

	ch := rateLimiter(t, reqPerTime)

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:18512/asd", nil)
	reqCtx, _ := context.WithTimeout(context.Background(), time.Second*10)
	req = req.WithContext(reqCtx)

	var client HTTPClient
	client = http.DefaultClient
	client = DecorateRateLimit(client, ch)
	client = DecorateCustomHeader(client, "hello")
	_, _ = client.Do(req)
	_, _ = client.Do(req)
	_, _ = client.Do(req)

	var client2 HTTPClient
	client2 = http.DefaultClient
	client2 = DecorateRateLimit(client2, ch)
	client2 = DecorateCustomHeader(client2, "hihihihi")
	_, _ = client2.Do(req)
	_, _ = client2.Do(req)
	_, _ = client2.Do(req)
}

func runServer(ctx context.Context, addr string) {
	mux := http.NewServeMux()
	mux.Handle("/asd", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Header.Get("X-MY-CUSTOM-HEADER"))
	}))

	srv := &http.Server{
		Addr:        addr,
		Handler:     mux,
		IdleTimeout: time.Minute,
	}

	go func() {
		go func() {
			<-ctx.Done()
			log.Println("Server stopped")
			_ = srv.Close()
		}()

		log.Printf("Listening on %s …\n", srv.Addr)
		_ = srv.ListenAndServe()

	}()

}
