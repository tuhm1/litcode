package controllers

import (
	"bytes"
	"io"
	"net/http"
	"oj/db"
	"runtime"
	"strconv"
	"sync"
	"time"

	"oj/exec"

	"github.com/gin-gonic/gin"
)

func PostJudge(c *gin.Context) {
	u := GetUser(c)
	if u == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	var p *db.Problem
	e := db.DB.Preload("Tests").First(&p, c.Param("problemID")).Error
	if e != nil {
		panic(e)
	}
	for _, tc := range p.Tests {
		tc.Problem = p
	}
	code := &db.Code{
		ProblemID: p.ID,
		Problem:   p,
		UserID:    u.ID,
		User:      u,
		Content:   c.PostForm("code"),
		Language:  c.PostForm("language"),
	}
	e = db.DB.Omit("User", "Problem").Create(code).Error
	if e != nil {
		panic(e)
	}
	c.Redirect(http.StatusSeeOther, "/submissions/"+strconv.Itoa(code.ID))
	r := &db.Result{
		CodeID: code.ID,
	}
	go judgeProgram(code, p, r)
}

func GetSubmission(c *gin.Context) {
	u := GetUser(c)
	var code *db.Code
	db.DB.Preload("Problem").
		Preload("User").
		Preload("Result.TestResults.Test").
		Find(&code, c.Param("codeID"))
	if (u == nil || u.ID != code.User.ID) && !HasAuthority(u, "admin") {
		c.HTML(http.StatusForbidden, "error.html", gin.H{"error": "Forbidden"})
		return
	}
	c.HTML(http.StatusOK, "submission.html", gin.H{
		"code": code,
	})
}

func judgeProgram(code *db.Code, p *db.Problem, r *db.Result) {
	var prog *exec.Program
	switch code.Language {
	case db.CPP:
		prog, r.CompileError = exec.NewCpp(code.Content)
	case db.JAVA:
		prog, r.CompileError = exec.NewJava(code.Content)
	case db.PYTHON:
		prog = exec.NewPython(code.Content)
	default:
		r.CompileError = "language not supported"
	}
	if prog == nil {
		r.Verdict = db.COMPILE_ERROR
		e := db.DB.Create(r).Error
		if e != nil {
			panic(e)
		}
		return
	}
	defer prog.Delete()
	r.TestResults = make([]*db.TestResult, len(p.Tests))
	r.Verdict = db.ACCEPTED
	var wg sync.WaitGroup
	for i := range p.Tests {
		wg.Add(1)
		pool <- true
		go func(i int) {
			defer func() {
				wg.Done()
				<-pool
			}()
			r.TestResults[i] = &db.TestResult{
				ResultID: r.ID,
				TestID:   p.Tests[i].ID,
			}
			judgeTest(prog, p.Tests[i], r.TestResults[i])
		}(i)
	}
	wg.Wait()
	for _, tr := range r.TestResults {
		if priority[r.Verdict] < priority[tr.Verdict] {
			r.Verdict = tr.Verdict
		}
	}
	e := db.DB.Create(r).Error
	if e != nil {
		panic(e)
	}
	if r.Verdict == db.ACCEPTED {
		e := db.DB.Model(code.User).Association("Solved").Append(code.Problem)
		if e != nil {
			panic(e)
		}
	}
}

func judgeTest(prog *exec.Program, t *db.Test, r *db.TestResult) {
	var fe, fo bytes.Buffer
	tl := time.Duration(t.Problem.Time) * time.Millisecond
	proc := prog.Run(tl, t.Problem.Memory, &fo, &fe)
	defer proc.Delete()
	go io.WriteString(proc.Stdin, t.Input+"\n")
	proc.Wait()
	r.Output = fo.String()
	r.Error = fe.String()
	stats := proc.Stats()
	r.ExitCode = stats.ExitCode
	if stats.ExitCode == 0 {
		if r.Output == t.Answer {
			r.Verdict = db.ACCEPTED
		} else {
			r.Verdict = db.WRONG_ANSWER
		}
	} else if stats.Time > tl {
		r.Verdict = db.TIME_LIMIT_EXCEEDED
	} else if stats.OOM {
		r.Verdict = db.MEMORY_LIMIT_EXCEEDED
	} else {
		r.Verdict = db.RUNTIME_ERROR
	}
}

var pool = make(chan bool, runtime.NumCPU())

var priority = map[string]int{
	db.ACCEPTED:              0,
	db.WRONG_ANSWER:          1,
	db.TIME_LIMIT_EXCEEDED:   2,
	db.MEMORY_LIMIT_EXCEEDED: 3,
	db.RUNTIME_ERROR:         4,
	db.COMPILE_ERROR:         5,
}
