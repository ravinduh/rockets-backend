package response

// APIResponse represents the standard API response format
type APIResponse struct {
	RequestID string      `json:"request_id"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// New creates a unified API response
func New(requestID string, data interface{}, err error) APIResponse {
	response := APIResponse{
		RequestID: requestID,
	}
	
	if err != nil {
		response.Error = err.Error()
	} else {
		response.Data = data
	}
	
	return response
}

// Success creates a successful API response (legacy compatibility)
func Success(requestID string, data interface{}) APIResponse {
	return New(requestID, data, nil)
}

// Error creates an error API response (legacy compatibility)
func Error(requestID string, err string) APIResponse {
	return APIResponse{
		RequestID: requestID,
		Error:     err,
	}
}