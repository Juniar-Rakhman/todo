package postgres

import (
	"database/sql"
	"fmt"

	"github.com/fharding1/todo/internal/store"

	// for the postgres sql driver
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type service struct {
	db *sql.DB
}

// Options holds information for connecting to a postgresql server
type Options struct {
	User, Pass string
	Host       string
	Port       int
	DBName     string
	SSLMode    string
}

func (o Options) connectionInfo() string {
	return fmt.Sprintf("host='%s' port='%d' user='%s' password='%s' dbname='%s' sslmode='%s'",
		o.Host, o.Port, o.User, o.Pass, o.DBName, o.SSLMode)
}

const todoTableCreationQuery = `
CREATE TABLE IF NOT EXISTS todos (
	id          SERIAL PRIMARY KEY,
	description varchar(256),
	isCompleted BOOLEAN
)`

// New connects to a postgres server with specified options and returns a store.Service
func New(options Options) (store.Service, error) {
	db, err := sql.Open("postgres", options.connectionInfo())
	if err != nil {
		return nil, errors.Wrap(err, "connecting to postgres database")
	}

	_, err = db.Exec(todoTableCreationQuery)
	if err != nil {
		return nil, errors.Wrap(err, "creating todos table")
	}

	return &service{db: db}, nil
}

func (s *service) CreateTodo(todo store.Todo) (id int64, err error) {
	err = s.db.QueryRow(
		"INSERT INTO todos (description, isCompleted) VALUES ($1, $2) RETURNING id",
		todo.Description, todo.IsCompleted).Scan(&id)
	return
}

func (s *service) GetTodo(id int64) (store.Todo, error) {
	todo := store.Todo{ID: id}
	err := s.db.QueryRow("SELECT description, isCompleted FROM todos WHERE id = $1", id).Scan(
		&todo.Description, &todo.IsCompleted)
	if err == sql.ErrNoRows {
		err = store.ErrNoResults
	}
	return todo, err
}

func (s *service) GetTodos() ([]store.Todo, error) {
	rows, err := s.db.Query("SELECT id, description, isCompleted FROM todos")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := []store.Todo{}

	for rows.Next() {
		var todo store.Todo
		if err := rows.Scan(&todo.ID, &todo.Description, &todo.IsCompleted); err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return todos, nil
}

func (s *service) UpdateTodo(todo store.Todo) error {
	_, err := s.db.Exec("UPDATE todos SET description = $1, completedAt = $2 WHERE id = $3",
		todo.Description, todo.IsCompleted, todo.ID)
	return err
}

func (s *service) PatchTodo(nt store.NullableTodo) error {
	_, err := s.db.Exec(`
		UPDATE todos SET
		description = COALESCE($1, description),
		isCompleted = COALESCE($2, isCompleted)
		WHERE id = $3
		`, nt.Description, nt.IsCompleted, nt.ID)
	return err
}

func (s *service) DeleteTodo(id int64) error {
	_, err := s.db.Exec("DELETE FROM todos WHERE id = $1", id)
	return err
}

func (s *service) Close() error {
	return s.db.Close()
}
