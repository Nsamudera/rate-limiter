# Rate Limiter
Rate limiter using the sliding window algorithm. It will limit request per client based on defined parameters

## Local Setup
1. Clone the project 
```
git clone https://github.com/Nsamudera/rate-limiter.git
```

2. Go to the project directory
```
cd rate-limiter
```

3. Run the code
```
go run .
```

4. Call the API
```
curl --header "clientID: PT A" localhost:8080
```
Calling the API multiple times using a single `clientID` will eventually lead to rate limit error. Waiting for the rate limit to refresh and try again, which will have the call be successful

There is a set of mock data in main file. Any client outside this will have a default limit of 3 requests per 5 seconds. To create a specific rate limiter config, see below

5. Create rate limiter config
```
curl -X POST -H "Content-type: application/json" -H "clientID: PT A" -d '{ "limit": 5,"window": 10}' 'http://localhost:8080/config'
```

## Running Tests
1. Run all test file with `-coverprofile` option to generate coverage report
```
go test --cover ./... -coverprofile=coverage.out
```

2. View the coverage report
```
go tool cover -html=coverage.out
```

## Design Choices
The rate limiter considered was either the fixed window algorithm or the sliding window algorithm as we want to limit based on a fixed window. The sliding window algorithm was chosen, which is explained below

Fixed window consideration
* For example, every 1 hour will mean that the limit will refresh at 00:00, 01:00, 02:00, etc
    * Pros:
      *  Refresh window is uniform (easy to remember). For example, if it refreshes every hour, we know it'll refresh on 00:00, 01:00, etc. If it refreshes every 10 minutes, we know it'll refresh on 00:00, 00:10, etc 
    * Cons:
      * Will need a cron to refresh the limit
        * This can be expensive as the number of clients grow
        * This might need more effort to maintain, as we need to consider if clients are no longer active
        * Additional dependency on the cron. If there is issue with the cron, it will impact the rate limit
      * Refresh window is uniform (can't further customize per client)

Sliding window consideration
* Refresh rate is based on first request time per refresh cycle
  *  For example, given the following configuration: refresh window = 1 hour, request limit = 100
  *  At 00:00 the client made 20 requests (first request will be set to 00:00)
  *  At 00:30 the client made 30 requests (first request is still at 00:00)
  *  At 01:15 the client made 80 requests, all 80 request will succeed
      * Since 01:15 is greater than the 1 hour refresh window (01:15 - 00:00 = 1hr 15min), we will refresh the rate during this call
  * However this means that the next refresh window is between 01:15 - 02:15
  * Pros:
    *  No cron needed, which means no additional resources/dependency
    *  Only use the refresh logic when needed. If a client no longer calls the API, we don't need to worry about them
  * Cons:
    * Harder to keep track of when the limit refreshes since it is dynamic
    * May need a way to expose to client so they know when the limit will refresh


## Assumptions and Limitations
1. Different clients are identified by their id (`clientID`), which is assumed to be known already before calling the API
2. `clientID` will be sent via the header "clientID"
3. Time window is assumed to be using Seconds. This is done for the simplicity:
   * We might want to consider UX in the implementation. For example, when creating the config, we should allow the user to define the limit as 1 Hour rather than 3600 Seconds. But for the sake of simplicity we just use seconds since we can still achieve the same functionality

4. POST /config will override any previously defined config and refresh the rate limit, allowing the client to immediately call the API. This is done for simplicity. In the future, we will need to add validation and ensure that any existing configuration can only be changed via a PATCH endpoint

## How it Works
This section will focus on the flow for the "/" endpoint, where the rate limiter is used

1. Get the `clientID`and throw error if it does not exist
2. Create a new RateLimiter based on the default value
3. Check whether the rate limit has been reached
  * Lock using mutex to ensure accuracy if there are concurrent request
  * Check to see if data exist for the given client
    * If data exist, check to see whether the time elapsed between now and when the first request is made is greater than the rate limit window for the client
      * If the time elapsed is greater than the rate limit window, we refresh the request count (refreshing the rate limit) and update the first request time
    * If data does not exist, we create a new RateLimiterConfig using the default values   
  * Check whether the number of request has exceeded the limit
  * If request has not exceeded the limit, increase the request count of the client by 1
4. Return the response

## API Reference
### Testing the rate limiter

| Method | URL |
| :---:  | :-: |
| ANY    | /   |

#### Description:
This will return a response with a message containing the `clientID` (which is defined in the header `"clientID": "PT A`"). The rate limiter logic is implemented in this endpoint. When the limit has been reached, an error will occur

#### Response example
```
{
  "status": 200,
  "message": "Hello PT A"
}
```

#### Error Codes
| Error Code | Message              | Description |
| :--------- | :------------------- | :---------- |
| 400        | No clientID provided | No client ID is provided, which is needed to determine the configuration used for the rate limiter |
| 429        | Too Many Requests for `<clientID>` | Rate limit has been reached. Client will need to wait for the limit to refresh |

### Creating new rate limiter config

| Method | URL     |
| :---  | :------- |
| POST   | /config |

#### Description:
This will simply create a new configuration based on the limit and window provided in the request body. This will override any existing config. ClientID is defined in the header (`"clientID": "PT A`")

#### Request body
| Name   | type     |  Description                                            |
| :----- | :------- | :------------------------------------------------------ |
| limit  | int      | The maximum number of request allowed per refresh cycle |
| window | int      | The time (in seconds) when the rate limit is refreshed  |

#### Request body example
```
{
  "limit": 10,
  "window": 1
}
```

#### Response example
```
{
  "status": 200,
  "message": "New config created for PT A"
}
```

#### Error Codes
| Error Code | Message             | Description |
| :-------   | :------------------ | :---------- |
| 400        | No clientID provided | No client ID is provided, which is needed to know who the rate limiter config is for |
| 400        | Config data must be greater than 0 | General error to show that there is something incorrect in the request body sent. For example, the body is sent using string instead of int. There are other cases, but is generalized for current build |


## Additional Notes
Tested to see whether mutex was correctly implemented. Based on testing done, mutex is correct and it should be able to handle concurrent requests correctly:
![Screenshot](rate-limiter/screenshot/mutex_test.png)