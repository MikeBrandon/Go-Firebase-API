package main

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	firebase "firebase.google.com/go"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var app *firebase.App
var firestoreClient *firestore.Client

type Task struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Members []*User `json:"members"`
}

type User struct {
	ID    string `json:"id"`
	UName string `json:"uName"`
}

func (u *User) updateUName(newName string) {
	u.UName = newName
}

func getUsers(c *gin.Context) {
	var users []interface{}
	iter := firestoreClient.Collection("users").Documents(context.Background())
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}

		users = append(users, doc.Data())
	}

	c.IndentedJSON(http.StatusOK, users)
}

func userById(c *gin.Context) {
	id, ok := c.Params.Get("id")
	if !ok {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "no id provided"})
		return
	}

	doc, err := firestoreClient.Collection("users").Doc(id).Get(context.Background())
	if err != nil {
		log.Fatalf("Failed getting user: %v", err)
		return
	}
	c.IndentedJSON(http.StatusOK, doc.Data())
}

func addUser(c *gin.Context) {
	newName, uOk := c.GetQuery("username")
	if !uOk {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "missing username parameter"})
		return
	}

	ref, _, err := firestoreClient.Collection("users").Add(context.Background(), map[string]interface{}{
		"uName": newName,
	})
	if err != nil {
		log.Fatalf("Failed adding user: %v", err)
		return
	}
	_, err = firestoreClient.Collection("users").Doc(ref.ID).Update(context.Background(), []firestore.Update{
		{
			Path:  "id",
			Value: ref.ID,
		},
	})
	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred: %s", err)
		return
	}

	c.IndentedJSON(http.StatusCreated, gin.H{"message": "User Added Succesfully"})
}

func patchUName(c *gin.Context) {
	id, iOk := c.GetQuery("id")
	username, uOk := c.GetQuery("username")
	if !iOk && !uOk {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no id and username provided"})
		return
	}
	if !iOk {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no id provided"})
		return
	}
	if !uOk {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no username provided"})
		return
	}

	_, err := firestoreClient.Collection("users").Doc(id).Update(context.Background(), []firestore.Update{
		{
			Path:  "uName",
			Value: username,
		},
	})
	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred: %s", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": "User Name Changed Successfully"})
}

func deleteUser(c *gin.Context) {
	id, ok := c.Params.Get("id")
	if !ok {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "no id provided"})
		return
	}

	_, err := firestoreClient.Collection("users").Doc(id).Delete(context.Background())
	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred: %s", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err})
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "User Deleted Successfully"})
}

func addTask(c *gin.Context) {
	var newTask Task

	if err := c.BindJSON(&newTask); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	ref, _, err := firestoreClient.Collection("tasks").Add(context.Background(), newTask)
	if err != nil {
		log.Fatalf("Failed adding user: %v", err)
		return
	}
	_, err = firestoreClient.Collection("tasks").Doc(ref.ID).Set(context.Background(), map[string]interface{}{
		"id":      ref.ID,
		"name":    newTask.Name,
		"members": []User{},
	})
	if err != nil {
		log.Fatalf("Failed adding user: %v", err)
		return
	}

	c.IndentedJSON(http.StatusCreated, gin.H{"message": "Task Added Succesfully"})
}

func addMember(c *gin.Context) {
	id, iOk := c.Params.Get("id")
	memberId, mOk := c.Params.Get("member")
	if !iOk && !mOk {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no task id and member id provided"})
		return
	}
	if !iOk {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no task id provided"})
		return
	}
	if !mOk {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no member id provided"})
		return
	}

	_, err := firestoreClient.Collection("tasks").Doc(id).Update(context.Background(), []firestore.Update{
		{
			Path:  "members",
			Value: memberId,
		},
	})
	if err != nil {
		log.Fatalf("Failed updating task: %v", err)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "Added Member"})
}

func getTasks(c *gin.Context) {
	var tasks []interface{}
	iter := firestoreClient.Collection("tasks").Documents(context.Background())
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}

		tasks = append(tasks, doc.Data())
	}

	c.IndentedJSON(http.StatusOK, tasks)
}

func init() {
	opt := option.WithCredentialsFile("firebase.json")
	app, err := firebase.NewApp(context.Background(), &firebase.Config{ProjectID: "go-test-api-b4c5d"}, opt)
	if err != nil {
		log.Fatalf("error initializing firebase app client: %v\n", err)
	}

	firestoreClient, err = app.Firestore(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	return
}

func main() {
	router := gin.Default()

	router.GET("/users", getUsers)
	router.GET("/tasks", getTasks)
	router.POST("/users", addUser)
	router.GET("/users/:id", userById)
	router.PATCH("/users", patchUName)
	router.DELETE("/users/:id", deleteUser)
	router.POST("/tasks", addTask)
	router.POST("/tasks/:id/:member", addMember)

	router.Run("localhost:8080")
}
