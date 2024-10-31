package validator

import (
	"log"
	"rate_limiter/config"
	"sync"
	"time"
)

type RateLimiterData struct {
	Requests         int
	Limit            int
	Window           time.Duration
	FirstRequestTime time.Time
}

type CreateData struct {
	Limit  int
	Window int
}

type RateLimiter struct {
	RateLimiterData
	Mutex sync.Mutex
}

type RateLimitCheckResult struct {
	Status bool
	Data   RateLimiterData
}

func ValidateClientID(clientID string) bool {
	return clientID != ""
}

func (rl *RateLimiter) ValidateRequestLimit(clientID string, currentTime time.Time, data map[string]RateLimiterData) RateLimitCheckResult {
	rl.Mutex.Lock()
	defer rl.Mutex.Unlock()

	// Use config or default value depending if client data exist
	// For future improvement, use database/Redis for data storage
	requests := config.DefaultRequest
	limit := config.DefaultLimit
	window := config.DefaultWindow

	clientData, ok := data[clientID]
	if ok {
		log.Printf("Starting Request: %v / %v", clientData.Requests, clientData.Limit)
		log.Printf("currentTime: %v\n", currentTime)
		log.Printf("firstRequestTime: %v\n", clientData.FirstRequestTime)
		log.Printf("difference: %v\n", currentTime.Sub(clientData.FirstRequestTime))
		log.Printf("window: %v\n", clientData.Window)

		// If first request has already exceeded the time window, refresh the request to 0
		if currentTime.Sub(clientData.FirstRequestTime) > clientData.Window {
			clientData.Requests = 0
			clientData.FirstRequestTime = currentTime
			data[clientID] = clientData
			log.Println("Refreshing the rate limit")
		}

		requests = clientData.Requests
		limit = clientData.Limit
	} else {
		// Create new config so we can keep track of future requests
		data[clientID] = RateLimiterData{requests, limit, window, currentTime}
	}

	// Check to see if client has reached the limit
	if requests >= limit {
		log.Println("Limit reached, process will not continue")
		return RateLimitCheckResult{
			false, data[clientID],
		}
	}

	// Add to the request count
	if clientData, ok := data[clientID]; ok {
		clientData.Requests++
		data[clientID] = clientData
	}
	log.Printf("Ending Request: %v / %v", data[clientID].Requests, data[clientID].Limit)

	return RateLimitCheckResult{
		true, data[clientID],
	}
}

func ValidateConfig(config CreateData) bool {
	if config.Limit <= 0 || config.Window <= 0 {
		return false
	}
	return true
}
