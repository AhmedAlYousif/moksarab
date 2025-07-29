package models

type PageModel[T any] struct {
	Content       []T  `json:"content"`
	First         bool `json:"first"`
	Last          bool `json:"last"`
	TotalElements int  `json:"totalElements"`
	TotalPages    int  `json:"totalPages"`
	Size          int  `json:"size"`
	Page          int  `json:"page"`
}

func PageOf[T any](content []T, pageNumber int, pageSize int, totalElements int) *PageModel[T] {
	totalPages := (totalElements + pageSize - 1) / pageSize
	return &PageModel[T]{
		Content:       content,
		Page:          pageNumber,
		Size:          pageSize,
		TotalElements: totalElements,
		TotalPages:    totalPages,
		First:         pageNumber == 0,
		Last:          (pageNumber + 1) >= totalPages,
	}
}
