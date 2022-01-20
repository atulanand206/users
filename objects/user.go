package objects

type (
	// Definition of the UserRequest object.
	UserRequest struct {
		Username          string         `json:"username" bson:"username"`
		Password          string         `json:"password" bson:"-"`
		EncryptedPassword [32]byte       `bson:"password"`
		Name              string         `json:"name" bson:"name, omitempty"`
		Rating            map[string]int `json:"rating" bson:"rating, omitempty"`
	}

	// Definition of the User object.
	User struct {
		Id       string         `json:"id" bson:"_id, omitempty"`
		Username string         `json:"username" bson:"username"`
		Name     string         `json:"name" bson:"name, omitempty"`
		Rating   map[string]int `json:"rating" bson:"rating, omitempty"`
	}

	// Definition of the UpdateUser object.
	UpdateUser struct {
		Username string         `json:"username" bson:"username"`
		Name     string         `json:"name" bson:"name, omitempty"`
		Rating   map[string]int `json:"rating" bson:"rating, omitempty"`
	}

	// Definition of the AuthorizeRequest object.
	AuthorizeRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// Definition of the AuthenticationResponse object.
	AuthenticationResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
)
