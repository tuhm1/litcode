package db

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	var e error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	DB, e = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if e != nil {
		panic(e)
	}
	DB.AutoMigrate(&Problem{}, &Test{}, &Code{}, &Result{}, &TestResult{}, &User{}, &Authority{})
	ensureAdmin()
}

func ensureAdmin() {
	u := &User{Username: os.Getenv("ADMIN_USERNAME")}
	r := DB.Where(u).Limit(1).Find(u)
	if r.Error != nil {
		panic(r.Error)
	}
	if r.RowsAffected != 0 {
		return
	}
	u.Name = os.Getenv("ADMIN_NAME")
	u.Authorities = append(u.Authorities, &Authority{Authority: "admin"})
	pw := os.Getenv("ADMIN_PASSWORD")
	h, e := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if e != nil {
		panic(e)
	}
	u.PasswordHash = string(h)
	r = DB.Save(u)
	if r.Error != nil {
		panic(r.Error)
	}
}

type Problem struct {
	gorm.Model
	ID          int
	Name        string
	Description string
	Time        int64
	Memory      int64
	Tests       []*Test
}

type Test struct {
	gorm.Model
	ProblemID int `gorm:"index"`
	Problem   *Problem
	ID        int
	Name      string
	Input     string
	Answer    string
}

type Code struct {
	gorm.Model
	ID        int
	ProblemID int
	Problem   *Problem
	UserID    int
	User      *User
	Result    *Result
	Content   string
	Language  string
}

type Result struct {
	gorm.Model
	ID           int
	CodeID       int `gorm:"index"`
	Code         *Code
	Verdict      string
	CompileError string
	TestResults  []*TestResult
}

type TestResult struct {
	gorm.Model
	ID       int
	ResultID int `gorm:"index"`
	Result   *Result
	TestID   int
	Test     *Test
	Output   string
	Error    string
	ExitCode int
	Verdict  string
}

type User struct {
	gorm.Model
	ID           int
	Name         string
	Username     string `gorm:"size:30;uniqueIndex"`
	PasswordHash string
	Authorities  []*Authority
	Solved       []*Problem `gorm:"many2many:solved"`
}

type Authority struct {
	gorm.Model
	ID        int
	UserID    int `gorm:"index"`
	User      User
	Authority string
}

const (
	ACCEPTED              = "accepted"
	WRONG_ANSWER          = "wrong answer"
	COMPILE_ERROR         = "compile error"
	RUNTIME_ERROR         = "runtime error"
	TIME_LIMIT_EXCEEDED   = "time limit exceeded"
	MEMORY_LIMIT_EXCEEDED = "memory limit exceeded"
)

const (
	CPP    = "c++"
	JAVA   = "java"
	PYTHON = "python"
)
