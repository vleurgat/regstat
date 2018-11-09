package registry

import (
	"net/http"
						)

// MockHttpClient: mock implementation of HttpClient.
type MockHttpClient struct {
	responses *[]http.Response
	err      error
}

// CreateMockHttpClientErr creates a MockHttpClient that returns errors.
func CreateMockHttpClientErr(err error) MockHttpClient {
	return MockHttpClient{
		responses: &[]http.Response{{}},
		err: err,
	}
}

// CreateMockHttpClient creates a MockHttpClient that returns http.Responses.
func CreateMockHttpClient(res ...http.Response) MockHttpClient {
	return MockHttpClient{
		responses: &res,
	}
}

// Do is the mock implementation of the real http.Client.Do method.
func (m MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	response := (*m.responses)[0]
	if len(*m.responses) > 1 {
		*m.responses = (*m.responses)[1:]
	}
	return &response, m.err
}
