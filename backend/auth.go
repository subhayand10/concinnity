package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func IsAuthenticated(w http.ResponseWriter, r *http.Request, t *Token) *User {
	token := r.Header.Get("Authentication")
	if cookie, err := r.Cookie("token"); err == nil {
		token = cookie.Value
	}

	res, err := findUserByTokenStmt.Query(token)
	if err != nil {
		handleInternalServerError(w, err)
		return nil
	} else if !res.Next() {
		if w != nil {
			http.Error(w, errorJson("You are not authenticated to access this resource!"),
				http.StatusUnauthorized)
		}
		return nil
	} else {
		var (
			username  string
			password  string
			email     string
			id        []byte
			token     string
			createdAt time.Time
		)
		err := res.Scan(&username, &password, &email, &id, &token, &createdAt)
		if err != nil {
			handleInternalServerError(w, err)
			return nil
		} else if t != nil {
			t.CreatedAt = createdAt
			t.Token = token
			t.ID = id
		}
		return &User{
			Username: username,
			Password: password,
			Email:    email,
			ID:       id,
		}
	}
}

func StatusEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, errorJson("Method Not Allowed!"), http.StatusMethodNotAllowed)
	} else if user := IsAuthenticated(nil, r, nil); user != nil {
		usernameJson, _ := json.Marshal(user.Username)
		w.Write([]byte("{\"online\":true,\"authenticated\":true,\"username\":" + string(usernameJson) + "}"))
	} else {
		w.Write([]byte("{\"online\":true,\"authenticated\":false}"))
	}
}

func LoginEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Check the body for JSON containing username and password and return a token.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, errorJson("Unable to read body!"), http.StatusBadRequest)
			return
		}
		var data struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			http.Error(w, errorJson("Unable to read body!"), http.StatusBadRequest)
			return
		} else if data.Username == "" || data.Password == "" {
			http.Error(w, errorJson("No username or password provided!"), http.StatusBadRequest)
			return
		}
		var user User
		err = findUserByNameOrEmailStmt.QueryRow(data.Username, data.Username).Scan(
			&user.Username, &user.Password, &user.Email, &user.ID)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			http.Error(w, errorJson("No account with this username/email exists!"), http.StatusBadRequest)
			return
		} else if err != nil {
			handleInternalServerError(w, err)
			return
		}
		tokenBytes := make([]byte, 64)
		_, _ = rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)
		result, err := insertTokenStmt.Exec(token, time.Now().UTC(), user.ID)
		if err != nil {
			handleInternalServerError(w, err)
			return
		} else if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			handleInternalServerError(w, err) // nil err solved by Ostrich algorithm
			return
		}
		// Add cookie to browser.
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			HttpOnly: true,
			Secure:   secureCookies,
			MaxAge:   3600 * 24 * 31,
			SameSite: http.SameSiteStrictMode,
		})
		json.NewEncoder(w).Encode(struct {
			Token    string `json:"token"`
			Username string `json:"username"`
		}{Token: token, Username: user.Username})
	} else {
		http.Error(w, errorJson("Method Not Allowed!"), http.StatusMethodNotAllowed)
	}
}

func LogoutEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		token := Token{}
		if IsAuthenticated(w, r, &token) == nil {
			return
		}
		result, err := deleteTokenStmt.Exec(token.Token)
		if err != nil {
			handleInternalServerError(w, err)
			return
		}
		rows, err := result.RowsAffected()
		if err != nil {
			handleInternalServerError(w, err)
			return
		} else if rows == 0 {
			http.Error(w, errorJson("You are not authenticated to access this resource!"),
				http.StatusUnauthorized)
			return
		}
		// Delete cookie on browser.
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "null",
			HttpOnly: true,
			Secure:   secureCookies,
			MaxAge:   -1,
			SameSite: http.SameSiteStrictMode,
		})
		w.Write([]byte("{\"success\":true}"))
	} else {
		http.Error(w, errorJson("Method Not Allowed!"), http.StatusMethodNotAllowed)
	}
}

func RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Check the body for JSON containing username, password and email, and return a token.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, errorJson("Unable to read body!"), http.StatusBadRequest)
			return
		}
		var data struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			http.Error(w, errorJson("Unable to read body!"), http.StatusBadRequest)
			return
		} else if data.Username == "" || data.Password == "" || data.Email == "" {
			http.Error(w, errorJson("No username, e-mail or password provided!"), http.StatusBadRequest)
			return
		} // TODO: Check if username, password and e-mail are valid.
		// TODO: Implement e-mail verification in some form
		// Check if an account with this username or email already exists.
		var u User
		err = findUserByEmailStmt.QueryRow(data.Email).Scan(&u.Username, &u.Password, &u.Email, &u.ID)
		if err == nil {
			http.Error(w, errorJson("An account with this e-mail already exists!"), http.StatusConflict)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			handleInternalServerError(w, err)
			return
		}
		err = findUserByUsernameStmt.QueryRow(data.Username).Scan(&u.Username, &u.Password, &u.Email, &u.ID)
		if err == nil {
			http.Error(w, errorJson("An account with this username already exists!"), http.StatusConflict)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			handleInternalServerError(w, err)
			return
		}
		// Create the account.
		hash := HashPassword(data.Password, GenerateSalt())
		uuid := uuid.New()
		result, err := insertUserStmt.Exec(data.Username, hash, data.Email, uuid)
		if err != nil {
			handleInternalServerError(w, err)
			return
		} else if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			handleInternalServerError(w, err) // nil err solved by Ostrich algorithm
			return
		}
		// Log the user in.
		tokenBytes := make([]byte, 64)
		_, _ = rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)
		result, err = insertTokenStmt.Exec(token, time.Now().UTC(), uuid)
		if err != nil {
			handleInternalServerError(w, err)
			return
		} else if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			handleInternalServerError(w, err) // nil err solved by Ostrich algorithm
			return
		}
		// Add cookie to browser.
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			HttpOnly: true,
			Secure:   secureCookies,
			MaxAge:   3600 * 24 * 31,
			SameSite: http.SameSiteStrictMode,
		})
		json.NewEncoder(w).Encode(struct {
			Token    string `json:"token"`
			Username string `json:"username"`
		}{Token: token, Username: data.Username})
	} else {
		http.Error(w, errorJson("Method Not Allowed!"), http.StatusMethodNotAllowed)
	}
}
