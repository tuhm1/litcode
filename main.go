package main

import (
	"oj/controllers"
	"oj/db"
	"os"

	"github.com/Masterminds/sprig"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	e := godotenv.Load()
	if e != nil {
		panic(e)
	}

	db.Init()

	r := gin.Default()
	r.SetFuncMap(sprig.FuncMap())
	r.LoadHTMLGlob("views/**")

	s := memstore.NewStore([]byte(os.Getenv("SECRET")))
	r.Use(sessions.Sessions("session", s))

	r.GET("/", controllers.GetProblems)
	r.GET("/problems", controllers.GetProblems)
	r.GET("/problems/create", controllers.GetCreateProblem)
	r.POST("/problems/create", controllers.PostCreateProblem)
	r.GET("/problems/edit/:ID", controllers.GetEditProblem)
	r.POST("/problems/edit/:ID", controllers.PostEditProblem)
	r.POST("/problems/delete/:ID", controllers.PostDeleteProblem)
	r.GET("/problems/details/:ID", controllers.GetProblem)
	r.GET("/problems/submissions/:ID", controllers.GetSubmissions)

	r.GET("/tests/create/:problemID", controllers.GetCreateTest)
	r.POST("/tests/create/:problemID", controllers.PostCreateTest)
	r.GET("/tests/edit/:ID", controllers.GetEditTest)
	r.POST("/tests/edit/:ID", controllers.PostEditTest)
	r.POST("/tests/delete/:ID", controllers.PostDeleteTest)

	r.POST("/judge/:problemID", controllers.PostJudge)
	r.GET("/submissions/:codeID", controllers.GetSubmission)

	r.GET("/register", controllers.GetRegister)
	r.POST("/register", controllers.PostRegister)
	r.GET("/login", controllers.GetLogin)
	r.POST("/login", controllers.PostLogin)
	r.POST("/logout", controllers.PostLogout)
	r.GET("/account", controllers.GetAccount)

	r.Run()
}
