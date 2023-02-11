# Concept 
- [How I Write HTTP Clients After (almost) 7 Years](https://github.com/tflyons/httpx/tree/main/gophercon22)

# Core
- http.Client 구조체의 Do 메서드를 오버라이딩 하여, client-side-request를 decorating 함
- time.Ticker 와 empty struct를 활용한 rate-limiter 구현

# Decorator
HTTPClient 인터페이스는 Do 메서드를 추상화 한다. 
HTTPClient 를 구현한 DoerFunc라는 함수타입은 Do 메서드를 구현한다. 
`DoerFunc` 는 함수이고, 그 함수의 `Do` 메서드는 인자로 입력된 `req *http.Request`을 받는다. 
이 `DoerFunc`의 `Do` 메서드는 req를 매개변수로 하여 DoerFunc를 실행한다.
결국 `DoerFunc`를 반환하는 두개의 데코레이터 함수 `DecorateCustomHeader`과 `DecorateRateLimit`은 호출되는 순서에 따라 다음의 방식으로 작동한다.
1. `http.DefaultClient`이 선언 된다. `http.DefaultClient` 도 `Do` 메서드를 구현하고 있으니 `HTTPClient`의 구현체이다. 
2. `DecorateRateLimit`로 위에 선언된 `http.DefaultClient`를 __Decorating__ 한 뒤 그것을 다시 client에 할당한다.
3. 여기서 __Decorating__ 이라는 행위는 `DecorateRateLimit` 함수에 정의된 행동을 수행한 뒤 client의 `Do` 메서드를 실행하는 또다른 함수를 만드는 것을 뜻한다.
    ```go
    var client HTTPClient
    client = http.DefaultClient
    client = DecorateRateLimit(client, ch)
    // 여기서 client는 DecorateRateLimit의 동작을 한 뒤 http.DefaultClient를 통해 Do()를 호출하는 함수를 반환한다. 이 함수의 Do 메서드는 이 함수를 실행시킨다.
    ```
   `http.DefaultClient`와, `DecorateCustomHeader`가 반환한 `DoerFunc` 모두 `HTTPClient` 인터페이스를 구현하고있기 때문에 이것이 가능하다.

4. `DecorateCustomHeader`로 위에 선언된 `DecorateRateLimit`로 __Decorating__ 된 `http.DefaultClient`로 `Do()` 하는  함수(`DoerFunc`)를 또 한번 __Decorating__ 한다.
   ```go
   client = DecorateRateLimit(client, ch)
   client = DecorateCustomHeader(client, "hello")
   // 여기서 client는 DecorateCustomHeader의 동작을 한뒤 DecorateRateLimit을 통해 반환된 함수 
   // (DecorateRateLimit의 동작을 한 뒤 http.DefaultClient를 통해 Do()를 호출하는 함수)
   // 의 Do() 메서드를 호출하는 함수를 반환한다. 
   ```
5. 그렇다면 `client` 는 RateLimit 로직이 포함된 동작을 한 뒤 `DefaultClient`의 `Do(req)`를 하는 함수를 실행시키기 전에, CustomHeader를 추가하는 함수인 것이다.


이렇게 함수의 동작을 꾸미는 함수를 반환하는 방식으로 여러 동작을 하는 함수가 체이닝되어 실행된다.

# RateLimiting
`rateLimiter` 함수는 time.Duration 타입의 t와, int 타입의 reqPerTime을 인자로 받는다.
t는 제한시간. reqPerTime은 그 제한 시간동안 최대 몇번의 요청이 가능한지를 뜻한다.

`rateLimiter`의 동작방식은 다음과 같다.
1. 채널을 만들고 정해진 시간마다 정해진 수의 빈 구조체를 넣는다.
2. `DecorateRateLimit` 데코레이터로 감싼 요청은 요청하기 전 위에서 만든 채널에서 빈구조체를 꺼낸다. 채널에 빈구조체가 있다면 꺼내진 뒤 이후 로직 (`client.Do(req)`)이 수행된다.
3. 만약 채널에서 빈구조체가 나오지 않는다면 계속 대기한다. req의 컨텍스트에 Done 시그널이 온다면 대기하지 않고 리퀘스트를 멈춘다.

