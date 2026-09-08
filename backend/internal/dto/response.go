package dto

type Response struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    interface{}  `json:"data,omitempty"`
	Errors  []FieldError `json:"errors,omitempty"`
	Meta    *Meta        `json:"meta,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginationQuery struct {
	Page    int `json:"page" query:"page"`
	PerPage int `json:"per_page" query:"per_page"`
}

func (p *PaginationQuery) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}
	return (p.Page - 1) * p.PerPage
}

func NewSuccessResponse(message string, data interface{}) Response {
	return Response{Success: true, Message: message, Data: data}
}

func NewErrorResponse(message string, errors []FieldError) Response {
	return Response{Success: false, Message: message, Errors: errors}
}

func NewPaginatedResponse(message string, data interface{}, meta *Meta) Response {
	return Response{Success: true, Message: message, Data: data, Meta: meta}
}
