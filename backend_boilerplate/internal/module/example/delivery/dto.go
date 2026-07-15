package delivery

type ItemResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type CreateItemRequest struct {
	Name string `json:"name" binding:"required"`
}
