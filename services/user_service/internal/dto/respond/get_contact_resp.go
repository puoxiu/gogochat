package respond


type GetUserContactResponse struct {
	UserId      string         `json:"user_id"`
	ContactId   string         `json:"contact_id"`
	ContactType int8           `json:"contact_type"`
	Status      int8           `json:"status"`
}