package postgresql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/gxkxv/restapi-pet/internal/config"
	_ "github.com/lib/pq"
	"log/slog"
	"net/http"
	"strconv"
)

type User struct {
	id     int
	Name   string
	Age    int
	Gender string
	Nation string
}
type Storage struct {
	db *sql.DB
}

func New(cfg *config.Config) (*Storage, error) {
	const op = "storage.postgresql.New"

	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS users (
		    id SERIAL,
			name TEXT,
			age INTEGER,
			gender TEXT,
			nation TEXT,
			PRIMARY KEY (id, name, age)
		)
	 `)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) GetInfoFromURL(name string) error {
	person := User{Name: name}

	resp, err := http.Get(fmt.Sprintf("https://api.agify.io/?name=%s", name))
	if err != nil {
		return fmt.Errorf("get age: %w", err)
	}
	defer resp.Body.Close()

	var ageData struct {
		Age int `json:"age"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ageData); err != nil {
		return fmt.Errorf("decode age: %w", err)
	}
	person.Age = ageData.Age

	resp, err = http.Get(fmt.Sprintf("https://api.genderize.io/?name=%s", name))
	if err != nil {
		return fmt.Errorf("get gender: %w", err)
	}
	defer resp.Body.Close()

	var genderData struct {
		Gender string `json:"gender"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&genderData); err != nil {
		return fmt.Errorf("decode gender: %w", err)
	}
	person.Gender = genderData.Gender

	resp, err = http.Get(fmt.Sprintf("https://api.nationalize.io/?name=%s", name))
	if err != nil {
		return fmt.Errorf("get nationality: %w", err)
	}
	defer resp.Body.Close()

	var nationData struct {
		Country []struct {
			CountryID   string  `json:"country_id"`
			Probability float64 `json:"probability"`
		} `json:"country"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nationData); err != nil {
		return fmt.Errorf("decode nationality: %w", err)
	}
	if len(nationData.Country) > 0 {
		person.Nation = nationData.Country[0].CountryID
	}

	stmt, err := s.db.Exec(fmt.Sprintf("INSERT INTO users(name,age,gender,nation) VALUES ('%s', %d, '%s', '%s')", person.Name, person.Age, person.Gender, person.Nation))
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	_ = stmt
	return nil
}

func GetUsers(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stmt, err := s.db.Prepare("SELECT name, age, gender, nation FROM users")
		if err != nil {
			slog.Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
		defer stmt.Close()
		rows, err := stmt.Query()
		if err != nil {
			slog.Error(err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		for rows.Next() {
			var user User
			if err := rows.Scan(&user.Name, &user.Age, &user.Gender, &user.Nation); err != nil {
				slog.Error(err.Error())
			}
			if err = json.NewEncoder(w).Encode(user); err != nil {
				slog.Error(err.Error())
			}
		}

		return
	}
}

func GetUser(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "id")
		if name == "" {
			http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
			return
		}
		stmt, err := s.db.Prepare("SELECT name, age, gender, nation FROM users WHERE id = $1")
		if err != nil {
			slog.Error(fmt.Sprintf("prepare user: %w", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()
		_ = stmt
		var user User
		if err := stmt.QueryRow(name).Scan(&user.Name, &user.Age, &user.Gender, &user.Nation); err != nil {
			slog.Error(fmt.Sprintf("query user: %w", err))
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(user)
		if err != nil {
			slog.Error(fmt.Sprintf("encode user: %w", err))
		}
		return
	}
}

func CreateUser(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Missing 'name' query parameter", http.StatusBadRequest)
		}
		err := s.GetInfoFromURL(name)
		if err != nil {
			slog.Error(err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(s)
		if err != nil {
			slog.Error(err.Error())
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

func UpdateUser(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := chi.URLParam(r, "id")
		field := chi.URLParam(r, "field")
		newValue := chi.URLParam(r, "new_value")

		if idStr == "" || field == "" || newValue == "" {
			http.Error(w, "Missing parameters", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		validFields := map[string]string{
			"name":   "string",
			"age":    "int",
			"gender": "string",
			"nation": "string",
		}
		fieldType, ok := validFields[field]
		if !ok {
			http.Error(w, "Invalid field name", http.StatusBadRequest)
			return
		}
		var query string

		if fieldType == "int" {
			valueInt, err := strconv.Atoi(newValue)
			if err != nil {
				http.Error(w, "Invalid integer value", http.StatusBadRequest)
				return
			}
			query = fmt.Sprintf("UPDATE users SET %s = $1 WHERE id = $2", field)
			_, err = s.db.Exec(query, valueInt, id)
		} else {
			query = fmt.Sprintf("UPDATE users SET %s = $1 WHERE id = $2", field)
			_, err = s.db.Exec(query, newValue, id)
		}

		if err != nil {
			slog.Error("Update error:", err)
			http.Error(w, "Database update error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func AddFriends(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		firstFriend := chi.URLParam(r, "firstFriend")
		secondFriend := chi.URLParam(r, "secondFriend")
		if firstFriend == "" {
			http.Error(w, "Missing 'firstFriend' query parameter", http.StatusBadRequest)
		} else if secondFriend == "" {
			http.Error(w, "Missing 'secondFriend' query parameter", http.StatusBadRequest)
		} else if firstFriend == "" && secondFriend == "" {
			http.Error(w, "Missing 'firstFriend' and 'secondFriend' query parameters", http.StatusBadRequest)
		}
		f, err := strconv.Atoi(firstFriend)
		if err != nil {
			slog.Error(err.Error())
		}
		fs, err := strconv.Atoi(secondFriend)
		if err != nil {
			slog.Error(err.Error())
		}
		stmt, err := s.db.Prepare(fmt.Sprintf("INSERT INTO friendships(user_id,friend_id) VALUES (%d,%d),(%d,%d)", f, fs, fs, f))
		if err != nil {
			slog.Error(err.Error())
		}
		defer stmt.Close()
		_, err = stmt.Exec()
		if err != nil {
			slog.Error(err.Error())
		}
	}
}

func GetFriends(s *Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		intId, err := strconv.Atoi(id)
		if err != nil {
			slog.Error(err.Error())
		}
		stmt, err := s.db.Prepare("SELECT u.name, f.friend_id FROM friendships f INNER JOIN users u ON u.id = f.user_id WHERE u.id = $1")
		if err != nil {
			slog.Error(err.Error())
		}
		defer stmt.Close()
		_, err = stmt.Exec(intId)
		if err != nil {
			slog.Error(err.Error())
		}
		rows, err := stmt.Query(intId)
		if err != nil {
			slog.Error(err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		for rows.Next() {
			var user User
			if err := rows.Scan(&user.id, &user.Name); err != nil {
				slog.Error(err.Error())
			}
			if err = json.NewEncoder(w).Encode(user); err != nil {
				slog.Error(err.Error())
			}
		}

		return
	}
}
