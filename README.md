# Concept 
- [How I Write HTTP Clients After (almost) 7 Years](https://github.com/tflyons/httpx/tree/main/gophercon22)

# Core
- http.Client 구조체의 Do 메서드를 오버라이딩 하여, client-side-request를 decorating 함
- time.Ticker 와 empty struct를 활용한 rate-limiter 구현
