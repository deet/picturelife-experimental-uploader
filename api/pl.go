package api

type BasicResponse interface {
	GetStatus() int64
}

type ApiResponse struct {
	Response_Time int64 `json:"response_time"`
	Status        int64 `json:"status"`
}

func (s *ApiResponse) GetStatus() int64 {
	return s.Status
}
