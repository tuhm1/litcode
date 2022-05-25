package controllers

import (
	"net/http"
	"oj/db"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func GetAccount(c *gin.Context) {
	u := GetUser(c)
	if u == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	c.HTML(http.StatusOK, "account.html", gin.H{"user": u})
}

func GetRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{})
}

func PostRegister(c *gin.Context) {
	if c.PostForm("password") != c.PostForm("confirm") {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Password confirmation does not match"})
		return
	}
	var u *db.User
	e := db.DB.Where(&db.User{Username: c.PostForm("username")}).Limit(1).Find(&u).Error
	if e != nil {
		panic(e)
	}
	if u != nil {
		c.HTML(http.StatusConflict, "error.html", gin.H{"error": "User already exists"})
		return
	}
	h, e := bcrypt.GenerateFromPassword([]byte(c.PostForm("password")), bcrypt.DefaultCost)
	if e != nil {
		panic(e)
	}
	u = &db.User{
		Username:     c.PostForm("username"),
		Name:         c.PostForm("name"),
		PasswordHash: string(h),
	}
	e = db.DB.Save(u).Error
	if e != nil {
		panic(e)
	}
	Login(c, u)
	s := sessions.Default(c)
	c.Redirect(http.StatusSeeOther, s.Get("login-redirect").(string))
	s.Delete("login-redirect")
	s.Save()
}

func GetLogin(c *gin.Context) {
	session := sessions.Default(c)
	session.Set("login-redirect", c.Request.Referer())
	session.Save()
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func PostLogin(c *gin.Context) {
	var u *db.User
	e := db.DB.Where(&db.User{Username: c.PostForm("username")}).Limit(1).Find(&u).Error
	if e != nil {
		panic(e)
	}
	if u == nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid username or password"})
		return
	}
	e = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(c.PostForm("password")))
	if e != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid username or password"})
		return
	}
	Login(c, u)
	s := sessions.Default(c)
	c.Redirect(http.StatusSeeOther, s.Get("login-redirect").(string))
	s.Delete("login-redirect")
	s.Save()
}

func PostLogout(c *gin.Context) {
	Logout(c)
	c.Redirect(http.StatusSeeOther, "/")
}

func Login(c *gin.Context, u *db.User) {
	s := sessions.Default(c)
	s.Set("userID", u.ID)
	e := s.Save()
	if e != nil {
		panic(e)
	}
}

func Logout(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	e := s.Save()
	if e != nil {
		panic(e)
	}
}

func GetUser(c *gin.Context) *db.User {
	s := sessions.Default(c)
	uID := s.Get("userID")
	if uID == nil {
		return nil
	}
	var u *db.User
	e := db.DB.Preload("Authorities").First(&u, uID).Error
	if e != nil {
		panic(e)
	}
	return u
}

func HasAuthority(u *db.User, au string) bool {
	if u == nil {
		return false
	}
	for _, uau := range u.Authorities {
		if uau.Authority == au {
			return true
		}
	}
	return false
}
