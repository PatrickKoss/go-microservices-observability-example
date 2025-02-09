package domain

type Order struct {
	ID         string   `json:"id"`
	CustomerID string   `json:"customerId"`
	ProductIDs []string `json:"productIds"`
}

type Product struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
