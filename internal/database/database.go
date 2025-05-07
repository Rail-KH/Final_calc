package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Rail-KH/Final_calc/internal/auth"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID       int64
	Login    string
	Password string
}

type Expression struct {
	UserID     int
	ID         int      `json:"id"`
	Expression string   `json:"expression"`
	Status     string   `json:"status"`
	Result     *float64 `json:"result"`
}

type Task struct {
	ID            string
	ExprID        int
	Arg1          float64
	Arg2          float64
	Operation     string
	OperationTime int
	Completed     bool
	Result        sql.NullFloat64
}

type DataBase struct {
	DB *sql.DB
}

func CreateTable() (*DataBase, error) {
	const (
		usersTable = `
	CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            login TEXT NOT NULL UNIQUE,
            password TEXT NOT NULL
	);`

		expressionsTable = `
	CREATE TABLE IF NOT EXISTS expressions (
        	id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            expression TEXT NOT NULL,
            status TEXT NOT NULL,
            result REAL,
            FOREIGN KEY(user_id) REFERENCES users(id)
	);`
		tasksTable = `CREATE TABLE IF NOT EXISTS tasks (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            expression_id INTEGER NOT NULL,
            arg1 REAL NOT NULL,
            arg2 REAL NOT NULL,
            operation TEXT NOT NULL,
            operation_time INTEGER NOT NULL,
            completed BOOLEAN DEFAULT FALSE,
            result REAL,
            FOREIGN KEY(expression_id) REFERENCES expressions(id)
        );`
	)

	ctx := context.TODO()
	db, err := sql.Open("sqlite3", "exp.db")
	if err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, usersTable); err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, expressionsTable); err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, tasksTable); err != nil {
		return nil, err
	}

	return &DataBase{DB: db}, nil
}

func (d *DataBase) InsertUser(login, password string) (int64, error) {
	var q = `
	INSERT INTO users (login, password) values ($1, $2)
	`
	hash, err := auth.HashPass(password)
	if err != nil {
		return 0, err
	}
	result, err := d.DB.Exec(q, login, hash)
	if err != nil {
		if isDuplicate(err) {
			return 0, errors.New("already exists")
		}
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (d *DataBase) SelectUser(login string) (*User, error) {
	user := &User{}
	var q = "SELECT id, login, password FROM users WHERE login=$1"
	err := d.DB.QueryRow(q, login).Scan(&user.ID, &user.Login, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	return user, nil
}

func (d *DataBase) CreateExpression(userID int, expr string) (*Expression, error) {
	e := &Expression{
		UserID:     userID,
		Expression: expr,
		Status:     "pending",
	}

	err := d.DB.QueryRow(
		`INSERT INTO expressions 
		(user_id, expression, status) 
		VALUES (?, ?, ?) 
		RETURNING id`,
		e.UserID, e.Expression, e.Status,
	).Scan(&e.ID)

	if err != nil {
		return nil, err
	}
	return e, nil
}

func (d *DataBase) UpdateExpression(e *Expression) error {
	var result interface{}
	if e.Result != nil {
		result = *e.Result
	}

	_, err := d.DB.Exec(
		`UPDATE expressions 
		SET status = ?, result = ? 
		WHERE id = ? AND user_id = ?`,
		e.Status, result, e.ID, e.UserID,
	)
	return err
}

func (d *DataBase) GetExpressions(userID int) ([]*Expression, error) {
	rows, err := d.DB.Query(
		`SELECT id, expression, status, result 
		FROM expressions WHERE user_id = ? `,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exprs []*Expression
	for rows.Next() {
		e := &Expression{UserID: userID}
		var result sql.NullFloat64
		err := rows.Scan(&e.ID, &e.Expression, &e.Status, &result)
		if err != nil {
			return nil, err
		}
		if result.Valid {
			e.Result = &result.Float64
		}
		exprs = append(exprs, e)
	}
	return exprs, nil
}

func (d *DataBase) GetExpressionByID(id, userID int) (*Expression, error) {
	e := &Expression{ID: id, UserID: userID}
	var result sql.NullFloat64
	err := d.DB.QueryRow(
		`SELECT expression, status, result 
		FROM expressions WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&e.Expression, &e.Status, &result)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		return nil, err
	}

	if result.Valid {
		e.Result = &result.Float64
	}
	return e, nil
}

func isDuplicate(err error) bool {
	return err != nil && err.Error() == "UNIQUE constraint failed: users.login"
}
