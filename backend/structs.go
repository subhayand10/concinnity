package main

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	ID       []byte `json:"id"`
}

/* type Room struct { // TODO
	ID        []byte    `json:id`
	Timestamp time.Time `json:"timestamp"`
} */