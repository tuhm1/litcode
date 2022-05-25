package controllers

import (
	"net/http"
	"oj/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetProblems(c *gin.Context) {
	u := GetUser(c)
	var ps *[]struct {
		db.Problem
		Solved bool
	}
	if u != nil {
		solved := "select problem_id from solved where user_id = ?"
		in := "select id in (" + solved + ")"
		e := db.DB.Raw("select id, name, ("+in+") as solved from problems where deleted_at is NULL", u.ID).Scan(&ps).Error
		if e != nil {
			panic(e)
		}
	} else {
		e := db.DB.Model(&db.Problem{}).Omit("solved").Find(&ps).Error
		if e != nil {
			panic(e)
		}
	}
	c.HTML(http.StatusOK, "problems.html", gin.H{
		"problems": ps,
		"user":     u,
		"isAdmin":  HasAuthority(u, "admin"),
	})
}

func GetCreateProblem(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	c.HTML(http.StatusOK, "problem-create.html", gin.H{})
}

func PostCreateProblem(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	p := &db.Problem{}
	var e error
	p.Name = c.PostForm("name")
	p.Description = c.PostForm("description")
	p.Time, _ = strconv.ParseInt(c.PostForm("time"), 10, 64)
	p.Memory, _ = strconv.ParseInt(c.PostForm("memory"), 10, 64)
	e = db.DB.Save(p).Error
	if e != nil {
		panic(e)
	}
	c.Redirect(http.StatusSeeOther, "/problems/edit/"+strconv.Itoa(p.ID))
}

func GetEditProblem(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var p *db.Problem
	e := db.DB.Preload("Tests").First(&p, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	c.HTML(http.StatusOK, "problem-edit.html", gin.H{
		"problem": p,
	})
}

func PostEditProblem(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var p *db.Problem
	e := db.DB.First(&p, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	p.Name = c.PostForm("name")
	p.Description = c.PostForm("description")
	p.Time, _ = strconv.ParseInt(c.PostForm("time"), 10, 64)
	p.Memory, _ = strconv.ParseInt(c.PostForm("memory"), 10, 64)
	e = db.DB.Save(&p).Error
	if e != nil {
		panic(e)
	}
	c.Redirect(http.StatusSeeOther, "/problems/edit/"+strconv.Itoa(p.ID))
}

func GetProblem(c *gin.Context) {
	var p *db.Problem
	e := db.DB.First(&p, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	u := GetUser(c)
	c.HTML(http.StatusOK, "problem.html", gin.H{
		"problem": p,
		"user":    u,
		"isAdmin": HasAuthority(u, "admin"),
	})
}

func PostDeleteProblem(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var p *db.Problem
	e := db.DB.First(&p, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	e = db.DB.Delete(p).Error
	if e != nil {
		panic(e)
	}
	c.Redirect(http.StatusSeeOther, "/problems")
}

func GetSubmissions(c *gin.Context) {
	u := GetUser(c)
	if u == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	var p *db.Problem
	e := db.DB.First(&p, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	var subs []*db.Code
	e = db.DB.Preload("Result").Where(&db.Code{UserID: u.ID, ProblemID: p.ID}).Find(&subs).Error
	if e != nil {
		panic(e)
	}
	c.HTML(http.StatusOK, "submissions.html", gin.H{
		"user":        u,
		"problem":     p,
		"submissions": subs,
	})
}
