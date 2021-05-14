package objects

type (
	UserRequest struct {
		Username          string   `json:"username" bson:"username"`
		Password          string   `json:"password" bson:"-"`
		EncryptedPassword [32]byte `bson:"password"`
		Name              string   `json:"name" bson:"name, omitempty"`
		Rating            int      `json:"rating" bson:"rating, omitempty"`
	}

	User struct {
		Id       string `json:"id" bson:"_id, omitempty"`
		Username string `json:"username" bson:"username"`
		Name     string `json:"name" bson:"name, omitempty"`
		Rating   int    `json:"rating" bson:"rating, omitempty"`
	}

	UpdateUser struct {
		Username string `json:"username" bson:"username"`
		Name     string `json:"name" bson:"name, omitempty"`
		Rating   int    `json:"rating" bson:"rating, omitempty"`
	}

	AuthorizeRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	AuthenticateRequest struct {
		Token string `json:"token"`
	}
)
