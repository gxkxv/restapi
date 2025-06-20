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
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Missing 'name' query parameter", http.StatusBadRequest)
			return
		}
		stmt, err := s.db.Prepare("SELECT name, age, gender, nation FROM users WHERE name = $1")
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
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Missing 'name' query parameter", http.StatusBadRequest)
		}
		var patch struct {
			Name   *string `json:"name"`
			Age    *int    `json:"age"`
			Gender *string `json:"gender"`
			Nation *string `json:"nation"`
		}
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			slog.Error(err.Error())
			return
		}
		query := "UPDATE users SET "
		args := []interface{}{}
		i := 1

		if patch.Name != nil {
			query += fmt.Sprintf("name = %d ", *patch.Name)
			args = append(args, *patch.Name)
			i++
		}
		if patch.Age != nil {
			query += fmt.Sprintf("age = %d ", *patch.Age)
			args = append(args, *patch.Age)
			i++
		}
		if patch.Gender != nil {
			query += fmt.Sprintf("gender = '%s' ", *patch.Gender)
			args = append(args, *patch.Gender)
			i++
		}
		if patch.Nation != nil {
			query += fmt.Sprintf("nation = '%s' ", *patch.Nation)
			args = append(args, *patch.Nation)
			i++
		}
		if len(args) == 0 {
			http.Error(w, "Missing  query parameters", http.StatusBadRequest)
			return
		}
		query = query[:len(query)-1] + fmt.Sprintf(" WHERE name = $%d", i)
		args = append(args, name)
		stmt, err := s.db.Exec(query, args...)
		_ = stmt
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
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
		_, err := strconv.Atoi(firstFriend)
		if err != nil {
			slog.Error(err.Error())
		}
		_, err = strconv.Atoi(secondFriend)
		if err != nil {
			slog.Error(err.Error())
		}
		stmt, err := s.db.Prepare(fmt.Sprintf("INSERT INTO friends(firstFriend,secondFriend) VALUES ('%d','%d'),('%d','%d')", firstFriend, secondFriend, secondFriend, firstFriend))
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
		_, err := strconv.Atoi(id)
		if err != nil {
			slog.Error(err.Error())
		}
		stmt, err := s.db.Prepare("SELECT u.name, f.friend_id FROM friendships INNER JOIN u users ON u.id = f.user_id WHERE u.id = $1")
		if err != nil {
			slog.Error(err.Error())
		}
		defer stmt.Close()
		_, err = stmt.Exec(id)
		if err != nil {
			slog.Error(err.Error())
		}
		rows, err := stmt.Query(id)
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
