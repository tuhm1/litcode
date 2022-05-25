package controllers

import (
	"net/http"
	"oj/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetCreateTest(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var p *db.Problem
	e := db.DB.First(&p, c.Param("problemID")).Error
	if e != nil {
		panic(e)
	}
	c.HTML(http.StatusOK, "test-create.html", gin.H{"problem": p})
}

func PostCreateTest(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var p *db.Problem
	e := db.DB.First(&p, c.Param("problemID")).Error
	if e != nil {
		panic(e)
	}
	t := &db.Test{
		ProblemID: p.ID,
		Name:      c.PostForm("name"),
		Input:     c.PostForm("input"),
		Answer:    c.PostForm("answer"),
	}
	e = db.DB.Save(t).Error
	if e != nil {
		panic(e)
	}
	c.Redirect(http.StatusSeeOther, "/problems/edit/"+strconv.Itoa(p.ID))
}

func GetEditTest(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var t *db.Test
	e := db.DB.Preload("Problem").First(&t, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	c.HTML(http.StatusOK, "test-edit.html", gin.H{
		"test": t,
	})
}

func PostEditTest(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var t *db.Test
	e := db.DB.First(&t, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	t.Name = c.PostForm("name")
	t.Input = c.PostForm("input")
	t.Answer = c.PostForm("answer")
	e = db.DB.Save(t).Error
	if e != nil {
		panic(e)
	}
	c.Redirect(http.StatusSeeOther, "/problems/edit/"+strconv.Itoa(t.ProblemID))
}

func PostDeleteTest(c *gin.Context) {
	u := GetUser(c)
	if !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	var t *db.Test
	e := db.DB.First(&t, c.Param("ID")).Error
	if e != nil {
		panic(e)
	}
	e = db.DB.Delete(t).Error
	if e != nil {
		panic(e)
	}
	c.Redirect(http.StatusSeeOther, "/problems/edit/"+strconv.Itoa(t.ProblemID))
}
