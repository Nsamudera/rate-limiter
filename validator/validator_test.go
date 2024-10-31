package validator

import (
	"fmt"
	"io"
	"log"
	"rate_limiter/config"
	"sync"
	"testing"
	"time"
)

type Body struct {
	Status  int
	Message string
}

func TestValidateClientIDFail(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Run("no clientID provided", func(t *testing.T) {
		expectedStatus := false
		response := ValidateClientID("")

		if response {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response)
		}
	})
}

func TestValidateClientIDSuccess(t *testing.T) {
	t.Run("clientID provided", func(t *testing.T) {
		expectedStatus := true
		response := ValidateClientID("PT A")

		if !response {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response)
		}
	})
}

func TestValidateCreateDataFail(t *testing.T) {
	// log.SetOutput(io.Discard)
	t.Run("both properties are 0", func(t *testing.T) {
		expectedStatus := false
		config := CreateData{0, 0}
		response := ValidateConfig(config)

		if response {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response)
		}
	})

	t.Run("one property is 0", func(t *testing.T) {
		expectedStatus := false
		config := CreateData{1, 0}
		response := ValidateConfig(config)

		if response {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response)
		}
	})
}

func TestValidateCreateDataSuccess(t *testing.T) {
	t.Run("[SUCCESS] both properties are greater than 0", func(t *testing.T) {
		expectedStatus := true
		config := CreateData{2, 5}
		response := ValidateConfig(config)

		if !response {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response)
		}
	})
}

var mockedRateLimiterData = map[string]RateLimiterData{
	"PT A":         {Requests: 2, Limit: 3, Window: 5 * time.Second, FirstRequestTime: time.Now()},
	"PT Limit Max": {Requests: 3, Limit: 3, Window: 3 * time.Second, FirstRequestTime: time.Now()},
}

func TestValidateRequestLimitFail(t *testing.T) {
	t.Run("Rate limit reached", func(t *testing.T) {
		currentTime := time.Now()
		clientId := "PT Limit Max"
		expectedStatus := false
		expectedData := mockedRateLimiterData[clientId]
		RateLimiterData := RateLimiterData{
			Requests:         config.DefaultRequest,
			Limit:            config.DefaultLimit,
			Window:           config.DefaultWindow,
			FirstRequestTime: currentTime,
		}

		rateLimiter := RateLimiter{
			RateLimiterData: RateLimiterData,
			Mutex:           sync.Mutex{},
		}

		response := rateLimiter.ValidateRequestLimit(clientId, currentTime, mockedRateLimiterData)

		fmt.Println(response.Data == mockedRateLimiterData[clientId])

		if response.Status {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response.Status)
		}

		if response.Data != expectedData {
			t.Errorf("Expected data to be unchanged %v, but got %v", expectedData, response.Data)
		}
	})
}

func TestValidateRequestLimitSuccess(t *testing.T) {
	t.Run("New client", func(t *testing.T) {
		currentTime := time.Now()
		clientId := "PT New"
		expectedStatus := true
		expectedData := map[string]RateLimiterData{
			clientId: {
				Requests: config.DefaultRequest + 1, Limit: config.DefaultLimit,
				Window: config.DefaultWindow, FirstRequestTime: currentTime},
		}
		RateLimiterData := RateLimiterData{
			Requests:         config.DefaultRequest,
			Limit:            config.DefaultLimit,
			Window:           config.DefaultWindow,
			FirstRequestTime: currentTime,
		}
		rateLimiter := RateLimiter{
			RateLimiterData: RateLimiterData,
			Mutex:           sync.Mutex{},
		}

		response := rateLimiter.ValidateRequestLimit(clientId, currentTime, mockedRateLimiterData)

		if !response.Status {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response.Status)
		}

		if response.Data != expectedData[clientId] {
			t.Errorf("Expected data to be updated %v, but got %v", expectedData[clientId], response.Data)
		}
	})

	t.Run("Existing client - has not passed refresh window", func(t *testing.T) {
		currentTime := time.Now()
		clientId := "PT A"
		expectedStatus := true
		expectedData := map[string]RateLimiterData{
			clientId: {
				Requests:         mockedRateLimiterData[clientId].Requests + 1,
				Limit:            mockedRateLimiterData[clientId].Limit,
				Window:           mockedRateLimiterData[clientId].Window,
				FirstRequestTime: mockedRateLimiterData[clientId].FirstRequestTime,
			},
		}
		RateLimiterData := RateLimiterData{
			Requests:         config.DefaultRequest,
			Limit:            config.DefaultLimit,
			Window:           config.DefaultWindow,
			FirstRequestTime: currentTime,
		}
		rateLimiter := RateLimiter{
			RateLimiterData: RateLimiterData,
			Mutex:           sync.Mutex{},
		}

		response := rateLimiter.ValidateRequestLimit(clientId, currentTime, mockedRateLimiterData)

		if !response.Status {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response.Status)
		}

		if response.Data != expectedData[clientId] {
			t.Errorf("Expected data to be updated %v, but got %v", expectedData[clientId], response.Data)
		}
	})

	t.Run("Existing client - has passed refresh window", func(t *testing.T) {
		currentTime := time.Now().AddDate(1, 0, 0)
		clientId := "PT A"
		expectedStatus := true
		expectedData := map[string]RateLimiterData{
			clientId: {
				Requests:         1,
				Limit:            mockedRateLimiterData[clientId].Limit,
				Window:           mockedRateLimiterData[clientId].Window,
				FirstRequestTime: currentTime,
			},
		}
		RateLimiterData := RateLimiterData{
			Requests:         config.DefaultRequest,
			Limit:            config.DefaultLimit,
			Window:           config.DefaultWindow,
			FirstRequestTime: currentTime,
		}
		rateLimiter := RateLimiter{
			RateLimiterData: RateLimiterData,
			Mutex:           sync.Mutex{},
		}

		response := rateLimiter.ValidateRequestLimit(clientId, currentTime, mockedRateLimiterData)

		if !response.Status {
			t.Errorf("Expect validation to be %v, but got %v", expectedStatus, response.Status)
		}

		if response.Data != expectedData[clientId] {
			t.Errorf("Expected data to be updated %v, but got %v", expectedData[clientId], response.Data)
		}
	})
}
