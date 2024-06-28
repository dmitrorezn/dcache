package domain

type Owner struct {
	ID            ID
	Name          string
	Surname       string
	ProductIDs    []ID
	EmailVerified bool
	PhoneVerified bool
}

type Product struct {
	ID      ID
	OwnerID ID
	Name    string
	Domain  string
	APIKey  string
}
