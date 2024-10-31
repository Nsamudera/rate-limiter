package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type Body struct {
	Status  int
	Message string
}

func TestRequestHandlerFail(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Run("no clientID provided", func(t *testing.T) {
		expectedStatus := http.StatusBadRequest
		expectedMessage := "No clientID provided"

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		requestHandler(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != "No clientID provided" {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		expectedStatus := http.StatusTooManyRequests
		clientID := "PT TEST"
		expectedMessage := fmt.Sprintf("Too Many Requests for %v", clientID)

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandler(response, request)
		// reset the Body as we are only interested in the second call
		response.Body.Reset()
		requestHandler(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})
}

func TestRequestHandlerSuccess(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		expectedStatus := http.StatusOK
		clientID := "PT B"
		expectedMessage := fmt.Sprintf("Hello %v", clientID)

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandler(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("successful request if config does not exist", func(t *testing.T) {
		expectedStatus := http.StatusOK
		clientID := "PT ABC"
		expectedMessage := fmt.Sprintf("Hello %v", clientID)

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandler(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("successful request after limit has refreshed", func(t *testing.T) {
		expectedStatus1 := http.StatusTooManyRequests
		expectedStatusFinal := http.StatusOK
		clientID := "PT B"
		expectedMessage1 := fmt.Sprintf("Too Many Requests for %v", clientID)
		expectedMessageFinal := fmt.Sprintf("Hello %v", clientID)

		// test to make sure that we exceed rate limit first
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandler(response, request)
		requestHandler(response, request)
		requestHandler(response, request)
		response.Body.Reset()
		requestHandler(response, request)

		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus1 {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus1, body.Status)
		}

		if body.Message != expectedMessage1 {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage1, body.Message)
		}

		// Test again once the rate limit has been refreshed
		// For this demo, config is currently hard coded in "main.go"
		time.Sleep(3 * time.Second)
		response.Body.Reset()
		requestHandler(response, request)
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatusFinal {
			t.Errorf("Expect status to be %v, but got %v", expectedStatusFinal, body.Status)
		}

		if body.Message != expectedMessageFinal {
			t.Errorf("Expect message to be %v, but got %v", expectedMessageFinal, body.Message)
		}
	})
}

func TestRequestHandlerConfigFail(t *testing.T) {
	t.Run("incorrect method provided", func(t *testing.T) {
		expectedStatus := http.StatusMethodNotAllowed
		expectedMessage := "Method not allowed"

		request := httptest.NewRequest(http.MethodGet, "/config", nil)
		response := httptest.NewRecorder()
		requestHandlerConfig(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("no clientID provided", func(t *testing.T) {
		expectedStatus := http.StatusBadRequest
		expectedMessage := "No clientID provided"

		request := httptest.NewRequest(http.MethodPost, "/config", nil)
		response := httptest.NewRecorder()
		requestHandlerConfig(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != "No clientID provided" {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("no body provided", func(t *testing.T) {
		expectedStatus := http.StatusBadRequest
		expectedMessage := "Config data must be greater than 0"

		request := httptest.NewRequest(http.MethodPost, "/config", nil)
		request.Header.Set("clientID", "PT A")
		response := httptest.NewRecorder()
		requestHandlerConfig(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("body contains string", func(t *testing.T) {
		clientID := "PT A"
		expectedStatus := http.StatusBadRequest
		expectedMessage := "Config data must be greater than 0"
		requestBody := strings.NewReader(`{
			"limit": "10",
			"window": "1"
		}`)

		request := httptest.NewRequest(http.MethodPost, "/config", requestBody)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandlerConfig(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("body contains array", func(t *testing.T) {
		clientID := "PT A"
		expectedStatus := http.StatusBadRequest
		expectedMessage := "Config data must be greater than 0"
		requestBody := strings.NewReader(`{
			"limit": [10],
			"window": [1]
		}`)

		request := httptest.NewRequest(http.MethodPost, "/config", requestBody)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandlerConfig(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})

	t.Run("body contains struct", func(t *testing.T) {
		clientID := "PT A"
		expectedStatus := http.StatusBadRequest
		expectedMessage := "Config data must be greater than 0"
		requestBody := strings.NewReader(`{
			"limit": {},
			"window": {}
		}`)

		request := httptest.NewRequest(http.MethodPost, "/config", requestBody)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandlerConfig(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})
}

func TestRequestHandlerConfigSuccess(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		clientID := "PT A"
		expectedStatus := http.StatusOK
		expectedMessage := fmt.Sprintf("New config created for %v", clientID)
		requestBody := strings.NewReader(`{
			"limit": 10,
			"window": 1
		}`)

		request := httptest.NewRequest(http.MethodPost, "/config", requestBody)
		request.Header.Set("clientID", clientID)
		response := httptest.NewRecorder()
		requestHandlerConfig(response, request)
		var body Body
		json.Unmarshal(response.Body.Bytes(), &body)

		if body.Status != expectedStatus {
			t.Errorf("Expect status to be %v, but got %v", expectedStatus, body.Status)
		}

		if body.Message != expectedMessage {
			t.Errorf("Expect message to be %v, but got %v", expectedMessage, body.Message)
		}
	})
}
